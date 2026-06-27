# mcpify — design

`mcpify <openapi-spec>` turns any OpenAPI 3.x spec into a working MCP server in
one command. No code generation, no config: point it at a spec (file or URL) and
every API operation becomes an MCP tool that proxies the real HTTP call.

## 1. Problem

MCP is everywhere in 2026, but standing up a server still means writing one by
hand: wiring the SDK, declaring tool schemas, mapping each tool to an HTTP call,
handling auth. Meanwhile almost every API a developer would want an agent to use
already ships an OpenAPI spec. That spec already describes the operations, their
parameters, and their shapes. The work of turning it into an MCP server is
mechanical and should be a single command. Today it is a project.

## 2. Non-goals

- **Not a code generator.** It does not emit a server you then edit and own. It
  runs the proxy directly. (A future `mcpify eject` could print Go, out of scope.)
- **Not an API gateway / auth broker.** Auth is pass-through: headers you supply
  are forwarded upstream. No token storage, no OAuth dance, no rate limiting.
- **Not a Swagger 2.0 / GraphQL / gRPC tool.** OpenAPI 3.0 and 3.1 only for v1;
  anything else fails with a clear message, not a half-parse.
- **Not a transformer of API semantics.** One operation maps to one tool, 1:1.
  No merging, no synthetic workflows, no response reshaping beyond formatting.

## 3. Alternatives considered

- **Generate server source code** (à la openapi-generator). Lost because: it
  makes the user own and maintain generated code, and drifts from the spec the
  moment the API changes. A live proxy re-reads the spec every start.
- **A hosted converter / SaaS.** Lost because: it puts a third party in the path
  of every API call (latency, trust, cost) for a job that is pure local glue.
  The whole appeal is a static binary you run next to the API.
- **Wrap an existing heavy OpenAPI library** (kin-openapi). Lost because: it
  pulls a large dependency tree and full $ref/validation machinery we do not
  need. We parse the subset required to build tool schemas and HTTP requests.

## 4. Chosen approach

```
  openapi spec (file|url)
        │  parse + resolve local $refs
        ▼
  []openapi.Operation ───► internal/tool ───► []ToolDef {name, desc, jsonschema, request-plan}
        │                                            │
        ▼                                            ▼  per tools/call
  mcp.Server (official go-sdk) ◄── AddTool ── ToolHandler ──► build *http.Request ──► upstream API
        │                                                              │
        ├── stdio transport (default, for Claude Desktop/clients)      ▼
        └── streamable-HTTP (--http)                          format response → CallToolResult
```

Trust boundary: the agent/MCP client talks to mcpify locally; mcpify is the only
thing that talks to the upstream API, with the operator's injected headers. The
spec is data, never executed.

One operation → one tool. The tool's input schema is an object merging the
operation's path/query/header parameters and (when the request body is a JSON
object) the body's fields, flattened to top-level args for a friendlier LLM
surface. The handler routes each arg back to path / query / header / body when
building the upstream request.

## 5. Tradeoffs

- Live proxy means mcpify must be running and reachable; there is no standalone
  artifact to hand off. Acceptable: that is true of every MCP server.
- Flattening body fields to top-level args is friendlier but risks name
  collisions with parameters; resolved by giving parameters precedence and
  keeping non-object bodies under a single `body` arg.
- Hand-rolled OpenAPI subset means some exotic specs (deep remote $refs,
  `oneOf`/`allOf` composition) degrade to a permissive schema rather than a
  precise one. We prefer a working tool with a loose schema over a hard failure.

## 6. Success / failure metrics

- Success: `mcpify <spec>` exposes one tool per operation, and a real MCP client
  (mctop) can list and successfully call a tool that reaches the upstream API.
- Failure signal: a spec that should work produces zero tools, or a tool call
  builds a malformed upstream request (wrong path/method/body).

## 7. Rollout

Local-first. Build and verify on this box against a self-contained example API.
Public release (repo, GoReleaser, installer) is a separate, explicitly-approved
step — not done autonomously.

## 8. Rollback

Pre-release there is nothing shipped to roll back; the project is a local dir
under git. If a release is later cut and is bad: delete the GitHub release/tag,
the prior tag's binaries remain installable; no server-side state exists.

## 9. Monitoring / runbook

N/A while local. mcpify itself logs upstream request failures to stderr; the MCP
client surfaces tool errors (IsError) inline.

## 10. Bus factor

Everything is in this repo: DESIGN.md (why), small `internal/*` packages each
with one job (parse, build tool, serve), and tests. Someone else could pick it
up from the package boundaries alone.
