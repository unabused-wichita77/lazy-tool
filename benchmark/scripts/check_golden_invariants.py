#!/usr/bin/env python3
"""Semantic checks for benchmark golden JSONL (complements validate_benchmark_jsonl schema checks).

Every non-empty row must declare a task that has a rule below—no silent skips.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any, Callable


def _err(msg: str) -> None:
    print(msg, file=sys.stderr)


def _expect_no_tools(r: dict[str, Any]) -> str | None:
    if r.get("tool_call_count") != 0:
        return f"no_tool expects tool_call_count==0, got {r.get('tool_call_count')}"
    return None


def _expect_search_tools_family(r: dict[str, Any]) -> str | None:
    if r.get("tool_call_count", 0) < 1 or r.get("task_success") is not True:
        return "search_tools family expects tool_call_count>=1 and task_success"
    return None


def _expect_tool_task(r: dict[str, Any]) -> str | None:
    if r.get("tool_call_count", 0) < 1 or r.get("task_success") is not True:
        return "tool task expects tool_call_count>=1 and task_success"
    return None


# Every task string appearing in benchmark/golden must be listed here.
RULES: dict[str, Callable[[dict[str, Any]], str | None]] = {
    "no_tool": _expect_no_tools,
    "search_tools_smoke": _expect_search_tools_family,
    "ambiguous_search": _expect_search_tools_family,
    "search_tools_prompt": _expect_search_tools_family,
    "search_tools_resource": _expect_search_tools_family,
    "everything_echo": _expect_tool_task,
    "filesystem_list": _expect_tool_task,
    "filesystem_read": _expect_tool_task,
}


def _check_row(row: Any, line_no: int) -> list[str]:
    errs: list[str] = []
    if not isinstance(row, dict):
        return [f"line {line_no}: expected object"]
    task = row.get("task")
    if not isinstance(task, str):
        return [f"line {line_no}: task must be str"]
    fn = RULES.get(task)
    if fn is None:
        return [
            f"line {line_no}: unknown task {task!r} — add a rule in RULES or drop the row from golden",
        ]
    msg = fn(row)
    if msg:
        errs.append(f"line {line_no} task={task}: {msg}")
    if row.get("task_success") is True and row.get("failure_reason") is not None:
        errs.append(
            f"line {line_no}: task_success true requires failure_reason null, got {row.get('failure_reason')!r}",
        )
    if row.get("task_success") is False and row.get("failure_reason") is None:
        errs.append(f"line {line_no}: task_success false requires failure_reason string (golden failure rows)")
    return errs


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("file", type=Path, help="JSONL golden file")
    args = ap.parse_args()
    path: Path = args.file
    if not path.is_file():
        _err(f"{path}: not a file")
        return 1
    text = path.read_text(encoding="utf-8")
    all_errs: list[str] = []
    any_row = False
    for i, line in enumerate(text.splitlines(), start=1):
        line = line.strip()
        if not line:
            continue
        any_row = True
        try:
            obj = json.loads(line)
        except json.JSONDecodeError as e:
            all_errs.append(f"{path}:{i}: {e}")
            continue
        all_errs.extend(_check_row(obj, i))
    if not any_row:
        _err(f"{path}: no JSONL rows")
        return 1
    if all_errs:
        for e in all_errs:
            _err(e)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
