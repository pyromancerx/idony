package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mmcdole/gofeed"
)

type RSSStore interface {
	AddRSSFeed(url, title, category string) error
	GetRSSFeeds() ([]map[string]string, error)
	GetRSSFeedsByCategory(category string) ([]map[string]string, error)
	IsRSSItemProcessed(guid string) (bool, error)
	MarkRSSItemProcessed(guid, feedURL string) error
}

type RSSTool struct {
	store RSSStore
}

func NewRSSTool(store RSSStore) *RSSTool {
	return &RSSTool{store: store}
}

func (r *RSSTool) Name() string {
	return "rss"
}

func (r *RSSTool) Description() string {
	return `Manages RSS feeds. Actions: "add", "list", "fetch".
JSON Input: {"action": "add|list|fetch", "url": "feed_url", "title": "optional", "category": "optional"}`
}

func (r *RSSTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action   string `json:"action"`
		URL      string `json:"url"`
		Title    string `json:"title"`
		Category string `json:"category"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	switch req.Action {
	case "add":
		if req.URL == "" {
			return "", fmt.Errorf("URL is required for add action")
		}
		err := r.store.AddRSSFeed(req.URL, req.Title, req.Category)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Feed added to category '%s': %s", req.Category, req.URL), nil

	case "list":
		feeds, err := r.store.GetRSSFeeds()
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		sb.WriteString("Current RSS Subscriptions:\n")
		for _, f := range feeds {
			sb.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", f["category"], f["title"], f["url"]))
		}
		if len(feeds) == 0 {
			return "No feeds found.", nil
		}
		return sb.String(), nil

	case "fetch":
		return r.fetchFeeds(ctx, req.Category)

	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}
}

func (r *RSSTool) fetchFeeds(ctx context.Context, category string) (string, error) {
	var feeds []map[string]string
	var err error

	if category != "" {
		feeds, err = r.store.GetRSSFeedsByCategory(category)
	} else {
		feeds, err = r.store.GetRSSFeeds()
	}

	if err != nil {
		return "", err
	}

	fp := gofeed.NewParser()
	var output strings.Builder
	title := "Latest RSS Items"
	if category != "" { title += " (Category: " + category + ")" }
	output.WriteString(title + ":\n")

	for _, f := range feeds {
		feedURL := f["url"]
		feed, err := fp.ParseURLWithContext(feedURL, ctx)
		if err != nil {
			output.WriteString(fmt.Sprintf("\n[Error parsing %s: %v]\n", feedURL, err))
			continue
		}

		output.WriteString(fmt.Sprintf("\n--- %s ---\n", feed.Title))
		count := 0
		for _, item := range feed.Items {
			if count >= 3 { break } 
			
			guid := item.GUID
			if guid == "" { guid = item.Link }
			
			processed, _ := r.store.IsRSSItemProcessed(guid)
			if !processed {
				output.WriteString(fmt.Sprintf("* %s\n  Link: %s\n  Summary: %s\n", item.Title, item.Link, item.Description))
				r.store.MarkRSSItemProcessed(guid, feedURL)
				count++
			}
		}
		if count == 0 {
			output.WriteString("No new items.\n")
		}
	}

	return output.String(), nil
}

func (r *RSSTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "RSS Feed Manager",
		"actions": []map[string]interface{}{
			{
				"name":  "add",
				"label": "Add RSS Feed",
				"fields": []map[string]interface{}{
					{"name": "url", "label": "Feed URL", "type": "string", "required": true},
					{"name": "title", "label": "Title", "type": "string"},
					{"name": "category", "label": "Category", "type": "string", "hint": "Tech, News, Personal"},
				},
			},
			{
				"name":  "list",
				"label": "List All Subscriptions",
				"fields": []map[string]interface{}{},
			},
			{
				"name":  "fetch",
				"label": "Fetch New Items",
				"fields": []map[string]interface{}{
					{"name": "category", "label": "Category", "type": "string", "hint": "Empty for all"},
				},
			},
		},
	}
}
