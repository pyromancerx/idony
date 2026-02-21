package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pyromancer/idony/internal/agent"
	"github.com/pyromancer/idony/internal/db"
)

type Server struct {
	Agent          *agent.Agent
	SubManager     *agent.SubAgentManager
	CouncilManager *agent.CouncilManager
	Store          *db.Store
	APIKey         string
}

func NewServer(a *agent.Agent, sm *agent.SubAgentManager, cm *agent.CouncilManager, s *db.Store, apiKey string) *Server {
	return &Server{
		Agent:          a,
		SubManager:     sm,
		CouncilManager: cm,
		Store:          s,
		APIKey:         apiKey,
	}
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.APIKey != "" {
			key := r.Header.Get("X-API-Key")
			if key != s.APIKey {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) registerRoutes() {
	http.HandleFunc("/chat", s.auth(s.handleChat))
	http.HandleFunc("/status", s.auth(s.handleStatus))
	http.HandleFunc("/history", s.auth(s.handleHistory))
	http.HandleFunc("/agents", s.auth(s.handleAgents))
	http.HandleFunc("/councils", s.auth(s.handleCouncils))
	http.HandleFunc("/tools", s.auth(s.handleTools))
	http.HandleFunc("/projects", s.auth(s.handleProjects))
	http.HandleFunc("/tasks", s.auth(s.handleTasks))
	http.HandleFunc("/assign_task", s.auth(s.handleAssignTask))
	http.HandleFunc("/ui/schemas", s.auth(s.handleUISchemas))
	
	// Webhooks (No Auth required? Or maybe API key? Webhooks usually public or secret in URL)
	// The ID acts as the secret.
	http.HandleFunc("POST /webhooks/{id}", s.handleWebhook)

	// Serve PWA static files
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/", fs)
}

func (s *Server) Start(addr string) error {
	s.registerRoutes()
	fmt.Printf("Idony Server starting on %s...\n", addr)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) StartSecure(addr, certFile, keyFile string) error {
	s.registerRoutes()
	if certFile != "" && keyFile != "" {
		fmt.Printf("Idony Server starting SECURELY (HTTPS) on %s...\n", addr)
		return http.ListenAndServeTLS(addr, certFile, keyFile, nil)
	}
	fmt.Printf("Idony Server starting (HTTP) on %s...\n", addr)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleUISchemas(w http.ResponseWriter, r *http.Request) {
	tools := s.Agent.GetTools()
	schemas := make(map[string]interface{})
	for name, tool := range tools {
		schemas[name] = tool.Schema()
	}
	json.NewEncoder(w).Encode(schemas)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing webhook ID", http.StatusBadRequest)
		return
	}

	hook, err := s.Store.GetWebhook(id)
	if err != nil || hook == nil {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	body, _ := io.ReadAll(r.Body)
	payload := string(body)

	prompt := strings.ReplaceAll(hook.PromptTemplate, "{{payload}}", payload)
	fmt.Printf("[Webhook Triggered] %s: %s\n", hook.Name, prompt)

	// Run async
	go func() {
		ctx := context.Background()
		if hook.TargetAgent == "main" {
			s.Agent.Run(ctx, prompt)
		} else {
			s.SubManager.SpawnNamed(ctx, hook.TargetAgent, prompt, nil)
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook accepted"))
}

func (s *Server) handleAssignTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TaskID string `json:"task_id"`
		Agent  string `json:"agent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.Store.AssignAgentToTask(req.TaskID, req.Agent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	projects, _ := s.Store.GetProjects()
	json.NewEncoder(w).Encode(projects)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id required", http.StatusBadRequest)
		return
	}
	tasks, _ := s.Store.GetTasks(projectID)
	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text   string   `json:"text"`
		Images []string `json:"images,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[Server]: JSON Decode Error: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("[Server]: Received: %s\n", req.Text)

	var response string
	var err error

	if strings.HasPrefix(req.Text, "/") {
		parts := strings.SplitN(req.Text[1:], " ", 2)
		toolName := parts[0]
		toolInput := ""
		if len(parts) > 1 {
			toolInput = parts[1]
		}

		if tool, ok := s.Agent.GetTools()[toolName]; ok {
			fmt.Printf("[Server]: Calling tool: %s\n", toolName)
			if len(req.Images) > 0 {
				s.Agent.SetLastUserImages(req.Images)
			}
			response, err = tool.Execute(r.Context(), toolInput)
		} else {
			response = "Command not recognized."
		}
	} else {
		if len(req.Images) > 0 {
			fmt.Printf("[Server]: Running Vision (%d images)\n", len(req.Images))
			response, err = s.Agent.RunVision(r.Context(), req.Text, req.Images)
		} else {
			fmt.Printf("[Server]: Running Agent...\n")
			response, err = s.Agent.Run(r.Context(), req.Text)
		}
	}

	if err != nil {
		fmt.Printf("[Server]: Agent Error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[Server]: Success. Response: %s\n", response)
	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	active, _ := s.SubManager.GetActive()
	thinking := s.Agent.IsThinking()
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"thinking": thinking,
		"active_subagents": active,
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	activities, _ := s.Store.GetRecentActivity()
	json.NewEncoder(w).Encode(activities)
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	defs, _ := s.SubManager.ListDefinitions()
	json.NewEncoder(w).Encode(defs)
}

func (s *Server) handleCouncils(w http.ResponseWriter, r *http.Request) {
	councils, _ := s.CouncilManager.ListCouncils()
	json.NewEncoder(w).Encode(councils)
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	tools := s.Agent.GetTools()
	var names []string
	for k := range tools {
		names = append(names, k)
	}
	json.NewEncoder(w).Encode(names)
}
