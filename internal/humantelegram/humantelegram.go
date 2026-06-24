package humantelegram

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	telegramAPIBase = "https://api.telegram.org"
	pollInterval    = time.Second
	maxPollTimeout  = 30 * time.Second

	TokenEnvVar  = "TELEGRAM_BOT_TOKEN"
	ChatIDEnvVar = "ASK_HUMAN_TELEGRAM_CHAT_ID"

	configDirName  = "ask-human-telegram"
	configFileName = "config.env"

	DefaultTimeout = 10 * time.Minute
)

var ErrUsageRequested = errors.New("usage requested")

type AskCLI struct {
	TelegramChat int64
	Timeout      time.Duration
	Prompt       string
}

type NotifyCLI struct {
	TelegramChat int64
	Message      string
}

type Config struct {
	TelegramBotToken string
	TelegramChatID   string
}

type Client struct {
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
	Message  *Message `json:"message"`
}

type Message struct {
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

func RunAsk(ctx context.Context, args []string, stdout io.Writer) error {
	cli, err := ParseAskCLI(args)
	if err != nil {
		if errors.Is(err, ErrUsageRequested) {
			_, writeErr := io.WriteString(stdout, AskUsageText())
			return writeErr
		}
		return err
	}

	client, err := NewEnvClient()
	if err != nil {
		return err
	}

	offset, err := client.LatestUpdateOffset(ctx)
	if err != nil {
		return err
	}

	sentMessage, err := client.SendMessage(ctx, cli.TelegramChat, cli.Prompt)
	if err != nil {
		return err
	}

	reply, err := client.WaitForReply(ctx, cli.TelegramChat, sentMessage, offset, cli.Timeout)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, reply)
	return err
}

func RunNotify(ctx context.Context, args []string, stdout io.Writer) error {
	return runNotify(ctx, args, stdout, NewEnvClient)
}

func runNotify(ctx context.Context, args []string, stdout io.Writer, clientFactory func() (Client, error)) error {
	cli, err := ParseNotifyCLI(args)
	if err != nil {
		if errors.Is(err, ErrUsageRequested) {
			_, writeErr := io.WriteString(stdout, NotifyUsageText())
			return writeErr
		}
		return err
	}

	client, err := clientFactory()
	if err != nil {
		return err
	}

	_, err = client.SendMessage(ctx, cli.TelegramChat, cli.Message)
	return err
}

func NewEnvClient() (Client, error) {
	token, err := DefaultTelegramBotToken()
	if err != nil {
		return Client{}, err
	}
	if token == "" {
		return Client{}, fmt.Errorf("%s environment variable or persisted config value is required", TokenEnvVar)
	}

	return NewClient(token, telegramAPIBase, &http.Client{Timeout: 35 * time.Second}), nil
}

func NewClient(token string, baseURL string, httpClient *http.Client) Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		token:      token,
	}
}

func RunConfig(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("ask-human-config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	token := fs.String("telegram-bot-token", "", "Telegram bot token")
	chatID := fs.String("telegram-chat", "", "Telegram chat ID")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := io.WriteString(stdout, ConfigUsageText())
			return writeErr
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}

	scanner := bufio.NewScanner(stdin)
	if strings.TrimSpace(*token) == "" {
		prompt, err := readPrompt(scanner, stdout, "Telegram bot token: ")
		if err != nil {
			return err
		}
		*token = prompt
	}
	if strings.TrimSpace(*chatID) == "" {
		prompt, err := readPrompt(scanner, stdout, "Telegram chat ID: ")
		if err != nil {
			return err
		}
		*chatID = prompt
	}

	config := Config{
		TelegramBotToken: strings.TrimSpace(*token),
		TelegramChatID:   strings.TrimSpace(*chatID),
	}
	if config.TelegramBotToken == "" {
		return fmt.Errorf("%s is required", TokenEnvVar)
	}
	if config.TelegramChatID == "" {
		return fmt.Errorf("%s is required", ChatIDEnvVar)
	}
	if _, err := parseChatID(config.TelegramChatID, ChatIDEnvVar); err != nil {
		return err
	}

	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	if err := WriteConfigFile(configPath, config); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Saved Telegram configuration to %s\n", configPath)
	return err
}

func readPrompt(scanner *bufio.Scanner, stdout io.Writer, prompt string) (string, error) {
	if _, err := io.WriteString(stdout, prompt); err != nil {
		return "", err
	}
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", errors.New("input ended before all required values were provided")
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func DefaultConfigPath() (string, error) {
	if xdgConfigHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, configDirName, configFileName), nil
	}

	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", errors.New("HOME environment variable is required to locate the config file")
		}
		home = userHome
	}
	if home == "" {
		return "", errors.New("HOME environment variable is required to locate the config file")
	}

	return filepath.Join(home, ".config", configDirName, configFileName), nil
}

func LoadDefaultConfig() (Config, error) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return Config{}, nil
	}

	config, err := ReadConfigFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	return config, nil
}

func ReadConfigFile(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var config Config
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("parse config %q line %d: expected KEY=value", path, lineNumber)
		}

		switch strings.TrimSpace(key) {
		case TokenEnvVar:
			config.TelegramBotToken = strings.TrimSpace(value)
		case ChatIDEnvVar:
			config.TelegramChatID = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	return config, nil
}

func WriteConfigFile(path string, config Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("set config directory permissions %q: %w", dir, err)
	}

	content := fmt.Sprintf("%s=%s\n%s=%s\n", TokenEnvVar, config.TelegramBotToken, ChatIDEnvVar, config.TelegramChatID)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("set config file permissions %q: %w", path, err)
	}

	return nil
}

func DefaultTelegramBotToken() (string, error) {
	if value := strings.TrimSpace(os.Getenv(TokenEnvVar)); value != "" {
		return value, nil
	}

	config, err := LoadDefaultConfig()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(config.TelegramBotToken), nil
}

func ParseAskCLI(args []string) (AskCLI, error) {
	fs := flag.NewFlagSet("ask-human", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	defaultChatID, err := DefaultTelegramChatID()
	if err != nil {
		return AskCLI{}, err
	}

	telegramChat := fs.Int64("telegram-chat", defaultChatID, "Telegram chat ID")
	timeoutSeconds := fs.Int("timeout", int(DefaultTimeout/time.Second), "Timeout in seconds")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return AskCLI{}, ErrUsageRequested
		}
		return AskCLI{}, err
	}
	if *telegramChat == 0 {
		return AskCLI{}, errors.New("--telegram-chat is required")
	}
	if *timeoutSeconds <= 0 {
		return AskCLI{}, errors.New("--timeout must be greater than zero")
	}

	prompt := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if prompt == "" {
		return AskCLI{}, errors.New("prompt text is required")
	}

	return AskCLI{
		TelegramChat: *telegramChat,
		Timeout:      time.Duration(*timeoutSeconds) * time.Second,
		Prompt:       prompt,
	}, nil
}

func ParseNotifyCLI(args []string) (NotifyCLI, error) {
	fs := flag.NewFlagSet("notify-human", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	defaultChatID, err := DefaultTelegramChatID()
	if err != nil {
		return NotifyCLI{}, err
	}

	telegramChat := fs.Int64("telegram-chat", defaultChatID, "Telegram chat ID")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return NotifyCLI{}, ErrUsageRequested
		}
		return NotifyCLI{}, err
	}
	if *telegramChat == 0 {
		return NotifyCLI{}, errors.New("--telegram-chat is required")
	}

	message := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if message == "" {
		return NotifyCLI{}, errors.New("message text is required")
	}

	return NotifyCLI{
		TelegramChat: *telegramChat,
		Message:      message,
	}, nil
}

func DefaultTelegramChatID() (int64, error) {
	value := strings.TrimSpace(os.Getenv(ChatIDEnvVar))
	if value == "" {
		config, err := LoadDefaultConfig()
		if err != nil {
			return 0, err
		}
		value = strings.TrimSpace(config.TelegramChatID)
	}
	if value == "" {
		return 0, nil
	}

	return parseChatID(value, ChatIDEnvVar)
}

func parseChatID(value string, name string) (int64, error) {
	chatID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer chat ID", name)
	}

	return chatID, nil
}

func AskUsageText() string {
	return "Usage: ask-human --telegram-chat <CHAT_ID> [--timeout <SECONDS>] \"<prompt>\"\n\n" +
		"Options:\n" +
		"  --telegram-chat <CHAT_ID>  Telegram chat ID to send the prompt to (defaults to ASK_HUMAN_TELEGRAM_CHAT_ID or persisted config)\n" +
		"  --timeout <SECONDS>        How long to wait for a reply (default: 600)\n" +
		"  --help                     Show this help text\n"
}

func NotifyUsageText() string {
	return "Usage: notify-human --telegram-chat <CHAT_ID> \"<message>\"\n\n" +
		"Options:\n" +
		"  --telegram-chat <CHAT_ID>  Telegram chat ID to send the message to (defaults to ASK_HUMAN_TELEGRAM_CHAT_ID or persisted config)\n" +
		"  --help                     Show this help text\n"
}

func ConfigUsageText() string {
	return "Usage: ask-human-config [--telegram-bot-token <TOKEN>] [--telegram-chat <CHAT_ID>]\n\n" +
		"Options:\n" +
		"  --telegram-bot-token <TOKEN>  Telegram bot token from BotFather\n" +
		"  --telegram-chat <CHAT_ID>     Telegram chat ID from getUpdates\n" +
		"  --help                        Show this help text\n"
}

func (c Client) LatestUpdateOffset(ctx context.Context) (int64, error) {
	updates, err := c.getUpdates(ctx, 0, 0)
	if err != nil {
		return 0, err
	}
	if len(updates) == 0 {
		return 0, nil
	}
	return updates[len(updates)-1].UpdateID + 1, nil
}

func (c Client) SendMessage(ctx context.Context, chatID int64, text string) (Message, error) {
	response, err := doTelegramRequest[sendMessageRequest, Message](ctx, c, "sendMessage", sendMessageRequest{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		return Message{}, fmt.Errorf("failed to send Telegram message: %w", err)
	}
	return response, nil
}

func (c Client) WaitForReply(ctx context.Context, chatID int64, sentMessage Message, offset int64, timeout time.Duration) (string, error) {
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

func extractReply(update update, chatID int64, sentMessage Message) (string, bool) {
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

func (c Client) getUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]update, error) {
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

func doTelegramRequest[TReq any, TRes any](ctx context.Context, client Client, method string, payload TReq) (TRes, error) {
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
