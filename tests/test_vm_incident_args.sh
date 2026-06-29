#!/bin/bash
#
# Bash tests for --vm-incident argument validation.
# These run locally without a cluster — they only test that the gather
# entrypoint validates inputs correctly before attempting any collection.
#
# Usage: bash tests/test_vm_incident_args.sh

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
GATHER="${SCRIPT_DIR}/../collection-scripts/gather"

PASS=0
FAIL=0

assert_error() {
    local description="$1"
    local expected_msg="$2"
    shift 2

    if output=$("$@" 2>&1); then
        echo "FAIL: ${description} — expected non-zero exit, got success"
        echo "  output: ${output}"
        FAIL=$(( FAIL + 1 ))
        return
    fi

    if echo "${output}" | grep -q "${expected_msg}"; then
        echo "PASS: ${description}"
        PASS=$(( PASS + 1 ))
    else
        echo "FAIL: ${description} — expected '${expected_msg}' in output"
        echo "  output: ${output}"
        FAIL=$(( FAIL + 1 ))
    fi
}

# --vm-incident without --incident-time
assert_error \
    "--vm-incident without --incident-time should fail" \
    "requires --incident-time" \
    env USR_BIN_GATHER=1 NS=testns VM=testvm bash "${GATHER}" --vm-incident

# --vm-incident without NS
assert_error \
    "--vm-incident without NS should fail" \
    "requires the NS environment variable" \
    env USR_BIN_GATHER=1 VM=testvm bash "${GATHER}" --vm-incident --incident-time=2026-06-06T06:06:00Z

# --vm-incident without VM
assert_error \
    "--vm-incident without VM should fail" \
    "requires the VM environment variable" \
    env USR_BIN_GATHER=1 NS=testns bash "${GATHER}" --vm-incident --incident-time=2026-06-06T06:06:00Z

# --vm-incident without NS and VM
assert_error \
    "--vm-incident without NS and VM should fail" \
    "requires the NS environment variable" \
    env USR_BIN_GATHER=1 bash "${GATHER}" --vm-incident --incident-time=2026-06-06T06:06:00Z

echo ""
echo "========================================"
echo "Results: ${PASS} passed, ${FAIL} failed"
echo "========================================"

if [[ ${FAIL} -gt 0 ]]; then
    exit 1
fi
