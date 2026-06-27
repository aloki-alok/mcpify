package openapi

import "strings"

// resolveRef looks up a local JSON pointer like "#/components/schemas/Pet"
// against the document root. Only same-document refs are supported; anything
// else returns ok=false and the caller degrades gracefully.
func resolveRef(root map[string]any, ref string) (any, bool) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, false
	}
	var cur any = root
	for _, part := range strings.Split(ref[2:], "/") {
		part = decodePointer(part)
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// decodePointer reverses RFC 6901 escaping (~1 -> "/", ~0 -> "~").
func decodePointer(s string) string {
	s = strings.ReplaceAll(s, "~1", "/")
	return strings.ReplaceAll(s, "~0", "~")
}

// inlineSchemaMap deep-copies a schema with all local $refs resolved inline, so
// the result stands alone for MCP clients that do not resolve references.
func inlineSchemaMap(root, schema map[string]any) map[string]any {
	v := inline(root, schema, map[string]bool{})
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// inline returns a deep copy of node with local $refs followed. seen holds the
// refs currently being expanded on this branch; revisiting one means a cycle,
// which collapses to an empty (permissive) schema rather than recursing forever.
func inline(root map[string]any, node any, seen map[string]bool) any {
	switch n := node.(type) {
	case map[string]any:
		if ref, ok := n["$ref"].(string); ok {
			if seen[ref] {
				return map[string]any{}
			}
			target, ok := resolveRef(root, ref)
			if !ok {
				return cloneMapWithout(n, "$ref")
			}
			next := copySet(seen)
			next[ref] = true
			return inline(root, target, next)
		}
		out := make(map[string]any, len(n))
		for k, v := range n {
			out[k] = inline(root, v, seen)
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i, v := range n {
			out[i] = inline(root, v, seen)
		}
		return out
	default:
		return node
	}
}

func copySet(s map[string]bool) map[string]bool {
	out := make(map[string]bool, len(s)+1)
	for k := range s {
		out[k] = true
	}
	return out
}

// cloneMapWithout copies m, dropping one key. Used when a $ref cannot be
// resolved (e.g. a remote ref): keep any sibling keys, drop the dead pointer.
func cloneMapWithout(m map[string]any, drop string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if k == drop {
			continue
		}
		out[k] = v
	}
	return out
}
