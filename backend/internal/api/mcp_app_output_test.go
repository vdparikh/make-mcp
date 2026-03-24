package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

func TestWrapToolOutputForMCPAppCard_ConfigKeys(t *testing.T) {
	cfg := &models.OutputDisplayConfig{ContentKey: "body", TitleKey: "heading"}
	raw := map[string]interface{}{"body": "hello", "heading": "H1", "extra": "x"}
	out := wrapToolOutputForMCPAppCard(raw, cfg)
	if out == nil {
		t.Fatal("expected wrap")
	}
	m := out.(map[string]interface{})
	app := m["_mcp_app"].(map[string]interface{})
	props := app["props"].(map[string]interface{})
	if props["content"] != "hello" {
		t.Fatalf("content: %v", props["content"])
	}
	if props["title"] != "H1" {
		t.Fatalf("title: %v", props["title"])
	}
}

func TestWrapToolOutputForMCPAppImage_URLKey(t *testing.T) {
	cfg := &models.OutputDisplayConfig{ImageURLKey: "media", TitleKey: "t"}
	raw := map[string]interface{}{"media": "https://example.com/x.png", "t": "Pic"}
	out := wrapToolOutputForMCPAppImage(raw, cfg)
	if out == nil {
		t.Fatal("expected wrap")
	}
	b, _ := json.Marshal(out)
	if !json.Valid(b) {
		t.Fatal("invalid json")
	}
}

func TestWrapToolOutputForMCPAppImage_RejectsNonHTTP(t *testing.T) {
	cfg := &models.OutputDisplayConfig{ImageURLKey: "u"}
	raw := map[string]interface{}{"u": "javascript:alert(1)"}
	if wrapToolOutputForMCPAppImage(raw, cfg) != nil {
		t.Fatal("expected nil")
	}
}

func TestWrapToolOutputForMCPAppMap_LatLng(t *testing.T) {
	raw := map[string]interface{}{"lat": 37.7749, "lng": -122.4194, "title": "SF"}
	out := wrapToolOutputForMCPAppMap(raw, nil)
	if out == nil {
		t.Fatal("expected wrap")
	}
	m := out.(map[string]interface{})
	app := m["_mcp_app"].(map[string]interface{})
	if app["widget"] != "map" {
		t.Fatalf("widget: %v", app["widget"])
	}
	props := app["props"].(map[string]interface{})
	u := props["embedUrl"].(string)
	if !strings.Contains(u, "google.com/maps") || !strings.Contains(u, "output=embed") {
		t.Fatalf("embedUrl: %s", u)
	}
	if props["title"] != "SF" {
		t.Fatalf("title: %v", props["title"])
	}
}

func TestWrapToolOutputForMCPAppMap_EmbedURLKey(t *testing.T) {
	cfg := &models.OutputDisplayConfig{EmbedURLKey: "map_url"}
	good := "https://www.google.com/maps/embed?pb=test"
	raw := map[string]interface{}{"map_url": good}
	out := wrapToolOutputForMCPAppMap(raw, cfg)
	if out == nil {
		t.Fatal("expected wrap")
	}
	m := out.(map[string]interface{})
	props := m["_mcp_app"].(map[string]interface{})["props"].(map[string]interface{})
	if props["embedUrl"].(string) != good {
		t.Fatalf("embedUrl: %v", props["embedUrl"])
	}
}

func TestWrapToolOutputForMCPAppMap_RejectsBadEmbedURL(t *testing.T) {
	cfg := &models.OutputDisplayConfig{EmbedURLKey: "map_url"}
	raw := map[string]interface{}{"map_url": "https://evil.example.com/maps"}
	if wrapToolOutputForMCPAppMap(raw, cfg) != nil {
		t.Fatal("expected nil")
	}
}

func TestWrapToolOutputForMCPAppMap_NestedArrayPath(t *testing.T) {
	cfg := &models.OutputDisplayConfig{
		LatKey:  "places.0.latitude",
		LngKey:  "places.0.longitude",
		MapZoom: 14,
	}
	raw := map[string]interface{}{
		"places": []interface{}{
			map[string]interface{}{
				"latitude":  "37.7749",
				"longitude": "-122.4194",
			},
		},
	}
	out := wrapToolOutputForMCPAppMap(raw, cfg)
	if out == nil {
		t.Fatal("expected wrap")
	}
	m := out.(map[string]interface{})
	props := m["_mcp_app"].(map[string]interface{})["props"].(map[string]interface{})
	if props["embedUrl"] == nil || props["embedUrl"] == "" {
		t.Fatalf("embedUrl: %v", props["embedUrl"])
	}
}
