package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"syscall/js"
	"time"
)

var (
	document = js.Global().Get("document")
	chat     = document.Call("getElementById", "chat")
	input    = document.Call("getElementById", "userInput")
	sendBtn  = document.Call("getElementById", "sendBtn")
	
	loginScreen = document.Call("getElementById", "loginScreen")
	appContent  = document.Call("getElementById", "appContent")
	apiKeyInput = document.Call("getElementById", "apiKeyInput")
	loginBtn    = document.Call("getElementById", "loginBtn")
	logoutBtn   = document.Call("getElementById", "logoutBtn")
	loginError  = document.Call("getElementById", "loginError")

	toolboxBtn        = document.Call("getElementById", "toolboxBtn")
	toolboxModalEl    = document.Call("getElementById", "toolboxModal")
	toolListEl        = document.Call("getElementById", "toolList")
	toolFormContainer = document.Call("getElementById", "toolFormContainer")
	activeToolTitle   = document.Call("getElementById", "activeToolTitle")
	dynamicFields     = document.Call("getElementById", "dynamicFields")
	executeToolBtn    = document.Call("getElementById", "executeToolBtn")
	backToToolboxBtn  = document.Call("getElementById", "backToToolboxBtn")

	historyPanel = document.Call("getElementById", "historyPanel")
	agentsPanel  = document.Call("getElementById", "agentsPanel")
	plannerPanel = document.Call("getElementById", "plannerPanel")

	sidebar          = document.Call("getElementById", "sidebar")
	sidebarOverlay   = document.Call("getElementById", "sidebarOverlay")
	toggleSidebarBtn = document.Call("getElementById", "toggleSidebarBtn")

	currentApiKey = ""
	cachedSchemas map[string]interface{}
	selectedTool  string
	
	isSending = false
)

func main() {
	c := make(chan struct{}, 0)

	// Check for stored key
	storedKey := js.Global().Get("localStorage").Call("getItem", "idony_api_key")
	if !storedKey.IsNull() && !storedKey.IsUndefined() && storedKey.String() != "" {
		currentApiKey = storedKey.String()
		go validateAndLogin(currentApiKey)
	}

	// UI Handlers
	loginBtn.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		key := apiKeyInput.Get("value").String()
		if key != "" { go validateAndLogin(key) }
		return nil
	}))

	logoutBtn.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Get("localStorage").Call("removeItem", "idony_api_key")
		currentApiKey = ""
		appContent.Get("style").Set("display", "none")
		loginScreen.Get("style").Set("display", "flex")
		return nil
	}))

	toolboxBtn.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go showToolbox()
		return nil
	}))

	backToToolboxBtn.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		toolFormContainer.Get("style").Set("display", "none")
		toolListEl.Get("style").Set("display", "flex")
		return nil
	}))

	executeToolBtn.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go executeSelectedTool()
		return nil
	}))

	sendBtn.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go sendMessage()
		return nil
	}))

	input.Set("onkeypress", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].Get("key").String() == "Enter" { go sendMessage() }
		return nil
	}))

	// Sidebar Toggle Logic
	toggleSidebarBtn.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		sidebar.Get("classList").Call("toggle", "show")
		sidebar.Get("classList").Call("toggle", "collapsed")
		sidebarOverlay.Get("classList").Call("toggle", "show")
		return nil
	}))

	sidebarOverlay.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		sidebar.Get("classList").Call("remove", "show")
		sidebar.Get("classList").Call("add", "collapsed")
		sidebarOverlay.Get("classList").Call("remove", "show")
		return nil
	}))

	// Background Polling
	go startPolling()

	fmt.Println("Idony Go-Wasm Frontend Loaded")
	<-c
}

func startPolling() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		if currentApiKey != "" && !isSending {
			updateHistory()
			updateAgents()
			updatePlanner()
		}
		<-ticker.C
	}
}

func updateHistory() {
	resp, err := apiGet("/history")
	if err != nil { return }
	var activities []map[string]interface{}
	if err := json.Unmarshal(resp, &activities); err != nil { return }

	historyPanel.Set("innerHTML", "")
	for _, a := range activities {
		div := document.Call("createElement", "div")
		div.Get("classList").Call("add", "sidebar-item")
		icon := "📝"
		if a["Type"] == "sub-agent" { icon = "🤖" }
		div.Set("innerText", fmt.Sprintf("%s %s", icon, a["Title"]))
		historyPanel.Call("appendChild", div)
	}
}

func updateAgents() {
	resp, err := apiGet("/agents")
	if err != nil { return }
	var agents []map[string]interface{}
	if err := json.Unmarshal(resp, &agents); err != nil { return }

	agentsPanel.Set("innerHTML", "")
	for _, a := range agents {
		div := document.Call("createElement", "div")
		div.Get("classList").Call("add", "sidebar-item")
		div.Set("innerText", fmt.Sprintf("👤 %s", a["Name"]))
		agentsPanel.Call("appendChild", div)
	}
}

func updatePlanner() {
	resp, err := apiGet("/projects")
	if err != nil { return }
	var projects []map[string]interface{}
	if err := json.Unmarshal(resp, &projects); err != nil { return }

	plannerPanel.Set("innerHTML", "")
	for _, p := range projects {
		div := document.Call("createElement", "div")
		div.Get("classList").Call("add", "sidebar-item")
		div.Set("innerText", fmt.Sprintf("📁 %s (%s)", p["Name"], p["Status"]))
		plannerPanel.Call("appendChild", div)
	}
}

func validateAndLogin(key string) {
	loginError.Get("style").Set("display", "none")
	
	currentApiKey = key // Set temporarily for validation
	resp, err := apiGet("/tools")
	
	if err != nil {
		loginError.Set("innerText", "Connection Error: "+err.Error())
		loginError.Get("style").Set("display", "block")
		currentApiKey = ""
		return
	}
	
	var tools []string
	if err := json.Unmarshal(resp, &tools); err != nil {
		loginError.Set("innerText", "Invalid Key")
		loginError.Get("style").Set("display", "block")
		currentApiKey = ""
		return
	}

	js.Global().Get("localStorage").Call("setItem", "idony_api_key", key)
	loginScreen.Get("style").Set("display", "none")
	appContent.Get("style").Set("display", "flex")
	appendMessage("assistant", "Identity verified. Secure link established.")
	
	go updateHistory()
	go updateAgents()
	go updatePlanner()
}

func showToolbox() {
	resp, err := apiGet("/ui/schemas")
	if err != nil { return }
	json.Unmarshal(resp, &cachedSchemas)

	toolListEl.Set("innerHTML", "")
	toolListEl.Get("style").Set("display", "flex")
	toolFormContainer.Get("style").Set("display", "none")

	for name, schema := range cachedSchemas {
		s := schema.(map[string]interface{})
		col := document.Call("createElement", "div")
		col.Get("classList").Call("add", "col-6", "col-md-4")
		card := document.Call("createElement", "div")
		card.Get("classList").Call("add", "tool-card", "text-center")
		card.Set("innerHTML", fmt.Sprintf("<div class='fw-bold'>%s</div><div class='small text-secondary'>%s</div>", name, s["title"]))
		card.Set("onclick", js.FuncOf(func(n string) func(js.Value, []js.Value) interface{} {
			return func(this js.Value, args []js.Value) interface{} { showToolForm(n); return nil }
		}(name)))
		col.Call("appendChild", card)
		toolListEl.Call("appendChild", col)
	}
	js.Global().Get("bootstrap").Get("Modal").Call("getOrCreateInstance", toolboxModalEl).Call("show")
}

func showToolForm(name string) {
	selectedTool = name
	schema := cachedSchemas[name].(map[string]interface{})
	activeToolTitle.Set("innerText", name)
	dynamicFields.Set("innerHTML", "")
	toolListEl.Get("style").Set("display", "none")
	toolFormContainer.Get("style").Set("display", "block")

	fields, ok := schema["fields"].([]interface{})
	if !ok { return }
	for _, f := range fields {
		field := f.(map[string]interface{})
		label := document.Call("createElement", "label")
		label.Get("classList").Call("add", "form-label", "mt-2")
		label.Set("innerText", field["label"].(string))
		var input js.Value
		if field["type"] == "longtext" {
			input = document.Call("createElement", "textarea")
		} else {
			input = document.Call("createElement", "input")
			input.Set("type", "text")
		}
		input.Get("classList").Call("add", "form-control", "bg-dark", "text-white", "border-secondary")
		input.Set("id", "field_"+field["name"].(string))
		dynamicFields.Call("appendChild", label)
		dynamicFields.Call("appendChild", input)
	}
}

func executeSelectedTool() {
	schema := cachedSchemas[selectedTool].(map[string]interface{})
	fields, _ := schema["fields"].([]interface{})
	data := make(map[string]string)
	for _, f := range fields {
		name := f.(map[string]interface{})["name"].(string)
		val := document.Call("getElementById", "field_"+name).Get("value").String()
		data[name] = val
	}
	js.Global().Get("bootstrap").Get("Modal").Call("getOrCreateInstance", toolboxModalEl).Call("hide")
	jsonInput, _ := json.Marshal(data)
	input.Set("value", fmt.Sprintf("/%s %s", selectedTool, string(jsonInput)))
	sendMessage()
}

func appendMessage(role, text string) {
	div := document.Call("createElement", "div")
	div.Get("classList").Call("add", "message", role)
	div.Set("innerText", text)
	chat.Call("appendChild", div)
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		chat.Set("scrollTop", chat.Get("scrollHeight"))
		return nil
	}), 50)
}

func sendMessage() {
	if isSending { return }
	text := input.Get("value").String()
	if text == "" { return }
	
	isSending = true
	input.Set("value", "")
	appendMessage("user", text)
	
	loader := document.Call("getElementById", "loader")
	loader.Get("style").Set("display", "block")
	
	resp, err := apiPost("/chat", map[string]interface{}{"text": text})
	
	loader.Get("style").Set("display", "none")
	isSending = false

	if err != nil {
		appendMessage("assistant", "Terminal Error: "+err.Error())
		return
	}
	
	var data map[string]string
	if err := json.Unmarshal(resp, &data); err != nil {
		appendMessage("assistant", "Malformed Response: "+string(resp))
		return
	}
	appendMessage("assistant", data["response"])
}

func apiPost(path string, body interface{}) ([]byte, error) {
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(jsonBody))
	if err != nil { return nil, err }
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", currentApiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	return io.ReadAll(resp.Body)
}

func apiGet(path string) ([]byte, error) {
	if currentApiKey == "" { return nil, fmt.Errorf("not logged in") }
	
	req, err := http.NewRequest("GET", path, nil)
	if err != nil { return nil, err }
	
	req.Header.Set("X-API-Key", currentApiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	return io.ReadAll(resp.Body)
}
