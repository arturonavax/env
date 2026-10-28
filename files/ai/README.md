# AI Configuration & Tools

This directory stores transferable configurations, harnesses, optimization proxies, and automation tools for AI coding agents and engineers.

## Directory Structure

* `aitools/`: Complete multi-model AI productivity CLI suite (`aitools`, `config.default.json`, README). Manages live quota telemetry, smart rolling window wakeups, RTC hardware auto-wake with suspension, and Conventional Git Commit generation (`gcm`).
* `rtk/`: Rust Token Killer proxy configuration (`config.toml`). Compresses noisy terminal outputs by up to 90% before returning to LLM agent context.
* `mcp/`: Configuration files, schemas, and server declarations for the Model Context Protocol (MCP).
* `prompts/`: System instructions, prompt templates, engineering guidelines (`AGENTS.md`), and persona specifications.
* `harnesses/`: Configuration and evaluation scripts for agent evaluation harnesses and orchestrators.

## Synchronization

* When `./install.sh ai` or `./install.sh all` is executed, these configurations, tools, and environments are automatically initialized and synchronized.
* To synchronize AI tooling independently:
  ```sh
  ./utils/sync_config.sh devtools
  ```
