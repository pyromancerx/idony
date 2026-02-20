package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/llm"
	"github.com/pyromancer/idony/internal/tools/base"
)

// ThoughtProcess represents the structured reasoning format expected from the LLM.
type ThoughtProcess struct {
	Thought string `json:"thought"`
	Tool    string `json:"tool,omitempty"`
	Input   string `json:"input,omitempty"`
	Final   string `json:"final,omitempty"`
}

// Agent is the core logic engine responsible for the loop.
type Agent struct {
	client      *llm.OllamaClient
	tools       map[string]base.Tool
	history     []llm.Message
	store       *db.Store
	isThinking  bool
	personality string
	model       string
}

// NewAgent initializes a new Agent with a client and a persistence store.
func NewAgent(client *llm.OllamaClient, store *db.Store) *Agent {
	a := &Agent{
		client:      client,
		tools:       make(map[string]base.Tool),
		store:       store,
		isThinking:  false,
		personality: "",
		model:       "",
	}
	a.loadHistory()
	return a
}

func (a *Agent) IsThinking() bool {
	return a.isThinking
}

func (a *Agent) loadHistory() {
	if a.store == nil {
		return
	}
	msgs, err := a.store.LoadLastMessages(20) // Load last 20 messages for context
	if err != nil {
		fmt.Printf("Warning: Could not load history from DB: %v\n", err)
		return
	}
	for _, m := range msgs {
		a.history = append(a.history, llm.Message{Role: m.Role, Content: m.Content})
	}
}

// RegisterTool adds a tool to the agent's repertoire.
func (a *Agent) RegisterTool(tool base.Tool) {
	a.tools[tool.Name()] = tool
}

// GetTools returns the map of registered tools.
func (a *Agent) GetTools() map[string]base.Tool {
	return a.tools
}

// SetModel updates the underlying model.
func (a *Agent) SetModel(model string) {
	a.model = model
	if a.client != nil {
		a.client.SetModel(model)
	}
}

// Run processes a user input through the agentic loop.
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	a.isThinking = true
	defer func() { a.isThinking = false }()

	// If a specific model is set for this agent instance, ensure the client uses it
	originalModel := a.client.Model
	if a.model != "" {
		a.client.SetModel(a.model)
	}
	defer func() { a.client.SetModel(originalModel) }()

	a.history = append(a.history, llm.Message{Role: "user", Content: userInput})
	if a.store != nil {
		a.store.SaveMessage("user", userInput)
	}

	for {
		// Construct system prompt with tool descriptions
		systemPrompt := a.buildSystemPrompt()
		
		messages := append([]llm.Message{{Role: "system", Content: systemPrompt}}, a.history...)

		rawResponse, err := a.client.GenerateResponse(ctx, messages)
		if err != nil {
			return "", err
		}

		// Attempt to parse the LLM's thought process
		var tp ThoughtProcess
		err = json.Unmarshal([]byte(a.extractJSON(rawResponse)), &tp)
		if err != nil {
			// If JSON parsing fails, the model might just be talking; treat as final
			a.history = append(a.history, llm.Message{Role: "assistant", Content: rawResponse})
			if a.store != nil {
				a.store.SaveMessage("assistant", rawResponse)
			}
			return rawResponse, nil
		}

		// If the model provides a final answer, return it
		if tp.Final != "" {
			a.history = append(a.history, llm.Message{Role: "assistant", Content: tp.Final})
			if a.store != nil {
				a.store.SaveMessage("assistant", tp.Final)
			}
			return tp.Final, nil
		}

		// Execute tool if requested
		if tp.Tool != "" {
			tool, ok := a.tools[tp.Tool]
			if !ok {
				errorMsg := fmt.Sprintf("Error: Tool '%s' not found.", tp.Tool)
				a.history = append(a.history, llm.Message{Role: "assistant", Content: errorMsg})
				continue
			}

			fmt.Printf("\n[Idony Thought]: %s\n", tp.Thought)
			fmt.Printf("[Executing Tool]: %s with input: %s\n", tp.Tool, tp.Input)

			result, err := tool.Execute(ctx, tp.Input)
			if err != nil {
				result = fmt.Sprintf("Tool error: %v", err)
			}

			// Add observation back to history
			observation := fmt.Sprintf("Observation: %s", result)
			a.history = append(a.history, llm.Message{Role: "assistant", Content: observation})
			// We don't usually save observations to permanent DB to avoid cluttering, 
			// but we keep them in 'a.history' for the current loop's context.
			continue
		}

		// Fallback
		return rawResponse, nil
	}
}

func (a *Agent) buildSystemPrompt() string {
	var toolDocs []string
	for _, t := range a.tools {
		toolDocs = append(toolDocs, fmt.Sprintf("- %s: %s", t.Name(), t.Description()))
	}

	personality := a.personality
	if personality == "" {
		if a.store != nil {
			personality, _ = a.store.GetSetting("personality")
		}
	}
	if personality == "" {
		personality = "You are Idony, a highly opinionated AI assistant."
	}

	return fmt.Sprintf("%s\n"+
		"You operate in a Think -> Plan -> Act -> Observe loop.\n"+
		"You MUST respond ONLY with a valid JSON object in the following format:\n"+
		"{\n"+
		"  \"thought\": \"your reasoning about the current state\",\n"+
		"  \"tool\": \"the name of the tool to use (optional)\",\n"+
		"  \"input\": \"the input to the tool (optional)\",\n"+
		"  \"final\": \"your final answer to the user (optional)\"\n"+
		"}\n\n"+
		"Available Tools:\n"+
		"%s\n\n"+
		"If you have the final answer, use the \"final\" field and omit \"tool\" and \"input\".\n"+
		"If you need more information, use a \"tool\" and provide \"input\".",
		personality,
		strings.Join(toolDocs, "\n"))
}

// extractJSON is a helper to find a JSON block in the model's output if it's not pure JSON.
func (a *Agent) extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}
