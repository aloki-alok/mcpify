package cli

import "testing"

func TestEndpointURL(t *testing.T) {
	cases := map[string]string{
		":8080":          "http://localhost:8080/mcp",
		"0.0.0.0:8080":   "http://localhost:8080/mcp",
		"127.0.0.1:8077": "http://127.0.0.1:8077/mcp",
		"localhost:9000": "http://localhost:9000/mcp",
	}
	for addr, want := range cases {
		if got := endpointURL(addr); got != want {
			t.Errorf("endpointURL(%q) = %q, want %q", addr, got, want)
		}
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"OmniDimension API": "omnidimension-api",
		"Petstore":          "petstore",
		"  --weird-- ":      "weird",
		"":                  "mcpify",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConfigArgs(t *testing.T) {
	o := options{
		spec:     "spec.yaml",
		base:     "https://api.example.com",
		readOnly: true,
		headers:  map[string]string{"Authorization": "Bearer x"},
	}
	args := configArgs(o)
	// spec must be last; base and read-only flags must be present.
	if args[len(args)-1] != "spec.yaml" {
		t.Fatalf("spec should be last: %v", args)
	}
	joined := ""
	for _, a := range args {
		joined += a + " "
	}
	for _, want := range []string{"--base", "https://api.example.com", "--read-only", "-H", "Authorization: Bearer x"} {
		if !contains2(args, want) {
			t.Errorf("missing %q in %v", want, args)
		}
	}
	_ = joined
}

func contains2(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
