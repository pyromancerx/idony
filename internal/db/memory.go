package db

import (
	"time"
)

type Memory struct {
	ID        int
	Content   string
	Type      string
	Tags      string
	CreatedAt time.Time
}

func (s *Store) SaveMemory(content, memType, tags string) error {
	_, err := s.DB.Exec("INSERT INTO memories (content, type, tags) VALUES (?, ?, ?)", content, memType, tags)
	return err
}

func (s *Store) SearchMemories(query string, limit int) ([]Memory, error) {
	// Simple LIKE search for now
	rows, err := s.DB.Query("SELECT id, content, type, tags, created_at FROM memories WHERE content LIKE ? OR tags LIKE ? ORDER BY created_at DESC LIMIT ?", 
		"%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.Content, &m.Type, &m.Tags, &m.CreatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, nil
}

func (s *Store) GetAllMemories() ([]Memory, error) {
	rows, err := s.DB.Query("SELECT id, content, type, tags, created_at FROM memories ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.Content, &m.Type, &m.Tags, &m.CreatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, nil
}
