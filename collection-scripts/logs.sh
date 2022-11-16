#!/bin/bash -x

DIR_NAME=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${DIR_NAME}/common.sh"
check_command

NAMESPACE_FILE=/var/run/secrets/kubernetes.io/serviceaccount/namespace

# if running in a pod
if [[ -f ${NAMESPACE_FILE} ]]; then
  POD=$(oc status | grep "^pod" | sed -E "s|pod/([^ ]+).*|\1|")
  oc logs --timestamps=true -n "$(cat ${NAMESPACE_FILE})" "${POD}" -c gather > "${BASE_COLLECTION_PATH}/must-gather.log"
fi
