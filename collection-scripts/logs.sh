#!/bin/bash -x

export BASE_COLLECTION_PATH="${BASE_COLLECTION_PATH:-/must-gather}"
NAMESPACE_FILE=/var/run/secrets/kubernetes.io/serviceaccount/namespace

# if running in a pod
if [[ -f ${NAMESPACE_FILE} ]]; then
  POD=$(oc status | grep "^pod" | sed -E "s|pod/([^ ]+).*|\1|")
  oc logs --timestamps=true -n "$(cat ${NAMESPACE_FILE})" "${POD}" -c gather > "${BASE_COLLECTION_PATH}/must-gather.log"
fi
