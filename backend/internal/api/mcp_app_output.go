package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

func parseToolOutputDisplayConfig(raw json.RawMessage) *models.OutputDisplayConfig {
	if len(raw) == 0 {
		return nil
	}
	var c models.OutputDisplayConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil
	}
	return &c
}

// isPathIndexSegment returns true if p is a 1–3 digit non-negative index (for paths like places.0.latitude).
func isPathIndexSegment(p string) bool {
	if len(p) < 1 || len(p) > 3 {
		return false
	}
	for _, r := range p {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// getValueAtPath walks obj using a dot-separated path; segments that are all digits index into []interface{}.
func getValueAtPath(obj map[string]interface{}, path string) (interface{}, bool) {
	if path == "" || obj == nil {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var cur interface{} = obj
	for _, p := range parts {
		if p == "" {
			return nil, false
		}
		if isPathIndexSegment(p) {
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 {
				return nil, false
			}
			arr, ok := cur.([]interface{})
			if !ok || idx >= len(arr) {
				return nil, false
			}
			cur = arr[idx]
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		v, ok := m[p]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func valueToDisplayString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func isAllowedHTTPURL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return false
	}
	switch u.Scheme {
	case "http", "https":
		return true
	default:
		return false
	}
}

// Preferred keys for card main content when content_key is not set (first non-empty string wins).
var cardContentKeys = []string{"joke", "text", "content", "message", "body", "description", "quote"}

// wrapToolOutputForMCPAppCard wraps a single object as MCP App card.
// If cfg provides content_key / title_key, those object properties are used; otherwise legacy heuristics apply.
func wrapToolOutputForMCPAppCard(result interface{}, cfg *models.OutputDisplayConfig) interface{} {
	obj, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}

	var content string
	if cfg != nil && cfg.ContentKey != "" {
		if v, exists := obj[cfg.ContentKey]; exists {
			content = strings.TrimSpace(valueToDisplayString(v))
		}
	}
	if content == "" {
		for _, key := range cardContentKeys {
			if v, exists := obj[key]; exists {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					content = strings.TrimSpace(s)
					break
				}
			}
		}
	}
	if content == "" {
		for _, v := range obj {
			if s, ok := v.(string); ok && len(s) > len(content) {
				content = s
			}
		}
	}
	if content == "" {
		return nil
	}

	title := "Result"
	if cfg != nil && cfg.TitleKey != "" {
		if v, ok := getValueAtPath(obj, cfg.TitleKey); ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				title = strings.TrimSpace(s)
			}
		}
	}
	if title == "Result" {
		if v, ok := obj["title"].(string); ok && strings.TrimSpace(v) != "" {
			title = strings.TrimSpace(v)
		} else if v, ok := obj["name"].(string); ok && strings.TrimSpace(v) != "" {
			title = strings.TrimSpace(v)
		}
	}

	textFallback, _ := json.Marshal(result)
	return map[string]interface{}{
		"text": string(textFallback),
		"_mcp_app": map[string]interface{}{
			"widget": "card",
			"props": map[string]interface{}{
				"content": content,
				"title":   title,
			},
		},
	}
}

var imageURLFallbackKeys = []string{"url", "image_url", "imageUrl", "image", "href", "link", "src"}

// wrapToolOutputForMCPAppImage wraps a single object as MCP App image widget (http/https URL only).
func wrapToolOutputForMCPAppImage(result interface{}, cfg *models.OutputDisplayConfig) interface{} {
	obj, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}

	var imageURL string
	if cfg != nil && cfg.ImageURLKey != "" {
		if v, ok := getValueAtPath(obj, cfg.ImageURLKey); ok {
			imageURL = strings.TrimSpace(valueToDisplayString(v))
		}
	}
	if imageURL == "" {
		for _, key := range imageURLFallbackKeys {
			if v, exists := obj[key]; exists {
				s := strings.TrimSpace(valueToDisplayString(v))
				if s != "" {
					imageURL = s
					break
				}
			}
		}
	}
	if !isAllowedHTTPURL(imageURL) {
		return nil
	}

	title := ""
	if cfg != nil && cfg.TitleKey != "" {
		if v, ok := getValueAtPath(obj, cfg.TitleKey); ok {
			if s, ok := v.(string); ok {
				title = strings.TrimSpace(s)
			}
		}
	}
	if title == "" {
		if v, ok := obj["title"].(string); ok {
			title = strings.TrimSpace(v)
		} else if v, ok := obj["name"].(string); ok {
			title = strings.TrimSpace(v)
		}
	}

	alt := title
	if alt == "" {
		alt = "Image"
	}

	textFallback, _ := json.Marshal(result)
	props := map[string]interface{}{
		"imageUrl": imageURL,
		"alt":      alt,
	}
	if title != "" {
		props["title"] = title
	}

	return map[string]interface{}{
		"text": string(textFallback),
		"_mcp_app": map[string]interface{}{
			"widget": "image",
			"props":  props,
		},
	}
}

// wrapToolOutputForMCPAppForm wraps a single object as MCP App interactive form widget.
// cfg must have submit_tool and fields (validated via NormalizeOutputDisplayConfigRaw).
func wrapToolOutputForMCPAppForm(result interface{}, cfg *models.OutputDisplayConfig) interface{} {
	if cfg == nil || cfg.SubmitTool == "" || len(cfg.Fields) == 0 {
		return nil
	}
	obj, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}

	initial := make(map[string]interface{})
	for _, f := range cfg.Fields {
		if v, exists := obj[f.Name]; exists {
			initial[f.Name] = v
		} else if f.Default != nil {
			initial[f.Name] = f.Default
		}
	}

	fieldsOut := make([]map[string]interface{}, 0, len(cfg.Fields))
	for _, f := range cfg.Fields {
		fieldsOut = append(fieldsOut, map[string]interface{}{
			"name":        f.Name,
			"label":       f.Label,
			"type":        f.Type,
			"required":    f.Required,
			"placeholder": f.Placeholder,
			"default":     f.Default,
		})
	}

	title := cfg.Title
	submitLabel := cfg.SubmitLabel
	if submitLabel == "" {
		submitLabel = "Submit"
	}

	textFallback, _ := json.Marshal(result)
	return map[string]interface{}{
		"text": string(textFallback),
		"_mcp_app": map[string]interface{}{
			"widget": "form",
			"props": map[string]interface{}{
				"title":          title,
				"submitLabel":    submitLabel,
				"submitTool":     cfg.SubmitTool,
				"fields":         fieldsOut,
				"initialValues":  initial,
			},
		},
	}
}

const (
	maxChartLabels = 64
	maxChartSeries = 8
	maxChartPoints = 256
)

func mcpAppToFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// wrapToolOutputForMCPAppChart expects result object with labels[] and datasets[{label, data[]}].
// Optional cfg: chart_type (bar|line), labels_key, datasets_key, title_key.
func wrapToolOutputForMCPAppChart(result interface{}, cfg *models.OutputDisplayConfig) interface{} {
	obj, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}

	labelsKey := "labels"
	datasetsKey := "datasets"
	chartType := "bar"
	if cfg != nil {
		if cfg.LabelsKey != "" {
			labelsKey = cfg.LabelsKey
		}
		if cfg.DatasetsKey != "" {
			datasetsKey = cfg.DatasetsKey
		}
		if cfg.ChartType == "line" {
			chartType = "line"
		}
	}

	vlab, ok := getValueAtPath(obj, labelsKey)
	if !ok {
		return nil
	}
	rawLabels, ok := vlab.([]interface{})
	if !ok || len(rawLabels) == 0 {
		return nil
	}
	labels := make([]string, 0, len(rawLabels))
	for i, v := range rawLabels {
		if i >= maxChartLabels {
			break
		}
		labels = append(labels, strings.TrimSpace(fmt.Sprint(v)))
	}
	if len(labels) == 0 {
		return nil
	}

	vds, ok := getValueAtPath(obj, datasetsKey)
	if !ok {
		return nil
	}
	rawDS, ok := vds.([]interface{})
	if !ok || len(rawDS) == 0 {
		return nil
	}

	datasets := make([]map[string]interface{}, 0, len(rawDS))
	for si, item := range rawDS {
		if si >= maxChartSeries {
			break
		}
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		lbl := ""
		if s, ok := m["label"].(string); ok {
			lbl = strings.TrimSpace(s)
		}
		rawData, ok := m["data"].([]interface{})
		if !ok || len(rawData) == 0 {
			continue
		}
		data := make([]float64, 0, len(rawData))
		for pi, p := range rawData {
			if pi >= maxChartPoints {
				break
			}
			if f, ok := mcpAppToFloat(p); ok {
				data = append(data, f)
			}
		}
		if len(data) == 0 {
			continue
		}
		datasets = append(datasets, map[string]interface{}{
			"label": lbl,
			"data":  data,
		})
	}
	if len(datasets) == 0 {
		return nil
	}

	n := len(labels)
	for _, ds := range datasets {
		arr := ds["data"].([]float64)
		if len(arr) < n {
			n = len(arr)
		}
	}
	if n <= 0 {
		return nil
	}
	labels = labels[:n]
	for i := range datasets {
		d := datasets[i]["data"].([]float64)
		if len(d) > n {
			datasets[i]["data"] = d[:n]
		}
	}

	title := ""
	if cfg != nil && cfg.TitleKey != "" {
		if v, ok := getValueAtPath(obj, cfg.TitleKey); ok {
			title = strings.TrimSpace(valueToDisplayString(v))
		}
	}
	if title == "" {
		if v, ok := obj["title"].(string); ok {
			title = strings.TrimSpace(v)
		}
	}

	textFallback, _ := json.Marshal(result)
	props := map[string]interface{}{
		"chartType": chartType,
		"labels":    labels,
		"datasets":  datasets,
	}
	if title != "" {
		props["title"] = title
	}

	return map[string]interface{}{
		"text": string(textFallback),
		"_mcp_app": map[string]interface{}{
			"widget": "chart",
			"props":  props,
		},
	}
}

const maxMapEmbedURLLen = 2048

// allowedGoogleMapsEmbedURL permits only https URLs on Google Maps hosts (defense-in-depth for iframe src).
func allowedGoogleMapsEmbedURL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > maxMapEmbedURLLen {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host != "www.google.com" && host != "google.com" && host != "maps.google.com" {
		return false
	}
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	if strings.HasPrefix(p, "/maps") {
		return true
	}
	if host == "maps.google.com" && (p == "/" || p == "") {
		q := u.Query()
		return q.Get("q") != "" || q.Get("pb") != ""
	}
	return false
}

func buildGoogleMapsEmbedURLFromCoords(lat, lng float64, zoom int) string {
	if zoom < 1 {
		zoom = 1
	}
	if zoom > 20 {
		zoom = 20
	}
	v := url.Values{}
	v.Set("q", fmt.Sprintf("%f,%f", lat, lng))
	v.Set("z", strconv.Itoa(zoom))
	v.Set("output", "embed")
	return "https://www.google.com/maps?" + v.Encode()
}

func objectLookupFloat(obj map[string]interface{}, keys []string) (float64, bool) {
	for _, k := range keys {
		if v, ok := getValueAtPath(obj, k); ok {
			if f, ok := mcpAppToFloat(v); ok {
				return f, true
			}
		}
	}
	return 0, false
}

func parseMapZoom(obj map[string]interface{}, cfg *models.OutputDisplayConfig) int {
	z := 14
	if cfg != nil && cfg.MapZoom >= 1 && cfg.MapZoom <= 20 {
		z = cfg.MapZoom
	}
	if cfg != nil && cfg.ZoomKey != "" {
		if v, ok := getValueAtPath(obj, cfg.ZoomKey); ok {
			if f, ok := mcpAppToFloat(v); ok {
				zi := int(f)
				if zi >= 1 && zi <= 20 {
					return zi
				}
			}
		}
	}
	for _, k := range []string{"zoom", "z"} {
		if v, ok := obj[k]; ok {
			if f, ok := mcpAppToFloat(v); ok {
				zi := int(f)
				if zi >= 1 && zi <= 20 {
					return zi
				}
			}
		}
	}
	return z
}

// wrapToolOutputForMCPAppMap embeds Google Maps: either embed_url (allowlisted) or lat/lng → maps?q=…&output=embed.
func wrapToolOutputForMCPAppMap(result interface{}, cfg *models.OutputDisplayConfig) interface{} {
	obj, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}

	var embedSrc string
	if cfg != nil && cfg.EmbedURLKey != "" {
		if v, ok := getValueAtPath(obj, cfg.EmbedURLKey); ok {
			if s, ok := v.(string); ok && allowedGoogleMapsEmbedURL(s) {
				embedSrc = strings.TrimSpace(s)
			}
		}
	}
	if embedSrc == "" {
		latKeys := []string{"lat", "latitude"}
		lngKeys := []string{"lng", "lon", "longitude"}
		if cfg != nil && cfg.LatKey != "" {
			latKeys = []string{cfg.LatKey}
		}
		if cfg != nil && cfg.LngKey != "" {
			lngKeys = []string{cfg.LngKey}
		}
		lat, ok1 := objectLookupFloat(obj, latKeys)
		lng, ok2 := objectLookupFloat(obj, lngKeys)
		if !ok1 || !ok2 {
			return nil
		}
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			return nil
		}
		zoom := parseMapZoom(obj, cfg)
		embedSrc = buildGoogleMapsEmbedURLFromCoords(lat, lng, zoom)
	}

	title := ""
	if cfg != nil && cfg.TitleKey != "" {
		if v, ok := getValueAtPath(obj, cfg.TitleKey); ok {
			title = strings.TrimSpace(valueToDisplayString(v))
		}
	}
	if title == "" {
		if v, ok := obj["title"].(string); ok {
			title = strings.TrimSpace(v)
		}
	}

	textFallback, _ := json.Marshal(result)
	props := map[string]interface{}{
		"embedUrl": embedSrc,
	}
	if title != "" {
		props["title"] = title
	}

	return map[string]interface{}{
		"text": string(textFallback),
		"_mcp_app": map[string]interface{}{
			"widget": "map",
			"props":  props,
		},
	}
}
