package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type BrowserManager struct {
	browser *rod.Browser
	page    *rod.Page
	mu      sync.Mutex
}

func NewBrowserManager() *BrowserManager {
	return &BrowserManager{}
}

func (m *BrowserManager) ensurePage() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.browser == nil {
		path, _ := launcher.LookPath()
		u := launcher.New().Bin(path).MustLaunch()
		m.browser = rod.New().ControlURL(u).MustConnect()
	}
	if m.page == nil {
		m.page = m.browser.MustPage()
	}
	return nil
}

type BrowserNativeTool struct {
	manager *BrowserManager
}

func NewBrowserNativeTool(m *BrowserManager) *BrowserNativeTool {
	return &BrowserNativeTool{manager: m}
}

func (b *BrowserNativeTool) Name() string {
	return "browser_native"
}

func (b *BrowserNativeTool) Description() string {
	return `Control a real browser. Actions: navigate, click, type, screenshot, content.
Input: {"action": "navigate", "url": "..."} or {"action": "click", "selector": "..."}`
}

func (b *BrowserNativeTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action   string `json:"action"`
		URL      string `json:"url"`
		Selector string `json:"selector"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", err
	}

	if err := b.manager.ensurePage(); err != nil {
		return "", fmt.Errorf("failed to start browser: %w", err)
	}

	page := b.manager.page
	// Set a timeout for the action
	// page.Timeout(30 * time.Second) // Go-rod timeouts are handled differently, context is better

	switch req.Action {
	case "navigate":
		err := page.Navigate(req.URL)
		if err != nil { return "", err }
		page.MustWaitLoad()
		return fmt.Sprintf("Navigated to %s", req.URL), nil

	case "click":
		el, err := page.Element(req.Selector)
		if err != nil { return "", err }
		if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil { return "", err }
		return fmt.Sprintf("Clicked %s", req.Selector), nil

	case "type":
		el, err := page.Element(req.Selector)
		if err != nil { return "", err }
		if err := el.Input(req.Text); err != nil { return "", err }
		return fmt.Sprintf("Typed '%s' into %s", req.Text, req.Selector), nil

	case "content":
		text, err := page.MustElement("body").Text()
		if err != nil { return "", err }
		if len(text) > 2000 { text = text[:2000] + "..." }
		return text, nil

	case "screenshot":
		data, err := page.Screenshot(true, nil)
		if err != nil { return "", err }
		filename := fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
		os.WriteFile(filename, data, 0644)
		return fmt.Sprintf("Screenshot saved to %s", filename), nil

	default:
		return "", fmt.Errorf("unknown action: %s", req.Action)
	}
}

func (b *BrowserNativeTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Browser Automation",
		"actions": []map[string]interface{}{
			{
				"name": "navigate",
				"label": "Navigate",
				"fields": []map[string]interface{}{
					{"name": "url", "label": "URL", "type": "string", "required": true},
				},
			},
			{
				"name": "click",
				"label": "Click Element",
				"fields": []map[string]interface{}{
					{"name": "selector", "label": "CSS Selector", "type": "string", "required": true},
				},
			},
			{
				"name": "type",
				"label": "Type Text",
				"fields": []map[string]interface{}{
					{"name": "selector", "label": "CSS Selector", "type": "string", "required": true},
					{"name": "text", "label": "Text", "type": "string", "required": true},
				},
			},
			{
				"name": "content",
				"label": "Get Page Text",
				"fields": []map[string]interface{}{},
			},
			{
				"name": "screenshot",
				"label": "Take Screenshot",
				"fields": []map[string]interface{}{},
			},
		},
	}
}
