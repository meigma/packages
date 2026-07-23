#!/usr/bin/env bash
set -euo pipefail

actionlint -shellcheck=shellcheck
shellcheck scripts/*.sh spikes/phase0/*.sh
python3 scripts/validate_publish_event_test.py
