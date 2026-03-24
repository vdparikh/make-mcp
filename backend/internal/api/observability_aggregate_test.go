package api

import (
	"testing"
	"time"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

func TestAggregateObservabilityExecutionStats(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	t1 := t0.Add(time.Hour)

	tt := []struct {
		name            string
		events          []models.ToolExecution
		wantLatency     int
		wantFailures    int
		wantRepair      int
		firstRepairTool string
	}{
		{
			name:        "empty",
			events:      nil,
			wantLatency: 0, wantFailures: 0, wantRepair: 0,
		},
		{
			name: "success_only",
			events: []models.ToolExecution{
				{ToolName: "a", ToolID: "id1", DurationMs: 10, Success: true},
				{ToolName: "a", ToolID: "id1", DurationMs: 30, Success: true},
			},
			wantLatency: 1, wantFailures: 0, wantRepair: 0,
		},
		{
			name: "failure_with_repair",
			events: []models.ToolExecution{
				{ToolName: "x", ToolID: "x1", Success: false, Error: "e1", RepairSuggestion: "fix1", CreatedAt: t0},
				{ToolName: "x", ToolID: "x1", Success: false, Error: "e2", RepairSuggestion: "fix2", CreatedAt: t1},
			},
			wantLatency: 1, wantFailures: 1, wantRepair: 2,
			// Newest repair first after sort
			firstRepairTool: "x",
		},
		{
			name: "tool_key_falls_back_to_id",
			events: []models.ToolExecution{
				{ToolName: "", ToolID: "only-id", DurationMs: 5, Success: true},
			},
			wantLatency: 1, wantFailures: 0, wantRepair: 0,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lat, fail, rep := aggregateObservabilityExecutionStats(tc.events)
			if len(lat) != tc.wantLatency {
				t.Fatalf("latency count: got %d want %d", len(lat), tc.wantLatency)
			}
			if len(fail) != tc.wantFailures {
				t.Fatalf("failure count: got %d want %d", len(fail), tc.wantFailures)
			}
			if len(rep) != tc.wantRepair {
				t.Fatalf("repair count: got %d want %d", len(rep), tc.wantRepair)
			}
			if tc.wantRepair > 0 && rep[0].ToolName != tc.firstRepairTool {
				t.Fatalf("first repair tool: got %q want %q", rep[0].ToolName, tc.firstRepairTool)
			}
			if tc.name == "failure_with_repair" && len(rep) >= 2 && rep[0].Suggestion != "fix2" {
				t.Fatalf("expected newest suggestion first, got %q", rep[0].Suggestion)
			}
		})
	}
}

func TestAggregateObservabilityExecutionStatsRepairCap(t *testing.T) {
	t.Parallel()
	base := time.Now().UTC()
	var evs []models.ToolExecution
	for i := range 25 {
		evs = append(evs, models.ToolExecution{
			ToolName: "t", ToolID: "id", Success: false,
			RepairSuggestion: "s", CreatedAt: base.Add(time.Duration(i) * time.Minute),
		})
	}
	_, _, rep := aggregateObservabilityExecutionStats(evs)
	if len(rep) != 20 {
		t.Fatalf("repair cap: got %d want 20", len(rep))
	}
}
