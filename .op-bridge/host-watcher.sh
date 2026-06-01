#!/bin/bash
BRIDGE="$(cd "$(dirname "$0")" && pwd)"

rm -f "$BRIDGE/request.txt" "$BRIDGE/result.txt" "$BRIDGE/status.txt"

echo "op-bridge: waiting for requests at $BRIDGE ..."
while true; do
  if [ -f "$BRIDGE/request.txt" ]; then
    cmd=$(cat "$BRIDGE/request.txt")
    rm "$BRIDGE/request.txt"

    # Rewrite __BRIDGE_FILE__ markers to actual bridge directory paths
    # and clean up transferred files after the command runs
    bridge_files=()
    resolved_cmd="$cmd"
    while [[ "$resolved_cmd" == *"__BRIDGE_FILE__"* ]]; do
      # Extract the filename after the marker
      fname="${resolved_cmd#*__BRIDGE_FILE__}"
      fname="${fname%%[[:space:]]*}"
      fname="${fname%%\'*}"
      bridge_path="$BRIDGE/$fname"
      resolved_cmd="${resolved_cmd/__BRIDGE_FILE__$fname/$bridge_path}"
      bridge_files+=("$bridge_path")
    done

    echo "op-bridge: running: op $resolved_cmd"
    eval op $resolved_cmd > "$BRIDGE/result.txt" 2>&1

    # Clean up transferred files
    for f in "${bridge_files[@]}"; do
      rm -f "$f"
    done

    echo "done" > "$BRIDGE/status.txt"
  fi
  sleep 0.2
done
