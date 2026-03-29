#!/usr/bin/env python3
"""Stdlib unit tests for benchmark harness semantics (no Groq, no network)."""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path


def _load_harness():
    root = Path(__file__).resolve().parents[1]
    path = root / "run_groq_benchmark_v2.py"
    spec = importlib.util.spec_from_file_location("bench_v2", path)
    mod = importlib.util.module_from_spec(spec)
    loader = spec.loader
    assert loader is not None
    loader.exec_module(mod)
    return mod


class TestHarnessSemantics(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.b = _load_harness()

    def test_failure_reason_wrong_tool_family_filesystem(self):
        b = self.b
        self.assertFalse(b._used_expected_tool_family("filesystem_list", ["echo"]))
        fr = b._failure_reason(
            task_name="filesystem_list",
            tool_names=["echo"],
            output_preview="ok",
            answer_format_success=True,
            expected_tool_family=False,
        )
        self.assertEqual(fr, "unexpected_tool_family")

    def test_failure_reason_search_tools_task_ok(self):
        b = self.b
        fr = b._failure_reason(
            task_name="ambiguous_search",
            tool_names=["search_tools"],
            output_preview="AMBIG_OK 2",
            answer_format_success=True,
            expected_tool_family=True,
        )
        self.assertIsNone(fr)

def main() -> int:
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(TestHarnessSemantics)
    r = unittest.TextTestRunner(verbosity=1).run(suite)
    return 0 if r.wasSuccessful() else 1


if __name__ == "__main__":
    raise SystemExit(main())
