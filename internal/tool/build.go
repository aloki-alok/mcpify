// Package tool turns OpenAPI operations into MCP tool definitions and builds the
// upstream HTTP request for a tool call. One operation becomes one tool: its
// input schema merges the operation's parameters with the JSON body fields, and
// a Def remembers where each argument belongs (path, query, header, or body) so
// the request can be reconstructed faithfully.
package tool

import (
	"sort"
	"strings"

	"github.com/aloki-alok/mcpify/internal/openapi"
)

// Def is a built MCP tool: its wire-facing name/description/schema plus the
// routing knowledge needed to turn arguments back into an HTTP request.
type Def struct {
	Name        string
	Description string
	InputSchema map[string]any

	op         openapi.Operation
	bodyFields map[string]bool // top-level args that are flattened JSON body fields
	bodyArg    bool            // true when the whole body is a single "body" argument
}

// Method and Path expose the underlying operation for previews.
func (d Def) Method() string { return d.op.Method }
func (d Def) Path() string   { return d.op.Path }

// Defs builds tool definitions for every operation in the document. When
// readOnly is set, only safe (GET, HEAD) operations are exposed. Names are made
// unique and stable.
func Defs(doc *openapi.Document, readOnly bool) []Def {
	used := map[string]bool{}
	var out []Def
	for _, op := range doc.Operations {
		if readOnly && op.Method != "GET" && op.Method != "HEAD" {
			continue
		}
		out = append(out, build(op, used))
	}
	return out
}

func build(op openapi.Operation, used map[string]bool) Def {
	d := Def{
		op:          op,
		Name:        uniqueName(op, used),
		Description: describe(op),
		bodyFields:  map[string]bool{},
	}

	properties := map[string]any{}
	var required []string
	paramNames := map[string]bool{}

	for _, p := range op.Parameters {
		schema := withContext(p.Schema, p.Description, "Location: "+p.In+" parameter.")
		properties[p.Name] = schema
		paramNames[p.Name] = true
		if p.Required {
			required = append(required, p.Name)
		}
	}

	if rb := op.RequestBody; rb != nil && rb.Schema != nil {
		if t, _ := rb.Schema["type"].(string); t == "object" && rb.Schema["properties"] != nil {
			bodyProps, _ := rb.Schema["properties"].(map[string]any)
			bodyReq := stringSet(rb.Schema["required"])
			for _, name := range sortedKeys(bodyProps) {
				if paramNames[name] {
					continue // a parameter of the same name wins
				}
				properties[name] = bodyProps[name]
				d.bodyFields[name] = true
				if bodyReq[name] {
					required = append(required, name)
				}
			}
		} else {
			properties["body"] = withContext(rb.Schema, rb.Description, "Request body.")
			d.bodyArg = true
			if rb.Required {
				required = append(required, "body")
			}
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		sort.Strings(required)
		schema["required"] = required
	}
	d.InputSchema = schema
	return d
}

// describe builds the tool description shown to the model: the method and path
// as an anchor, then the operation's summary and description.
func describe(op openapi.Operation) string {
	parts := []string{op.Method + " " + op.Path}
	if op.Summary != "" {
		parts = append(parts, op.Summary)
	}
	if op.Description != "" && op.Description != op.Summary {
		parts = append(parts, op.Description)
	}
	return strings.Join(parts, "\n")
}

// withContext returns a copy of a schema with the parameter/body description and
// a routing note folded into its "description", without mutating the input.
func withContext(schema map[string]any, desc, note string) map[string]any {
	out := make(map[string]any, len(schema)+1)
	for k, v := range schema {
		out[k] = v
	}
	existing, _ := out["description"].(string)
	combined := strings.TrimSpace(strings.Join(nonEmpty(desc, existing, note), " "))
	if combined != "" {
		out["description"] = combined
	}
	return out
}

func nonEmpty(ss ...string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func stringSet(v any) map[string]bool {
	out := map[string]bool{}
	if list, ok := v.([]any); ok {
		for _, it := range list {
			if s, ok := it.(string); ok {
				out[s] = true
			}
		}
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
