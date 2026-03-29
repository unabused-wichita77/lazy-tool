#!/usr/bin/env python3
"""Rewrite benchmark/golden/offline_benchmark_snapshot.json from sample_benchmark_rows.jsonl.

Run after editing the JSONL golden so the offline array snapshot stays in sync.
"""

from __future__ import annotations

import json
from pathlib import Path

_REPO = Path(__file__).resolve().parents[2]
_JSONL = _REPO / "benchmark/golden/sample_benchmark_rows.jsonl"
_OUT = _REPO / "benchmark/golden/offline_benchmark_snapshot.json"


def main() -> None:
    rows: list[dict] = []
    for line in _JSONL.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line:
            continue
        rows.append(json.loads(line))
    _OUT.write_text(json.dumps(rows, indent=2, sort_keys=False) + "\n", encoding="utf-8")
    print(f"wrote {_OUT} ({len(rows)} rows)")


if __name__ == "__main__":
    main()
