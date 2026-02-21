package db

import (
	"fmt"
	"strings"
)

type GraphNode struct {
	ID    string
	Label string
	Type  string
}

type GraphEdge struct {
	Source   string
	Target   string
	Relation string
}

func (s *Store) AddGraphNode(id, label, nodeType string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO graph_nodes (id, label, type) VALUES (?, ?, ?)", id, label, nodeType)
	return err
}

func (s *Store) AddGraphEdge(source, target, relation string) error {
	// Ensure nodes exist first? Or assume caller did it. 
	// SQLite foreign keys are enforced if PRAGMA foreign_keys=ON. 
	// For simplicity, we'll try to insert ignore on nodes if they don't exist, using label=id.
	_, _ = s.DB.Exec("INSERT OR IGNORE INTO graph_nodes (id, label, type) VALUES (?, ?, 'auto')", source, source)
	_, _ = s.DB.Exec("INSERT OR IGNORE INTO graph_nodes (id, label, type) VALUES (?, ?, 'auto')", target, target)

	_, err := s.DB.Exec("INSERT INTO graph_edges (source_id, target_id, relation) VALUES (?, ?, ?)", source, target, relation)
	return err
}

func (s *Store) QueryGraph(nodeID string) ([]string, error) {
	// Find all edges connected to this node (incoming and outgoing)
	rows, err := s.DB.Query(`
		SELECT source_id, relation, target_id FROM graph_edges 
		WHERE source_id = ? OR target_id = ?
	`, nodeID, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var src, rel, tgt string
		if err := rows.Scan(&src, &rel, &tgt); err != nil {
			return nil, err
		}
		if src == nodeID {
			results = append(results, fmt.Sprintf("-> [%s] -> %s", rel, tgt))
		} else {
			results = append(results, fmt.Sprintf("<- [%s] <- %s", rel, src))
		}
	}
	return results, nil
}

func (s *Store) VisualizeGraph() (string, error) {
	// Return a simple text representation or DOT format
	rows, err := s.DB.Query("SELECT source_id, relation, target_id FROM graph_edges LIMIT 50")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("digraph G {\n")
	for rows.Next() {
		var s, r, t string
		rows.Scan(&s, &r, &t)
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\"];\n", s, t, r))
	}
	sb.WriteString("}")
	return sb.String(), nil
}
