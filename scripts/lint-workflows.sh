#!/usr/bin/env bash
set -euo pipefail

actionlint -shellcheck=shellcheck
shellcheck scripts/*.sh
