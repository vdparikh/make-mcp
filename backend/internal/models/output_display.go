package models

import (
	"encoding/json"
	"regexp"
	"strings"
)

// OutputDisplayConfig maps tool result fields to MCP Apps widgets (card / image / form).
type OutputDisplayConfig struct {
	ContentKey  string `json:"content_key,omitempty"`   // main text for card
	TitleKey    string `json:"title_key,omitempty"`     // headline for card or image
	ImageURLKey string `json:"image_url_key,omitempty"` // URL string field for image widget

	// Form widget: first tool returns field definitions + optional defaults from result object;
	// submit_tool is the name of another tool on the same server to call with form values.
	SubmitTool  string       `json:"submit_tool,omitempty"`
	Title       string       `json:"title,omitempty"`        // form heading (optional)
	SubmitLabel string       `json:"submit_label,omitempty"` // submit button label
	Fields      []FormField  `json:"fields,omitempty"`

	// Chart widget: tool result object includes labels + datasets (see wrapToolOutputForMCPAppChart).
	ChartType   string `json:"chart_type,omitempty"`   // bar | line
	LabelsKey   string `json:"labels_key,omitempty"`   // default "labels"
	DatasetsKey string `json:"datasets_key,omitempty"` // default "datasets"

	// Map widget (Google Maps embed): lat/lng on result object or embed_url from tool (see wrapToolOutputForMCPAppMap).
	LatKey      string `json:"lat_key,omitempty"`       // default lat, latitude
	LngKey      string `json:"lng_key,omitempty"`       // default lng, lon, longitude
	ZoomKey     string `json:"zoom_key,omitempty"`      // optional numeric field for zoom 1–20
	EmbedURLKey string `json:"embed_url_key,omitempty"` // optional field with full https://…google…/maps… URL
	MapZoom     int    `json:"zoom,omitempty"`          // default zoom when not using zoom_key (1–20)
}

// FormField describes one input in the MCP App form widget.
type FormField struct {
	Name        string      `json:"name"`
	Label       string      `json:"label"`
	Type        string      `json:"type"` // text, textarea, boolean, number, date, time, datetime-local, color
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Placeholder string      `json:"placeholder,omitempty"`
}

var outputDisplayFieldKeyRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,127}$`)

var pathSegmentIdentRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,127}$`)

// pathSegmentIndexRE matches array indices in paths (e.g. places.0.latitude).
var pathSegmentIndexRE = regexp.MustCompile(`^[0-9]{1,3}$`)

// SanitizeOutputDisplayFieldKey returns key if it is a safe single object property name (form field names), else empty.
func SanitizeOutputDisplayFieldKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || !outputDisplayFieldKeyRE.MatchString(s) {
		return ""
	}
	return s
}

// SanitizeOutputDisplayPathKey returns a dotted path for mapping tool result fields (card, chart, map, etc.).
// Each segment is either a safe identifier ([a-zA-Z][a-zA-Z0-9_]*) or a 1–3 digit array index.
// Rejects "..", empty segments, and overly long paths.
func SanitizeOutputDisplayPathKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 256 || strings.Contains(s, "..") {
		return ""
	}
	parts := strings.Split(s, ".")
	if len(parts) > 24 {
		return ""
	}
	for _, p := range parts {
		if p == "" {
			return ""
		}
		if pathSegmentIndexRE.MatchString(p) {
			continue
		}
		if !pathSegmentIdentRE.MatchString(p) {
			return ""
		}
	}
	return s
}

var allowedFormFieldTypes = map[string]struct{}{
	"text": {}, "textarea": {}, "boolean": {}, "number": {},
	"date": {}, "time": {}, "datetime-local": {}, "color": {},
}

const maxFormFields = 50

// SanitizeSubmitToolName returns a valid MCP tool name or empty.
func SanitizeSubmitToolName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || !outputDisplayFieldKeyRE.MatchString(s) {
		return ""
	}
	return s
}

// SanitizeFormFields trims and bounds form field definitions from untrusted JSON.
func SanitizeFormFields(fields []FormField) []FormField {
	if len(fields) == 0 {
		return nil
	}
	out := make([]FormField, 0, len(fields))
	for i, f := range fields {
		if i >= maxFormFields {
			break
		}
		name := SanitizeOutputDisplayFieldKey(f.Name)
		if name == "" {
			continue
		}
		label := strings.TrimSpace(f.Label)
		if len(label) > 200 {
			label = label[:200]
		}
		if label == "" {
			label = name
		}
		ft := strings.TrimSpace(strings.ToLower(f.Type))
		if _, ok := allowedFormFieldTypes[ft]; !ok {
			ft = "text"
		}
		ph := strings.TrimSpace(f.Placeholder)
		if len(ph) > 500 {
			ph = ph[:500]
		}
		out = append(out, FormField{
			Name:        name,
			Label:       label,
			Type:        ft,
			Default:     f.Default,
			Required:    f.Required,
			Placeholder: ph,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NormalizeOutputDisplay returns a supported output_display value or json.
func NormalizeOutputDisplay(s string) string {
	switch strings.TrimSpace(s) {
	case OutputDisplayTable, OutputDisplayCard, OutputDisplayImage, OutputDisplayForm, OutputDisplayChart, OutputDisplayMap:
		return strings.TrimSpace(s)
	default:
		return OutputDisplayJSON
	}
}

// NormalizeOutputDisplayConfigRaw parses untrusted JSON, sanitizes keys, and returns nil if empty.
func NormalizeOutputDisplayConfigRaw(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var c OutputDisplayConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.ContentKey = SanitizeOutputDisplayPathKey(c.ContentKey)
	c.TitleKey = SanitizeOutputDisplayPathKey(c.TitleKey)
	c.ImageURLKey = SanitizeOutputDisplayPathKey(c.ImageURLKey)

	c.SubmitTool = SanitizeSubmitToolName(c.SubmitTool)
	c.Fields = SanitizeFormFields(c.Fields)
	c.Title = strings.TrimSpace(c.Title)
	if len(c.Title) > 200 {
		c.Title = c.Title[:200]
	}
	c.SubmitLabel = strings.TrimSpace(c.SubmitLabel)
	if len(c.SubmitLabel) > 80 {
		c.SubmitLabel = c.SubmitLabel[:80]
	}

	hasForm := c.SubmitTool != "" && len(c.Fields) > 0
	if !hasForm {
		c.SubmitTool = ""
		c.Fields = nil
		c.Title = ""
		c.SubmitLabel = ""
	}

	ct := strings.ToLower(strings.TrimSpace(c.ChartType))
	if ct == "bar" || ct == "line" {
		c.ChartType = ct
	} else {
		c.ChartType = ""
	}
	c.LabelsKey = SanitizeOutputDisplayPathKey(c.LabelsKey)
	c.DatasetsKey = SanitizeOutputDisplayPathKey(c.DatasetsKey)

	hasChart := c.ChartType != "" || c.LabelsKey != "" || c.DatasetsKey != ""
	if !hasChart {
		c.ChartType = ""
		c.LabelsKey = ""
		c.DatasetsKey = ""
	}

	c.LatKey = SanitizeOutputDisplayPathKey(c.LatKey)
	c.LngKey = SanitizeOutputDisplayPathKey(c.LngKey)
	c.ZoomKey = SanitizeOutputDisplayPathKey(c.ZoomKey)
	c.EmbedURLKey = SanitizeOutputDisplayPathKey(c.EmbedURLKey)
	if c.MapZoom < 1 || c.MapZoom > 20 {
		c.MapZoom = 0
	}

	hasMap := c.LatKey != "" || c.LngKey != "" || c.ZoomKey != "" || c.EmbedURLKey != "" || c.MapZoom != 0
	if !hasMap {
		c.LatKey = ""
		c.LngKey = ""
		c.ZoomKey = ""
		c.EmbedURLKey = ""
		c.MapZoom = 0
	}

	hasCard := c.ContentKey != "" || c.TitleKey != "" || c.ImageURLKey != ""
	if !hasCard && !hasForm && !hasChart && !hasMap {
		return nil, nil
	}
	return json.Marshal(c)
}
