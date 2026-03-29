#!/usr/bin/env python3
"""Fail CI if harness TASKS, golden JSONL `task` values, and invariant RULES drift apart."""

from __future__ import annotations

import importlib.util
import json
import re
import sys
from pathlib import Path

_REPO = Path(__file__).resolve().parents[2]
_HARNESS = _REPO / "benchmark" / "run_groq_benchmark_v2.py"
_GOLDEN = _REPO / "benchmark" / "golden" / "sample_benchmark_rows.jsonl"
_INVARIANTS = _REPO / "benchmark" / "scripts" / "check_golden_invariants.py"


def _harness_tasks() -> set[str]:
    text = _HARNESS.read_text(encoding="utf-8")
    marker = "TASKS: dict[str, dict[str, Any]] = {"
    start = text.index(marker) + len(marker) - 1  # position at '{'
    depth = 0
    for j in range(start, len(text)):
        c = text[j]
        if c == "{":
            depth += 1
        elif c == "}":
            depth -= 1
            if depth == 0:
                block = text[start : j + 1]
                break
    else:
        raise RuntimeError("unclosed TASKS dict")
    keys = set(re.findall(r'^\s{4}"([a-z][a-z0-9_]*)":\s*\{', block, re.MULTILINE))
    if not keys:
        raise RuntimeError("no TASKS keys parsed")
    return keys


def _golden_tasks() -> set[str]:
    out: set[str] = set()
    for line in _GOLDEN.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line:
            continue
        row = json.loads(line)
        t = row.get("task")
        if not isinstance(t, str):
            raise RuntimeError(f"golden row missing string task: {line[:80]}")
        out.add(t)
    return out


def _rules_tasks() -> set[str]:
    spec = importlib.util.spec_from_file_location("golden_invariants", _INVARIANTS)
    if spec is None or spec.loader is None:
        raise RuntimeError("cannot load check_golden_invariants.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    rules = getattr(mod, "RULES", None)
    if not isinstance(rules, dict):
        raise RuntimeError("RULES dict missing")
    return set(rules.keys())


def main() -> int:
    h = _harness_tasks()
    g = _golden_tasks()
    r = _rules_tasks()
    errs: list[str] = []
    if h != g:
        errs.append(f"harness != golden: only_harness={sorted(h - g)} only_golden={sorted(g - h)}")
    if h != r:
        errs.append(f"harness != RULES: only_harness={sorted(h - r)} only_rules={sorted(r - h)}")
    if errs:
        for e in errs:
            print(e, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
