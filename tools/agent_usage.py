#!/usr/bin/env python3
"""Report how Claude Code agents used op, from their session transcripts.

    python3 tools/agent_usage.py [--days 7] [--since 2026-11-17]

Rerun after a release and compare: each counter is a turn an agent wasted.
"""
import argparse
import glob
import json
import os
import re
import time
from collections import Counter
from datetime import datetime, timezone

OP = re.compile(r"(?:^|[\s;&|(`]|\$\()(?:rtk proxy\s+)?op\s+([a-z][\w-]*)")
RAW_API = re.compile(r"curl\b[^\n]*/api/v3/")
UNKNOWN = re.compile(r'Error: unknown (command|flag|shorthand flag)[: ]+"?([^"\s]+)')
HOOK_DENIED = re.compile(r"PreToolUse:Bash hook error: This command posts or edits an OpenProject comment")
SHOW_PIPED = re.compile(r"\bop\s+show\s+[^|;\n]*\|\s*(head|grep|sed|tail)\b")
ID_LOOP = re.compile(r"for \w+ in [\d\s$]+;\s*do[^\n]*\bop\s+(show|comment)\b")


def tool_calls(path):
    pending = {}
    with open(path, errors="replace") as fh:
        for line in fh:
            try:
                event = json.loads(line)
            except ValueError:
                continue
            content = (event.get("message") or {}).get("content")
            if not isinstance(content, list):
                continue
            for item in content:
                if not isinstance(item, dict):
                    continue
                if item.get("type") == "tool_use" and item.get("name") == "Bash":
                    pending[item["id"]] = (event.get("timestamp") or "", (item.get("input") or {}).get("command", ""))
                elif item.get("type") == "tool_result" and item.get("tool_use_id") in pending:
                    out = item.get("content")
                    if isinstance(out, list):
                        out = "\n".join(x.get("text", "") for x in out if isinstance(x, dict))
                    yield (*pending.pop(item["tool_use_id"]), out or "")


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--days", type=int, default=7)
    parser.add_argument("--since", help="ISO date; overrides --days")
    parser.add_argument("--root", default=os.path.expanduser("~/.claude/projects"))
    args = parser.parse_args()
    since = (datetime.fromisoformat(args.since).replace(tzinfo=timezone.utc).timestamp()
             if args.since else time.time() - args.days * 86400)

    counts, subcommands, unknown = Counter(), Counter(), Counter()
    sessions = set()
    for path in glob.glob(f"{args.root}/*/*.jsonl") + glob.glob(f"{args.root}/*/*/subagents/*.jsonl"):
        if os.path.getmtime(path) < since:
            continue
        for ts, cmd, out in tool_calls(path):
            if ts and datetime.fromisoformat(ts.replace("Z", "+00:00")).timestamp() < since:
                continue
            used = OP.findall(cmd)
            raw_api = bool(RAW_API.search(cmd))
            if not used and not raw_api:
                continue
            sessions.add(path)
            counts["calls"] += 1
            subcommands.update(set(used))
            counts["raw API curl"] += raw_api
            counts["comment reads denied by hook"] += bool(HOOK_DENIED.search(out)) and not re.search(
                r"comment\s+\S+\s+[\"']", cmd)
            counts["op show piped to head/grep/sed/tail"] += len(SHOW_PIPED.findall(cmd))
            counts["for-loops over ticket ids"] += bool(ID_LOOP.search(cmd))
            for kind, name in UNKNOWN.findall(out):
                unknown[f"{kind} {name}"] += 1

    print(f"{counts['calls']} op/API calls in {len(sessions)} transcripts since "
          f"{datetime.fromtimestamp(since).date()}")
    for key in ("raw API curl", "comment reads denied by hook", "op show piped to head/grep/sed/tail",
                "for-loops over ticket ids"):
        print(f"  {key:40} {counts[key]}")
    print(f"  {'unknown command/flag errors':40} {sum(unknown.values())}")
    for name, n in unknown.most_common(10):
        print(f"      {n:4}  {name}")
    print("  subcommands:", ", ".join(f"{k} {v}" for k, v in subcommands.most_common(12)))


if __name__ == "__main__":
    main()
