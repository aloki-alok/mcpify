package tool

import (
	"testing"

	"github.com/aloki-alok/mcpify/internal/openapi"
)

func mkTaggedOp(method, path, operationID string, tags ...string) openapi.Operation {
	op := mkOp(method, path)
	op.OperationID = operationID
	op.Tags = tags
	return op
}

func TestFilterEmptyKeepsEverything(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		mkTaggedOp("GET", "/pets", "listPets", "pet"),
		mkTaggedOp("POST", "/users", "createUser", "user"),
	}}
	defs := Defs(doc, false, Filter{})
	if len(defs) != 2 {
		t.Fatalf("got %d defs, want 2", len(defs))
	}
}

func TestFilterIncludeSubstringIsCaseInsensitive(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		mkTaggedOp("GET", "/pets", "listPets"),
		mkTaggedOp("POST", "/users", "createUser"),
	}}
	defs := Defs(doc, false, Filter{Include: []string{"PET"}})
	if len(defs) != 1 || defs[0].Name != "listPets" {
		t.Fatalf("defs = %+v", defs)
	}
}

func TestFilterIncludeMatchesOperationID(t *testing.T) {
	op := mkTaggedOp("DELETE", "/things/{id}", "removeThing")
	doc := &openapi.Document{Operations: []openapi.Operation{op}}
	defs := Defs(doc, false, Filter{Include: []string{"removeThing"}})
	if len(defs) != 1 {
		t.Fatalf("operationId match failed: %+v", defs)
	}
}

func TestFilterIncludeMatchesPath(t *testing.T) {
	op := mkTaggedOp("GET", "/orders/{orderId}/items", "listItems")
	doc := &openapi.Document{Operations: []openapi.Operation{op}}
	defs := Defs(doc, false, Filter{Include: []string{"/orders/"}})
	if len(defs) != 1 {
		t.Fatalf("path match failed: %+v", defs)
	}
}

func TestFilterIncludeGlob(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		mkTaggedOp("GET", "/pets", "listPets"),
		mkTaggedOp("GET", "/pets/{petId}", "getPet"),
		mkTaggedOp("POST", "/users", "createUser"),
	}}
	defs := Defs(doc, false, Filter{Include: []string{"get*"}})
	if len(defs) != 1 || defs[0].Name != "getPet" {
		t.Fatalf("glob match = %+v", defs)
	}
}

func TestFilterIncludeGlobOnPath(t *testing.T) {
	op := mkTaggedOp("GET", "/pets/{petId}", "getPet")
	doc := &openapi.Document{Operations: []openapi.Operation{op}}
	// path.Match glob against the literal path template, '*' does not cross '/'.
	defs := Defs(doc, false, Filter{Include: []string{"/pets/*"}})
	if len(defs) != 1 {
		t.Fatalf("path glob failed: %+v", defs)
	}
}

func TestFilterTagExactCaseInsensitive(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		mkTaggedOp("GET", "/pets", "listPets", "Pet"),
		mkTaggedOp("POST", "/users", "createUser", "User"),
	}}
	defs := Defs(doc, false, Filter{Tags: []string{"pet"}})
	if len(defs) != 1 || defs[0].Name != "listPets" {
		t.Fatalf("tag match = %+v", defs)
	}
}

func TestFilterTagIsExactNotSubstring(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		mkTaggedOp("GET", "/pets", "listPets", "pets"), // tag "pets", filter "pet" should not match
	}}
	defs := Defs(doc, false, Filter{Tags: []string{"pet"}})
	if len(defs) != 0 {
		t.Fatalf("expected no match for exact tag semantics, got %+v", defs)
	}
}

func TestFilterIncludeOrTagCombineWithOR(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		mkTaggedOp("GET", "/pets", "listPets", "pet"),
		mkTaggedOp("POST", "/users", "createUser", "user"),
		mkTaggedOp("GET", "/orders", "listOrders", "order"),
	}}
	// --include matches listOrders by path, --tag matches pet: both operations
	// should be kept even though neither flag alone covers both.
	defs := Defs(doc, false, Filter{Include: []string{"/orders"}, Tags: []string{"pet"}})
	if len(defs) != 2 {
		t.Fatalf("OR-combined filter = %+v", defs)
	}
}

func TestFilterCombinesWithReadOnly(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		mkTaggedOp("GET", "/pets", "listPets", "pet"),
		mkTaggedOp("POST", "/pets", "createPet", "pet"),
	}}
	defs := Defs(doc, true, Filter{Tags: []string{"pet"}})
	if len(defs) != 1 || defs[0].Name != "listPets" {
		t.Fatalf("read-only + tag filter = %+v", defs)
	}
}

func TestFilterNoMatchesYieldsEmpty(t *testing.T) {
	doc := &openapi.Document{Operations: []openapi.Operation{
		mkTaggedOp("GET", "/pets", "listPets", "pet"),
	}}
	defs := Defs(doc, false, Filter{Include: []string{"nonexistent"}})
	if len(defs) != 0 {
		t.Fatalf("expected empty result, got %+v", defs)
	}
}
