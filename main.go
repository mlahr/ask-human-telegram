package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	telegramAPIBase = "https://api.telegram.org"
	pollInterval    = time.Second
	maxPollTimeout  = 30 * time.Second
	tokenEnvVar     = "TELEGRAM_BOT_TOKEN"
)

type cli struct {
	telegramChat int64
	timeout      time.Duration
	prompt       string
}

type telegramClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

type telegramResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description"`
}

type update struct {
	UpdateID int64    `json:"update_id"`
	Message  *message `json:"message"`
}

type message struct {
	MessageID      int64          `json:"message_id"`
	Date           int64          `json:"date"`
	Text           string         `json:"text"`
	Caption        string         `json:"caption"`
	Chat           chat           `json:"chat"`
	From           *user          `json:"from"`
	ReplyToMessage *messageHeader `json:"reply_to_message"`
}

type messageHeader struct {
	MessageID int64 `json:"message_id"`
}

type chat struct {
	ID int64 `json:"id"`
}

type user struct {
	IsBot bool `json:"is_bot"`
}

type sendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type getUpdatesRequest struct {
	Offset         int64    `json:"offset"`
	Timeout        int      `json:"timeout"`
	AllowedUpdates []string `json:"allowed_updates"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	cli, err := parseCLI(args)
	if err != nil {
		return err
	}

	token := os.Getenv(tokenEnvVar)
	if token == "" {
		return fmt.Errorf("%s environment variable is required", tokenEnvVar)
	}

	client := telegramClient{
		httpClient: &http.Client{Timeout: 35 * time.Second},
		baseURL:    telegramAPIBase,
		token:      token,
	}

	offset, err := client.latestUpdateOffset(ctx)
	if err != nil {
		return err
	}

	sentMessage, err := client.sendPrompt(ctx, cli.telegramChat, cli.prompt)
	if err != nil {
		return err
	}

	reply, err := client.waitForReply(ctx, cli.telegramChat, sentMessage, offset, cli.timeout)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, reply)
	return err
}

func parseCLI(args []string) (cli, error) {
	fs := flag.NewFlagSet("ask-human", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	telegramChat := fs.Int64("telegram-chat", 0, "Telegram chat ID")
	timeoutSeconds := fs.Int("timeout", 120, "Timeout in seconds")

	if err := fs.Parse(args); err != nil {
		return cli{}, err
	}
	if *telegramChat == 0 {
		return cli{}, errors.New("--telegram-chat is required")
	}
	if *timeoutSeconds <= 0 {
		return cli{}, errors.New("--timeout must be greater than zero")
	}

	prompt := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if prompt == "" {
		return cli{}, errors.New("prompt text is required")
	}

	return cli{
		telegramChat: *telegramChat,
		timeout:      time.Duration(*timeoutSeconds) * time.Second,
		prompt:       prompt,
	}, nil
}

func (c telegramClient) latestUpdateOffset(ctx context.Context) (int64, error) {
	updates, err := c.getUpdates(ctx, 0, 0)
	if err != nil {
		return 0, err
	}
	if len(updates) == 0 {
		return 0, nil
	}
	return updates[len(updates)-1].UpdateID + 1, nil
}

func (c telegramClient) sendPrompt(ctx context.Context, chatID int64, prompt string) (message, error) {
	response, err := doTelegramRequest[sendMessageRequest, message](ctx, c, "sendMessage", sendMessageRequest{
		ChatID: chatID,
		Text:   prompt,
	})
	if err != nil {
		return message{}, fmt.Errorf("failed to send Telegram prompt: %w", err)
	}
	return response, nil
}

func (c telegramClient) waitForReply(ctx context.Context, chatID int64, sentMessage message, offset int64, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timed out waiting for a reply after %d seconds", int(timeout.Seconds()))
		}

		remaining := time.Until(deadline)
		pollTimeout := minDuration(remaining, maxPollTimeout)
		updates, err := c.getUpdates(ctx, offset, pollTimeout)
		if err != nil {
			return "", err
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			if reply, ok := extractReply(update, chatID, sentMessage); ok {
				return reply, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func extractReply(update update, chatID int64, sentMessage message) (string, bool) {
	if update.Message == nil || update.Message.Chat.ID != chatID || update.Message.From == nil || update.Message.From.IsBot {
		return "", false
	}

	msg := update.Message
	if msg.Date < sentMessage.Date {
		return "", false
	}

	isDirectReply := msg.ReplyToMessage != nil && msg.ReplyToMessage.MessageID == sentMessage.MessageID
	if !isDirectReply && msg.Date <= sentMessage.Date {
		return "", false
	}

	if text := strings.TrimSpace(msg.Text); text != "" {
		return text, true
	}
	if caption := strings.TrimSpace(msg.Caption); caption != "" {
		return caption, true
	}

	return "", false
}

func (c telegramClient) getUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]update, error) {
	seconds := int(timeout.Seconds())
	if seconds < 0 {
		seconds = 0
	}

	updates, err := doTelegramRequest[getUpdatesRequest, []update](ctx, c, "getUpdates", getUpdatesRequest{
		Offset:         offset,
		Timeout:        seconds,
		AllowedUpdates: []string{"message"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Telegram updates: %w", err)
	}
	return updates, nil
}

func doTelegramRequest[TReq any, TRes any](ctx context.Context, client telegramClient, method string, payload TReq) (TRes, error) {
	var zero TRes

	body, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/bot%s/%s", client.baseURL, client.token, method), bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return zero, fmt.Errorf("request to Telegram method %q failed: %w", method, err)
	}
	defer resp.Body.Close()

	var telegramResp telegramResponse[TRes]
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return zero, fmt.Errorf("failed to decode Telegram response for %q: %w", method, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !telegramResp.OK {
		description := telegramResp.Description
		if description == "" {
			description = resp.Status
		}
		return zero, fmt.Errorf("Telegram API %q error: %s", method, description)
	}

	return telegramResp.Result, nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
