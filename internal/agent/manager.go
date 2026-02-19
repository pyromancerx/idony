package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/llm"
	"github.com/pyromancer/idony/internal/tools/base"
)

type SubAgentManager struct {
	client *llm.OllamaClient
	store  *db.Store
	tools  map[string]base.Tool
	mu     sync.Mutex
}

func NewSubAgentManager(client *llm.OllamaClient, store *db.Store, tools map[string]base.Tool) *SubAgentManager {
	return &SubAgentManager{
		client: client,
		store:  store,
		tools:  tools,
	}
}

func (m *SubAgentManager) Spawn(ctx context.Context, prompt string) (string, error) {
	id := uuid.New().String()[:8] // Short ID for convenience
	err := m.store.SaveSubAgent(id, prompt, "running")
	if err != nil {
		return "", err
	}

	// Run in background
	go m.runSubAgent(id, prompt)

	return id, nil
}

func (m *SubAgentManager) runSubAgent(id, prompt string) {
	// Create a fresh agent for this task
	// We don't pass the store to the sub-agent's constructor to avoid polluting main history
	subAgent := &Agent{
		client: m.client,
		tools:  m.tools,
		store:  nil, 
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Printf("\n[Sub-Agent %s]: Starting task: %s\n", id, prompt)
	
	result, err := subAgent.Run(ctx, prompt)
	status := "completed"
	if err != nil {
		status = "failed"
		result = fmt.Sprintf("Error: %v", err)
	}

	err = m.store.UpdateSubAgent(id, status, result)
	if err != nil {
		log.Printf("Error updating sub-agent %s in DB: %v", id, err)
	}

	fmt.Printf("\n[Sub-Agent %s]: Task %s\n", id, status)
}

func (m *SubAgentManager) List() ([]db.SubAgentTask, error) {
	return m.store.GetSubAgents()
}

func (m *SubAgentManager) GetActive() ([]db.SubAgentTask, error) {
	return m.store.GetActiveSubAgents()
}

func (m *SubAgentManager) UpdateProgress(id string, progress int) error {
	return m.store.UpdateSubAgentProgress(id, progress)
}
