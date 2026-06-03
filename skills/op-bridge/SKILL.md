---
name: op-bridge
description: How to use the op-bridge to run op CLI commands from inside Docker (Claude Code container)
---

# op-bridge

Claude Code runs inside Docker. The `op` CLI and its config (`~/.oprc`) live on the host. The **op-bridge** lets Claude Code call `op` on the host via shared files.

## Architecture

```
Docker container (Claude Code)          Host machine
┌─────────────────────────┐            ┌──────────────────────────┐
│  .op-bridge/op.sh       │            │  host-watcher.sh (bg)    │
│  writes request.txt  ───┼──────────► │  reads request.txt       │
│  waits for status.txt   │            │  runs: eval op <cmd>     │
│  reads result.txt    ◄──┼────────────│  writes result.txt       │
└─────────────────────────┘            └──────────────────────────┘
         shared volume: $(pwd)/.op-bridge/
```

## How to Call op from Inside Docker

```bash
bash /path/to/.op-bridge/op.sh <op command and args>

# Examples:
bash /Users/yuying/work/op-cli/.op-bridge/op.sh show 81455
bash /Users/yuying/work/op-cli/.op-bridge/op.sh board -p app --sprint="App_06/02/2026"
bash /Users/yuying/work/op-cli/.op-bridge/op.sh update 81455 --status=closed
bash /Users/yuying/work/op-cli/.op-bridge/op.sh create bug "[iOS][EET] Title" --priority=P1
```

## File Attachments

When using `op create --attach` or `op attach`, files must be accessible to the host. The bridge handles this automatically by copying the file into the bridge directory before sending the request.

```bash
# op.sh detects file paths, copies them to bridge dir, and uses __BRIDGE_FILE__ markers
bash .op-bridge/op.sh attach 81455 "/path/to/screenshot.png"

# For create with --attach, same mechanism applies
bash .op-bridge/op.sh create bug "Title" --attach="/path/to/file.png"
```

The host-watcher resolves `__BRIDGE_FILE__<name>` back to the real path before running `op`, then cleans up the copied file.

## Bridge Files

| File | Purpose |
|------|---------|
| `request.txt` | op.sh writes shell-escaped command args here |
| `status.txt` | host-watcher writes `done` when command finishes |
| `result.txt` | host-watcher writes stdout+stderr of the op command |

## Limitations

- **No parallelism**: the bridge is single-threaded (one request at a time). Running two `op.sh` calls concurrently will cause race conditions.
- **op only**: the bridge only supports `op` commands. Tools like `glab` are not available inside Docker — run them on the host directly, or use `! <command>` in the Claude Code prompt to run on the host.
- **10-second timeout**: op.sh polls for `status.txt` up to 50 times × 0.2s = 10 seconds. Slow commands will appear to timeout.

## Troubleshooting

**"error: timeout waiting for host-watcher"**
→ host-watcher is not running. On the host, check if `run_cc.sh` was used to start Docker. If you started Docker manually, run `bash .op-bridge/host-watcher.sh &` on the host in a separate terminal.

**Stale bridge files causing wrong results**
→ The `run_cc.sh` script clears stale files on startup. If running manually, delete `request.txt`, `result.txt`, `status.txt` from the bridge directory before starting.

**Command runs but returns wrong sprint**
→ `op board` and `op sprint progress` use `FindActiveSprint`, which returns the first "open" version in the project. Specify the sprint explicitly:
```bash
bash .op-bridge/op.sh board -p app --sprint="App_06/02/2026"
```
