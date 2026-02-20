package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/pyromancer/idony/internal/agent"
	"github.com/pyromancer/idony/internal/config"
	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/llm"
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
	idony.RegisterTool(tools.NewConfigUpdateTool("config.txt"))
	idony.RegisterTool(tools.NewPersonalityTool(store))
	idony.RegisterTool(tools.NewSubAgentTool(subManager))
	idony.RegisterTool(tools.NewCouncilTool(councilManager))
	
	swarmPath := conf.GetWithDefault("SWARMUI_PATH", "/home/pyromancer/swarmconnector/swarmui")
	swarmURL := conf.GetWithDefault("SWARMUI_URL", "http://localhost:7801")
	swarmModel := conf.GetWithDefault("SWARMUI_DEFAULT_MODEL", "v1-5-pruned-emaonly.safetensors")
	idony.RegisterTool(tools.NewSwarmUITool(swarmPath, swarmURL, swarmModel))

	// Register Browser Tool
	browserBin := conf.GetWithDefault("BROWSER_BIN", "/home/pyromancer/browser-connector/idony-browser")
	idony.RegisterTool(tools.NewBrowserTool(browserBin))

	// TUI Setup
	app := tview.NewApplication()
	
	outputView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	outputView.SetBorder(true).SetTitle(" Idony Chat ")

	// Side panel sections
	historyView := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	historyView.SetBorder(true).SetTitle(" Activity (24h) ")

	agentsView := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	agentsView.SetBorder(true).SetTitle(" Specialized Agents ")

	councilsView := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	councilsView.SetBorder(true).SetTitle(" Councils ")

	sideFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(historyView, 0, 1, false).
		AddItem(agentsView, 0, 1, false).
		AddItem(councilsView, 0, 1, false)
	
	// Command input
	inputField := tview.NewInputField().
		SetLabel("> ").
		SetFieldWidth(0)

	// Status bar at the bottom
	statusBar := tview.NewTextView().SetDynamicColors(true)

	// Layout
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	
	contentFlex := tview.NewFlex().
		AddItem(outputView, 0, 2, true).
		AddItem(sideFlex, 0, 1, false)

	mainFlex.AddItem(contentFlex, 0, 1, true).
		AddItem(inputField, 1, 0, true).
		AddItem(statusBar, 1, 0, false)

	// Hotkeys
	sideHidden := false
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlH {
			if sideHidden {
				contentFlex.AddItem(sideFlex, 0, 1, false)
			} else {
				contentFlex.RemoveItem(sideFlex)
			}
			sideHidden = !sideHidden
			return nil
		}
		return event
	})

	// Periodic update loop
	go func() {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			// Update History
			activities, _ := store.GetRecentActivity()
			var histText strings.Builder
			for _, a := range activities {
				histText.WriteString(fmt.Sprintf("[yellow]%s[white] %s\n", a.Timestamp.Format("15:04"), a.Title))
			}
			
			// Update Specialized Agents
			defs, _ := subManager.ListDefinitions()
			var agentsText strings.Builder
			for _, d := range defs {
				agentsText.WriteString(fmt.Sprintf("[green]%s[white]: %s (Model: %s)\n\n", d.Name, d.Personality, d.Model))
			}

			// Update Councils
			councils, _ := councilManager.ListCouncils()
			var councilsText strings.Builder
			for _, c := range councils {
				councilsText.WriteString(fmt.Sprintf("[blue]%s[white]: %s\n\n", c.Name, c.Members))
			}

			app.QueueUpdateDraw(func() {
				historyView.SetText(histText.String())
				agentsView.SetText(agentsText.String())
				councilsView.SetText(councilsText.String())
			})

			// Update Status Bar
			active, _ := subManager.GetActive()
			thinking := idony.IsThinking()
			status := ""
			if thinking {
				status = fmt.Sprintf("%s Thinking...", spinner[i%len(spinner)])
			}
			if len(active) > 0 {
				if status != "" { status += " | " }
				status += fmt.Sprintf("%d Sub-Agents Active", len(active))
			}
			app.QueueUpdateDraw(func() {
				statusBar.SetText(status)
			})

			i++
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// Handle input
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter { return }
		
		text := strings.TrimSpace(inputField.GetText())
		inputField.SetText("")
		if text == "" { return }

		if text == "/exit" || text == "/quit" {
			app.Stop()
			return
		}

		fmt.Fprintf(outputView, "[green]You:[white] %s\n", text)

		go func() {
			var response string
			var err error

			if strings.HasPrefix(text, "/") {
				parts := strings.SplitN(text[1:], " ", 2)
				toolName := parts[0]
				toolInput := ""
				if len(parts) > 1 { toolInput = parts[1] }

				if tool, ok := idony.GetTools()[toolName]; ok {
					app.QueueUpdateDraw(func() {
						fmt.Fprintf(outputView, "[blue]Tool Execution:[white] %s\n", toolName)
					})
					response, err = tool.Execute(context.Background(), toolInput)
				} else {
					response = "Command not recognized."
				}
			} else {
				response, err = idony.Run(context.Background(), text)
			}

			app.QueueUpdateDraw(func() {
				if err != nil {
					fmt.Fprintf(outputView, "[red]Error:[white] %v\n", err)
				} else {
					fmt.Fprintf(outputView, "\n[yellow]Idony:[white] %s\n\n", response)
				}
			})
		}()
	})

	fmt.Fprintf(outputView, "[yellow]Idony AI Bot v1.3.0 (Multi-Target Scheduling) Initialized[white]\n")
	fmt.Fprintf(outputView, "Model: %s\n", model)
	fmt.Fprintf(outputView, "Press [yellow]Ctrl+H[white] to toggle side panel.\n\n")

	if err := app.SetRoot(mainFlex, true).Run(); err != nil {
		fmt.Printf("Error running application: %v\n", err)
	}
}
