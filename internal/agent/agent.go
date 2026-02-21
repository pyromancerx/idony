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
	Thought string          `json:"thought"`
	Tool    string          `json:"tool,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	Final   string          `json:"final,omitempty"`
}

// Agent is the core logic engine responsible for the loop.
type Agent struct {
	client         *llm.OllamaClient
	tools          map[string]base.Tool
	history        []llm.Message
	store          *db.Store
	isThinking     bool
	personality    string
	model          string
	lastUserImages []string
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

func (a *Agent) GetLastUserImages() []string {
	return a.lastUserImages
}

func (a *Agent) SetLastUserImages(images []string) {
	a.lastUserImages = images
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

// SetBaseURL updates the underlying LLM client's base URL.
func (a *Agent) SetBaseURL(url string) {
	if a.client != nil {
		a.client.BaseURL = url
	}
}

// Run processes a user input through the agentic loop.
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	a.isThinking = true
	a.lastUserImages = nil
	defer func() { a.isThinking = false }()

	a.history = append(a.history, llm.Message{Role: "user", Content: userInput})
	if a.store != nil {
		a.store.SaveMessage("user", userInput)
	}

	return a.internalLoop(ctx)
}

// RunVision processes a user input with one or more base64 images.
func (a *Agent) RunVision(ctx context.Context, userInput string, b64Images []string) (string, error) {
	a.isThinking = true
	a.lastUserImages = b64Images
	defer func() { a.isThinking = false }()

	a.history = append(a.history, llm.Message{Role: "user", Content: userInput, Images: b64Images})
	if a.store != nil {
		a.store.SaveMessage("user", "[Image Attached] "+userInput)
	}

	return a.internalLoop(ctx)
}

func (a *Agent) internalLoop(ctx context.Context) (string, error) {
	// If a specific model is set for this agent instance, ensure the client uses it
	originalModel := a.client.Model
	if a.model != "" {
		a.client.SetModel(a.model)
	}
	defer func() { a.client.SetModel(originalModel) }()

	for {
		// Construct system prompt with tool descriptions
		systemPrompt := a.buildSystemPrompt()
		
		messages := append([]llm.Message{{Role: "system", Content: systemPrompt}}, a.history...)

		rawResponse, err := a.client.GenerateResponse(ctx, messages)
		if err != nil {
			return "", err
		}
		fmt.Printf("\n[LLM Raw Response]: %s\n", rawResponse)

		if strings.TrimSpace(rawResponse) == "" {
			return "Error: The model returned an empty response. It may be too small for this task or experiencing an error.", nil
		}

		// Attempt to parse the LLM's thought process
		var tp ThoughtProcess
		extracted := a.extractJSON(rawResponse)
		err = json.Unmarshal([]byte(extracted), &tp)
		if err != nil || (tp.Final == "" && tp.Tool == "" && tp.Thought == "") {
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
			inputStr := string(tp.Input)
			// Remove surrounding quotes if it's just a string, otherwise keep as JSON
			if strings.HasPrefix(inputStr, "\"") && strings.HasSuffix(inputStr, "\"") {
				var s string
				if err := json.Unmarshal(tp.Input, &s); err == nil {
					inputStr = s
				}
			}
			fmt.Printf("[Executing Tool]: %s with input: %s\n", tp.Tool, inputStr)

			result, err := tool.Execute(ctx, inputStr)
			if err != nil {
				result = fmt.Sprintf("Tool error: %v", err)
			}
			fmt.Printf("[Tool Result]: %s\n", result)

			// Add observation back to history
			observation := fmt.Sprintf("Observation: %s", result)
			a.history = append(a.history, llm.Message{Role: "assistant", Content: observation})
			continue
		}

		// Fallback: if we have a thought but no action/final, return the raw response
		// to preserve any conversational text outside the JSON.
		a.history = append(a.history, llm.Message{Role: "assistant", Content: rawResponse})
		if a.store != nil {
			a.store.SaveMessage("assistant", rawResponse)
		}
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

	// Inject Memories
	memoryContext := ""
	if a.store != nil {
		// Fetch recent/relevant memories. 
		// For now, we fetch the last 10 memories as context.
		// Future improvement: Vector search based on current input.
		memories, _ := a.store.SearchMemories("", 10) 
		if len(memories) > 0 {
			var mems []string
			for _, m := range memories {
				mems = append(mems, fmt.Sprintf("- [%s] %s", m.Type, m.Content))
			}
			memoryContext = "\n\nRELEVANT MEMORIES:\n" + strings.Join(mems, "\n")
		}
	}

	return fmt.Sprintf("%s\n"+
		"You operate in a strict Think -> Plan -> Act -> Observe loop.\n"+
		"You MUST wrap your response in a single <json> block. Do NOT include any text outside this block.\n"+
		"FORMAT:\n"+
		"<json>\n"+
		"{\n"+
		"  \"thought\": \"reasoning about the current state\",\n"+
		"  \"tool\": \"tool_name\",\n"+
		"  \"input\": \"tool_input\",\n"+
		"  \"final\": \"final answer\"\n"+
		"}\n"+
		"</json>\n"+
		"%s\n\n"+
		"INTERACTIVE MODE:\n"+
		"If a tool requires parameters you do not have, ask the user for them using the 'final' field.\n\n"+
		"IMAGE ANALYSIS:\n"+
		"You can analyze images directly or use the 'subagent' tool.\n\n"+
		"Available Tools:\n"+
		"%s\n\n"+
		"If you have the final answer, use \"final\". If you need a tool, use \"tool\" and \"input\".",
		personality,
		memoryContext,
		strings.Join(toolDocs, "\n"))
}

// extractJSON is a helper to find a JSON block in the model's output.
func (a *Agent) extractJSON(s string) string {
	// Try to find <json> tags first
	if start := strings.Index(s, "<json>"); start != -1 {
		if end := strings.Index(s[start:], "</json>"); end != -1 {
			return s[start+6 : start+end]
		}
	}

	// Fallback to first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}
