# mcpify

Turn any OpenAPI 3.x spec into a working MCP server in one command.

Point it at a spec (a file or a URL) and every API operation becomes an MCP
tool. There is no code to generate and nothing to wire up: mcpify reads the
spec, exposes one tool per operation, and proxies each tool call to the real
API.

```
mcpify ls openapi.yaml        # preview the tools a spec exposes
mcpify openapi.yaml           # serve it as an MCP server over stdio
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

Serve a spec over stdio, the transport MCP clients use when they spawn a
server:

```
mcpify ./petstore.yaml
```

Serve over HTTP instead:

```
mcpify --http :8080 ./petstore.yaml
```

Forward auth and override the upstream base URL:

```
mcpify --base https://api.example.com -H "Authorization: Bearer $TOKEN" spec.json
```

Expose only read operations (GET and HEAD):

```
mcpify --read-only spec.yaml
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
| `--http <addr>` | serve over HTTP at `addr` instead of stdio |
| `--read-only` | expose only GET and HEAD operations |
| `--timeout <dur>` | upstream request timeout (default `30s`) |

## Scope

OpenAPI 3.0 and 3.1, in JSON or YAML. One operation maps to one tool. Request
bodies are JSON. Templated server URLs resolve from their variable defaults, or
override the base with `--base`. Auth is pass-through: the headers you supply are
forwarded upstream. Swagger 2.0 is not supported.

## License

MIT
