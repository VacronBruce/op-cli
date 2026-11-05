# op-bridge: running `op` from inside Docker

Read this only when the working directory has a `.op-bridge/` folder, which means
Claude Code runs in a Docker container while the `op` binary and `~/.oprc` live on the
host. The bridge relays each `op` call through files in that shared folder.

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

## Calling `op`

Replace `op` with `bash .op-bridge/op.sh` (path relative to the project root that holds
`.op-bridge/`):

```bash
bash .op-bridge/op.sh show 81455
bash .op-bridge/op.sh board -p app --sprint="App_06/02/2026"
bash .op-bridge/op.sh update 81455 --status=closed
bash .op-bridge/op.sh create bug "[iOS][EET] Title" --priority=P1
```

## Attachments

Files must be reachable from the host. `op.sh attach <id> <file>...` copies each file
into the bridge folder and sends a `__BRIDGE_FILE__<name>` marker; the host watcher
resolves it to the real path, runs `op`, and deletes the copy.

```bash
bash .op-bridge/op.sh attach 81455 "/path/to/screenshot.png"
```

Only the `attach` subcommand gets this file copying. `create --attach=<path>` is passed
through unchanged and the host can't see a container path, so create the ticket
without `--attach`, then `op.sh attach <new-id> <file>`.

## Bridge files

| File | Purpose |
|---|---|
| `request.txt` | op.sh writes the shell-escaped command args here |
| `status.txt` | host-watcher writes `done` when the command finishes |
| `result.txt` | host-watcher writes the command's stdout+stderr |

## Limits

- One request at a time: two concurrent `op.sh` calls race on the same files.
- `op` only. `glab` and other host tools aren't available inside Docker; run them on
  the host, or with `! <command>` in the Claude Code prompt.
- About 10 seconds per call: op.sh polls `status.txt` 50 times at 0.2 s, so a slow
  command looks like a timeout.

## Troubleshooting

- **`error: timeout waiting for host-watcher`** — the watcher isn't running. `run_cc.sh`
  starts it with Docker; if Docker was started by hand, run
  `bash .op-bridge/host-watcher.sh &` on the host.
- **Stale results** — `run_cc.sh` clears old bridge files on startup. When running by
  hand, delete `request.txt`, `result.txt` and `status.txt` from the bridge folder first.
- **Wrong sprint** — without `--sprint`, `op board` and `op sprint progress` use the
  first open version whose dates cover today, falling back to the first open version.
  Pass the sprint explicitly: `bash .op-bridge/op.sh board -p app --sprint="App_06/02/2026"`.
