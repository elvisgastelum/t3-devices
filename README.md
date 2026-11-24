# t3-devices

Work is tracked in Kaneo, reached through the `kaneo` MCP server.

## Setup

Copy `.mcp.json.example` to `.mcp.json` and set the `kaneo` server URL to your own Kaneo instance. `.mcp.json` is git-ignored, so your server URL stays local. `CLAUDE.md`, `.claude/` and `.plans/` are git-ignored too and stay local.

## Build

Install the pinned toolchain with `mise install`, then:

- `mise run build` builds `bin/t3-devices` with version info embedded.
- `mise run test` runs the unit tests.
- `mise run check` runs the full gate: tracked-file guard, `go vet`, `staticcheck`, `gofmt` and tests.
- `mise run install` builds and installs to `~/.local/bin/t3-devices`.
