package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const flowHTTPMaxBody = 4 << 20 // 4 MiB

// flowNodeData matches React Flow node.data (label, description, config).
type flowNodeData struct {
	Config struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
		// Transform
		Expression string `json:"expression"`
	} `json:"config"`
}

func parseFlowNodeData(raw json.RawMessage) (flowNodeData, error) {
	var d flowNodeData
	if err := json.Unmarshal(raw, &d); err != nil {
		return d, err
	}
	return d, nil
}

// flowSubstituteURL replaces {{key}} placeholders using string values from input object.
func flowSubstituteURL(tpl string, input json.RawMessage) (string, error) {
	var obj map[string]interface{}
	if len(input) == 0 || string(input) == "null" {
		return tpl, nil
	}
	if err := json.Unmarshal(input, &obj); err != nil {
		return tpl, err
	}
	out := tpl
	for k, v := range obj {
		placeholder := "{{" + k + "}}"
		if !strings.Contains(out, placeholder) {
			continue
		}
		var s string
		switch t := v.(type) {
		case string:
			s = t
		case float64, bool, json.Number:
			s = fmt.Sprint(t)
		default:
			b, err := json.Marshal(t)
			if err != nil {
				continue
			}
			s = string(b)
		}
		out = strings.ReplaceAll(out, placeholder, s)
	}
	return out, nil
}

func flowExecuteHTTP(ctx context.Context, nodeData json.RawMessage, input json.RawMessage) (json.RawMessage, string) {
	d, err := parseFlowNodeData(nodeData)
	if err != nil {
		return nil, "api node: invalid node data: " + err.Error()
	}
	method := strings.ToUpper(strings.TrimSpace(d.Config.Method))
	if method == "" {
		method = http.MethodGet
	}
	rawURL := strings.TrimSpace(d.Config.URL)
	if rawURL == "" {
		return nil, "api node: url is required"
	}
	finalURL, err := flowSubstituteURL(rawURL, input)
	if err != nil {
		return nil, "api node: url substitution: " + err.Error()
	}
	parsed, err := url.Parse(finalURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, "api node: only http and https URLs are allowed"
	}

	var body io.Reader
	if method != http.MethodGet && method != http.MethodHead {
		body = bytes.NewReader(input)
	}

	req, err := http.NewRequestWithContext(ctx, method, finalURL, body)
	if err != nil {
		return nil, "api node: " + err.Error()
	}
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range d.Config.Headers {
		k = strings.TrimSpace(k)
		if k != "" {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "api node: request failed: " + err.Error()
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, flowHTTPMaxBody))
	if err != nil {
		return nil, "api node: read body: " + err.Error()
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Sprintf("api node: HTTP %d: %s", resp.StatusCode, truncateRunes(string(bodyBytes), 500))
	}

	bodyBytes = bytes.TrimSpace(bodyBytes)
	if len(bodyBytes) == 0 {
		return json.RawMessage(`null`), ""
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") || json.Valid(bodyBytes) {
		var v interface{}
		if err := json.Unmarshal(bodyBytes, &v); err != nil {
			return json.RawMessage(bodyBytes), ""
		}
		out, err := json.Marshal(v)
		if err != nil {
			return json.RawMessage(bodyBytes), ""
		}
		return out, ""
	}
	// Non-JSON: wrap as string payload for downstream transforms
	wrapped, _ := json.Marshal(map[string]interface{}{
		"body":    string(bodyBytes),
		"raw":     true,
		"status":  resp.StatusCode,
		"headers": headersToSimpleMap(resp.Header),
	})
	return wrapped, ""
}

func headersToSimpleMap(h http.Header) map[string]string {
	out := make(map[string]string)
	for k, vals := range h {
		if len(vals) > 0 {
			out[k] = vals[0]
		}
	}
	return out
}

func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// flowTransformApply applies a simple dot-path expression to JSON (e.g. "origin", "args.foo").
// Empty expression, "input", or "result" passes data through. Strips a leading "return " and optional
// "result." / "input." prefix so snippets like "return result.origin" work when the param is the prior JSON.
func flowTransformApply(data json.RawMessage, expr string) (json.RawMessage, string) {
	e := strings.TrimSpace(expr)
	if e == "" {
		return data, ""
	}
	e = strings.TrimSuffix(strings.TrimSpace(e), ";")
	e = strings.TrimSpace(e)
	low := strings.ToLower(e)
	if low == "result" || low == "input" {
		return data, ""
	}
	if strings.HasPrefix(strings.ToLower(e), "return ") {
		e = strings.TrimSpace(e[7:])
		e = strings.TrimSuffix(strings.TrimSpace(e), ";")
		e = strings.TrimSpace(e)
	}
	if strings.EqualFold(e, "result") || strings.EqualFold(e, "input") {
		return data, ""
	}
	// Strip leading result. / input. (generator-style pseudo-root)
	if dot := strings.IndexByte(e, '.'); dot > 0 {
		first := strings.ToLower(e[:dot])
		if first == "result" || first == "input" {
			e = e[dot+1:]
		}
	}
	e = strings.Trim(e, ".")
	if e == "" {
		return data, ""
	}

	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, "transform: invalid JSON from previous node: " + err.Error()
	}
	parts := strings.Split(e, ".")
	cur := root
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, "transform: empty path segment"
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, fmt.Sprintf("transform: not an object at %q", p)
		}
		next, ok := m[p]
		if !ok {
			return nil, fmt.Sprintf("transform: missing path %q", e)
		}
		cur = next
	}
	out, err := json.Marshal(cur)
	if err != nil {
		return nil, "transform: " + err.Error()
	}
	return out, ""
}
