package telegram

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pyromancer/idony/internal/agent"
	"github.com/pyromancer/idony/internal/config"
	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/tools"
)

type Bridge struct {
	token        string
	bot          *tgbotapi.BotAPI
	agent        *agent.Agent
	store        *db.Store
	conf         *config.Config
	transcriber  *tools.TranscribeTool
	tts          *tools.TTSTool
}

func NewBridge(token string, a *agent.Agent, store *db.Store, conf *config.Config) (*Bridge, error) {
	return &Bridge{
		token:        token,
		agent:        a,
		store:        store,
		conf:         conf,
		transcriber:  tools.NewTranscribeTool(conf, store),
		tts:          tools.NewTTSTool(conf),
	}, nil
}

func (b *Bridge) isAllowed(userID string) bool {
	allowedStr := b.conf.Get("TELEGRAM_ALLOWED_USERS")
	if allowedStr == "*" {
		return true
	}
	users := strings.Split(allowedStr, ",")
	for _, u := range users {
		if strings.TrimSpace(u) == userID {
			return true
		}
	}
	return false
}

func (b *Bridge) Start() {
	backoff := 5 * time.Second
	maxBackoff := 5 * time.Minute

	for {
		log.Println("Attempting to connect to Telegram...")
		bot, err := tgbotapi.NewBotAPI(b.token)
		if err != nil {
			log.Printf("Telegram connection failed: %v. Retrying in %v...", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff on success
		backoff = 5 * time.Second
		b.bot = bot
		log.Printf("Authorized on account %s", b.bot.Self.UserName)

		// Sync commands
		b.registerCommands()

		// Start listening
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		updates := b.bot.GetUpdatesChan(u)

		for update := range updates {
			if update.Message == nil {
				continue
			}

			userID := fmt.Sprintf("%d", update.Message.From.ID)
			if !b.isAllowed(userID) {
				log.Printf("Unauthorized user: %s", userID)
				continue
			}

			go b.handleMessage(update.Message)
		}

		log.Println("Telegram update channel closed. Reconnecting...")
		time.Sleep(time.Second)
	}
}

func (b *Bridge) registerCommands() {
	var tgCommands []tgbotapi.BotCommand
	for _, tool := range b.agent.GetTools() {
		name := strings.ToLower(tool.Name())
		desc := tool.Description()
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		tgCommands = append(tgCommands, tgbotapi.BotCommand{
			Command:     name,
			Description: desc,
		})
	}

	config := tgbotapi.NewSetMyCommands(tgCommands...)
	if _, err := b.bot.Request(config); err != nil {
		log.Printf("Failed to register Telegram commands: %v", err)
	}
}

func (b *Bridge) handleMessage(m *tgbotapi.Message) {
	var input string
	var b64Images []string
	var err error

	if m.Voice != nil {
		input, err = b.processVoice(m.Voice)
		if err != nil {
			b.sendText(m.Chat.ID, fmt.Sprintf("Error processing voice: %v", err))
			return
		}
		b.sendText(m.Chat.ID, fmt.Sprintf("Transcribed: %s", input))
	} else if m.Text != "" {
		input = m.Text
	} else if m.Photo != nil {
		input = m.Caption
		if input == "" {
			input = "Describe this image."
		}
		photo := m.Photo[len(m.Photo)-1]
		b64, err := b.downloadAsBase64(photo.FileID)
		if err != nil {
			b.sendText(m.Chat.ID, fmt.Sprintf("Error downloading photo: %v", err))
			return
		}
		b64Images = append(b64Images, b64)
	}

	if input == "" {
		return
	}

	var response string
	if strings.HasPrefix(input, "/") {
		parts := strings.SplitN(input[1:], " ", 2)
		toolName := parts[0]
		toolInput := ""
		if len(parts) > 1 { toolInput = parts[1] }

		if tool, ok := b.agent.GetTools()[toolName]; ok {
			b.sendText(m.Chat.ID, fmt.Sprintf("[Direct Tool Execution]: %s", toolName))
			if len(b64Images) > 0 {
				b.agent.SetLastUserImages(b64Images)
			}
			response, err = tool.Execute(context.Background(), toolInput)
		} else {
			response = "Command not recognized."
		}
	} else if len(b64Images) > 0 {
		response, err = b.agent.RunVision(context.Background(), input, b64Images)
	} else {
		response, err = b.agent.Run(context.Background(), input)
	}

	if err != nil {
		b.sendText(m.Chat.ID, fmt.Sprintf("Agent Error: %v", err))
		return
	}

	if m.Voice != nil || strings.Contains(strings.ToLower(input), "speak") {
		b.sendVoice(m.Chat.ID, response)
	} else {
		b.sendText(m.Chat.ID, response)
	}
}

func (b *Bridge) processVoice(v *tgbotapi.Voice) (string, error) {
	fileURL, err := b.bot.GetFileDirectURL(v.FileID)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tempFile := filepath.Join("temp_audio", v.FileID+".ogg")
	os.MkdirAll("temp_audio", 0755)
	out, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer out.Close()
	io.Copy(out, resp.Body)

	inputJSON := fmt.Sprintf(`{"action": "file", "path": "%s"}`, tempFile)
	return b.transcriber.Execute(context.Background(), inputJSON)
}

func (b *Bridge) downloadAsBase64(fileID string) (string, error) {
	fileURL, err := b.bot.GetFileDirectURL(fileID)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (b *Bridge) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.bot.Send(msg)
}

func (b *Bridge) sendVoice(chatID int64, text string) {
	cleaned := b.cleanResponseForTTS(text)
	wavPath, err := b.tts.Execute(context.Background(), cleaned)
	if err != nil {
		b.sendText(chatID, text)
		return
	}
	voice := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(wavPath))
	b.bot.Send(voice)
	os.Remove(wavPath)
}

func (b *Bridge) cleanResponseForTTS(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "temp_audio/") || 
		   strings.Contains(trimmed, "tts_") || 
		   strings.Contains(trimmed, ".wav") ||
		   strings.HasPrefix(trimmed, "Observation:") {
			continue
		}
		result = append(result, trimmed)
	}
	return strings.Join(result, " ")
}
