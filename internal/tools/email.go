package tools

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/smtp"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/pyromancer/idony/internal/config"
)

type EmailTool struct {
	conf *config.Config
}

func NewEmailTool(conf *config.Config) *EmailTool {
	return &EmailTool{conf: conf}
}

func (e *EmailTool) Name() string {
	return "email"
}

func (e *EmailTool) Description() string {
	return `Manages emails. Actions: "send", "check".
JSON Input: {"action": "send|check", "to": "recipient", "subject": "sub", "body": "msg", "account": "standard|gmail"}`
}

func (e *EmailTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action  string `json:"action"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
		Account string `json:"account"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	account := req.Account
	if account == "" {
		account = e.conf.GetWithDefault("EMAIL_DEFAULT_ACCOUNT", "standard")
	}

	switch req.Action {
	case "send":
		return e.sendEmail(req.To, req.Subject, req.Body, account)
	case "check":
		return e.checkEmails(account)
	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}
}

func (e *EmailTool) sendEmail(to, subject, body, account string) (string, error) {
	if to == "" {
		to = e.conf.Get("EMAIL_TO_ADDRESS")
	}
	
	var host, port, user, pass string
	useSSL := e.conf.Get("SMTP_USE_SSL") == "true"

	if account == "gmail" {
		host = "smtp.gmail.com"
		port = "587"
		user = e.conf.Get("GMAIL_USER")
		pass = e.conf.Get("GMAIL_PASS")
	} else {
		host = e.conf.Get("SMTP_HOST")
		port = e.conf.Get("SMTP_PORT")
		user = e.conf.Get("SMTP_USER")
		pass = e.conf.Get("SMTP_PASS")
	}

	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)
	addr := host + ":" + port
	auth := smtp.PlainAuth("", user, pass, host)

	var err error
	if useSSL {
		tlsConfig := &tls.Config{InsecureSkipVerify: false, ServerName: host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return "", err
		}
		c, err := smtp.NewClient(conn, host)
		if err != nil {
			return "", err
		}
		if err = c.Auth(auth); err != nil {
			return "", err
		}
		if err = c.Mail(user); err != nil {
			return "", err
		}
		if err = c.Rcpt(to); err != nil {
			return "", err
		}
		w, err := c.Data()
		if err != nil {
			return "", err
		}
		_, err = w.Write([]byte(msg))
		if err != nil {
			return "", err
		}
		err = w.Close()
		if err != nil {
			return "", err
		}
		c.Quit()
	} else {
		err = smtp.SendMail(addr, auth, user, []string{to}, []byte(msg))
	}

	if err != nil {
		return "", err
	}
	return "Email sent successfully.", nil
}

func (e *EmailTool) checkEmails(account string) (string, error) {
	var host, port, user, pass string
	useSSL := e.conf.Get("IMAP_USE_SSL") == "true"
	trusted := strings.Split(e.conf.Get("EMAIL_TRUSTED_SENDERS"), ",")

	if account == "gmail" {
		host = "imap.gmail.com"
		port = "993"
		user = e.conf.Get("GMAIL_USER")
		pass = e.conf.Get("GMAIL_PASS")
		useSSL = true
	} else {
		host = e.conf.Get("IMAP_HOST")
		port = e.conf.Get("IMAP_PORT")
		user = e.conf.Get("IMAP_USER")
		pass = e.conf.Get("IMAP_PASS")
	}

	addr := host + ":" + port
	var c *client.Client
	var err error

	if useSSL {
		c, err = client.DialTLS(addr, nil)
	} else {
		c, err = client.Dial(addr)
	}
	if err != nil {
		return "", err
	}
	defer c.Logout()

	if err := c.Login(user, pass); err != nil {
		return "", err
	}

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return "", err
	}

	if mbox.Messages == 0 {
		return "No messages in inbox.", nil
	}

	from := uint32(1)
	if mbox.Messages > 10 {
		from = mbox.Messages - 9
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, mbox.Messages)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody + "[]"}, messages)
	}()

	var output strings.Builder
	output.WriteString("New/Recent Trusted Messages:\n")
	found := false

	for msg := range messages {
		isTrusted := false
		sender := msg.Envelope.From[0].Address()
		for _, t := range trusted {
			if strings.TrimSpace(t) == sender {
				isTrusted = true
				break
			}
		}

		if isTrusted {
			found = true
			section := &imap.BodySectionName{}
			r := msg.GetBody(section)
			mr, err := mail.CreateReader(r)
			if err != nil {
				continue
			}

			bodyText := ""
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				} else if err != nil {
					break
				}
				switch p.Header.(type) {
				case *mail.InlineHeader:
					b, _ := io.ReadAll(p.Body)
					bodyText = string(b)
				}
			}
			output.WriteString(fmt.Sprintf("- From: %s\n  Subject: %s\n  Body: %s\n", sender, msg.Envelope.Subject, bodyText))
		}
	}

	if err := <-done; err != nil {
		return "", err
	}

	if !found {
		return "No messages from trusted senders found.", nil
	}

	return output.String(), nil
}

func (e *EmailTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Email Manager",
		"actions": []map[string]interface{}{
			{
				"name":  "send",
				"label": "Send Email",
				"fields": []map[string]interface{}{
					{"name": "to", "label": "To", "type": "string", "hint": "recipient@example.com"},
					{"name": "subject", "label": "Subject", "type": "string", "required": true},
					{"name": "body", "label": "Body", "type": "longtext", "required": true},
					{"name": "account", "label": "Account", "type": "choice", "options": []string{"standard", "gmail"}},
				},
			},
			{
				"name":  "check",
				"label": "Check Inbox",
				"fields": []map[string]interface{}{
					{"name": "account", "label": "Account", "type": "choice", "options": []string{"standard", "gmail"}},
				},
			},
		},
	}
}
