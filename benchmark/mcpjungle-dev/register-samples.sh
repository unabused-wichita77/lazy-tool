#!/usr/bin/env bash
# Register sample MCPs into a local MCPJungle registry.
# Run AFTER: docker compose up (MCPJungle). See benchmark/README.md
#
# mcpjungle requires either --conf <file.json> (stdio or HTTP) OR --name + --url (HTTP only).
# A bare "mcpjungle register" fails with: supply a configuration file or set --name
# set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p /tmp/lazy-tool-mcpjungle-fs

if ! command -v mcpjungle >/dev/null 2>&1; then
  echo "Install mcpjungle CLI: https://github.com/mcpjungle/MCPJungle" >&2
  exit 1
fi

register() {
  local f="$1"
  if [[ ! -f "$f" ]]; then
    echo "Missing config file: $f" >&2
    exit 1
  fi
  echo "==> mcpjungle register --conf $f"
  mcpjungle register --conf "$f"
}

register "$DIR/everything.json"
register "$DIR/filesystem.json"

if command -v uvx >/dev/null 2>&1; then
  register "$DIR/time.json"
else
  echo "Skipping time.json (uvx not found)"
fi

echo "Done. Tool count (stderr merged): mcpjungle list tools 2>&1 | grep -cE '^[[:space:]]*[0-9]+\\.'"
