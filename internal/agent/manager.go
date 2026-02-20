package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
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

	// Run in background with default personality and model
	go m.runSubAgent(id, prompt, "", "", nil)

	return id, nil
}

func (m *SubAgentManager) SpawnNamed(ctx context.Context, agentName, prompt string) (string, error) {
	def, err := m.store.GetSubAgentDefinition(agentName)
	if err != nil {
		return "", err
	}
	if def == nil {
		return "", fmt.Errorf("sub-agent definition for '%s' not found", agentName)
	}

	id := uuid.New().String()[:8]
	err = m.store.SaveSubAgent(id, fmt.Sprintf("[%s]: %s", agentName, prompt), "running")
	if err != nil {
		return "", err
	}

	// Filter tools if specified
	var allowedTools map[string]base.Tool
	if def.Tools != "" && def.Tools != "*" {
		allowedTools = make(map[string]base.Tool)
		toolList := strings.Split(def.Tools, ",")
		for _, tn := range toolList {
			tn = strings.TrimSpace(tn)
			if t, ok := m.tools[tn]; ok {
				allowedTools[tn] = t
			}
		}
	} else {
		allowedTools = m.tools
	}

	go m.runSubAgent(id, prompt, def.Personality, def.Model, allowedTools)

	return id, nil
}

func (m *SubAgentManager) runSubAgent(id, prompt, personality, model string, tools map[string]base.Tool) {
	// Create a fresh agent for this task
	if tools == nil {
		tools = m.tools
	}

	subAgent := &Agent{
		client:      m.client,
		tools:       tools,
		store:       nil, 
		personality: personality,
		model:       model,
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

func (m *SubAgentManager) ListDefinitions() ([]db.SubAgentDefinition, error) {
	return m.store.GetSubAgentDefinitions()
}

func (m *SubAgentManager) DefineAgent(name, personality, tools, model string) error {
	return m.store.SaveSubAgentDefinition(name, personality, tools, model)
}

func (m *SubAgentManager) GetActive() ([]db.SubAgentTask, error) {
	return m.store.GetActiveSubAgents()
}

func (m *SubAgentManager) UpdateProgress(id string, progress int) error {
	return m.store.UpdateSubAgentProgress(id, progress)
}
