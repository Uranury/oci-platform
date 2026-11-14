#!/usr/bin/env bash
# Runs site.yml and fails unless every host reports changed=0.
# Usage: scripts/check-idempotence.sh [extra ansible-playbook args]
#        e.g. scripts/check-idempotence.sh --ask-vault-pass
set -uo pipefail
cd "$(dirname "$0")/.."

log=$(mktemp)
trap 'rm -f "$log"' EXIT

ansible-playbook site.yml "$@" | tee "$log"
status=${PIPESTATUS[0]}
[[ $status -eq 0 ]] || { echo "playbook failed (exit $status)" >&2; exit "$status"; }

echo
if grep -E '^\S+\s+:\s+ok=' "$log" | grep -vE '\schanged=0\s' ; then
    echo "NOT idempotent: the hosts above reported changes" >&2
    exit 1
fi
echo "idempotent: every host reports changed=0"
