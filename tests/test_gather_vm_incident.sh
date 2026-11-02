#!/bin/bash
#
# Unit tests for incident-era virt-launcher readiness and virsh retry handling.

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
COLLECTOR="${SCRIPT_DIR}/../collection-scripts/gather_vm_incident"
TEST_DIR=$(mktemp -d /tmp/gather-vm-incident.XXXXXX)
CALLS="${TEST_DIR}/oc.calls"

trap 'rm -rf "${TEST_DIR}"' EXIT

PASS=0
FAIL=0

pass() { echo "PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); }

assert_equals() {
    local description=$1 expected=$2 actual=$3
    if [[ "${actual}" == "${expected}" ]]; then
        pass "${description}"
    else
        fail "${description} — expected '${expected}', got '${actual}'"
    fi
}

assert_file_contains() {
    local description=$1 expected=$2 file=$3
    if grep -Fq "${expected}" "${file}"; then
        pass "${description}"
    else
        fail "${description} — '${expected}' was not found in ${file}"
    fi
}

count_calls() {
    local action=$1
    grep -c "^${action} " "${CALLS}" || true
}

timeout() { shift; "$@"; }
sleep() { :; }

scenario=""
oc() {
    local action=$1
    shift
    echo "${action} $*" >> "${CALLS}"

    case "${scenario}:${action}" in
        ready-after-retry:wait)
            [[ $(count_calls wait) -gt 1 ]]
            ;;
        never-ready:wait)
            return 1
            ;;
        virsh-retry:exec)
            if (( $(count_calls exec) == 1 )); then
                echo "temporary libvirt failure" >&2
                return 1
            fi
            echo "incident virsh data"
            ;;
        virsh-empty:exec)
            return 0
            ;;
        *)
            return 0
            ;;
    esac
}

export USR_BIN_GATHER=1
export GATHER_VM_INCIDENT_LIBRARY=true
export OC_BIN=oc
export INCIDENT_VIRSH_RETRY_ATTEMPTS=3
export INCIDENT_VIRSH_RETRY_DELAY=0
# shellcheck disable=SC1090 # The collector path is derived from this test's location.
source "${COLLECTOR}"
collected() { :; }
skipped() { :; }

: > "${CALLS}"
scenario=ready-after-retry
if wait_for_ready_incident_launcher virt-launcher-incident testns; then
    pass "incident-era launcher becomes ready after retry"
else
    fail "incident-era launcher becomes ready after retry"
fi
assert_equals "incident launcher readiness is retried" "2" "$(count_calls wait)"

: > "${CALLS}"
scenario=never-ready
if wait_for_ready_incident_launcher virt-launcher-incident testns; then
    fail "replaced incident-era launcher is rejected"
else
    pass "replaced incident-era launcher is rejected"
fi
assert_equals "replaced incident launcher uses bounded retries" "3" "$(count_calls wait)"

: > "${CALLS}"
scenario=never-ready
incident_path="${TEST_DIR}/incident-vm"
mkdir -p "${incident_path}"
if collect_incident_virsh_data virt-launcher-incident testns vm "${incident_path}"; then
    fail "unready incident-era launcher skips live VM state"
else
    pass "unready incident-era launcher skips live VM state"
fi
assert_equals "unready incident-era launcher does not run virsh" "0" "$(count_calls exec)"
assert_file_contains "unready incident-era launcher records a diagnostic" "did not become Ready" \
    "${incident_path}/virt-launcher-incident.virsh-collection.error.txt"

: > "${CALLS}"
scenario=virsh-retry
retry_output="${TEST_DIR}/retry.txt"
if collect_incident_virsh_output "${retry_output}" virt-launcher-incident -n testns -c compute -- virsh -r dumpxml testns_vm; then
    pass "incident virsh command succeeds after retry"
else
    fail "incident virsh command succeeds after retry"
fi
assert_equals "incident virsh command retries once" "2" "$(count_calls exec)"
assert_equals "incident virsh output is preserved" "incident virsh data" "$(<"${retry_output}")"

: > "${CALLS}"
scenario=virsh-empty
empty_output="${TEST_DIR}/empty.txt"
if collect_incident_virsh_output "${empty_output}" virt-launcher-incident -n testns -c compute -- virsh -r dumpxml testns_vm; then
    fail "empty incident virsh output reports failure"
else
    pass "empty incident virsh output reports failure"
fi
assert_equals "empty incident virsh output uses all retries" "3" "$(count_calls exec)"
assert_file_contains "empty incident output gets a diagnostic" "returned no output" "${empty_output}"

echo ""
echo "========================================"
echo "Results: ${PASS} passed, ${FAIL} failed"
echo "========================================"

if (( FAIL > 0 )); then
    exit 1
fi
