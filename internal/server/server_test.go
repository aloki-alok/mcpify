package server

import (
	"net/http"
	"testing"

	"github.com/aloki-alok/mcpify/internal/openapi"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResolveBase(t *testing.T) {
	cases := []struct {
		name     string
		servers  []openapi.Server
		override string
		specURL  string
		want     string
		wantErr  bool
	}{
		{name: "override wins", servers: []openapi.Server{{URL: "https://a"}}, override: "https://b", want: "https://b"},
		{name: "override must be absolute", override: "a.com", wantErr: true},
		{name: "absolute server", servers: []openapi.Server{{URL: "https://api.example.com/v1"}}, want: "https://api.example.com/v1"},
		{name: "relative server resolved against spec url", servers: []openapi.Server{{URL: "/v2"}}, specURL: "https://host.tld/openapi.json", want: "https://host.tld/v2"},
		{name: "relative server without spec url errors", servers: []openapi.Server{{URL: "/v2"}}, wantErr: true},
		{name: "templated server errors", servers: []openapi.Server{{URL: "https://{region}.api.com"}}, wantErr: true},
		{name: "no servers falls back to spec origin", specURL: "https://host.tld/spec.yaml", want: "https://host.tld"},
		{name: "no servers no spec url errors", wantErr: true},
		{name: "templated server filled from variable defaults", servers: []openapi.Server{{URL: "https://{host}/{base}", Variables: map[string]string{"host": "api.example.com", "base": "v2"}}}, want: "https://api.example.com/v2"},
		{name: "templated server missing default errors", servers: []openapi.Server{{URL: "https://{host}/v1", Variables: map[string]string{}}}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := &openapi.Document{Servers: tc.servers}
			got, err := ResolveBase(doc, tc.override, tc.specURL)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestFormatResponseSuccess(t *testing.T) {
	resp := &http.Response{StatusCode: 200, Status: "200 OK"}
	res := formatResponse(resp, []byte(`{"id":7,"name":"rex"}`))
	if res.IsError {
		t.Fatal("200 should not be an error")
	}
	if res.StructuredContent == nil {
		t.Fatal("json object should populate structured content")
	}
	obj := res.StructuredContent.(map[string]any)
	if obj["name"] != "rex" {
		t.Fatalf("structured = %v", obj)
	}
}

func TestFormatResponseError(t *testing.T) {
	resp := &http.Response{StatusCode: 404, Status: "404 Not Found"}
	res := formatResponse(resp, []byte(`{"error":"missing"}`))
	if !res.IsError {
		t.Fatal("404 should be an error")
	}
}

func TestFormatResponseEmptyBody(t *testing.T) {
	resp := &http.Response{StatusCode: 204, Status: "204 No Content"}
	res := formatResponse(resp, nil)
	if res.IsError {
		t.Fatal("204 should not be an error")
	}
	got := res.Content[0].(*sdk.TextContent).Text
	if got != "HTTP 204 No Content (no content)" {
		t.Fatalf("text = %q", got)
	}
}
