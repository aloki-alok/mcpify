# AGENTS.md

mcpify turns an OpenAPI spec (file or URL) into a working MCP server in one command. Go. Each operation becomes one tool that proxies the real HTTP call.

## Commands

- Build: `go build ./...`
- Test: `go test ./...`
- Vet: `go vet ./...`
- Run against a spec: `go run . ls examples/petstore.yaml` (lists tools)
- Serve over stdio: `go run . serve examples/petstore.yaml` (`mcpify help` lists commands)

## Layout

- `main.go`: entry point.
- `internal/openapi/`: parses the spec and inlines local `$ref`s. Includes Swagger 2.0 conversion.
- `internal/tool/`: turns each operation into a tool and builds its HTTP request. Also naming and filtering.
- `internal/server/`: wires the tools into an MCP server and proxies calls.
- `internal/cli/`: commands (`serve`, `ls`, `upgrade`, plus connect helpers) and flag parsing.
- `internal/upgrade/`, `internal/ui/`: self-update and terminal output.
- `examples/petstore.yaml`: a small spec for manual runs.
- `DESIGN.md`: the problem, non-goals and shape. Read it before a large change.
- `.goreleaser.yaml`, `install.sh`: release and install. Release runs on tags.

## Rules

- One operation maps to one tool, 1:1. No merged tools, synthetic workflows or response reshaping.
- Auth is pass-through. Forward the headers the user supplies. Do not store tokens or add OAuth or rate limiting.
- It is not a code generator. It runs the proxy and re-reads the spec on every start.
- Keep dependencies few. Give a reason in the PR for any new one.
- Add a test for changed behavior, mainly in `internal/openapi` and `internal/tool`. A wrong result there builds the wrong request.
- Redact tokens in issues, tests and examples.
- Keep each PR to one concern.

## Before you open a PR

CI runs these on every PR.

1. `go vet ./...`
2. `go test ./...`
3. `go build ./...`
