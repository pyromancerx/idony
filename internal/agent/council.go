package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/llm"
)

type CouncilManager struct {
	client     *llm.OllamaClient
	store      *db.Store
	subManager *SubAgentManager
}

func NewCouncilManager(client *llm.OllamaClient, store *db.Store, subManager *SubAgentManager) *CouncilManager {
	return &CouncilManager{
		client:     client,
		store:      store,
		subManager: subManager,
	}
}

func (m *CouncilManager) RunCouncilSession(ctx context.Context, councilName, problem string) (string, error) {
	council, err := m.store.GetCouncil(councilName)
	if err != nil {
		return "", err
	}
	if council == nil {
		return "", fmt.Errorf("council '%s' not found", councilName)
	}

	memberNames := strings.Split(council.Members, ",")
	var members []*db.SubAgentDefinition
	for _, name := range memberNames {
		def, _ := m.store.GetSubAgentDefinition(strings.TrimSpace(name))
		if def != nil {
			members = append(members, def)
		}
	}

	if len(members) == 0 {
		return "", fmt.Errorf("no valid members found for council '%s'", councilName)
	}

	id := uuid.New().String()[:8]
	sessionTitle := fmt.Sprintf("Council '%s' Session: %s", councilName, id)
	err = m.store.SaveSubAgent(id, sessionTitle, "running", "", "")
	if err != nil {
		return "", err
	}

	go m.executeCouncilSession(id, councilName, members, problem)

	return id, nil
}

func (m *CouncilManager) executeCouncilSession(id, councilName string, members []*db.SubAgentDefinition, problem string) {
	fmt.Printf("\n[Council %s]: Session Started - %s\n", councilName, problem)

	var transcript []string
	transcript = append(transcript, fmt.Sprintf("Council Problem: %s", problem))

	// We'll do 2 rounds of discussion
	for round := 1; round <= 2; round++ {
		for _, member := range members {
			// Update status with progress
			progress := (round-1)*100/2 + (100 / (2 * len(members)))
			m.store.UpdateSubAgentProgress(id, progress)

			// Construct a specialized prompt for the member
			memberPrompt := fmt.Sprintf("You are participating in a council meeting called '%s'.\n"+
				"The problem we are solving is: %s\n\n"+
				"Current Discussion Transcript:\n%s\n\n"+
				"Provide your thoughts or solutions based on your unique personality and expertise.",
				councilName, problem, strings.Join(transcript, "\n\n"))

			// Create temporary agent for this turn
			subAgent := &Agent{
				client:      m.client,
				tools:       m.subManager.tools,
				store:       nil,
				personality: member.Personality,
				model:       member.Model,
			}

			fmt.Printf("[Council %s] Member '%s' is thinking...\n", councilName, member.Name)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			response, err := subAgent.Run(ctx, memberPrompt)
			cancel()

			if err != nil {
				log.Printf("Error in council turn for %s: %v", member.Name, err)
				continue
			}

			contribution := fmt.Sprintf("[%s]: %s", member.Name, response)
			transcript = append(transcript, contribution)
		}
	}

	finalResult := strings.Join(transcript, "\n\n---\n\n")
	m.store.UpdateSubAgent(id, "completed", finalResult)
	fmt.Printf("\n[Council %s]: Session Completed\n", councilName)
}

func (m *CouncilManager) DefineCouncil(name string, members []string) error {
	return m.store.SaveCouncil(name, strings.Join(members, ","))
}

func (m *CouncilManager) ListCouncils() ([]db.Council, error) {
	return m.store.GetCouncils()
}
