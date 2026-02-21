package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/pyromancer/idony/internal/agent"
	"github.com/pyromancer/idony/internal/config"
	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/llm"
	"github.com/pyromancer/idony/internal/server"
	"github.com/pyromancer/idony/internal/telegram"
	"github.com/pyromancer/idony/internal/tools"
)

func main() {
	// Load configuration
	conf, err := config.LoadConfig("config.txt")
	if err != nil {
		fmt.Printf("Warning: Could not load config.txt: %v. Using defaults.\n", err)
	}

	model := conf.GetWithDefault("MODEL", "llama3.1")
	ollamaURL := conf.GetWithDefault("OLLAMA_URL", "http://localhost:11434")
	serverAddr := conf.GetWithDefault("SERVER_ADDR", "0.0.0.0:8080")
	
	apiKey := conf.Get("SERVER_API_KEY")
	if apiKey == "" {
		fmt.Println("No API Key found. Generating a secure one...")
		apiKey = uuid.New().String()
		conf.Set("SERVER_API_KEY", apiKey)
		if err := conf.SaveToFile("config.txt"); err != nil {
			fmt.Printf("Error saving generated API Key: %v\n", err)
		} else {
			fmt.Println("New API Key generated and saved to config.txt")
		}
	}

	// Initialize SQLite Store
	store, err := db.NewStore("idony.db")
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}

	// Initialize Ollama client
	client := llm.NewOllamaClient(ollamaURL, model)

	// Initialize Main Agent
	idony := agent.NewAgent(client, store)

	// Initialize Managers
	subManager := agent.NewSubAgentManager(client, store, idony.GetTools())
	councilManager := agent.NewCouncilManager(client, store, subManager)

	// Initialize Scheduler and start it
	scheduler := agent.NewScheduler(idony, store, subManager, councilManager)
	scheduler.Start(context.Background())

	// Register Tools
	idony.RegisterTool(&tools.TimeTool{})
	idony.RegisterTool(&tools.GeminiCoder{})
	idony.RegisterTool(tools.NewScheduleTool(store))
	idony.RegisterTool(tools.NewConfigUpdateTool(conf, "config.txt", idony))
	idony.RegisterTool(tools.NewReloadConfigTool(conf, "config.txt", idony))
	idony.RegisterTool(tools.NewPersonalityTool(store))
	idony.RegisterTool(tools.NewSubAgentTool(subManager, idony))
	idony.RegisterTool(tools.NewCouncilTool(councilManager))
	idony.RegisterTool(&tools.ListFilesTool{})
	idony.RegisterTool(&tools.ReadFileTool{})
	idony.RegisterTool(&tools.WriteFileTool{})
	idony.RegisterTool(&tools.DeleteFileTool{})
	idony.RegisterTool(&tools.SearchFileTool{})
	idony.RegisterTool(&tools.ShellExecTool{})
	
	browserManager := tools.NewBrowserManager()
	idony.RegisterTool(tools.NewBrowserNativeTool(browserManager))
	idony.RegisterTool(&tools.WebSearchTool{})

	idony.RegisterTool(tools.NewEmailTool(conf))
	idony.RegisterTool(tools.NewRSSTool(store))
	idony.RegisterTool(tools.NewPlannerTool(store))
	idony.RegisterTool(tools.NewKnowledgeTool(store, "./knowledge"))
	idony.RegisterTool(tools.NewTranscribeTool(conf, store))
	idony.RegisterTool(tools.NewMediaSearchTool(store))
	idony.RegisterTool(tools.NewTTSTool(conf))
	idony.RegisterTool(tools.NewDocsTool("./docs"))
	idony.RegisterTool(tools.NewModelListTool(client))
	idony.RegisterTool(tools.NewAgentListTool(subManager))
	idony.RegisterTool(&tools.OllamaLibraryTool{})
	idony.RegisterTool(tools.NewMemoryTool(store))
	idony.RegisterTool(tools.NewRecallTool(store))
	idony.RegisterTool(tools.NewGraphAddTool(store))
	idony.RegisterTool(tools.NewGraphQueryTool(store))
	idony.RegisterTool(tools.NewCompactTool(store, client))
	idony.RegisterTool(tools.NewOptimizeMemoryTool(store, client))
	idony.RegisterTool(tools.NewMessagingTool(store))
	idony.RegisterTool(tools.NewInboxTool(store))
	idony.RegisterTool(tools.NewWebhookTool(store))
	
	// Load MCP Tools
	mcpManager := tools.NewMCPManager()
	mcpTools, err := mcpManager.LoadFromConfig(conf)
	if err != nil {
		fmt.Printf("Warning: Failed to load MCP tools from config: %v\n", err)
	} else {
		for _, t := range mcpTools {
			idony.RegisterTool(t)
			fmt.Printf("Registered MCP Tool: %s\n", t.Name())
		}
	}

	swarmPath := conf.GetWithDefault("SWARMUI_PATH", "/home/pyromancer/swarmconnector/swarmui")
	swarmURL := conf.GetWithDefault("SWARMUI_URL", "http://localhost:7801")
	swarmModel := conf.GetWithDefault("SWARMUI_DEFAULT_MODEL", "v1-5-pruned-emaonly.safetensors")
	idony.RegisterTool(tools.NewSwarmUITool(swarmPath, swarmURL, swarmModel))

	browserBin := conf.GetWithDefault("BROWSER_BIN", "./idony-browser")
	idony.RegisterTool(tools.NewBrowserTool(browserBin))

	// Start Telegram Bridge if token exists
	tgToken := conf.Get("TELEGRAM_TOKEN")
	if tgToken != "" && tgToken != "your-telegram-bot-token" {
		fmt.Println("Starting Telegram Bridge...")
		tgBridge, err := telegram.NewBridge(tgToken, idony, store, conf)
		if err != nil {
			fmt.Printf("Failed to initialize Telegram: %v\n", err)
		} else {
			go tgBridge.Start()
		}
	}

	// Start Server
	srv := server.NewServer(idony, subManager, councilManager, store, apiKey)
	
	certFile := conf.Get("TLS_CERT_FILE")
	keyFile := conf.Get("TLS_KEY_FILE")

	if err := srv.StartSecure(serverAddr, certFile, keyFile); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
