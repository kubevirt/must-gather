#!/usr/bin/env bash

set -ex

HCO_VERSION=1.6.0
MINOR_VER="${HCO_VERSION%.*}"
CMD=oc
NS=kubevirt-hyperconverged

# Create a catalog source
cat <<EOF | ${CMD} apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: hco-catalog-source
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: "quay.io/kubevirt/hyperconverged-cluster-index:${HCO_VERSION}"
  displayName: Kubevirt Hyperconverged Cluster Operator
  publisher: Kubevirt Project
EOF

# Create the kubevirt-hyperconverged namespace
cat <<EOF | ${CMD} apply -f -
apiVersion: v1
kind: Namespace
metadata:
    name: "${NS}"
EOF

# Create operator group
cat <<EOF | ${CMD} apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
    name: kubevirt-hyperconverged-group
    namespace: "${NS}"
EOF

# Create Subscription
cat <<EOF | ${CMD} apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
    name: hco-operatorhub
    namespace: "${NS}"
spec:
    source: hco-catalog-source
    sourceNamespace: openshift-marketplace
    name: community-kubevirt-hyperconverged
    channel: "${HCO_VERSION}"
EOF

sleep 60

# Wait for the deployments to be available
"${CMD}" wait -n "${NS}" deployment hco-operator --for=condition=Available --timeout="1080s"
"${CMD}" wait -n "${NS}"  deployment hco-webhook --for=condition=Available --timeout="1080s"

# Create the HyperConverged resource
curl "https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/release-${MINOR_VER}/deploy/hco.cr.yaml" | ${CMD} -n "${NS}" apply -f -

# Wait for the HyperConverged CR to be available
"${CMD}" wait -n "${NS}" HyperConverged "kubevirt-hyperconverged" --for=condition=Available --timeout="1080s"
