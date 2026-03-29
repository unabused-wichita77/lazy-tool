#!/usr/bin/env python3
"""CI: offline_benchmark_snapshot.json must match sample_benchmark_rows.jsonl row-for-row."""

from __future__ import annotations

import json
import sys
from pathlib import Path

_REPO = Path(__file__).resolve().parents[2]
_JSONL = _REPO / "benchmark/golden/sample_benchmark_rows.jsonl"
_SNAP = _REPO / "benchmark/golden/offline_benchmark_snapshot.json"


def _load_jsonl() -> list[dict]:
    rows: list[dict] = []
    for line in _JSONL.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line:
            continue
        rows.append(json.loads(line))
    return rows


def main() -> int:
    if not _SNAP.is_file():
        print(f"missing {_SNAP}; run: make sync-benchmark-offline-snapshot", file=sys.stderr)
        return 1
    from_jsonl = _load_jsonl()
    from_snap = json.loads(_SNAP.read_text(encoding="utf-8"))
    if not isinstance(from_snap, list):
        print("offline_benchmark_snapshot.json must be a JSON array", file=sys.stderr)
        return 1
    if from_jsonl != from_snap:
        print(
            "golden JSONL and offline_benchmark_snapshot.json differ "
            f"(jsonl={len(from_jsonl)} snap={len(from_snap)}); "
            "run: make sync-benchmark-offline-snapshot",
            file=sys.stderr,
        )
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
