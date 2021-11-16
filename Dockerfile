FROM quay.io/openshift/origin-must-gather:4.9.0 as builder

FROM quay.io/centos/centos:8

ENV INSTALLATION_NAMESPACE kubevirt-hyperconverged

# For gathering data from nodes
RUN dnf update -y && dnf install iproute tcpdump pciutils util-linux nftables rsync -y && dnf clean all

COPY --from=builder /usr/bin/oc /usr/bin/oc

# Copy all collection scripts to /usr/bin
COPY collection-scripts/* /usr/bin/

# Copy node-gather resources to /etc
COPY node-gather /etc/

ENTRYPOINT /usr/bin/gather
