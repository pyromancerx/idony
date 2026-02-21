package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type WebSearchTool struct{}

func (s *WebSearchTool) Name() string { return "web_search" }
func (s *WebSearchTool) Description() string {
	return "Search the web using DuckDuckGo. Input: search query."
}

func (s *WebSearchTool) Execute(ctx context.Context, input string) (string, error) {
	query := strings.TrimSpace(input)
	if query == "" { return "", fmt.Errorf("query is empty") }

	// Use html.duckduckgo.com for easier parsing
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil { return "", err }
	req.Header.Set("User-Agent", "Mozilla/5.0 (Idony AI)")

	resp, err := (&http.Client{}).Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil { return "", err }

	var results []string
	count := 0
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		if count >= 5 { return }
		title := s.Find(".result__title").Text()
		link, _ := s.Find(".result__a").Attr("href")
		snippet := s.Find(".result__snippet").Text()
		
		if title != "" && link != "" {
			results = append(results, fmt.Sprintf("Title: %s\nLink: %s\nSnippet: %s\n", strings.TrimSpace(title), strings.TrimSpace(link), strings.TrimSpace(snippet)))
			count++
		}
	})

	if len(results) == 0 { return "No results found.", nil }
	return strings.Join(results, "---\n"), nil
}

func (s *WebSearchTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Web Search",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Query", "type": "string", "required": true},
		},
	}
}
