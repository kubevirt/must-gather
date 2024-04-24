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

get_log_collection_args() {
	# validation of MUST_GATHER_SINCE and MUST_GATHER_SINCE_TIME is done by the
	# caller (oc adm must-gather) so it's safe to use the values as they are.
	log_collection_args=""

	if [ -n "${MUST_GATHER_SINCE:-}" ]; then
		log_collection_args=--since="${MUST_GATHER_SINCE}"
	fi
	if [ -n "${MUST_GATHER_SINCE_TIME:-}" ]; then
		log_collection_args=--since-time="${MUST_GATHER_SINCE_TIME}"
	fi

	# oc adm node-logs `--since` parameter is not the same as oc adm inspect `--since`.
	# it takes a simplified duration in the form of '(+|-)[0-9]+(s|m|h|d)' or
	# an ISO formatted time. since MUST_GATHER_SINCE and MUST_GATHER_SINCE_TIME
	# are formatted differently, we re-format them so they can be used
	# transparently by node-logs invocations.
	node_log_collection_args=""

	if [ -n "${MUST_GATHER_SINCE:-}" ]; then
		# shellcheck disable=SC2001
		since=$(echo "${MUST_GATHER_SINCE:-}" | sed 's/\([0-9]*[dhms]\).*/\1/')
		node_log_collection_args=--since="-${since}"
	fi
	if [ -n "${MUST_GATHER_SINCE_TIME:-}" ]; then
	  # shellcheck disable=SC2001
		iso_time=$(echo "${MUST_GATHER_SINCE_TIME}" | sed 's/T/ /; s/Z//')
		node_log_collection_args=--since="${iso_time}"
	fi
	export log_collection_args
	export node_log_collection_args
}
