package db

import "time"

type MediaEntry struct {
	ID          int
	FilePath    string
	Description string
	MediaType   string
	CreatedAt   time.Time
}

func (s *Store) SaveMediaIndex(path, description, mediaType string) error {
	_, err := s.DB.Exec("INSERT INTO media_index (file_path, description, media_type) VALUES (?, ?, ?)", path, description, mediaType)
	return err
}

func (s *Store) SearchMedia(query string, limit int) ([]MediaEntry, error) {
	rows, err := s.DB.Query("SELECT id, file_path, description, media_type, created_at FROM media_index WHERE description LIKE ? ORDER BY created_at DESC LIMIT ?", 
		"%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MediaEntry
	for rows.Next() {
		var m MediaEntry
		if err := rows.Scan(&m.ID, &m.FilePath, &m.Description, &m.MediaType, &m.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, m)
	}
	return entries, nil
}
