package cli

import (
	"testing"
	"time"
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
