package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// BuildRequest turns a tool call's arguments into the upstream HTTP request for
// this operation, routing each argument to the path, query string, headers, or
// JSON body according to the Def. baseURL is the resolved API server root.
func BuildRequest(ctx context.Context, d Def, baseURL string, args map[string]any) (*http.Request, error) {
	path := d.op.Path
	query := url.Values{}
	header := http.Header{}
	var cookies []string

	for _, p := range d.op.Parameters {
		v, ok := args[p.Name]
		switch p.In {
		case "path":
			if !ok {
				return nil, fmt.Errorf("missing required path parameter %q", p.Name)
			}
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(valToStr(v)))
		case "query":
			if ok {
				for _, s := range valToStrings(v) {
					query.Add(p.Name, s)
				}
			}
		case "header":
			if ok {
				header.Set(p.Name, valToStr(v))
			}
		case "cookie":
			if ok {
				cookies = append(cookies, p.Name+"="+valToStr(v))
			}
		}
	}

	full := strings.TrimRight(baseURL, "/") + path
	if enc := query.Encode(); enc != "" {
		full += "?" + enc
	}

	var body []byte
	if d.bodyArg {
		if v, ok := args["body"]; ok {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("encode body: %w", err)
			}
			body = b
		}
	} else if len(d.bodyFields) > 0 {
		obj := map[string]any{}
		for name := range d.bodyFields {
			if v, ok := args[name]; ok {
				obj[name] = v
			}
		}
		if len(obj) > 0 {
			b, err := json.Marshal(obj)
			if err != nil {
				return nil, fmt.Errorf("encode body: %w", err)
			}
			body = b
		}
	}

	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, d.op.Method, full, reader)
	if err != nil {
		return nil, err
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if len(cookies) > 0 {
		req.Header.Set("Cookie", strings.Join(cookies, "; "))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// valToStr renders a scalar argument for a URL. JSON numbers arrive as float64;
// integral values print without a trailing ".0".
func valToStr(v any) string {
	switch n := v.(type) {
	case string:
		return n
	case bool:
		return strconv.FormatBool(n)
	case float64:
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'g', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// valToStrings expands an array argument into repeated query values; a scalar
// returns a single element.
func valToStrings(v any) []string {
	if list, ok := v.([]any); ok {
		out := make([]string, 0, len(list))
		for _, it := range list {
			out = append(out, valToStr(it))
		}
		return out
	}
	return []string{valToStr(v)}
}
