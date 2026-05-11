package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

type TelegramService struct{}

type telegramSendMessageRequest struct {
	ChatID                string               `json:"chat_id"`
	Text                  string               `json:"text"`
	ReplyMarkup           *telegramReplyMarkup `json:"reply_markup,omitempty"`
	DisableWebPagePreview bool                 `json:"disable_web_page_preview,omitempty"`
}

type telegramReplyMarkup struct {
	InlineKeyboard [][]telegramInlineButton `json:"inline_keyboard"`
}

type telegramInlineButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type telegramAPIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func (s TelegramService) SendHeartbeatReminder(settings models.Settings, remainingStr, recipients string) error {
	quickLink := BuildQuickHeartbeatURL(settings.HeartbeatToken, true)
	text := fmt.Sprintf("Check-in required.\n\nYour scheduled message for %s will be sent in %s unless you confirm.\n\nTap the button below to send a heartbeat.", recipients, remainingStr)
	return s.sendMessage(settings.TelegramBotToken, settings.TelegramChatID, text, "Send heartbeat", quickLink)
}

func (s TelegramService) SendTestHeartbeat(settings models.Settings) error {
	quickLink := BuildQuickHeartbeatURL(settings.HeartbeatToken, true)
	text := "Test message from Aeterna.\n\nTap the button below to send a heartbeat using your current account."
	return s.sendMessage(settings.TelegramBotToken, settings.TelegramChatID, text, "Send heartbeat", quickLink)
}

func (s TelegramService) sendMessage(botToken, chatID, text, buttonText, buttonURL string) error {
	botToken = strings.TrimSpace(botToken)
	chatID = strings.TrimSpace(chatID)
	if botToken == "" {
		return BadRequest("Telegram bot token is required", nil)
	}
	if chatID == "" {
		return BadRequest("Telegram chat ID is required", nil)
	}

	reqBody := telegramSendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		DisableWebPagePreview: true,
	}
	if buttonText != "" && buttonURL != "" {
		reqBody.ReplyMarkup = &telegramReplyMarkup{
			InlineKeyboard: [][]telegramInlineButton{{
				{Text: buttonText, URL: buttonURL},
			}},
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Internal("Failed to encode Telegram request", err)
	}

	apiBase := strings.TrimRight(os.Getenv("TELEGRAM_API_BASE"), "/")
	if apiBase == "" {
		apiBase = "https://api.telegram.org"
	}
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", apiBase, botToken)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Internal("Failed to create Telegram request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Internal("Telegram request failed", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return BadRequest("Telegram API returned non-2xx status", fmt.Errorf("%s", msg))
	}

	var apiResp telegramAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err == nil && !apiResp.OK {
		msg := strings.TrimSpace(apiResp.Description)
		if msg == "" {
			msg = "unknown Telegram API error"
		}
		return BadRequest("Telegram API rejected the request", fmt.Errorf("%s", msg))
	}

	return nil
}
