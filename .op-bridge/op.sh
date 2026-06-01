#!/bin/bash
BRIDGE="$(cd "$(dirname "$0")" && pwd)"

if [ -z "$*" ]; then
  echo "usage: op.sh <command> [args...]"
  exit 1
fi

rm -f "$BRIDGE/result.txt" "$BRIDGE/status.txt"
printf '%q ' "$@" > "$BRIDGE/request.txt"

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
