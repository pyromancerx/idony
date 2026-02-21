package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/pyromancer/idony/internal/config"
	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/llm"
)

type Client struct {
	BaseURL string
	APIKey  string
}

func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+path, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" { req.Header.Set("X-API-Key", c.APIKey) }
	client := &http.Client{Timeout: 30 * time.Second} // Long timeout for LLM
	return client.Do(req)
}

func (c *Client) Get(path string, target interface{}) error {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil { return err }
	if c.APIKey != "" { req.Header.Set("X-API-Key", c.APIKey) }
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("status %d", resp.StatusCode) }
	return json.NewDecoder(resp.Body).Decode(target)
}

func main() {
	conf, err := config.LoadConfig("config.txt")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}
	serverAddr := conf.GetWithDefault("SERVER_ADDR", "127.0.0.1:8080")
	if !strings.HasPrefix(serverAddr, "http") { serverAddr = "http://" + serverAddr }
	apiKey := conf.Get("SERVER_API_KEY")
	client := &Client{BaseURL: serverAddr, APIKey: apiKey}

	app := tview.NewApplication()
	
	tview.Borders.Horizontal = '-'
	tview.Borders.Vertical = '|'
	tview.Borders.TopLeft = '+'
	tview.Borders.TopRight = '+'
	tview.Borders.BottomLeft = '+'
	tview.Borders.BottomRight = '+'

	outputView := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	outputView.SetBorder(true).SetTitle(" Chat ")

	historyView := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	historyView.SetBorder(true).SetTitle(" History ")

	agentsView := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	agentsView.SetBorder(true).SetTitle(" Agents ")

	plannerTree := tview.NewTreeView()
	plannerTree.SetBorder(true).SetTitle(" Planner ")
	plannerRoot := tview.NewTreeNode("Projects").SetColor(tcell.ColorNames["Cyan"])
	plannerTree.SetRoot(plannerRoot)

	inputField := tview.NewInputField().SetLabel("> ").SetFieldWidth(0)

	// --- AUTOCOMPLETE LOGIC ---
	var availableTools []string
	inputField.SetAutocompleteFunc(func(currentText string) (entries []string) {
		if len(currentText) == 0 || !strings.HasPrefix(currentText, "/") {
			return nil
		}
		
		// If we don't have tools yet, try to fetch them once
		if len(availableTools) == 0 {
			client.Get("/tools", &availableTools)
		}

		for _, tool := range availableTools {
			cmd := "/" + tool
			if strings.HasPrefix(cmd, currentText) {
				entries = append(entries, cmd)
			}
		}
		return
	})

	visibility := map[string]bool{"history": true, "agents": true, "planner": true}
	statusMenu := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	menuOptions := []string{"History", "Agents", "Planner", "Quit"}
	menuIdx := 0

	sideFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	mainContent := tview.NewFlex()
	rootFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainContent, 0, 1, true).
		AddItem(inputField, 1, 0, false).
		AddItem(statusMenu, 1, 0, false)

	updateLayout := func() {
		mainContent.Clear()
		mainContent.AddItem(outputView, 0, 2, false)
		sideFlex.Clear()
		showSide := false
		if visibility["history"] { sideFlex.AddItem(historyView, 0, 1, false); showSide = true }
		if visibility["agents"] { sideFlex.AddItem(agentsView, 0, 1, false); showSide = true }
		if showSide { mainContent.AddItem(sideFlex, 0, 1, false) }
		if visibility["planner"] { mainContent.AddItem(plannerTree, 0, 1, false) }
	}

	focusList := []tview.Primitive{inputField, outputView, historyView, agentsView, plannerTree, statusMenu}
	focusIdx := 0

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			for {
				focusIdx = (focusIdx + 1) % len(focusList)
				p := focusList[focusIdx]
				if p == historyView && !visibility["history"] { continue }
				if p == agentsView && !visibility["agents"] { continue }
				if p == plannerTree && !visibility["planner"] { continue }
				app.SetFocus(p)
				break
			}
			return nil
		}
		if event.Key() == tcell.KeyCtrlC { app.Stop(); return nil }
		return event
	})

	statusMenu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft: menuIdx = (menuIdx - 1 + len(menuOptions)) % len(menuOptions); return nil
		case tcell.KeyRight: menuIdx = (menuIdx + 1) % len(menuOptions); return nil
		case tcell.KeyEnter:
			switch menuOptions[menuIdx] {
			case "History": visibility["history"] = !visibility["history"]
			case "Agents": visibility["agents"] = !visibility["agents"]
			case "Planner": visibility["planner"] = !visibility["planner"]
			case "Quit": app.Stop()
			}
			updateLayout(); return nil
		}
		return event
	})

	go func() {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			var activities []db.Activity
			if err := client.Get("/history", &activities); err == nil {
				var sb strings.Builder
				for _, a := range activities { sb.WriteString(fmt.Sprintf("[yellow]%s[white] %s\n", a.Timestamp.Format("15:04"), a.Title)) }
				app.QueueUpdateDraw(func() { historyView.SetText(sb.String()) })
			}
			
			var agents []db.SubAgentDefinition
			client.Get("/agents", &agents)
			var asb strings.Builder
			for _, d := range agents { asb.WriteString(fmt.Sprintf("[green]%s[white]\n", d.Name)) }
			app.QueueUpdateDraw(func() { agentsView.SetText(asb.String()) })

			var statusData struct {
				Thinking bool `json:"thinking"`
				Active   []db.SubAgentTask `json:"active_subagents"`
			}
			err := client.Get("/status", &statusData)

			app.QueueUpdateDraw(func() {
				var sb strings.Builder
				if err != nil {
					sb.WriteString(fmt.Sprintf("[red]OFFLINE: %v | ", err))
				} else {
					if statusData.Thinking { sb.WriteString(fmt.Sprintf("%s Thinking | ", spinner[i%len(spinner)])) }
					if len(statusData.Active) > 0 { sb.WriteString(fmt.Sprintf("%d Active | ", len(statusData.Active))) }
				}
				for idx, opt := range menuOptions {
					style := "[white]"
					if idx == menuIdx && app.GetFocus() == statusMenu { style = "[black:yellow]" }
					state := ""
					if opt == "History" && visibility["history"] { state = " (ON)" }
					if opt == "History" && !visibility["history"] { state = " (OFF)" }
					if opt == "Agents" && visibility["agents"] { state = " (ON)" }
					if opt == "Agents" && !visibility["agents"] { state = " (OFF)" }
					if opt == "Planner" && visibility["planner"] { state = " (ON)" }
					if opt == "Planner" && !visibility["planner"] { state = " (OFF)" }
					sb.WriteString(fmt.Sprintf("%s %s%s [-:-:-]  ", style, opt, state))
				}
				statusMenu.SetText(sb.String())
			})
			i++; time.Sleep(200 * time.Millisecond)
		}
	}()

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter { return }
		text := strings.TrimSpace(inputField.GetText())
		inputField.SetText("")
		if text == "" { return }
		fmt.Fprintf(outputView, "[green]You:[white] %s\n", text)
		go func() {
			var resp *http.Response
			var err error

			if strings.HasPrefix(text, "/image ") {
				parts := strings.SplitN(strings.TrimPrefix(text, "/image "), " ", 2)
				path := parts[0]
				prompt := "Describe this image."
				if len(parts) > 1 { prompt = parts[1] }

				b64, err := llm.EncodeImage(path)
				if err != nil {
					app.QueueUpdateDraw(func() { fmt.Fprintf(outputView, "[red]Error loading image: %v[white]\n", err) })
					return
				}
				resp, err = client.Post("/chat", map[string]interface{}{
					"text":   prompt,
					"images": []string{b64},
				})
			} else {
				resp, err = client.Post("/chat", map[string]string{"text": text})
			}

			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					app.QueueUpdateDraw(func() { fmt.Fprintf(outputView, "[red]Server Error: Status %d[white]\n", resp.StatusCode) })
					return
				}
				var data map[string]string
				json.NewDecoder(resp.Body).Decode(&data)
				app.QueueUpdateDraw(func() { fmt.Fprintf(outputView, "\n[yellow]Idony:[white] %s\n\n", data["response"]) })
			} else {
				app.QueueUpdateDraw(func() { fmt.Fprintf(outputView, "[red]Connection Error: %v[white]\n", err) })
			}
		}()
	})

	updateLayout()
	fmt.Fprintf(outputView, "[yellow]Idony TUI Client v1.5.1\n[white]Connected to: %s\n\n", serverAddr)
	if err := app.SetRoot(rootFlex, true).SetFocus(inputField).Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
