#!/usr/bin/env python3
"""Validate benchmark output lines match the schema from run_groq_benchmark_v2 (CI / golden checks, no API keys)."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def _check_row(obj: Any, line_no: int) -> list[str]:
    errs: list[str] = []
    if not isinstance(obj, dict):
        return [f"line {line_no}: expected JSON object, got {type(obj).__name__}"]
    required = {
        "run_index": int,
        "label": str,
        "model": str,
        "task": str,
        "strict_answers": bool,
        "input_tokens": int,
        "output_tokens": int,
        "total_tokens": int,
        "usage_missing": bool,
        "pseudo_tool_text": bool,
        "duration_s": (int, float),
        "output_preview": str,
        "tool_execution_success": bool,
        "answer_format_success": bool,
        "used_expected_tool_family": bool,
        "task_success": bool,
        "tool_call_count": int,
        "tool_names": list,
    }
    for key, typ in required.items():
        if key not in obj:
            errs.append(f"line {line_no}: missing key {key!r}")
            continue
        val = obj[key]
        if isinstance(typ, tuple):
            if not isinstance(val, typ):
                errs.append(f"line {line_no}: {key!r} must be one of {typ}, got {type(val).__name__}")
        elif not isinstance(val, typ):
            errs.append(f"line {line_no}: {key!r} must be {typ.__name__}, got {type(val).__name__}")
    if "failure_reason" in obj and obj["failure_reason"] is not None and not isinstance(obj["failure_reason"], str):
        errs.append(f"line {line_no}: failure_reason must be str or null")
    return errs


def validate_file(path: Path) -> list[str]:
    text = path.read_text(encoding="utf-8")
    errs: list[str] = []
    if not text.strip():
        return [f"{path}: empty file"]
    for i, line in enumerate(text.splitlines(), start=1):
        line = line.strip()
        if not line:
            continue
        try:
            obj = json.loads(line)
        except json.JSONDecodeError as e:
            errs.append(f"{path}:{i}: invalid JSON: {e}")
            continue
        errs.extend(_check_row(obj, i))
    return errs


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("files", nargs="+", type=Path, help="JSONL files to validate")
    args = ap.parse_args()
    all_errs: list[str] = []
    for f in args.files:
        if not f.is_file():
            all_errs.append(f"{f}: not a file")
            continue
        all_errs.extend(validate_file(f))
    if all_errs:
        for e in all_errs:
            print(e, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
