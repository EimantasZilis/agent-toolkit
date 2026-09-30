#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_TOOLKIT_HOME="${AGENT_TOOLKIT_HOME:-$(cd "$SCRIPT_DIR/.." && pwd)}"

source "$SCRIPT_DIR/config.sh"
source "$SCRIPT_DIR/guidance.sh"
source "$SCRIPT_DIR/paths.sh"
source "$SCRIPT_DIR/skills.sh"
