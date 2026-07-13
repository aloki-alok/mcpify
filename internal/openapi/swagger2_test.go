package openapi

import "testing"

const swagger2 = `
swagger: "2.0"
info:
  title: Petstore
  version: 1.0.0
host: api.example.com
basePath: /v2
schemes: [http, https]
paths:
  /pets:
    parameters:
      - name: tenant
        in: header
        required: true
        type: string
    get:
      operationId: listPets
      summary: List pets
      parameters:
        - name: limit
          in: query
          type: integer
        - name: status
          in: query
          type: string
          enum: [available, pending, sold]
          collectionFormat: multi
    post:
      operationId: createPet
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: '#/definitions/Pet'
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          type: string
definitions:
  Pet:
    type: object
    required: [name]
    properties:
      name: { type: string }
      owner: { $ref: '#/definitions/Pet' }
`

func TestLoadSwagger2(t *testing.T) {
	doc, err := Load([]byte(swagger2))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if doc.Title != "Petstore" || doc.APIVersion != "1.0.0" {
		t.Fatalf("info = %q %q", doc.Title, doc.APIVersion)
	}
	// https is preferred even though http is listed first.
	if len(doc.Servers) != 1 || doc.Servers[0].URL != "https://api.example.com/v2" {
		t.Fatalf("servers = %+v", doc.Servers)
	}
	if len(doc.Operations) != 3 {
		t.Fatalf("want 3 operations, got %d", len(doc.Operations))
	}

	get := findOp(doc, "listPets")
	if get == nil || get.Method != "GET" || get.Path != "/pets" {
		t.Fatalf("listPets = %+v", get)
	}
	// path-level header param merges onto the operation, and inline-typed query
	// params carry their type.
	if !hasParam(get, "tenant", "header") || !hasParam(get, "limit", "query") {
		t.Fatalf("listPets params = %+v", get.Parameters)
	}
	limit := paramNamed(get, "limit")
	if limit.Schema["type"] != "integer" {
		t.Fatalf("limit type = %+v", limit.Schema)
	}
	status := paramNamed(get, "status")
	if _, ok := status.Schema["enum"]; !ok {
		t.Fatalf("status enum dropped: %+v", status.Schema)
	}
	// collectionFormat is intentionally not carried into the schema.
	if _, ok := status.Schema["collectionFormat"]; ok {
		t.Fatalf("collectionFormat should be ignored: %+v", status.Schema)
	}

	pathOp := findOp(doc, "getPet")
	p := paramNamed(pathOp, "petId")
	if p == nil || !p.Required {
		t.Fatalf("path param petId should be required: %+v", p)
	}
}

func TestSwagger2BodyRefInlineAndCycle(t *testing.T) {
	doc, err := Load([]byte(swagger2))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	post := findOp(doc, "createPet")
	if post.RequestBody == nil || !post.RequestBody.Required {
		t.Fatalf("createPet body = %+v", post.RequestBody)
	}
	props, _ := post.RequestBody.Schema["properties"].(map[string]any)
	if _, ok := props["name"]; !ok {
		t.Fatalf("#/definitions ref not inlined: %+v", post.RequestBody.Schema)
	}
	owner, _ := props["owner"].(map[string]any)
	if len(owner) != 0 {
		t.Fatalf("cyclic owner should collapse to {}, got %+v", owner)
	}
}

func TestSwagger2NonObjectBody(t *testing.T) {
	const spec = `
swagger: "2.0"
info: { title: T, version: "1" }
host: api.example.com
paths:
  /tags:
    post:
      operationId: setTags
      parameters:
        - name: body
          in: body
          required: true
          schema:
            type: array
            items: { type: string }
`
	doc, err := Load([]byte(spec))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	op := findOp(doc, "setTags")
	if op.RequestBody == nil {
		t.Fatal("setTags body missing")
	}
	if op.RequestBody.Schema["type"] != "array" {
		t.Fatalf("non-object body schema = %+v", op.RequestBody.Schema)
	}
}

func TestSwagger2ServerURL(t *testing.T) {
	base := func(extra string) string {
		return "swagger: \"2.0\"\ninfo: { title: T, version: \"1\" }\n" + extra +
			"\npaths:\n  /x:\n    get: { operationId: x }\n"
	}
	cases := []struct {
		name  string
		extra string
		want  string // "" means no server should be set
	}{
		{"https preferred", "host: h.com\nbasePath: /b\nschemes: [http, https]", "https://h.com/b"},
		{"single http", "host: h.com\nbasePath: /b\nschemes: [http]", "http://h.com/b"},
		{"no schemes", "host: h.com\nbasePath: /b", "h.com/b"},
		{"no basePath", "host: h.com\nschemes: [https]", "https://h.com"},
		{"no host falls back", "basePath: /b\nschemes: [https]", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc, err := Load([]byte(base(c.extra)))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if c.want == "" {
				if len(doc.Servers) != 0 {
					t.Fatalf("want no server, got %+v", doc.Servers)
				}
				return
			}
			if len(doc.Servers) != 1 || doc.Servers[0].URL != c.want {
				t.Fatalf("server = %+v, want %q", doc.Servers, c.want)
			}
		})
	}
}

func TestSwagger2FormDataSkipped(t *testing.T) {
	const spec = `
swagger: "2.0"
info: { title: T, version: "1" }
host: api.example.com
paths:
  /upload:
    post:
      operationId: uploadFile
      parameters:
        - name: file
          in: formData
          type: file
  /pets:
    get:
      operationId: listPets
`
	doc, err := Load([]byte(spec))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if findOp(doc, "uploadFile") != nil {
		t.Fatal("formData operation should be skipped")
	}
	if findOp(doc, "listPets") == nil {
		t.Fatal("non-formData operation in the same spec must survive")
	}
	if len(doc.Warnings) != 1 {
		t.Fatalf("want one formData warning, got %+v", doc.Warnings)
	}
}

func TestSwagger2ParamRefResolved(t *testing.T) {
	const spec = `
swagger: "2.0"
info: { title: T, version: "1" }
host: api.example.com
parameters:
  LimitParam:
    name: limit
    in: query
    type: integer
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - $ref: '#/parameters/LimitParam'
`
	doc, err := Load([]byte(spec))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	op := findOp(doc, "listPets")
	if !hasParam(op, "limit", "query") {
		t.Fatalf("shared #/parameters ref not resolved: %+v", op.Parameters)
	}
	if paramNamed(op, "limit").Schema["type"] != "integer" {
		t.Fatalf("resolved param type wrong: %+v", op.Parameters)
	}
}
