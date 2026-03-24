package api

import (
	"encoding/json"
	"testing"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

func TestValidateServerJSONExportPayload(t *testing.T) {
	t.Parallel()

	base := ServerJSONExportPayload{
		SchemaVersion: serverJSONExportSchemaVersion,
		Server: ServerExportMeta{
			Name:        "demo",
			Description: "desc",
			Version:     "1.0.0",
		},
		Tools: []ExportTool{
			{
				Name:          "get_weather",
				ExecutionType: models.ExecutionTypeRestAPI,
			},
		},
	}

	cases := []struct {
		name    string
		payload ServerJSONExportPayload
		wantErr bool
	}{
		{name: "valid minimal", payload: base, wantErr: false},
		{
			name: "missing server name",
			payload: func() ServerJSONExportPayload {
				p := base
				p.Server.Name = ""
				return p
			}(),
			wantErr: true,
		},
		{
			name: "missing server version",
			payload: func() ServerJSONExportPayload {
				p := base
				p.Server.Version = ""
				return p
			}(),
			wantErr: true,
		},
		{
			name: "duplicate tool names",
			payload: func() ServerJSONExportPayload {
				p := base
				p.Tools = append(p.Tools, ExportTool{Name: "get_weather", ExecutionType: models.ExecutionTypeWebhook})
				return p
			}(),
			wantErr: true,
		},
		{
			name: "invalid policy rule fail action",
			payload: func() ServerJSONExportPayload {
				p := base
				p.Policies = []ExportPolicy{
					{
						ToolName: "get_weather",
						Name:     "p1",
						Rules: []ExportPolicyRule{
							{Type: models.PolicyRuleCustom, Priority: 1, FailAction: "block"},
						},
					},
				}
				return p
			}(),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateServerJSONExportPayload(&tc.payload)
			if tc.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestEnsureJSONObjectRaw(t *testing.T) {
	t.Parallel()

	got := ensureJSONObjectRaw(nil)
	if string(got) != "{}" {
		t.Fatalf("nil raw = %q, want {}", string(got))
	}

	in := json.RawMessage(`{"a":1}`)
	got = ensureJSONObjectRaw(in)
	if string(got) != string(in) {
		t.Fatalf("non-empty raw changed: got %s want %s", string(got), string(in))
	}
}

func TestParseImportServerJSONBody(t *testing.T) {
	t.Parallel()

	rawPayload := []byte(`{
		"schema_version": 1,
		"server": {"name":"x","version":"1.0.0"},
		"tools": [{"name":"t1","execution_type":"rest_api"}],
		"resources": [],
		"prompts": [],
		"context_configs": [],
		"policies": []
	}`)

	payload, wrapped, err := parseImportServerJSONBody(rawPayload)
	if err != nil {
		t.Fatalf("raw payload parse err: %v", err)
	}
	if payload.SchemaVersion != serverJSONExportSchemaVersion {
		t.Fatalf("schema_version = %d", payload.SchemaVersion)
	}
	if wrapped.Payload.SchemaVersion != 0 {
		t.Fatalf("expected empty wrapped payload for raw input, got %d", wrapped.Payload.SchemaVersion)
	}

	wrappedPayload := []byte(`{
		"payload": {
			"schema_version": 1,
			"server": {"name":"x","version":"1.0.0"},
			"tools": [{"name":"t1","execution_type":"rest_api"}],
			"resources": [],
			"prompts": [],
			"context_configs": [],
			"policies": []
		},
		"server_name_override": "clone-x"
	}`)

	payload, wrapped, err = parseImportServerJSONBody(wrappedPayload)
	if err != nil {
		t.Fatalf("wrapped payload parse err: %v", err)
	}
	if payload.SchemaVersion != serverJSONExportSchemaVersion {
		t.Fatalf("wrapped schema_version = %d", payload.SchemaVersion)
	}
	if wrapped.ServerNameOverride == nil || *wrapped.ServerNameOverride != "clone-x" {
		t.Fatalf("server_name_override not parsed: %#v", wrapped.ServerNameOverride)
	}

	if _, _, err := parseImportServerJSONBody([]byte(`{`)); err == nil {
		t.Fatal("expected parse error for invalid JSON")
	}
}
