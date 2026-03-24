package api

import (
	"sort"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// aggregateObservabilityExecutionStats builds latency, failure, and repair-suggestion summaries from raw events.
func aggregateObservabilityExecutionStats(events []models.ToolExecution) (
	latencyList []models.ToolLatencyStat,
	failureList []models.ToolFailureStat,
	repairSuggestions []models.RepairSuggestionItem,
) {
	latencyByTool := make(map[string]*models.ToolLatencyStat)
	failuresByTool := make(map[string]*models.ToolFailureStat)
	for i := range events {
		e := &events[i]
		toolKey := e.ToolName
		if toolKey == "" {
			toolKey = e.ToolID
		}
		if stat, ok := latencyByTool[toolKey]; ok {
			stat.Count++
			stat.AvgMs = (stat.AvgMs*float64(stat.Count-1) + float64(e.DurationMs)) / float64(stat.Count)
			if e.DurationMs > stat.P95Ms {
				stat.P95Ms = e.DurationMs
			}
		} else {
			latencyByTool[toolKey] = &models.ToolLatencyStat{ToolName: e.ToolName, ToolID: e.ToolID, Count: 1, AvgMs: float64(e.DurationMs), P95Ms: e.DurationMs}
		}
		if !e.Success {
			if f, ok := failuresByTool[toolKey]; ok {
				f.Count++
				if e.Error != "" {
					f.LastError = e.Error
				}
			} else {
				failuresByTool[toolKey] = &models.ToolFailureStat{ToolName: e.ToolName, ToolID: e.ToolID, Count: 1, LastError: e.Error}
			}
			if e.RepairSuggestion != "" {
				repairSuggestions = append(repairSuggestions, models.RepairSuggestionItem{
					ToolName: e.ToolName, ToolID: e.ToolID, Suggestion: e.RepairSuggestion, CreatedAt: e.CreatedAt,
				})
			}
		}
	}
	latencyList = make([]models.ToolLatencyStat, 0, len(latencyByTool))
	for _, s := range latencyByTool {
		latencyList = append(latencyList, *s)
	}
	failureList = make([]models.ToolFailureStat, 0, len(failuresByTool))
	for _, f := range failuresByTool {
		failureList = append(failureList, *f)
	}
	sort.Slice(repairSuggestions, func(i, j int) bool { return repairSuggestions[j].CreatedAt.Before(repairSuggestions[i].CreatedAt) })
	if len(repairSuggestions) > 20 {
		repairSuggestions = repairSuggestions[:20]
	}
	return latencyList, failureList, repairSuggestions
}
