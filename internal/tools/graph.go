package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pyromancer/idony/internal/db"
)

type GraphAddTool struct {
	store *db.Store
}

func NewGraphAddTool(store *db.Store) *GraphAddTool {
	return &GraphAddTool{store: store}
}

func (g *GraphAddTool) Name() string {
	return "graph_add"
}

func (g *GraphAddTool) Description() string {
	return `Adds a relationship to the knowledge graph.
Input: {"source": "EntityA", "relation": "is_a", "target": "EntityB", "source_type": "concept", "target_type": "concept"}`
}

func (g *GraphAddTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Source     string `json:"source"`
		Relation   string `json:"relation"`
		Target     string `json:"target"`
		SourceType string `json:"source_type"`
		TargetType string `json:"target_type"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if req.Source == "" || req.Target == "" || req.Relation == "" {
		return "", fmt.Errorf("source, target, and relation are required")
	}

	// Ensure nodes exist (update/create)
	_ = g.store.AddGraphNode(req.Source, req.Source, req.SourceType)
	_ = g.store.AddGraphNode(req.Target, req.Target, req.TargetType)

	err := g.store.AddGraphEdge(req.Source, req.Target, req.Relation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Graph updated: (%s) -[%s]-> (%s)", req.Source, req.Relation, req.Target), nil
}

func (g *GraphAddTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Add to Graph",
		"fields": []map[string]interface{}{
			{"name": "source", "label": "Source Entity", "type": "string", "required": true},
			{"name": "relation", "label": "Relation", "type": "string", "required": true},
			{"name": "target", "label": "Target Entity", "type": "string", "required": true},
			{"name": "source_type", "label": "Source Type", "type": "string", "hint": "concept"},
			{"name": "target_type", "label": "Target Type", "type": "string", "hint": "concept"},
		},
	}
}

type GraphQueryTool struct {
	store *db.Store
}

func NewGraphQueryTool(store *db.Store) *GraphQueryTool {
	return &GraphQueryTool{store: store}
}

func (g *GraphQueryTool) Name() string {
	return "graph_query"
}

func (g *GraphQueryTool) Description() string {
	return "Queries the knowledge graph for connections to an entity. Input: entity ID."
}

func (g *GraphQueryTool) Execute(ctx context.Context, input string) (string, error) {
	results, err := g.store.QueryGraph(strings.TrimSpace(input))
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No connections found.", nil
	}
	return "Connections:\n" + strings.Join(results, "\n"), nil
}

func (g *GraphQueryTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Query Graph",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Entity Name", "type": "string", "required": true},
		},
	}
}
