#!/usr/bin/env bash
set -euo pipefail

PATTERN="${KONTEKST_ARG_PATTERN:?pattern argument is required}"
SEARCH_PATH="${KONTEKST_ARG_PATH:-.}"
INCLUDE="${KONTEKST_ARG_INCLUDE:-}"

if command -v rg &>/dev/null; then
    args=(--line-number --no-heading "$PATTERN" "$SEARCH_PATH")
    if [ -n "$INCLUDE" ]; then
        args=(--glob "$INCLUDE" "${args[@]}")
    fi
    rg "${args[@]}" | head -200
else
    args=(-rn "$PATTERN" "$SEARCH_PATH")
    if [ -n "$INCLUDE" ]; then
        args=(--include="$INCLUDE" "${args[@]}")
    fi
    grep "${args[@]}" | head -200
fi
