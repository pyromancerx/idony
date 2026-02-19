package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // Using CGO-free sqlite
)

type Message struct {
	ID        int
	Role      string
	Content   string
	Timestamp time.Time
}

type Store struct {
	DB *sql.DB
}

// NewStore initializes a new SQLite store with the required tables.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables if they don't exist
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS scheduled_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_type TEXT NOT NULL, -- "one-shot" or "recurring"
		schedule TEXT NOT NULL,  -- Cron string or RFC3339 timestamp
		prompt TEXT NOT NULL,    -- The prompt Idony should run
		last_run DATETIME
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS sub_agents (
		id TEXT PRIMARY KEY,
		prompt TEXT NOT NULL,
		status TEXT NOT NULL, -- "running", "completed", "failed"
		progress INTEGER DEFAULT 0,
		result TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &Store{DB: db}, nil
}

type SubAgentTask struct {
	ID         string
	Prompt     string
	Status     string
	Progress   int
	Result     string
	CreatedAt  time.Time
	FinishedAt *time.Time
}

func (s *Store) SaveSubAgent(id, prompt, status string) error {
	_, err := s.DB.Exec("INSERT INTO sub_agents (id, prompt, status, progress) VALUES (?, ?, ?, 0)", id, prompt, status)
	return err
}

func (s *Store) UpdateSubAgentProgress(id string, progress int) error {
	_, err := s.DB.Exec("UPDATE sub_agents SET progress = ? WHERE id = ?", progress, id)
	return err
}

func (s *Store) UpdateSubAgent(id, status, result string) error {
	_, err := s.DB.Exec("UPDATE sub_agents SET status = ?, result = ?, progress = 100, finished_at = CURRENT_TIMESTAMP WHERE id = ?", status, result, id)
	return err
}

func (s *Store) GetActiveSubAgents() ([]SubAgentTask, error) {
	rows, err := s.DB.Query("SELECT id, prompt, status, progress, COALESCE(result, ''), created_at, finished_at FROM sub_agents WHERE status = 'running' ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []SubAgentTask
	for rows.Next() {
		var t SubAgentTask
		if err := rows.Scan(&t.ID, &t.Prompt, &t.Status, &t.Progress, &t.Result, &t.CreatedAt, &t.FinishedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Store) GetSubAgents() ([]SubAgentTask, error) {
	rows, err := s.DB.Query("SELECT id, prompt, status, progress, COALESCE(result, ''), created_at, finished_at FROM sub_agents ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []SubAgentTask
	for rows.Next() {
		var t SubAgentTask
		if err := rows.Scan(&t.ID, &t.Prompt, &t.Status, &t.Result, &t.CreatedAt, &t.FinishedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Store) GetSetting(key string) (string, error) {
	var val string
	err := s.DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

type ScheduledTask struct {
	ID       int
	Type     string
	Schedule string
	Prompt   string
	LastRun  *time.Time
}

func (s *Store) SaveTask(taskType, schedule, prompt string) error {
	_, err := s.DB.Exec("INSERT INTO scheduled_tasks (task_type, schedule, prompt) VALUES (?, ?, ?)", taskType, schedule, prompt)
	return err
}

func (s *Store) LoadTasks() ([]ScheduledTask, error) {
	rows, err := s.DB.Query("SELECT id, task_type, schedule, prompt, last_run FROM scheduled_tasks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []ScheduledTask
	for rows.Next() {
		var t ScheduledTask
		if err := rows.Scan(&t.ID, &t.Type, &t.Schedule, &t.Prompt, &t.LastRun); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Store) UpdateTaskLastRun(id int) error {
	_, err := s.DB.Exec("UPDATE scheduled_tasks SET last_run = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (s *Store) DeleteTask(id int) error {
	_, err := s.DB.Exec("DELETE FROM scheduled_tasks WHERE id = ?", id)
	return err
}

// SaveMessage persists a message into the database.
func (s *Store) SaveMessage(role, content string) error {
	_, err := s.DB.Exec("INSERT INTO messages (role, content) VALUES (?, ?)", role, content)
	return err
}

// LoadLastMessages retrieves the most recent n messages.
func (s *Store) LoadLastMessages(limit int) ([]Message, error) {
	rows, err := s.DB.Query("SELECT id, role, content, timestamp FROM messages ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.Timestamp); err != nil {
			return nil, err
		}
		// prepend to keep order correct
		msgs = append([]Message{m}, msgs...)
	}

	return msgs, nil
}
