package tool

import (
	"strings"

	"github.com/aloki-alok/mcpify/internal/openapi"
)

// uniqueName derives a stable, MCP-safe tool name from an operation, preferring
// operationId and falling back to method+path. Collisions get a numeric suffix.
func uniqueName(op openapi.Operation, used map[string]bool) string {
	base := sanitize(op.OperationID)
	if base == "" {
		base = sanitize(strings.ToLower(op.Method) + "_" + op.Path)
	}
	if base == "" {
		base = "op"
	}
	name := base
	for i := 2; used[name]; i++ {
		name = base + "_" + itoa(i)
	}
	used[name] = true
	return name
}

// sanitize reduces a string to the characters MCP tool names allow
// ([A-Za-z0-9_-]), turning path templates like /pets/{petId} into pets_petId
// and collapsing runs of separators.
func sanitize(s string) string {
	var b strings.Builder
	lastSep := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSep = false
		case r == '-' || r == '_':
			if !lastSep {
				b.WriteByte('_')
				lastSep = true
			}
		default: // '/', '{', '}', '.', spaces, etc. become a single separator
			if !lastSep {
				b.WriteByte('_')
				lastSep = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
