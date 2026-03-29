#!/usr/bin/env python3
"""
Benchmark Groq + PydanticAI against:

1. MCPJungle streamable HTTP directly
2. lazy-tool stdio (wrapping an indexed MCPJungle catalog)

Tailored for a local MCPJungle setup that exposes:
- everything
- filesystem (rooted at /tmp/lazy-tool-mcpjungle-fs)

What this harness does well:
- compares direct MCPJungle vs lazy-tool stdio
- tracks latency and token usage (when the provider returns it)
- distinguishes real API tool calls from fake text-mode "tool calls"
- records machine-readable failure reasons
- supports repeat runs for flaky-model detection
- can print raw message-part classes for debugging
- does not crash the whole benchmark if one mode fails

Requirements:
  - GROQ_API_KEY in environment
  - pydantic-ai installed
  - pydantic-ai[mcp] support installed
  - MCPJungle running locally
  - lazy-tool built locally if using --mode lazy or --mode both

Examples:

  python benchmark/run_groq_benchmark_v2.py --task no_tool --mode both
  python benchmark/run_groq_benchmark_v2.py --task filesystem_read --mode lazy --prepare-fs
  python benchmark/run_groq_benchmark_v2.py --task filesystem_read --mode both --prepare-fs --repeat 5
  python benchmark/run_groq_benchmark_v2.py --task filesystem_read --mode both --prepare-fs --debug-parts
"""

from __future__ import annotations

import argparse
import asyncio
import csv
import json
import os
import re
import sys
import time
from pathlib import Path
from typing import Any

_REPO_ROOT = Path(__file__).resolve().parent.parent
_BENCHMARK_DIR = Path(__file__).resolve().parent
_FS_ROOT = Path("/tmp/lazy-tool-mcpjungle-fs")

_INSTRUCTIONS_STRUCTURED_TOOLS_ONLY = (
    "When you need a tool, you MUST use the model's structured tool/function-calling channel. "
    "Never simulate tools by writing tags or pseudo-syntax in plain text "
    "(for example no <function=...>, no XML tool blocks, no markdown code fences that fake a call). "
    "If tools are available and required, use a real tool call instead of describing the call."
)

TASKS: dict[str, dict[str, Any]] = {
    "no_tool": {
        "prompt": "Reply with exactly the single word: ok. Do not call any tools unless required.",
        "expect_tool_calls": False,
        "expected_tool_hints": [],
        "description": "Baseline token cost when no tool should be needed.",
    },
    "search_tools_smoke": {
        "prompt": (
            "You must call the MCP tool named search_tools. "
            "Use query \"filesystem\" and limit 5. "
            "After the result returns, reply with exactly one line starting with SEARCH_OK "
            "followed by a short summary of how many hits were returned."
        ),
        "expect_tool_calls": True,
        "expected_tool_hints": ["search_tools"],
        "description": "Validate lazy-tool search_tools execution.",
    },
    "filesystem_list": {
        "prompt": (
            "Use available tools to inspect the filesystem under /tmp/lazy-tool-mcpjungle-fs. "
            "List the file names you find at the top level and reply with one short sentence. "
            "Do not make up files. Use tools."
        ),
        "expect_tool_calls": True,
        "expected_tool_hints": ["filesystem", "list", "directory", "read", "ls"],
        "description": "Checks whether the agent can use filesystem tools against the throwaway directory.",
    },
    "filesystem_read": {
        "prompt": (
            "Use available tools to read the file /tmp/lazy-tool-mcpjungle-fs/notes.txt "
            "and reply with exactly: FILE_OK <content>. "
            "Do not guess. Use tools."
        ),
        "expect_tool_calls": True,
        "expected_tool_hints": ["filesystem", "read", "file"],
        "description": "Checks whether the agent can read a known file from the test directory.",
    },
    "everything_echo": {
        "prompt": (
            "Use available tools from the 'everything' server to perform one simple tool call if possible, "
            "then reply with exactly one line starting with EVERYTHING_OK followed by a short summary."
        ),
        "expect_tool_calls": True,
        "expected_tool_hints": ["everything", "echo", "sample", "test"],
        "description": "Checks whether a tool from the everything server is callable.",
    },
    "ambiguous_search": {
        "prompt": (
            "Call search_tools with query \"echo\" and limit 3. "
            "Reply with one line AMBIG_OK and the number of results returned (digit only after AMBIG_OK)."
        ),
        "expect_tool_calls": True,
        "expected_tool_hints": ["search_tools"],
        "description": "Second happy path: discovery under an ambiguous query string.",
    },
    "search_tools_prompt": {
        "prompt": (
            "Call search_tools with query \"prompt\" and limit 8. "
            "Reply with one line SEARCH_PROMPT_OK and the approximate number of hits (a digit is enough)."
        ),
        "expect_tool_calls": True,
        "expected_tool_hints": ["search_tools"],
        "description": "Prompt/template discovery via search_tools (lazy-indexed catalog).",
    },
    "search_tools_resource": {
        "prompt": (
            "Call search_tools with query \"resource\" and limit 8. "
            "Reply with one line SEARCH_RES_OK and the approximate number of hits (a digit is enough)."
        ),
        "expect_tool_calls": True,
        "expected_tool_hints": ["search_tools"],
        "description": "Resource discovery via search_tools (lazy-indexed catalog).",
    },
}

_PSEUDO_TOOL_PATTERNS = [
    re.compile(r"<function=", re.IGNORECASE),
    re.compile(r"</function>", re.IGNORECASE),
    re.compile(r"assistant\s+to=functions?\.", re.IGNORECASE),
    re.compile(r"invoke_proxy_tool\s*\(", re.IGNORECASE),
    re.compile(r"search_tools\s*\(", re.IGNORECASE),
    re.compile(r"<\|tool", re.IGNORECASE),
    re.compile(r"\btool_call\b", re.IGNORECASE),
    re.compile(r'"name"\s*:\s*"[^"]+"\s*,\s*"arguments"\s*:', re.IGNORECASE),
]


def _default_lazy_binary() -> Path:
    return _REPO_ROOT / "bin" / "lazy-tool"


def _default_lazy_config() -> Path:
    return _BENCHMARK_DIR / "configs" / "mcpjungle-lazy-tool.yaml"


def _prepare_fs_fixture() -> None:
    _FS_ROOT.mkdir(parents=True, exist_ok=True)
    (_FS_ROOT / "notes.txt").write_text("hello from lazy-tool benchmark\n", encoding="utf-8")
    (_FS_ROOT / "todo.json").write_text(
        json.dumps({"tasks": ["benchmark", "search", "compare"]}, indent=2) + "\n",
        encoding="utf-8",
    )
    nested = _FS_ROOT / "nested"
    nested.mkdir(exist_ok=True)
    (nested / "info.txt").write_text("nested file\n", encoding="utf-8")


def _looks_like_pseudo_tool_output(text: str) -> bool:
    s = text.strip()
    if not s:
        return False
    return any(p.search(s) for p in _PSEUDO_TOOL_PATTERNS)


def _safe_int(v: Any) -> int:
    try:
        return int(v or 0)
    except Exception:
        return 0


def _tool_stats(result: object) -> dict[str, Any]:
    """Inspect PydanticAI run history for real structured tool calls."""
    from pydantic_ai.messages import ModelResponse, ToolCallPart

    names: list[str] = []
    for msg in result.all_messages():
        if isinstance(msg, ModelResponse):
            for part in getattr(msg, "parts", []):
                if isinstance(part, ToolCallPart):
                    names.append(part.tool_name)

    return {
        "tool_call_count": len(names),
        "tool_names": names,
    }


def _debug_message_parts(result: object) -> list[dict[str, Any]]:
    """Return a compact JSON-serializable view of message part classes."""
    from pydantic_ai.messages import ModelRequest, ModelResponse

    out: list[dict[str, Any]] = []
    for idx, msg in enumerate(result.all_messages()):
        entry: dict[str, Any] = {
            "index": idx,
            "message_class": type(msg).__name__,
            "parts": [],
        }
        if isinstance(msg, (ModelRequest, ModelResponse)):
            for part in getattr(msg, "parts", []):
                part_info = {
                    "part_class": type(part).__name__,
                }
                if hasattr(part, "tool_name"):
                    part_info["tool_name"] = getattr(part, "tool_name")
                if hasattr(part, "content"):
                    content = str(getattr(part, "content"))
                    part_info["content_preview"] = content[:200]
                entry["parts"].append(part_info)
        out.append(entry)
    return out


def _evaluate_answer_format(task_name: str, output_preview: str, strict: bool) -> bool:
    output_lower = output_preview.lower()
    if task_name == "no_tool":
        ok = output_lower.strip().startswith("ok")
        return ok and (not strict or len(output_lower.strip()) <= 8)
    if task_name == "search_tools_smoke":
        ok = "search_ok" in output_lower
        if strict and ok:
            return bool(re.search(r"search_ok\D", output_preview, re.IGNORECASE))
        return ok
    if task_name == "filesystem_read":
        return ("file_ok" in output_lower) and ("hello from lazy-tool benchmark" in output_lower)
    if task_name == "everything_echo":
        return "everything_ok" in output_lower
    if task_name == "filesystem_list":
        expected_names = {"notes.txt", "todo.json", "nested"}
        return any(name in output_preview for name in expected_names)
    if task_name == "ambiguous_search":
        ok = "ambig_ok" in output_lower
        if strict and ok:
            return bool(re.search(r"ambig_ok\D+\d", output_preview, re.IGNORECASE))
        return ok
    if task_name == "search_tools_prompt":
        ok = "search_prompt_ok" in output_lower
        if strict and ok:
            return bool(re.search(r"search_prompt_ok\D+\d", output_preview, re.IGNORECASE))
        return ok
    if task_name == "search_tools_resource":
        ok = "search_res_ok" in output_lower
        if strict and ok:
            return bool(re.search(r"search_res_ok\D+\d", output_preview, re.IGNORECASE))
        return ok
    return True


def _used_expected_tool_family(task_name: str, tool_names: list[str]) -> bool:
    cfg = TASKS[task_name]
    expect_tool_calls = bool(cfg["expect_tool_calls"])
    expected_tool_hints: list[str] = list(cfg["expected_tool_hints"])
    tool_names_lower = [t.lower() for t in tool_names]

    if not expected_tool_hints:
        return not expect_tool_calls

    return any(
        hint.lower() in tool_name
        for hint in expected_tool_hints
        for tool_name in tool_names_lower
    )


def _tool_execution_success(task_name: str, tool_names: list[str], pseudo_tool_text: bool) -> bool:
    expect_tool_calls = bool(TASKS[task_name]["expect_tool_calls"])
    if expect_tool_calls:
        return len(tool_names) > 0 and not pseudo_tool_text
    return len(tool_names) == 0 and not pseudo_tool_text


def _failure_reason(
    *,
    task_name: str,
    tool_names: list[str],
    output_preview: str,
    answer_format_success: bool,
    expected_tool_family: bool,
) -> str | None:
    expect_tool_calls = bool(TASKS[task_name]["expect_tool_calls"])
    pseudo_tool_text = _looks_like_pseudo_tool_output(output_preview)

    if expect_tool_calls and len(tool_names) == 0 and pseudo_tool_text:
        return "pseudo_tool_call_text"
    if expect_tool_calls and len(tool_names) == 0:
        return "no_tool_call"
    # Require tools to match TASKS["expected_tool_hints"] semantics even when names are lazy proxy tools.
    # (Previously proxy-only calls could pass filesystem-ish tasks with search_tools alone.)
    if expect_tool_calls and len(tool_names) > 0 and not expected_tool_family:
        return "unexpected_tool_family"
    if not answer_format_success:
        return "answer_format_failed"
    return None


def _exception_row(
    *,
    run_index: int,
    label: str,
    model: str,
    task_name: str,
    attachment_mode: str,
    discovery_mode: str,
    exc: Exception,
    jungle_url: str | None = None,
    lazy_config: str | None = None,
) -> dict[str, Any]:
    return {
        "run_index": run_index,
        "label": label,
        "model": model,
        "task": task_name,
        "input_tokens": 0,
        "output_tokens": 0,
        "total_tokens": 0,
        "usage_missing": True,
        "pseudo_tool_text": False,
        "duration_s": 0.0,
        "output_preview": "",
        "tool_execution_success": False,
        "answer_format_success": False,
        "used_expected_tool_family": False,
        "task_success": False,
        "failure_reason": "exception",
        "exception_type": type(exc).__name__,
        "exception_message": str(exc),
        "tool_call_count": 0,
        "tool_names": [],
        "attachment_mode": attachment_mode,
        "discovery_mode": discovery_mode,
        "jungle_url": jungle_url,
        "lazy_config": lazy_config,
    }


async def _run_agent(
    *,
    label: str,
    model: str,
    prompt: str,
    mcp_server: object,
    max_tokens: int,
    task_name: str,
    debug_parts: bool,
    run_index: int,
    strict_answers: bool,
) -> dict[str, Any]:
    from pydantic_ai import Agent

    expect_tools = bool(TASKS[task_name]["expect_tool_calls"])
    agent_kw: dict[str, Any] = {"model": model, "toolsets": [mcp_server]}
    if expect_tools:
        agent_kw["instructions"] = _INSTRUCTIONS_STRUCTURED_TOOLS_ONLY
        agent_kw["retries"] = 2

    agent = Agent(**agent_kw)

    started = time.perf_counter()
    async with agent:
        result = await agent.run(prompt, model_settings={"max_tokens": max_tokens})
    duration_s = round(time.perf_counter() - started, 3)

    usage = result.usage()
    inp = _safe_int(getattr(usage, "input_tokens", 0))
    out = _safe_int(getattr(usage, "output_tokens", 0))
    usage_missing = inp == 0 and out == 0

    stats = _tool_stats(result)
    output_preview = (str(result.output) if result.output is not None else "")[:800]
    pseudo_tool_text = _looks_like_pseudo_tool_output(output_preview)

    answer_format_success = _evaluate_answer_format(task_name, output_preview, strict_answers)
    expected_tool_family = _used_expected_tool_family(task_name, stats["tool_names"])
    tool_execution_success = _tool_execution_success(task_name, stats["tool_names"], pseudo_tool_text)
    failure_reason = _failure_reason(
        task_name=task_name,
        tool_names=stats["tool_names"],
        output_preview=output_preview,
        answer_format_success=answer_format_success,
        expected_tool_family=expected_tool_family,
    )
    task_success = (
        tool_execution_success
        and answer_format_success
        and not pseudo_tool_text
        and expected_tool_family
        and failure_reason is None
    )

    row = {
        "run_index": run_index,
        "label": label,
        "model": model,
        "task": task_name,
        "strict_answers": strict_answers,
        "input_tokens": inp,
        "output_tokens": out,
        "total_tokens": inp + out,
        "usage_missing": usage_missing,
        "pseudo_tool_text": pseudo_tool_text,
        "duration_s": duration_s,
        "output_preview": output_preview,
        "tool_execution_success": tool_execution_success,
        "answer_format_success": answer_format_success,
        "used_expected_tool_family": expected_tool_family,
        "task_success": task_success,
        "failure_reason": failure_reason,
        **stats,
    }

    if debug_parts:
        row["debug_message_parts"] = _debug_message_parts(result)

    return row


async def _run_jungle(
    *,
    jungle_url: str,
    model: str,
    prompt: str,
    max_tokens: int,
    task_name: str,
    debug_parts: bool,
    run_index: int,
    strict_answers: bool,
) -> dict[str, Any]:
    from pydantic_ai.mcp import MCPServerStreamableHTTP

    server = MCPServerStreamableHTTP(
        jungle_url,
        timeout=120.0,
        read_timeout=600.0,
    )
    row = await _run_agent(
        label="mcpjungle_direct",
        model=model,
        prompt=prompt,
        mcp_server=server,
        max_tokens=max_tokens,
        task_name=task_name,
        debug_parts=debug_parts,
        run_index=run_index,
        strict_answers=strict_answers,
    )
    row["attachment_mode"] = "direct_mcp"
    row["discovery_mode"] = "none"
    row["jungle_url"] = jungle_url
    row["lazy_config"] = None
    return row


async def _run_lazy(
    *,
    lazy_binary: Path,
    lazy_config: Path,
    workdir: Path,
    model: str,
    prompt: str,
    max_tokens: int,
    task_name: str,
    debug_parts: bool,
    run_index: int,
    strict_answers: bool,
) -> dict[str, Any]:
    from pydantic_ai.mcp import MCPServerStdio

    if not lazy_binary.is_file():
        raise FileNotFoundError(
            f"lazy-tool binary not found: {lazy_binary} (run make build from repo root)"
        )
    if not lazy_config.is_file():
        raise FileNotFoundError(f"config not found: {lazy_config}")

    env = {**os.environ, "LAZY_TOOL_CONFIG": str(lazy_config.resolve())}
    server = MCPServerStdio(
        str(lazy_binary.resolve()),
        ["serve"],
        env=env,
        cwd=workdir,
        timeout=120.0,
        read_timeout=600.0,
    )
    row = await _run_agent(
        label="lazy_tool_stdio",
        model=model,
        prompt=prompt,
        mcp_server=server,
        max_tokens=max_tokens,
        task_name=task_name,
        debug_parts=debug_parts,
        run_index=run_index,
        strict_answers=strict_answers,
    )
    row["attachment_mode"] = "lazy_stdio"
    row["discovery_mode"] = "static_search_proxy"
    row["jungle_url"] = None
    row["lazy_config"] = str(Path(lazy_config).resolve())
    return row


async def _async_main(args: argparse.Namespace) -> list[dict[str, Any]]:
    if not os.environ.get("GROQ_API_KEY"):
        print("GROQ_API_KEY is not set (https://console.groq.com/)", file=sys.stderr)
        raise SystemExit(1)

    if args.task not in TASKS:
        print(f"unknown task: {args.task}", file=sys.stderr)
        raise SystemExit(2)

    if args.prepare_fs:
        _prepare_fs_fixture()

    model = args.model
    if not model.startswith("groq:"):
        model = f"groq:{model}"

    task_cfg = TASKS[args.task]
    prompt = args.prompt if args.prompt else task_cfg["prompt"]
    max_tokens = args.max_tokens
    if task_cfg["expect_tool_calls"]:
        max_tokens = max(max_tokens, 512)

    modes: list[str] = []
    if args.mode in ("jungle", "both"):
        modes.append("jungle")
    if args.mode in ("lazy", "both"):
        modes.append("lazy")

    rows: list[dict[str, Any]] = []
    for run_index in range(1, args.repeat + 1):
        for mode in modes:
            try:
                if mode == "jungle":
                    row = await _run_jungle(
                        jungle_url=args.jungle_url,
                        model=model,
                        prompt=prompt,
                        max_tokens=max_tokens,
                        task_name=args.task,
                        debug_parts=args.debug_parts,
                        run_index=run_index,
                        strict_answers=args.strict_answers,
                    )
                else:
                    row = await _run_lazy(
                        lazy_binary=Path(args.lazy_binary),
                        lazy_config=Path(args.lazy_config),
                        workdir=Path(args.workdir),
                        model=model,
                        prompt=prompt,
                        max_tokens=max_tokens,
                        task_name=args.task,
                        debug_parts=args.debug_parts,
                        run_index=run_index,
                        strict_answers=args.strict_answers,
                    )
            except Exception as e:
                if mode == "jungle":
                    row = _exception_row(
                        run_index=run_index,
                        label="mcpjungle_direct",
                        model=model,
                        task_name=args.task,
                        attachment_mode="direct_mcp",
                        discovery_mode="none",
                        exc=e,
                        jungle_url=args.jungle_url,
                    )
                else:
                    row = _exception_row(
                        run_index=run_index,
                        label="lazy_tool_stdio",
                        model=model,
                        task_name=args.task,
                        attachment_mode="lazy_stdio",
                        discovery_mode="static_search_proxy",
                        exc=e,
                        lazy_config=str(Path(args.lazy_config).resolve()),
                    )
            rows.append(row)

    return rows


def _print_summary(rows: list[dict[str, Any]]) -> None:
    by_label: dict[str, list[dict[str, Any]]] = {}
    for row in rows:
        by_label.setdefault(row["label"], []).append(row)

    print("\nsummary:")
    for label, items in by_label.items():
        n = len(items)
        success_count = sum(1 for x in items if x["task_success"])
        pseudo_count = sum(1 for x in items if x["pseudo_tool_text"])
        missing_usage_count = sum(1 for x in items if x["usage_missing"])
        exception_count = sum(1 for x in items if x.get("failure_reason") == "exception")
        avg_duration = round(sum(float(x["duration_s"]) for x in items) / n, 3)
        avg_total_tokens = round(sum(int(x["total_tokens"]) for x in items) / n, 1)
        print(
            f"  {label}: runs={n} success={success_count}/{n} "
            f"pseudo_tool_text={pseudo_count}/{n} exceptions={exception_count}/{n} "
            f"usage_missing={missing_usage_count}/{n} avg_duration_s={avg_duration} "
            f"avg_total_tokens={avg_total_tokens}"
        )


def _print_human(rows: list[dict[str, Any]], args: argparse.Namespace) -> None:
    if not rows:
        print("no rows")
        return

    task_cfg = TASKS[args.task]
    print(f"task={args.task}")
    print(f"description={task_cfg['description']}")
    print(f"model={rows[0]['model']}")
    print(f"repeat={args.repeat}")
    if args.prompt:
        print(f"prompt_override={args.prompt!r}")

    for r in rows:
        print(
            f"\nrun={r['run_index']} {r['label']}: input_tokens={r['input_tokens']} "
            f"output_tokens={r['output_tokens']} total={r['total_tokens']} "
            f"duration_s={r['duration_s']}"
        )
        print(
            f"  success={r['task_success']} "
            f"tool_exec={r['tool_execution_success']} "
            f"answer_format={r['answer_format_success']} "
            f"expected_tool_family={r['used_expected_tool_family']} "
            f"attachment_mode={r['attachment_mode']} "
            f"discovery_mode={r['discovery_mode']}"
        )

        tc = r.get("tool_call_count", 0)
        tnames = r.get("tool_names") or []
        if tnames:
            print(f"  tool_calls: {tc} ({', '.join(tnames)})")
        else:
            print(f"  tool_calls: {tc} (none)")

        if r.get("failure_reason"):
            print(f"  failure_reason: {r['failure_reason']}")
        if r.get("exception_type"):
            print(f"  exception: {r['exception_type']}: {r.get('exception_message', '')}")
        if r.get("usage_missing"):
            print(
                "  warning: provider returned usage_tokens=0; run may still have hit the API, "
                "but token accounting is incomplete"
            )
        if r.get("pseudo_tool_text"):
            print(
                "  warning: output looks like a text-mode fake tool call; no real API tool call was captured"
            )
        if r.get("output_preview"):
            print(f"  output: {r['output_preview']!r}")

        if args.debug_parts and r.get("debug_message_parts"):
            print("  debug_message_parts:")
            for msg in r["debug_message_parts"]:
                print(f"    - {json.dumps(msg, ensure_ascii=False)}")

    if args.repeat > 1:
        _print_summary(rows)


def _write_jsonl(path: Path, rows: list[dict[str, Any]]) -> None:
    with path.open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")


def _write_csv(path: Path, rows: list[dict[str, Any]]) -> None:
    flat_rows: list[dict[str, Any]] = []
    for row in rows:
        flat = dict(row)
        if "tool_names" in flat and isinstance(flat["tool_names"], list):
            flat["tool_names"] = ",".join(flat["tool_names"])
        if "debug_message_parts" in flat:
            flat["debug_message_parts"] = json.dumps(flat["debug_message_parts"], ensure_ascii=False)
        flat_rows.append(flat)

    fieldnames: list[str] = []
    for row in flat_rows:
        for key in row.keys():
            if key not in fieldnames:
                fieldnames.append(key)

    with path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(flat_rows)


def main() -> None:
    p = argparse.ArgumentParser(
        description="Benchmark Groq + PydanticAI against MCPJungle direct vs lazy-tool stdio."
    )
    p.add_argument(
        "--mode",
        choices=("jungle", "lazy", "both"),
        default="both",
        help="Which attachment mode to benchmark (default: both)",
    )
    p.add_argument(
        "--task",
        choices=tuple(TASKS.keys()),
        default="no_tool",
        help="Predefined benchmark task",
    )
    p.add_argument(
        "--prompt",
        default="",
        help="Optional explicit prompt override (bypasses task default prompt)",
    )
    p.add_argument(
        "--jungle-url",
        default="http://127.0.0.1:8080/mcp",
        help="MCPJungle streamable HTTP endpoint",
    )
    p.add_argument(
        "--lazy-binary",
        default=str(_default_lazy_binary()),
        help="Path to lazy-tool executable",
    )
    p.add_argument(
        "--lazy-config",
        default=str(_default_lazy_config()),
        help="LAZY_TOOL_CONFIG YAML (indexed catalog)",
    )
    p.add_argument(
        "--workdir",
        default=str(_REPO_ROOT),
        help="Working directory for lazy-tool subprocess (repo root recommended)",
    )
    p.add_argument(
        "--model",
        default="llama-3.3-70b-versatile",
        help="Groq model id, with or without groq: prefix",
    )
    p.add_argument(
        "--max-tokens",
        type=int,
        default=256,
        help="Cap completion tokens per model step (tool tasks floor at 512)",
    )
    p.add_argument(
        "--repeat",
        type=int,
        default=1,
        help="Repeat each mode N times (default: 1)",
    )
    p.add_argument(
        "--prepare-fs",
        action="store_true",
        help="Create a small fixture under /tmp/lazy-tool-mcpjungle-fs before the run",
    )
    p.add_argument(
        "--debug-parts",
        action="store_true",
        help="Include raw message-part class info for debugging provider/tool behavior",
    )
    p.add_argument(
        "--json",
        action="store_true",
        help="Print one JSON array only",
    )
    p.add_argument(
        "--jsonl-out",
        default="",
        help="Optional path to write JSONL rows",
    )
    p.add_argument(
        "--csv-out",
        default="",
        help="Optional path to write CSV rows",
    )
    p.add_argument(
        "--strict-answers",
        action="store_true",
        help="Tighter answer_format checks (counts/delimiters) for benchmark tasks (part-3 rigor)",
    )
    args = p.parse_args()

    if args.repeat < 1:
        print("--repeat must be >= 1", file=sys.stderr)
        raise SystemExit(2)

    rows = asyncio.run(_async_main(args))

    if args.jsonl_out:
        _write_jsonl(Path(args.jsonl_out), rows)
    if args.csv_out:
        _write_csv(Path(args.csv_out), rows)

    if args.json:
        print(json.dumps(rows, indent=2, ensure_ascii=False))
        return

    _print_human(rows, args)


if __name__ == "__main__":
    main()