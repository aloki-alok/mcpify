package openapi

import "testing"

const petstore = `
openapi: 3.0.3
info:
  title: Petstore
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /pets:
    parameters:
      - name: tenant
        in: header
        required: true
        schema: { type: string }
    get:
      operationId: listPets
      summary: List pets
      parameters:
        - name: limit
          in: query
          schema: { type: integer }
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          schema: { type: string }
components:
  schemas:
    Pet:
      type: object
      required: [name]
      properties:
        name: { type: string }
        owner: { $ref: '#/components/schemas/Pet' }
`

func TestLoad(t *testing.T) {
	doc, err := Load([]byte(petstore))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if doc.Title != "Petstore" || doc.APIVersion != "1.0.0" {
		t.Fatalf("info = %q %q", doc.Title, doc.APIVersion)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].URL != "https://api.example.com/v1" {
		t.Fatalf("servers = %+v", doc.Servers)
	}
	if len(doc.Operations) != 3 {
		t.Fatalf("want 3 operations, got %d", len(doc.Operations))
	}

	get := findOp(doc, "listPets")
	if get == nil {
		t.Fatal("listPets missing")
	}
	if get.Method != "GET" || get.Path != "/pets" {
		t.Fatalf("listPets = %s %s", get.Method, get.Path)
	}
	// path-level header param is merged onto the operation
	if !hasParam(get, "tenant", "header") || !hasParam(get, "limit", "query") {
		t.Fatalf("listPets params = %+v", get.Parameters)
	}

	pathOp := findOp(doc, "getPet")
	p := paramNamed(pathOp, "petId")
	if p == nil || !p.Required {
		t.Fatalf("path param petId should be required: %+v", p)
	}

	post := findOp(doc, "createPet")
	if post.RequestBody == nil || !post.RequestBody.Required {
		t.Fatalf("createPet body = %+v", post.RequestBody)
	}
	props, _ := post.RequestBody.Schema["properties"].(map[string]any)
	if _, ok := props["name"]; !ok {
		t.Fatalf("body schema not inlined: %+v", post.RequestBody.Schema)
	}
	// the self-referential owner must terminate at an empty schema, not recurse
	owner, _ := props["owner"].(map[string]any)
	if len(owner) != 0 {
		t.Fatalf("cyclic owner should collapse to {}, got %+v", owner)
	}
}

func TestSwaggerRejected(t *testing.T) {
	_, err := Load([]byte(`{"swagger":"2.0","info":{}}`))
	if err == nil {
		t.Fatal("expected Swagger 2.0 to be rejected")
	}
}

func TestLoadJSON(t *testing.T) {
	const spec = `{"openapi":"3.1.0","info":{"title":"T","version":"9"},"paths":{"/x":{"get":{"operationId":"x"}}}}`
	doc, err := Load([]byte(spec))
	if err != nil {
		t.Fatalf("Load json: %v", err)
	}
	if len(doc.Operations) != 1 || doc.Operations[0].OperationID != "x" {
		t.Fatalf("ops = %+v", doc.Operations)
	}
}

func findOp(d *Document, id string) *Operation {
	for i := range d.Operations {
		if d.Operations[i].OperationID == id {
			return &d.Operations[i]
		}
	}
	return nil
}

func paramNamed(op *Operation, name string) *Parameter {
	if op == nil {
		return nil
	}
	for i := range op.Parameters {
		if op.Parameters[i].Name == name {
			return &op.Parameters[i]
		}
	}
	return nil
}

func hasParam(op *Operation, name, in string) bool {
	p := paramNamed(op, name)
	return p != nil && p.In == in
}
