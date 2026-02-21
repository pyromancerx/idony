package db

import (
	"database/sql"
	"time"
)

type Webhook struct {
	ID             string
	Name           string
	TargetAgent    string
	PromptTemplate string
	CreatedAt      time.Time
}

func (s *Store) SaveWebhook(w Webhook) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO webhooks (id, name, target_agent, prompt_template) VALUES (?, ?, ?, ?)",
		w.ID, w.Name, w.TargetAgent, w.PromptTemplate)
	return err
}

func (s *Store) GetWebhook(id string) (*Webhook, error) {
	var w Webhook
	err := s.DB.QueryRow("SELECT id, name, target_agent, prompt_template, created_at FROM webhooks WHERE id = ?", id).
		Scan(&w.ID, &w.Name, &w.TargetAgent, &w.PromptTemplate, &w.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &w, err
}

func (s *Store) ListWebhooks() ([]Webhook, error) {
	rows, err := s.DB.Query("SELECT id, name, target_agent, prompt_template, created_at FROM webhooks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.Name, &w.TargetAgent, &w.PromptTemplate, &w.CreatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, nil
}

func (s *Store) DeleteWebhook(id string) error {
	_, err := s.DB.Exec("DELETE FROM webhooks WHERE id = ?", id)
	return err
}
