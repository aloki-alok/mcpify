# Contributing to mcpify

Thanks for helping out. mcpify is a single Go binary; the loop is small.

## Build and test

```
go build ./...
go test ./...
go vet ./...
```

`go run . <spec>` runs your working copy against a spec. The `examples/`
directory has one to point it at:

```
go run . ls examples/petstore.yaml
```

## Layout

- `internal/openapi` parses the spec and inlines local `$ref`s.
- `internal/tool` turns each operation into a tool and builds its HTTP request.
- `internal/server` wires the tools into an MCP server and proxies calls.
- `internal/cli` holds the commands (`serve`, `ls`) and flag parsing.
- `DESIGN.md` is the why: the problem, the non-goals, and the shape. Read it
  before a large change so a feature lands where it belongs.

## Pull requests

- Keep each PR to one concern, with a clear title that says the intent.
- Add a test for behavior you change, especially in `internal/openapi` and
  `internal/tool` where a wrong result would build the wrong request.
- `go test ./...` and `go vet ./...` must pass. CI runs both on every PR.
- Match the surrounding style. No new dependency without a reason in the PR.

## Reporting bugs

Open an issue with the version (`mcpify version`), the spec (or a minimal
reproduction of it), and what tool call behaved unexpectedly. Redact any tokens.
