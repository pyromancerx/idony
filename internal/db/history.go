package db

import (
	"time"
)

type Activity struct {
	Timestamp time.Time
	Title     string
	Type      string // "task" or "sub-agent"
}

func (s *Store) GetRecentActivity() ([]Activity, error) {
	var activities []Activity
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")

	// Get messages (tasks)
	rows, err := s.DB.Query("SELECT timestamp, content FROM messages WHERE role = 'user' AND timestamp > ? ORDER BY timestamp DESC", yesterday)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a Activity
		var content string
		if err := rows.Scan(&a.Timestamp, &content); err != nil {
			return nil, err
		}
		a.Type = "task"
		// Summarize title (first 30 chars)
		if len(content) > 30 {
			a.Title = content[:27] + "..."
		} else {
			a.Title = content
		}
		activities = append(activities, a)
	}

	// Get sub-agent tasks
	rows, err = s.DB.Query("SELECT created_at, prompt FROM sub_agents WHERE created_at > ? ORDER BY created_at DESC", yesterday)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a Activity
		var content string
		if err := rows.Scan(&a.Timestamp, &content); err != nil {
			return nil, err
		}
		a.Type = "sub-agent"
		// Summarize title
		if len(content) > 30 {
			a.Title = content[:27] + "..."
		} else {
			a.Title = content
		}
		activities = append(activities, a)
	}

	return activities, nil
}
