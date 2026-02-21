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
		target_type TEXT DEFAULT 'main',
		target_name TEXT,
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
		model TEXT,
		personality TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS sub_agent_definitions (
		name TEXT PRIMARY KEY,
		personality TEXT NOT NULL,
		tools TEXT NOT NULL, -- Comma-separated list of tool names
		model TEXT           -- Optional model override
	);
	CREATE TABLE IF NOT EXISTS councils (
		name TEXT PRIMARY KEY,
		members TEXT NOT NULL -- Comma-separated list of sub-agent names
	);
	CREATE TABLE IF NOT EXISTS rss_feeds (
		url TEXT PRIMARY KEY,
		title TEXT,
		category TEXT
	);
	CREATE TABLE IF NOT EXISTS processed_rss_items (
		guid TEXT PRIMARY KEY,
		feed_url TEXT,
		processed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(feed_url) REFERENCES rss_feeds(url)
	);
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'planning',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		parent_id TEXT,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'pending',
		assigned_agent TEXT,
		result TEXT,
		FOREIGN KEY(project_id) REFERENCES projects(id),
		FOREIGN KEY(parent_id) REFERENCES tasks(id)
	);
	CREATE TABLE IF NOT EXISTS knowledge_base (
		key TEXT PRIMARY KEY,
		category TEXT,
		content TEXT NOT NULL,
		tags TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		type TEXT DEFAULT 'fact', -- fact, preference, observation
		tags TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS graph_nodes (
		id TEXT PRIMARY KEY,
		label TEXT NOT NULL,
		type TEXT DEFAULT 'concept',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS graph_edges (
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		relation TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(source_id) REFERENCES graph_nodes(id),
		FOREIGN KEY(target_id) REFERENCES graph_nodes(id)
	);
	CREATE TABLE IF NOT EXISTS media_index (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT,
		description TEXT, -- transcript or visual description
		media_type TEXT, -- image, audio, video
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS agent_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_agent TEXT,
		to_agent TEXT,
		content TEXT,
		read BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS webhooks (
		id TEXT PRIMARY KEY,
		name TEXT,
		target_agent TEXT, -- "main" or subagent name
		prompt_template TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Migrations: Add target_type and target_name if they are missing
	_, _ = db.Exec("ALTER TABLE scheduled_tasks ADD COLUMN target_type TEXT DEFAULT 'main'")
	_, _ = db.Exec("ALTER TABLE scheduled_tasks ADD COLUMN target_name TEXT")
	_, _ = db.Exec("ALTER TABLE sub_agents ADD COLUMN model TEXT")
	_, _ = db.Exec("ALTER TABLE sub_agents ADD COLUMN personality TEXT")

	return &Store{DB: db}, nil
}

type Project struct {
	ID          string
	Name        string
	Description string
	Status      string
	CreatedAt   time.Time
}

type Task struct {
	ID            string
	ProjectID     string
	ParentID      string
	Title         string
	Description   string
	Status        string
	AssignedAgent string
	Result        string
}

type KnowledgeEntry struct {
	Key       string
	Category  string
	Content   string
	Tags      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Store) SaveProject(p Project) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO projects (id, name, description, status) VALUES (?, ?, ?, ?)", p.ID, p.Name, p.Description, p.Status)
	return err
}

func (s *Store) GetProjects() ([]Project, error) {
	rows, err := s.DB.Query("SELECT id, name, description, status, created_at FROM projects ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Store) SaveTask(t Task) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO tasks (id, project_id, parent_id, title, description, status, assigned_agent, result) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.ProjectID, t.ParentID, t.Title, t.Description, t.Status, t.AssignedAgent, t.Result)
	return err
}

func (s *Store) GetTasks(projectID string) ([]Task, error) {
	rows, err := s.DB.Query("SELECT id, project_id, COALESCE(parent_id, ''), title, description, status, COALESCE(assigned_agent, ''), COALESCE(result, '') FROM tasks WHERE project_id = ?", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Title, &t.Description, &t.Status, &t.AssignedAgent, &t.Result); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Store) AssignAgentToTask(taskID, agentName string) error {
	_, err := s.DB.Exec("UPDATE tasks SET assigned_agent = ? WHERE id = ?", agentName, taskID)
	return err
}

func (s *Store) SaveKnowledge(k KnowledgeEntry) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO knowledge_base (key, category, content, tags, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)",
		k.Key, k.Category, k.Content, k.Tags)
	return err
}

func (s *Store) GetKnowledge(key string) (*KnowledgeEntry, error) {
	var k KnowledgeEntry
	err := s.DB.QueryRow("SELECT key, category, content, tags, created_at, updated_at FROM knowledge_base WHERE key = ?", key).
		Scan(&k.Key, &k.Category, &k.Content, &k.Tags, &k.CreatedAt, &k.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &k, err
}

func (s *Store) SearchKnowledge(query string) ([]KnowledgeEntry, error) {
	rows, err := s.DB.Query("SELECT key, category, content, tags, created_at, updated_at FROM knowledge_base WHERE key LIKE ? OR content LIKE ? OR tags LIKE ?",
		"%"+query+"%", "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []KnowledgeEntry
	for rows.Next() {
		var k KnowledgeEntry
		if err := rows.Scan(&k.Key, &k.Category, &k.Content, &k.Tags, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, k)
	}
	return entries, nil
}

func (s *Store) ListKnowledgeKeys() ([]string, error) {
	rows, err := s.DB.Query("SELECT key FROM knowledge_base ORDER BY key ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *Store) AddRSSFeed(url, title, category string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO rss_feeds (url, title, category) VALUES (?, ?, ?)", url, title, category)
	return err
}

func (s *Store) GetRSSFeeds() ([]map[string]string, error) {
	rows, err := s.DB.Query("SELECT url, title, category FROM rss_feeds")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []map[string]string
	for rows.Next() {
		var u, t, c string
		if err := rows.Scan(&u, &t, &c); err != nil {
			return nil, err
		}
		feeds = append(feeds, map[string]string{"url": u, "title": t, "category": c})
	}
	return feeds, nil
}

func (s *Store) GetRSSFeedsByCategory(category string) ([]map[string]string, error) {
	rows, err := s.DB.Query("SELECT url, title, category FROM rss_feeds WHERE category = ?", category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []map[string]string
	for rows.Next() {
		var u, t, c string
		if err := rows.Scan(&u, &t, &c); err != nil {
			return nil, err
		}
		feeds = append(feeds, map[string]string{"url": u, "title": t, "category": c})
	}
	return feeds, nil
}

func (s *Store) IsRSSItemProcessed(guid string) (bool, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM processed_rss_items WHERE guid = ?", guid).Scan(&count)
	return count > 0, err
}

func (s *Store) MarkRSSItemProcessed(guid, feedURL string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO processed_rss_items (guid, feed_url) VALUES (?, ?)", guid, feedURL)
	return err
}

type Council struct {
	Name    string
	Members string
}

func (s *Store) SaveCouncil(name, members string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO councils (name, members) VALUES (?, ?)", name, members)
	return err
}

func (s *Store) GetCouncils() ([]Council, error) {
	rows, err := s.DB.Query("SELECT name, members FROM councils")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var councils []Council
	for rows.Next() {
		var c Council
		if err := rows.Scan(&c.Name, &c.Members); err != nil {
			return nil, err
		}
		councils = append(councils, c)
	}
	return councils, nil
}

func (s *Store) GetCouncil(name string) (*Council, error) {
	var c Council
	err := s.DB.QueryRow("SELECT name, members FROM councils WHERE name = ?", name).Scan(&c.Name, &c.Members)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (s *Store) DeleteCouncil(name string) error {
	_, err := s.DB.Exec("DELETE FROM councils WHERE name = ?", name)
	return err
}

type SubAgentDefinition struct {
	Name        string
	Personality string
	Tools       string
	Model       string
}

func (s *Store) SaveSubAgentDefinition(name, personality, tools, model string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO sub_agent_definitions (name, personality, tools, model) VALUES (?, ?, ?, ?)", name, personality, tools, model)
	return err
}

func (s *Store) GetSubAgentDefinitions() ([]SubAgentDefinition, error) {
	rows, err := s.DB.Query("SELECT name, personality, tools, COALESCE(model, '') FROM sub_agent_definitions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []SubAgentDefinition
	for rows.Next() {
		var d SubAgentDefinition
		if err := rows.Scan(&d.Name, &d.Personality, &d.Tools, &d.Model); err != nil {
			return nil, err
		}
		defs = append(defs, d)
	}
	return defs, nil
}

func (s *Store) GetSubAgentDefinition(name string) (*SubAgentDefinition, error) {
	var d SubAgentDefinition
	err := s.DB.QueryRow("SELECT name, personality, tools, COALESCE(model, '') FROM sub_agent_definitions WHERE name = ?", name).Scan(&d.Name, &d.Personality, &d.Tools, &d.Model)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

type SubAgentTask struct {
	ID          string
	Prompt      string
	Status      string
	Progress    int
	Result      string
	Model       string
	Personality string
	CreatedAt   time.Time
	FinishedAt  *time.Time
}

func (s *Store) SaveSubAgent(id, prompt, status, model, personality string) error {
	_, err := s.DB.Exec("INSERT INTO sub_agents (id, prompt, status, progress, model, personality) VALUES (?, ?, ?, 0, ?, ?)", id, prompt, status, model, personality)
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
	rows, err := s.DB.Query("SELECT id, prompt, status, progress, COALESCE(result, ''), COALESCE(model, ''), COALESCE(personality, ''), created_at, finished_at FROM sub_agents WHERE status = 'running' ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []SubAgentTask
	for rows.Next() {
		var t SubAgentTask
		if err := rows.Scan(&t.ID, &t.Prompt, &t.Status, &t.Progress, &t.Result, &t.Model, &t.Personality, &t.CreatedAt, &t.FinishedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Store) GetSubAgents() ([]SubAgentTask, error) {
	rows, err := s.DB.Query("SELECT id, prompt, status, progress, COALESCE(result, ''), COALESCE(model, ''), COALESCE(personality, ''), created_at, finished_at FROM sub_agents ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []SubAgentTask
	for rows.Next() {
		var t SubAgentTask
		if err := rows.Scan(&t.ID, &t.Prompt, &t.Status, &t.Progress, &t.Result, &t.Model, &t.Personality, &t.CreatedAt, &t.FinishedAt); err != nil {
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
	ID         int
	Type       string
	Schedule   string
	Prompt     string
	TargetType string
	TargetName string
	LastRun    *time.Time
}

func (s *Store) SaveScheduledTask(taskType, schedule, prompt, targetType, targetName string) error {
	_, err := s.DB.Exec("INSERT INTO scheduled_tasks (task_type, schedule, prompt, target_type, target_name) VALUES (?, ?, ?, ?, ?)", taskType, schedule, prompt, targetType, targetName)
	return err
}

func (s *Store) LoadScheduledTasks() ([]ScheduledTask, error) {
	rows, err := s.DB.Query("SELECT id, task_type, schedule, prompt, COALESCE(target_type, 'main'), COALESCE(target_name, ''), last_run FROM scheduled_tasks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []ScheduledTask
	for rows.Next() {
		var t ScheduledTask
		if err := rows.Scan(&t.ID, &t.Type, &t.Schedule, &t.Prompt, &t.TargetType, &t.TargetName, &t.LastRun); err != nil {
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

func (s *Store) GetOldestMessages(limit int) ([]Message, error) {
	rows, err := s.DB.Query("SELECT id, role, content, timestamp FROM messages ORDER BY timestamp ASC LIMIT ?", limit)
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
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (s *Store) DeleteMessages(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	query := "DELETE FROM messages WHERE id IN ("
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"
	_, err := s.DB.Exec(query, args...)
	return err
}
