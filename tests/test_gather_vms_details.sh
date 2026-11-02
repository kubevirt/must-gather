#!/bin/bash
#
# Unit tests for the transient virt-launcher and virsh failure handling in
# gather_vms_details. They mock oc so they run locally without a cluster.

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
COLLECTOR="${SCRIPT_DIR}/../collection-scripts/gather_vms_details"
TEST_DIR=$(mktemp -d /tmp/gather-vms-details.XXXXXX)
CALLS="${TEST_DIR}/oc.calls"

trap 'rm -rf "${TEST_DIR}"' EXIT

PASS=0
FAIL=0

pass() {
  echo "PASS: $1"
  PASS=$((PASS + 1))
}

fail() {
  echo "FAIL: $1"
  FAIL=$((FAIL + 1))
}

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

timeout() {
  shift
  "$@"
}

sleep() {
  :
}

scenario=""
oc() {
  local action=$1
  shift
  echo "${action} $*" >> "${CALLS}"

  case "${scenario}:${action}" in
    replacement:get)
      if [[ $(count_calls get) == 1 ]]; then
        echo "virt-launcher-old vm"
      else
        echo "virt-launcher-new vm"
      fi
      ;;
    replacement:wait)
      if [[ "$*" == *"pod/virt-launcher-old"* ]]; then
        return 1
      fi
      echo "pod/virt-launcher-new condition met"
      ;;
    retry:exec)
      if (( $(count_calls exec) < 3 )); then
        echo "temporary libvirt failure" >&2
        return 1
      fi
      echo "virsh data"
      ;;
    permanent-failure:exec)
      echo "persistent libvirt failure" >&2
      return 1
      ;;
    empty-output:exec)
      return 0
      ;;
    never-ready:get)
      echo "virt-launcher-old vm"
      ;;
    never-ready:wait)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
}

export USR_BIN_GATHER=1
export GATHER_VMS_DETAILS_LIBRARY=true
export OC_BIN=oc
export VIRSH_RETRY_ATTEMPTS=3
export VIRSH_RETRY_DELAY=0
# shellcheck disable=SC1090 # The collector path is derived from this test's location.
source "${COLLECTOR}"

: > "${CALLS}"
scenario=replacement
pod=$(wait_for_ready_virt_launcher testns vm)
assert_equals "replaced launcher pod is selected" "virt-launcher-new" "${pod}"
assert_equals "replacement path waits twice" "2" "$(count_calls wait)"

: > "${CALLS}"
scenario=retry
retry_output="${TEST_DIR}/retry.txt"
if collect_virsh_output "${retry_output}" pod -n testns -c compute -- virsh -r capabilities; then
  pass "transient virsh failure succeeds after retry"
else
  fail "transient virsh failure succeeds after retry"
fi
assert_equals "virsh is attempted three times" "3" "$(count_calls exec)"
assert_equals "successful retry output is preserved" "virsh data" "$(<"${retry_output}")"

: > "${CALLS}"
scenario=permanent-failure
failure_output="${TEST_DIR}/failure.txt"
if collect_virsh_output "${failure_output}" pod -n testns -c compute -- virsh -r capabilities; then
  fail "persistent virsh failure reports failure"
else
  pass "persistent virsh failure reports failure"
fi
assert_equals "persistent virsh failure uses all retries" "3" "$(count_calls exec)"
assert_file_contains "persistent failure is recorded in output" "persistent libvirt failure" "${failure_output}"

: > "${CALLS}"
scenario=empty-output
empty_output="${TEST_DIR}/empty.txt"
if collect_virsh_output "${empty_output}" pod -n testns -c compute -- virsh -r capabilities; then
  fail "empty virsh output reports failure"
else
  pass "empty virsh output reports failure"
fi
assert_equals "empty output uses all retries" "3" "$(count_calls exec)"
assert_file_contains "empty output gets a diagnostic" "returned no output" "${empty_output}"

: > "${CALLS}"
scenario=empty-output
domblklist="${TEST_DIR}/domblklist.txt"
blockjob_output="${TEST_DIR}/blockjob.txt"
printf 'Type Device Target Source\n-------------------------\n' > "${domblklist}"
collect_blockjob_data "${blockjob_output}" "${domblklist}" pod testns vm
assert_equals "no blockjob command runs without disks" "0" "$(count_calls exec)"
assert_file_contains "empty disk list produces a blockjob explanation" "No disks were returned" "${blockjob_output}"

: > "${CALLS}"
scenario=never-ready
BASE_COLLECTION_PATH="${TEST_DIR}/archive"
gather_vm_info testns virt-launcher-old vm
assert_equals "no virsh command runs for an unready launcher" "0" "$(count_calls exec)"
assert_file_contains "unready launcher produces an error artifact" "no ready virt-launcher pod found" \
  "${BASE_COLLECTION_PATH}/namespaces/testns/vms/vm/virt-launcher-old.virsh-collection.error.txt"

echo ""
echo "========================================"
echo "Results: ${PASS} passed, ${FAIL} failed"
echo "========================================"

if (( FAIL > 0 )); then
  exit 1
fi
