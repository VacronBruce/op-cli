#!/bin/bash
BRIDGE="$(cd "$(dirname "$0")" && pwd)"

if [ -z "$*" ]; then
  echo "usage: op.sh <command> [args...]"
  exit 1
fi

rm -f "$BRIDGE/result.txt" "$BRIDGE/status.txt"

# For attach commands, copy files to the shared bridge directory
# so the host watcher can access them.
args=()
subcmd="$1"
if [ "$subcmd" = "attach" ]; then
  args+=("$1")  # "attach"
  shift
  args+=("$1")  # work package ID
  shift
  # Remaining args: file paths and flags
  for arg in "$@"; do
    if [[ "$arg" == --* ]]; then
      # Flags pass through unchanged
      args+=("$arg")
    elif [ -f "$arg" ]; then
      # Copy file to bridge directory, preserving the filename
      fname="$(basename "$arg")"
      cp "$arg" "$BRIDGE/$fname"
      args+=("__BRIDGE_FILE__$fname")
    else
      args+=("$arg")
    fi
  done
else
  args=("$@")
fi

printf '%q ' "${args[@]}" > "$BRIDGE/request.txt"

for i in $(seq 50); do
  [ -f "$BRIDGE/status.txt" ] && break
  sleep 0.2
done

if [ ! -f "$BRIDGE/status.txt" ]; then
  echo "error: timeout waiting for host-watcher" >&2
  exit 1
fi

cat "$BRIDGE/result.txt"
rm -f "$BRIDGE/status.txt"
