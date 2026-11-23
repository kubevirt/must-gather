#!/bin/bash
# Mode-gate checks for coredump collection. Usage: bash tests/test_coredump_gate.sh
set -euo pipefail
S="$(cd "$(dirname "$0")/../collection-scripts" && pwd)"

grep -q 'coredumpctl list' "$S/gather_nodes"
grep -q 'COLLECT_COREDUMPS' "$S/gather_nodes"
grep -q -- '--core_dumps)' "$S/gather"
grep -q 'COLLECT_COREDUMPS=1' "$S/gather"
grep -q 'COLLECT_COREDUMPS=1' "$S/gather_vm_incident"

echo OK
