package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/aloki-alok/mcpify/internal/openapi"
)

func TestParseFlagsBeforeAndAfterSpec(t *testing.T) {
	o, err := parse([]string{"--base", "https://api.example.com", "spec.yaml", "-H", "Authorization: Bearer tok", "--read-only"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if o.spec != "spec.yaml" {
		t.Fatalf("spec = %q", o.spec)
	}
	if o.base != "https://api.example.com" {
		t.Fatalf("base = %q", o.base)
	}
	if o.headers["Authorization"] != "Bearer tok" {
		t.Fatalf("headers = %v", o.headers)
	}
	if !o.readOnly {
		t.Fatal("read-only not set")
	}
}

func TestParseInlineAndTimeout(t *testing.T) {
	o, err := parse([]string{"--base=https://x", "--timeout=5s", "spec.json"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if o.base != "https://x" || o.timeout != 5*time.Second {
		t.Fatalf("o = %+v", o)
	}
}

func TestParseIncludeAndTagRepeatable(t *testing.T) {
	o, err := parse([]string{"--include", "pet", "--tag", "user", "spec.yaml", "--include", "order", "--tag", "store"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if o.spec != "spec.yaml" {
		t.Fatalf("spec = %q", o.spec)
	}
	wantInclude := []string{"pet", "order"}
	if len(o.include) != len(wantInclude) || o.include[0] != wantInclude[0] || o.include[1] != wantInclude[1] {
		t.Fatalf("include = %v, want %v", o.include, wantInclude)
	}
	wantTags := []string{"user", "store"}
	if len(o.tags) != len(wantTags) || o.tags[0] != wantTags[0] || o.tags[1] != wantTags[1] {
		t.Fatalf("tags = %v, want %v", o.tags, wantTags)
	}
}

func TestParseIncludeAndTagBeforeAndAfterSpec(t *testing.T) {
	before, err := parse([]string{"--include", "pet", "spec.yaml"})
	if err != nil {
		t.Fatalf("parse (before): %v", err)
	}
	after, err := parse([]string{"spec.yaml", "--include", "pet"})
	if err != nil {
		t.Fatalf("parse (after): %v", err)
	}
	if len(before.include) != 1 || before.include[0] != "pet" {
		t.Fatalf("before = %v", before.include)
	}
	if len(after.include) != 1 || after.include[0] != "pet" {
		t.Fatalf("after = %v", after.include)
	}
}

func TestParseIncludeInline(t *testing.T) {
	o, err := parse([]string{"--include=pet", "--tag=user", "spec.yaml"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(o.include) != 1 || o.include[0] != "pet" {
		t.Fatalf("include = %v", o.include)
	}
	if len(o.tags) != 1 || o.tags[0] != "user" {
		t.Fatalf("tags = %v", o.tags)
	}
}

func TestNoToolsErrorNamesTheFilter(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		{Method: "GET", Path: "/pets", OperationID: "listPets", Tags: []string{"pet"}},
		{Method: "POST", Path: "/users", OperationID: "createUser", Tags: []string{"user"}},
	}}

	o := options{include: []string{"nope"}}
	err := o.noToolsError(doc)
	if err == nil || !strings.Contains(err.Error(), "--include") {
		t.Fatalf("error should mention the filter: %v", err)
	}
	// It should suggest what the spec does offer so a typo is easy to fix.
	for _, want := range []string{"pet", "user", "listPets", "createUser"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error should list available %q: %v", want, err)
		}
	}

	ro := options{readOnly: true}
	if err := ro.noToolsError(doc); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("error should mention read-only: %v", err)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := parse([]string{"--http", ":8080"}); err == nil {
		t.Fatal("missing spec should error")
	}
	if _, err := parse([]string{"-H", "no-colon", "spec.yaml"}); err == nil {
		t.Fatal("malformed header should error")
	}
	if _, err := parse([]string{"--nope", "spec.yaml"}); err == nil {
		t.Fatal("unknown flag should error")
	}
	if _, err := parse([]string{"a.yaml", "b.yaml"}); err == nil {
		t.Fatal("two specs should error")
	}
}
