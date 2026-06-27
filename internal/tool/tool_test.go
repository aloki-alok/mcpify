package tool

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/aloki-alok/mcpify/internal/openapi"
)

func mkOp(method, path string) openapi.Operation {
	return openapi.Operation{Method: method, Path: path}
}

func TestBuildSchemaFlattensBody(t *testing.T) {
	op := mkOp("POST", "/pets")
	op.OperationID = "createPet"
	op.Parameters = []openapi.Parameter{
		{Name: "tenant", In: "header", Required: true, Schema: map[string]any{"type": "string"}},
	}
	op.RequestBody = &openapi.RequestBody{
		Required: true,
		Schema: map[string]any{
			"type":     "object",
			"required": []any{"name"},
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "integer"},
			},
		},
	}
	d := build(op, map[string]bool{})

	props := d.InputSchema["properties"].(map[string]any)
	for _, want := range []string{"tenant", "name", "age"} {
		if _, ok := props[want]; !ok {
			t.Fatalf("missing property %q in %v", want, props)
		}
	}
	req, _ := d.InputSchema["required"].([]string)
	if !contains(req, "name") || !contains(req, "tenant") {
		t.Fatalf("required = %v", req)
	}
	if !d.bodyFields["name"] || d.bodyArg {
		t.Fatalf("body should be flattened: fields=%v arg=%v", d.bodyFields, d.bodyArg)
	}
}

func TestBuildSchemaNonObjectBody(t *testing.T) {
	op := mkOp("PUT", "/blob")
	op.RequestBody = &openapi.RequestBody{
		Required: true,
		Schema:   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	}
	d := build(op, map[string]bool{})
	props := d.InputSchema["properties"].(map[string]any)
	if _, ok := props["body"]; !ok || !d.bodyArg {
		t.Fatalf("non-object body should be a single 'body' arg: %v", props)
	}
}

func TestUniqueNames(t *testing.T) {
	used := map[string]bool{}
	a := uniqueName(mkOp("GET", "/pets/{petId}"), used) // no operationId -> from method+path
	b := uniqueName(mkOp("GET", "/pets/{petId}"), used) // collision
	if a != "get_pets_petId" {
		t.Fatalf("derived name = %q", a)
	}
	if b != "get_pets_petId_2" {
		t.Fatalf("collision name = %q", b)
	}
}

func TestBuildRequestRouting(t *testing.T) {
	op := mkOp("POST", "/pets/{petId}/notes")
	op.OperationID = "addNote"
	op.Parameters = []openapi.Parameter{
		{Name: "petId", In: "path", Required: true, Schema: map[string]any{"type": "string"}},
		{Name: "limit", In: "query", Schema: map[string]any{"type": "integer"}},
		{Name: "X-Trace", In: "header", Schema: map[string]any{"type": "string"}},
	}
	op.RequestBody = &openapi.RequestBody{
		Schema: map[string]any{"type": "object", "properties": map[string]any{
			"text": map[string]any{"type": "string"},
		}},
	}
	d := build(op, map[string]bool{})

	args := map[string]any{
		"petId":   "p-7",
		"limit":   float64(5),
		"X-Trace": "abc",
		"text":    "hello",
	}
	req, err := BuildRequest(context.Background(), d, "https://api.example.com/v1/", args)
	if err != nil {
		t.Fatalf("BuildRequest: %v", err)
	}
	if req.Method != "POST" {
		t.Fatalf("method = %s", req.Method)
	}
	if got := req.URL.Path; got != "/v1/pets/p-7/notes" {
		t.Fatalf("path = %q", got)
	}
	if got := req.URL.Query().Get("limit"); got != "5" {
		t.Fatalf("query limit = %q", got)
	}
	if got := req.Header.Get("X-Trace"); got != "abc" {
		t.Fatalf("header = %q", got)
	}
	if ct := req.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
	bodyBytes, _ := io.ReadAll(req.Body)
	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("body json: %v (%s)", err, bodyBytes)
	}
	if body["text"] != "hello" || len(body) != 1 {
		t.Fatalf("body = %v", body)
	}
}

func TestBuildRequestMissingPathParam(t *testing.T) {
	op := mkOp("GET", "/pets/{petId}")
	op.Parameters = []openapi.Parameter{{Name: "petId", In: "path", Required: true}}
	d := build(op, map[string]bool{})
	if _, err := BuildRequest(context.Background(), d, "http://x", map[string]any{}); err == nil {
		t.Fatal("expected error for missing path parameter")
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
