#!/bin/bash
BRIDGE="$(cd "$(dirname "$0")" && pwd)"

rm -f "$BRIDGE/request.txt" "$BRIDGE/result.txt" "$BRIDGE/status.txt"

echo "op-bridge: waiting for requests at $BRIDGE ..."
while true; do
  if [ -f "$BRIDGE/request.txt" ]; then
    cmd=$(cat "$BRIDGE/request.txt")
    rm "$BRIDGE/request.txt"
    echo "op-bridge: running: op $cmd"
    eval op $cmd > "$BRIDGE/result.txt" 2>&1
    echo "done" > "$BRIDGE/status.txt"
  fi
  sleep 0.2
done
