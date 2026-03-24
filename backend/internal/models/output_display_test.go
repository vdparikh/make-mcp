package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeOutputDisplayFieldKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"title", "title"},
		{"  url  ", "url"},
		{"", ""},
		{"a-b", ""},
		{".x", ""},
		{"places.lat", ""},
		{strings.Repeat("a", 200), ""},
	}
	for _, tt := range tests {
		if got := SanitizeOutputDisplayFieldKey(tt.in); got != tt.want {
			t.Errorf("SanitizeOutputDisplayFieldKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSanitizeOutputDisplayPathKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"places.latitude", "places.latitude"},
		{"places.0.latitude", "places.0.latitude"},
		{"data.series.0.label", "data.series.0.label"},
		{"", ""},
		{"..a", ""},
		{"a..b", ""},
		{".bad", ""},
	}
	for _, tt := range tests {
		if got := SanitizeOutputDisplayPathKey(tt.in); got != tt.want {
			t.Errorf("SanitizeOutputDisplayPathKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeOutputDisplay(t *testing.T) {
	if got := NormalizeOutputDisplay("image"); got != OutputDisplayImage {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeOutputDisplay("form"); got != OutputDisplayForm {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeOutputDisplay("chart"); got != OutputDisplayChart {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeOutputDisplay("map"); got != OutputDisplayMap {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeOutputDisplay("nope"); got != OutputDisplayJSON {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeOutputDisplayConfigRaw(t *testing.T) {
	raw, err := NormalizeOutputDisplayConfigRaw(json.RawMessage(`{"content_key":"body","title_key":"x","image_url_key":"../bad"}`))
	if err != nil {
		t.Fatal(err)
	}
	var c OutputDisplayConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if c.ContentKey != "body" {
		t.Fatalf("content_key: %+v", c)
	}
	if c.ImageURLKey != "" {
		t.Fatalf("invalid key should be cleared: %+v", c)
	}
}

func TestNormalizeOutputDisplayConfigRaw_Form(t *testing.T) {
	raw, err := NormalizeOutputDisplayConfigRaw(json.RawMessage(`{
		"submit_tool": "save_prefs",
		"title": "Preferences",
		"submit_label": "Save",
		"fields": [
			{"name": "q", "label": "Query", "type": "text", "required": true},
			{"name": "x", "label": "Bad", "type": "invalid_type"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	var c OutputDisplayConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if c.SubmitTool != "save_prefs" {
		t.Fatalf("submit_tool: %+v", c)
	}
	if len(c.Fields) != 2 {
		t.Fatalf("fields len: %+v", c.Fields)
	}
	if c.Fields[1].Type != "text" {
		t.Fatalf("invalid type should become text: %+v", c.Fields[1])
	}
}

func TestNormalizeOutputDisplayConfigRaw_FormIncompleteDropped(t *testing.T) {
	raw, err := NormalizeOutputDisplayConfigRaw(json.RawMessage(`{"submit_tool":"","fields":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if raw != nil {
		t.Fatalf("expected nil, got %s", string(raw))
	}
}

func TestNormalizeOutputDisplayConfigRaw_Chart(t *testing.T) {
	raw, err := NormalizeOutputDisplayConfigRaw(json.RawMessage(`{
		"chart_type": "line",
		"labels_key": "x_labels",
		"datasets_key": "series"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	var c OutputDisplayConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if c.ChartType != "line" {
		t.Fatalf("chart_type: %+v", c)
	}
	if c.LabelsKey != "x_labels" || c.DatasetsKey != "series" {
		t.Fatalf("keys: %+v", c)
	}
}

func TestNormalizeOutputDisplayConfigRaw_Map(t *testing.T) {
	raw, err := NormalizeOutputDisplayConfigRaw(json.RawMessage(`{
		"lat_key": "latitude",
		"lng_key": "longitude",
		"zoom": 12
	}`))
	if err != nil {
		t.Fatal(err)
	}
	var c OutputDisplayConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if c.LatKey != "latitude" || c.LngKey != "longitude" || c.MapZoom != 12 {
		t.Fatalf("map config: %+v", c)
	}
}
