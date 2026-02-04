---
name: Codex CLI
description: Interaction guide for the Codex CLI tool
---

# Codex CLI

The `codex` CLI is available for executing agentic tasks, code reviews, and managing the Codex agent.

## Usage

```bash
codex [OPTIONS] [PROMPT]
codex [OPTIONS] <COMMAND> [ARGS]
```

## Key Commands

- `codex exec [prompt]`: Run Codex non-interactively.
- `codex review`: Run a code review.
- `codex login`: Manage login.
- `codex mcp-server`: Run the Codex MCP server.

## Common Options

- `-m, --model <MODEL>`: Specify the model (e.g., `go`, `claude-3-5-sonnet`).
- `--oss`: Use local open source model.
- `-s, --sandbox <MODE>`: Set sandbox mode (`read-only`, `workspace-write`, `danger-full-access`).
- `--web-search`: Enable web search.

## Examples

**Run a one-off task:**
```bash
codex exec "Refactor this file"
```

**Run with specific model:**
```bash
codex exec -m go "Explain this code"
```

**Start interactive session:**
```bash
codex
```
