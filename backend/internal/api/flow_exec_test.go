package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFlowSubstituteURL(t *testing.T) {
	t.Parallel()
	out, err := flowSubstituteURL(`https://ex.com/{{zip}}`, json.RawMessage(`{"zip":"94538"}`))
	if err != nil {
		t.Fatal(err)
	}
	if out != "https://ex.com/94538" {
		t.Fatalf("got %q", out)
	}
}

func TestFlowExecuteHTTP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"origin":"1.2.3.4","url":"https://ex/get"}`))
	}))
	defer srv.Close()

	node, _ := json.Marshal(map[string]interface{}{
		"label": "API",
		"config": map[string]interface{}{
			"method": "GET",
			"url":    srv.URL + "/get",
		},
	})
	raw, errStr := flowExecuteHTTP(context.Background(), node, json.RawMessage(`{}`))
	if errStr != "" {
		t.Fatal(errStr)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["origin"] != "1.2.3.4" {
		t.Fatalf("origin: %v", got["origin"])
	}
}

func TestFlowTransformApply(t *testing.T) {
	t.Parallel()
	in := json.RawMessage(`{"origin":"9.9.9.9","args":{}}`)

	t.Run("passthrough", func(t *testing.T) {
		t.Parallel()
		out, errStr := flowTransformApply(in, "")
		if errStr != "" {
			t.Fatal(errStr)
		}
		if string(out) != string(in) {
			t.Fatalf("got %s", out)
		}
	})

	t.Run("return_result_semicolon", func(t *testing.T) {
		t.Parallel()
		out, errStr := flowTransformApply(in, "return result;")
		if errStr != "" {
			t.Fatal(errStr)
		}
		if string(out) != string(in) {
			t.Fatalf("got %s", out)
		}
	})

	tests := []struct {
		name string
		expr string
		want string
		err  bool
	}{
		{"pick origin", "origin", `"9.9.9.9"`, false},
		{"return result.origin", "return result.origin", `"9.9.9.9"`, false},
		{"args object", "args", `{}`, false},
		{"missing", "missing", ``, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, errStr := flowTransformApply(in, tc.expr)
			if tc.err {
				if errStr == "" {
					t.Fatal("expected error")
				}
				return
			}
			if errStr != "" {
				t.Fatal(errStr)
			}
			if string(out) != tc.want {
				t.Fatalf("got %s want %s", out, tc.want)
			}
		})
	}
}
