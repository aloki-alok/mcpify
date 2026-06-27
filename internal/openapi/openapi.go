// Package openapi parses the subset of an OpenAPI 3.x document that mcpify needs
// to expose each operation as an MCP tool: the servers, and for every operation
// its method, path, parameters, and JSON request body. Schemas are returned as
// generic JSON-compatible trees with local $refs inlined, so a tool's input
// schema is self-contained for clients that cannot resolve references.
package openapi

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document is the parsed, mcpify-relevant view of an OpenAPI spec.
type Document struct {
	Title      string
	APIVersion string
	Servers    []Server
	Operations []Operation
}

// Server is one entry from the spec's top-level servers list.
type Server struct {
	URL         string
	Description string
}

// Operation is a single path+method, the unit that becomes one MCP tool.
type Operation struct {
	Method      string // upper-case: GET, POST, ...
	Path        string // template form, e.g. /pets/{petId}
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Parameters  []Parameter
	RequestBody *RequestBody
}

// Parameter is a path, query, header, or cookie parameter.
type Parameter struct {
	Name        string
	In          string // path | query | header | cookie
	Description string
	Required    bool
	Schema      map[string]any // JSON-schema fragment, $refs inlined
}

// RequestBody is the application/json request body of an operation.
type RequestBody struct {
	Description string
	Required    bool
	Schema      map[string]any // JSON-schema fragment, $refs inlined
}

// methodOrder is the fixed order operations are emitted in per path, so tool
// listings are deterministic regardless of map iteration order.
var methodOrder = []string{"get", "post", "put", "patch", "delete", "head", "options"}

// Load parses an OpenAPI 3.x document from JSON or YAML bytes. JSON is a subset
// of YAML, so one decoder handles both.
func Load(data []byte) (*Document, error) {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}
	if root == nil {
		return nil, errors.New("empty document")
	}
	if _, ok := root["swagger"]; ok {
		return nil, errors.New("Swagger 2.0 is not supported; convert the spec to OpenAPI 3.x")
	}
	ver, _ := root["openapi"].(string)
	if !strings.HasPrefix(ver, "3.") {
		return nil, fmt.Errorf("unsupported openapi version %q (need 3.x)", ver)
	}

	doc := &Document{}
	if info, ok := root["info"].(map[string]any); ok {
		doc.Title, _ = info["title"].(string)
		doc.APIVersion, _ = info["version"].(string)
	}
	doc.Servers = parseServers(root["servers"])
	doc.Operations = parsePaths(root)
	return doc, nil
}

func parseServers(v any) []Server {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []Server
	for _, it := range list {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		url, _ := m["url"].(string)
		if url == "" {
			continue
		}
		desc, _ := m["description"].(string)
		out = append(out, Server{URL: url, Description: desc})
	}
	return out
}

func parsePaths(root map[string]any) []Operation {
	paths, ok := root["paths"].(map[string]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var ops []Operation
	for _, path := range keys {
		item, ok := paths[path].(map[string]any)
		if !ok {
			continue
		}
		// Path-level parameters apply to every operation under the path.
		shared := parseParameters(root, item["parameters"])
		for _, method := range methodOrder {
			raw, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			op := Operation{
				Method: strings.ToUpper(method),
				Path:   path,
			}
			op.OperationID, _ = raw["operationId"].(string)
			op.Summary, _ = raw["summary"].(string)
			op.Description, _ = raw["description"].(string)
			op.Tags = parseTags(raw["tags"])
			op.Parameters = mergeParams(shared, parseParameters(root, raw["parameters"]))
			op.RequestBody = parseRequestBody(root, raw["requestBody"])
			ops = append(ops, op)
		}
	}
	return ops
}

func parseTags(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, it := range list {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// mergeParams overlays operation parameters on path-level ones; an operation
// parameter with the same name+location replaces the shared one.
func mergeParams(shared, own []Parameter) []Parameter {
	if len(shared) == 0 {
		return own
	}
	key := func(p Parameter) string { return p.In + "\x00" + p.Name }
	seen := make(map[string]bool, len(own))
	for _, p := range own {
		seen[key(p)] = true
	}
	out := make([]Parameter, 0, len(shared)+len(own))
	for _, p := range shared {
		if !seen[key(p)] {
			out = append(out, p)
		}
	}
	return append(out, own...)
}

func parseParameters(root map[string]any, v any) []Parameter {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []Parameter
	for _, it := range list {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if ref, ok := m["$ref"].(string); ok {
			if t, ok := resolveRef(root, ref); ok {
				if tm, ok := t.(map[string]any); ok {
					m = tm
				}
			}
		}
		name, _ := m["name"].(string)
		in, _ := m["in"].(string)
		if name == "" || in == "" {
			continue
		}
		p := Parameter{Name: name, In: in}
		p.Description, _ = m["description"].(string)
		p.Required, _ = m["required"].(bool)
		if in == "path" {
			p.Required = true // path parameters are always required
		}
		if sch, ok := m["schema"].(map[string]any); ok {
			p.Schema = inlineSchemaMap(root, sch)
		} else {
			p.Schema = map[string]any{"type": "string"}
		}
		out = append(out, p)
	}
	return out
}

func parseRequestBody(root map[string]any, v any) *RequestBody {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	if ref, ok := m["$ref"].(string); ok {
		if t, ok := resolveRef(root, ref); ok {
			if tm, ok := t.(map[string]any); ok {
				m = tm
			}
		}
	}
	content, ok := m["content"].(map[string]any)
	if !ok {
		return nil
	}
	mediaType := pickJSONMedia(content)
	if mediaType == "" {
		return nil
	}
	media, _ := content[mediaType].(map[string]any)
	schema, _ := media["schema"].(map[string]any)
	rb := &RequestBody{}
	rb.Description, _ = m["description"].(string)
	rb.Required, _ = m["required"].(bool)
	if schema != nil {
		rb.Schema = inlineSchemaMap(root, schema)
	}
	return rb
}

// pickJSONMedia returns the JSON media type to use from a content map,
// preferring application/json, then any */*+json, then any single entry.
func pickJSONMedia(content map[string]any) string {
	if _, ok := content["application/json"]; ok {
		return "application/json"
	}
	for ct := range content {
		if strings.HasSuffix(ct, "+json") || strings.Contains(ct, "json") {
			return ct
		}
	}
	for ct := range content {
		return ct
	}
	return ""
}
