package tool

import (
	"path"
	"strings"

	"github.com/aloki-alok/mcpify/internal/openapi"
)

// Filter narrows which operations become tools. An operation is kept if it
// matches any Include pattern or any Tag, so the two lists are OR'd together
// as one keep-set; an empty Filter keeps everything.
type Filter struct {
	Include []string // substring or glob, matched against tool name, operationId, and path
	Tags    []string // exact match (case-insensitive) against an operation's tags
}

// Empty reports whether the filter has no patterns and would keep everything.
func (f Filter) Empty() bool {
	return len(f.Include) == 0 && len(f.Tags) == 0
}

// matches reports whether an operation, once named, passes the filter. name is
// the tool name Defs already derived for it, so --include can match the same
// identifier a user sees in `ls` output.
func (f Filter) matches(name string, op openapi.Operation) bool {
	if f.Empty() {
		return true
	}
	for _, pat := range f.Include {
		if matchInclude(pat, name, op.OperationID, op.Path) {
			return true
		}
	}
	for _, tag := range f.Tags {
		for _, t := range op.Tags {
			if strings.EqualFold(tag, t) {
				return true
			}
		}
	}
	return false
}

// matchInclude applies one --include pattern to a tool name, its operationId,
// and its path. A pattern containing '*' is a path.Match glob checked against
// each candidate in turn; otherwise it is a case-insensitive substring check
// against any of them.
func matchInclude(pat string, candidates ...string) bool {
	glob := strings.Contains(pat, "*")
	lowerPat := strings.ToLower(pat)
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if glob {
			if ok, _ := path.Match(pat, c); ok {
				return true
			}
			continue
		}
		if strings.Contains(strings.ToLower(c), lowerPat) {
			return true
		}
	}
	return false
}
