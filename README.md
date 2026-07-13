# mcpify

Turn any OpenAPI 3.x spec into a working MCP server in one command.

Point it at a spec (a file or a URL) and every API operation becomes an MCP
tool. There is no code to generate and nothing to wire up: mcpify reads the
spec, exposes one tool per operation, and proxies each tool call to the real
API.

![mcpify listing the tools an OpenAPI spec exposes](./docs/demo.gif)

```
mcpify ls openapi.yaml        # preview the tools a spec exposes
mcpify openapi.yaml           # serve it as an MCP server
```

## Install

Homebrew:

```
brew install aloki-alok/tap/mcpify
```

Or the install script:

```
curl -fsSL https://raw.githubusercontent.com/aloki-alok/mcpify/main/install.sh | sh
```

Or with Go (1.26+):

```
go install github.com/aloki-alok/mcpify@latest
```

Or grab a prebuilt binary from the [releases page](https://github.com/aloki-alok/mcpify/releases), or build from source:

```
git clone https://github.com/aloki-alok/mcpify
cd mcpify
go build -o mcpify .
```

## Use

Preview what a spec turns into, without starting anything:

```
mcpify ls https://petstore3.swagger.io/api/v3/openapi.json
```

Serve a spec:

```
mcpify ./petstore.yaml
```

In a terminal this opens a short menu: run a local server and get a connect
URL (the default, just press enter), print a client config to paste, or list
the tools. When an MCP client spawns mcpify over a pipe, it serves stdio
directly, so the same command works inside a client config. Pass `--stdio` to
force stdio serving in a terminal too.

Serve over HTTP at a fixed address:

```
mcpify --http :8080 ./petstore.yaml
```

The MCP endpoint is `http://localhost:8080/mcp`. The root URL serves a short
plain-text page describing the server, for anyone who opens it in a browser.

Forward auth and override the upstream base URL:

```
mcpify --base https://api.example.com -H "Authorization: Bearer $TOKEN" spec.json
```

Expose only read operations (GET and HEAD):

```
mcpify --read-only spec.yaml
```

Cut down a large spec to the tools you actually want. `--include` keeps
operations whose tool name, operationId, or path match; without a `*` this is
a case insensitive substring check, with one it is a `path.Match` glob (whole
string, case sensitive, `*` does not cross a `/`) against each of the three.
`--tag` keeps operations carrying that OpenAPI tag (case insensitive, exact
match). Both are repeatable, and an operation is kept if it matches any
`--include` or any `--tag`:

```
mcpify --tag pet --tag store spec.yaml
mcpify --include "*Pet*" spec.yaml
mcpify --include order spec.yaml
```

Update to the latest release in place:

```
mcpify upgrade
```

### Plug into an MCP client

Any client that launches a server over stdio works. For example:

```json
{
  "mcpServers": {
    "petstore": {
      "command": "mcpify",
      "args": ["--base", "https://api.example.com", "/path/to/openapi.json"]
    }
  }
}
```

## How arguments map

Each operation's path, query, header, and cookie parameters become tool
arguments. A JSON request body whose schema is an object is flattened so its
fields are top-level arguments too; a parameter wins if it shares a name. Other
body shapes (an array, a scalar) are taken as a single `body` argument. mcpify
routes each argument back to the right place when it builds the upstream
request.

## Options

| Flag | Meaning |
| --- | --- |
| `--base <url>` | upstream base URL, overriding the spec's `servers` |
| `-H, --header "Name: value"` | header sent on every upstream request (repeatable) |
| `--http <addr>` | serve over HTTP at `addr` (MCP endpoint at `/mcp`) |
| `--stdio` | serve stdio even in a terminal, skipping the menu |
| `--read-only` | expose only GET and HEAD operations |
| `--include <pattern>` | keep only operations whose tool name, operationId, or path match (repeatable) |
| `--tag <tag>` | keep only operations with this OpenAPI tag (repeatable) |
| `--timeout <dur>` | upstream request timeout (default `30s`) |

## Scope

OpenAPI 3.0 and 3.1, in JSON or YAML. One operation maps to one tool. Request
bodies are JSON. Templated server URLs resolve from their variable defaults, or
override the base with `--base`. Auth is pass-through: the headers you supply are
forwarded upstream. Swagger 2.0 is not supported.

## License

MIT
