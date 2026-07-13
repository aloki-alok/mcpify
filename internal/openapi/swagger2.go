package openapi

import (
	"fmt"
	"sort"
	"strings"
)

// loadSwagger2 converts a Swagger 2.0 root document into the same Document model
// the 3.x path produces. The two specs differ mainly in how they express the
// server (schemes/host/basePath instead of a servers list), how non-body
// parameters carry their type (inline instead of under a "schema"), and where
// $refs point (#/definitions instead of #/components). Everything downstream
// (tool building, request routing) then treats both alike.
func loadSwagger2(root map[string]any) (*Document, error) {
	doc := &Document{}
	if info, ok := root["info"].(map[string]any); ok {
		doc.Title, _ = info["title"].(string)
		doc.APIVersion, _ = info["version"].(string)
	}
	if srv, ok := swagger2Server(root); ok {
		doc.Servers = []Server{srv}
	}
	doc.Operations = parseSwagger2Paths(root, doc)
	return doc, nil
}

// swagger2Server assembles the single server URL from schemes + host + basePath.
// https is preferred when offered; the first listed scheme is the fallback. When
// host is absent the server is left unset so ResolveBase falls back to the URL
// the spec was fetched from, matching how a 3.x relative server resolves.
func swagger2Server(root map[string]any) (Server, bool) {
	host, _ := root["host"].(string)
	if host == "" {
		return Server{}, false
	}
	basePath, _ := root["basePath"].(string)
	scheme := pickScheme(root["schemes"])
	url := host + basePath
	if scheme != "" {
		url = scheme + "://" + url
	}
	return Server{URL: url}, true
}

// pickScheme chooses the transport scheme, preferring https, then the first one
// listed. An empty result leaves the assembled URL scheme-relative, which
// ResolveBase then completes from the spec's own origin.
func pickScheme(v any) string {
	list, ok := v.([]any)
	if !ok {
		return ""
	}
	var first string
	for _, it := range list {
		s, ok := it.(string)
		if !ok {
			continue
		}
		if s == "https" {
			return "https"
		}
		if first == "" {
			first = s
		}
	}
	return first
}

func parseSwagger2Paths(root map[string]any, doc *Document) []Operation {
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
		sharedRaw := rawParams(item["parameters"])
		for _, method := range methodOrder {
			raw, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			merged := mergeSwagger2Params(sharedRaw, rawParams(raw["parameters"]))
			if consumesFormData(merged) {
				doc.Warnings = append(doc.Warnings, fmt.Sprintf(
					"skipping %s %s: formData parameters are not supported", strings.ToUpper(method), path))
				continue
			}
			op := Operation{Method: strings.ToUpper(method), Path: path}
			op.OperationID, _ = raw["operationId"].(string)
			op.Summary, _ = raw["summary"].(string)
			op.Description, _ = raw["description"].(string)
			op.Tags = parseTags(raw["tags"])
			op.Parameters, op.RequestBody = convertSwagger2Params(root, merged)
			ops = append(ops, op)
		}
	}
	return ops
}

// rawParams resolves each entry of a Swagger 2.0 parameter list, following any
// #/parameters/... $ref so location checks and merging see the real parameter.
func rawParams(v any) []map[string]any {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(list))
	for _, it := range list {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, m)
	}
	return out
}

// mergeSwagger2Params overlays operation parameters on path-level ones, keyed by
// name+location as the spec requires. It mirrors mergeParams but works on the
// raw maps, before conversion, since a formData check must happen on the merged
// set. Refs are resolved lazily here only to read name/in for the key.
func mergeSwagger2Params(shared, own []map[string]any) []map[string]any {
	if len(shared) == 0 {
		return own
	}
	seen := make(map[string]bool, len(own))
	for _, p := range own {
		seen[paramKey(p)] = true
	}
	out := make([]map[string]any, 0, len(shared)+len(own))
	for _, p := range shared {
		if !seen[paramKey(p)] {
			out = append(out, p)
		}
	}
	return append(out, own...)
}

func paramKey(m map[string]any) string {
	name, _ := m["name"].(string)
	in, _ := m["in"].(string)
	return in + "\x00" + name
}

func consumesFormData(params []map[string]any) bool {
	for _, m := range params {
		if in, _ := m["in"].(string); in == "formData" {
			return true
		}
	}
	return false
}

// convertSwagger2Params splits a merged parameter list into path/query/header
// parameters and, for a single body parameter, a RequestBody. In Swagger 2.0 a
// non-body parameter carries its type inline (type/format/enum/items), whereas a
// body parameter wraps a JSON schema under "schema"; both are inlined so local
// $refs to #/definitions are self-contained.
func convertSwagger2Params(root map[string]any, params []map[string]any) ([]Parameter, *RequestBody) {
	var out []Parameter
	var body *RequestBody
	for _, m := range params {
		if ref, ok := m["$ref"].(string); ok {
			if t, ok := resolveRef(root, ref); ok {
				if tm, ok := t.(map[string]any); ok {
					m = tm
				}
			}
		}
		name, _ := m["name"].(string)
		in, _ := m["in"].(string)
		if in == "body" {
			body = swagger2Body(root, m)
			continue
		}
		if name == "" || in == "" {
			continue
		}
		p := Parameter{Name: name, In: in}
		p.Description, _ = m["description"].(string)
		p.Required, _ = m["required"].(bool)
		if in == "path" {
			p.Required = true
		}
		p.Schema = swagger2ParamSchema(root, m)
		out = append(out, p)
	}
	return out, body
}

// swagger2Body turns a body parameter into a RequestBody. Its schema maps to
// what a 3.x requestBody produces: an object schema is later flattened into
// top-level fields by the tool builder, and any other shape becomes the single
// "body" argument. The same convention holds for both specs.
func swagger2Body(root, m map[string]any) *RequestBody {
	rb := &RequestBody{}
	rb.Description, _ = m["description"].(string)
	rb.Required, _ = m["required"].(bool)
	if sch, ok := m["schema"].(map[string]any); ok {
		rb.Schema = inlineSchemaMap(root, sch)
	}
	return rb
}

// swagger2ParamSchema builds a JSON-schema fragment for a non-body parameter
// from its inline type keywords. collectionFormat is intentionally ignored: it
// only affects how array values are serialized on the wire, which mcpify handles
// with repeated query values regardless. A parameter with no type defaults to
// string, matching the 3.x fallback.
func swagger2ParamSchema(root, m map[string]any) map[string]any {
	schema := map[string]any{}
	for _, k := range []string{"type", "format", "enum", "items", "default", "minimum", "maximum", "pattern"} {
		if v, ok := m[k]; ok {
			schema[k] = v
		}
	}
	if _, ok := schema["type"]; !ok {
		schema["type"] = "string"
	}
	return inlineSchemaMap(root, schema)
}
