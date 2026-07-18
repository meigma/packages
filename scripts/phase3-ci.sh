#!/usr/bin/env bash
set -euo pipefail

actionlint -shellcheck=shellcheck
shellcheck scripts/*.sh spikes/phase0/*.sh
python3 scripts/check_workflow_policy.py

echo 'Phase 3 workflow and shell policy proof passed.'
