#!/usr/bin/env python3
"""Report how Claude Code agents used op, from their session transcripts.

    python3 tools/agent_usage.py [--days 7] [--since 2026-11-17]

Rerun after a release and compare: each counter is a turn an agent wasted.
"comment reads denied by hook" counts denials from ~/.claude/hooks/confirm-outbound.py
and stays 0 where that hook isn't installed. "transcripts holding the API key" compares
against the key in ~/.oprc (or OP_API_KEY) and never prints it.
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
READS_ORC = re.compile(r"(?:~|\$HOME)/\.oprc(?![\w.])")
UNKNOWN = re.compile(r'Error: unknown (command|flag|shorthand flag)[: ]+"?([^"\s]+)')
BAD_ID = re.compile(r"invalid work package ID: (\S+)")
JIRA_MISS = re.compile(r'no work package found with JIRA ID "([^"]*)"')
SEARCH_WORDS_ERROR = re.compile(r"accepts 1 arg\(s\), received [2-9]|op search takes one value|quote the words: op search")
KEY_SHAPED = re.compile(r"-\d+$")
HOOK_DENIED = re.compile(r"PreToolUse:Bash hook error: This command posts or edits an OpenProject comment")
SHOW_PIPED = re.compile(r"\bop\s+show\s+[^|;\n]*\|\s*(head|grep|sed|tail)\b")
ID_LOOP = re.compile(r"for \w+ in [\d\s$]+;\s*do[^\n]*\bop\s+(show|comment)\b")
COUNTERS = ("raw API curl", "commands touching ~/.oprc", "comment reads denied by hook",
            "op show piped to head/grep/sed/tail", "for-loops over ticket ids",
            "op search given free text")


def api_key():
    key = os.environ.get("OP_API_KEY")
    if not key:
        try:
            with open(os.path.expanduser("~/.oprc")) as fh:
                match = re.search(r"^\s*api_key:\s*['\"]?([^'\"\s#]+)", fh.read(), re.M)
            key = match and match.group(1)
        except OSError:
            return None
    return key if key and len(key) >= 16 else None


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
    key = api_key()

    counts, subcommands, errors = Counter(), Counter(), Counter()
    sessions, leaked = set(), set()
    for path in glob.glob(f"{args.root}/*/*.jsonl") + glob.glob(f"{args.root}/*/*/subagents/*.jsonl"):
        if os.path.getmtime(path) < since:
            continue
        for ts, cmd, out in tool_calls(path):
            if ts and datetime.fromisoformat(ts.replace("Z", "+00:00")).timestamp() < since:
                continue
            if key and (key in cmd or key in out):
                leaked.add(path)
            used = OP.findall(cmd)
            raw_api = bool(RAW_API.search(cmd))
            reads_orc = bool(READS_ORC.search(cmd)) and "op setup" not in cmd
            if not used and not raw_api and not reads_orc:
                continue
            sessions.add(path)
            counts["calls"] += 1
            subcommands.update(set(used))
            counts["raw API curl"] += raw_api
            counts["commands touching ~/.oprc"] += reads_orc
            counts["comment reads denied by hook"] += bool(HOOK_DENIED.search(out)) and not re.search(
                r"comments?\s+\S+\s+[\"']", cmd)
            counts["op show piped to head/grep/sed/tail"] += len(SHOW_PIPED.findall(cmd))
            counts["for-loops over ticket ids"] += bool(ID_LOOP.search(cmd))
            counts["op search given free text"] += sum(
                not KEY_SHAPED.search(q) for q in JIRA_MISS.findall(out)) + (
                "search" in used and bool(SEARCH_WORDS_ERROR.search(out)))
            for kind, name in UNKNOWN.findall(out):
                errors[f"{kind} {name}"] += 1
            for word in BAD_ID.findall(out):
                errors[f"ticket ID {word}"] += 1

    print(f"{counts['calls']} op/API calls in {len(sessions)} transcripts since "
          f"{datetime.fromtimestamp(since).date()}")
    for name in COUNTERS:
        print(f"  {name:40} {counts[name]}")
    print(f"  {'transcripts holding the API key':40} {len(leaked) if key else 'n/a (no key found)'}")
    print(f"  {'unknown command/flag or bad ID errors':40} {sum(errors.values())}")
    for name, n in errors.most_common(10):
        print(f"      {n:4}  {name}")
    print("  subcommands:", ", ".join(f"{k} {v}" for k, v in subcommands.most_common(12)))


if __name__ == "__main__":
    main()
