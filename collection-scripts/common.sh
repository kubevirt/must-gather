#!/bin/bash -x

DIR_NAME=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
export BASE_COLLECTION_PATH="${BASE_COLLECTION_PATH:-/must-gather}"
export PROS=${PROS:-5}
export INSTALLATION_NAMESPACE=${INSTALLATION_NAMESPACE:-kubevirt-hyperconverged}

function check_command {
    if [[ -z "$USR_BIN_GATHER" ]]; then
        echo "This script should not be directly executed." 1>&2
        echo "Please check \"${DIR_NAME}/gather --help\" for execution options." 1>&2
        exit 1
    fi
}

