package humantelegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func useEmptyConfig(t *testing.T) string {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", filepath.Join(configHome, "home"))
	return filepath.Join(configHome, configDirName, configFileName)
}

func TestParseAskCLI(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	cli, err := ParseAskCLI([]string{"--telegram-chat", "1234", "--timeout", "45", "What", "time", "is", "it?"})
	if err != nil {
		t.Fatalf("ParseAskCLI returned error: %v", err)
	}

	if cli.TelegramChat != 1234 {
		t.Fatalf("TelegramChat = %d, want 1234", cli.TelegramChat)
	}
	if cli.Timeout != 45*time.Second {
		t.Fatalf("Timeout = %s, want 45s", cli.Timeout)
	}
	if cli.Prompt != "What time is it?" {
		t.Fatalf("Prompt = %q, want %q", cli.Prompt, "What time is it?")
	}
}

func TestParseAskCLIDefaultTimeout(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	cli, err := ParseAskCLI([]string{"--telegram-chat", "1234", "What time is it?"})
	if err != nil {
		t.Fatalf("ParseAskCLI returned error: %v", err)
	}

	if cli.Timeout != 10*time.Minute {
		t.Fatalf("Timeout = %s, want 10m0s", cli.Timeout)
	}
}

func TestParseAskCLIUsesTelegramChatEnvVar(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "4321")

	cli, err := ParseAskCLI([]string{"What time is it?"})
	if err != nil {
		t.Fatalf("ParseAskCLI returned error: %v", err)
	}

	if cli.TelegramChat != 4321 {
		t.Fatalf("TelegramChat = %d, want 4321", cli.TelegramChat)
	}
}

func TestParseAskCLIRejectsInvalidTelegramChatEnvVar(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "abc")

	_, err := ParseAskCLI([]string{"What time is it?"})
	if err == nil {
		t.Fatal("ParseAskCLI returned nil error, want invalid env var error")
	}
	if !strings.Contains(err.Error(), ChatIDEnvVar) {
		t.Fatalf("error = %q, want mention of %s", err.Error(), ChatIDEnvVar)
	}
}

func TestParseAskCLIHelp(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	_, err := ParseAskCLI([]string{"--help"})
	if !errors.Is(err, ErrUsageRequested) {
		t.Fatalf("ParseAskCLI error = %v, want %v", err, ErrUsageRequested)
	}
}

func TestRunAskHelpWritesUsage(t *testing.T) {
	useEmptyConfig(t)
	var stdout strings.Builder
	originalChatIDEnvValue, hadChatIDEnv := os.LookupEnv(ChatIDEnvVar)
	if hadChatIDEnv {
		defer func() {
			_ = os.Setenv(ChatIDEnvVar, originalChatIDEnvValue)
		}()
	} else {
		defer func() {
			_ = os.Unsetenv(ChatIDEnvVar)
		}()
	}
	_ = os.Unsetenv(ChatIDEnvVar)

	if err := RunAsk(context.Background(), []string{"--help"}, &stdout); err != nil {
		t.Fatalf("RunAsk returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Usage: ask-human") {
		t.Fatalf("help output = %q, want usage header", output)
	}
	if !strings.Contains(output, "--telegram-chat") {
		t.Fatalf("help output = %q, want telegram-chat flag", output)
	}
	if !strings.Contains(output, ChatIDEnvVar) {
		t.Fatalf("help output = %q, want env var hint", output)
	}
	if !strings.Contains(output, "default: 600") {
		t.Fatalf("help output = %q, want updated default timeout", output)
	}
}

func TestParseNotifyCLI(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	cli, err := ParseNotifyCLI([]string{"--telegram-chat", "1234", "Deploy", "finished"})
	if err != nil {
		t.Fatalf("ParseNotifyCLI returned error: %v", err)
	}

	if cli.TelegramChat != 1234 {
		t.Fatalf("TelegramChat = %d, want 1234", cli.TelegramChat)
	}
	if cli.Message != "Deploy finished" {
		t.Fatalf("Message = %q, want %q", cli.Message, "Deploy finished")
	}
}

func TestParseNotifyCLIUsesTelegramChatEnvVar(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "4321")

	cli, err := ParseNotifyCLI([]string{"Deploy finished"})
	if err != nil {
		t.Fatalf("ParseNotifyCLI returned error: %v", err)
	}

	if cli.TelegramChat != 4321 {
		t.Fatalf("TelegramChat = %d, want 4321", cli.TelegramChat)
	}
}

func TestParseNotifyCLIRejectsMissingMessage(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	_, err := ParseNotifyCLI([]string{"--telegram-chat", "1234"})
	if err == nil {
		t.Fatal("ParseNotifyCLI returned nil error, want missing message error")
	}
	if !strings.Contains(err.Error(), "message text") {
		t.Fatalf("error = %q, want missing message error", err.Error())
	}
}

func TestParseNotifyCLIRejectsMissingTelegramChat(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	_, err := ParseNotifyCLI([]string{"Deploy finished"})
	if err == nil {
		t.Fatal("ParseNotifyCLI returned nil error, want missing chat error")
	}
	if !strings.Contains(err.Error(), "--telegram-chat") {
		t.Fatalf("error = %q, want missing chat error", err.Error())
	}
}

func TestParseNotifyCLIRejectsInvalidTelegramChatEnvVar(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "abc")

	_, err := ParseNotifyCLI([]string{"Deploy finished"})
	if err == nil {
		t.Fatal("ParseNotifyCLI returned nil error, want invalid env var error")
	}
	if !strings.Contains(err.Error(), ChatIDEnvVar) {
		t.Fatalf("error = %q, want mention of %s", err.Error(), ChatIDEnvVar)
	}
}

func TestParseNotifyCLIHelp(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	_, err := ParseNotifyCLI([]string{"--help"})
	if !errors.Is(err, ErrUsageRequested) {
		t.Fatalf("ParseNotifyCLI error = %v, want %v", err, ErrUsageRequested)
	}
}

func TestRunNotifyHelpWritesUsage(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	var stdout strings.Builder

	if err := RunNotify(context.Background(), []string{"--help"}, &stdout); err != nil {
		t.Fatalf("RunNotify returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Usage: notify-human") {
		t.Fatalf("help output = %q, want usage header", output)
	}
	if !strings.Contains(output, "--telegram-chat") {
		t.Fatalf("help output = %q, want telegram-chat flag", output)
	}
	if !strings.Contains(output, ChatIDEnvVar) {
		t.Fatalf("help output = %q, want env var hint", output)
	}
}

func TestRunNotifySendsMessageAndDoesNotFetchUpdates(t *testing.T) {
	useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")

	var paths []string
	var sent sendMessageRequest
	var handlerErr error

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path != "/botfake-token/sendMessage" {
			handlerErr = errors.New("unexpected path: " + r.URL.Path)
			http.Error(w, handlerErr.Error(), http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			handlerErr = err
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":99,"date":1234,"chat":{"id":1234},"text":"Deploy finished"}}`))
	}))
	defer server.Close()

	var stdout strings.Builder
	err := runNotify(context.Background(), []string{"--telegram-chat", "1234", "Deploy", "finished"}, &stdout, func() (Client, error) {
		return NewClient("fake-token", server.URL, server.Client()), nil
	})
	if err != nil {
		t.Fatalf("runNotify returned error: %v", err)
	}
	if handlerErr != nil {
		t.Fatalf("handler error: %v", handlerErr)
	}

	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if len(paths) != 1 {
		t.Fatalf("paths = %v, want exactly one Telegram call", paths)
	}
	if sent.ChatID != 1234 {
		t.Fatalf("sent.ChatID = %d, want 1234", sent.ChatID)
	}
	if sent.Text != "Deploy finished" {
		t.Fatalf("sent.Text = %q, want %q", sent.Text, "Deploy finished")
	}
}

func TestDefaultConfigPathUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath returned error: %v", err)
	}

	want := filepath.Join(configHome, configDirName, configFileName)
	if path != want {
		t.Fatalf("DefaultConfigPath = %q, want %q", path, want)
	}
}

func TestDefaultConfigPathUsesHomeWhenXDGConfigHomeIsUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath returned error: %v", err)
	}

	want := filepath.Join(home, ".config", configDirName, configFileName)
	if path != want {
		t.Fatalf("DefaultConfigPath = %q, want %q", path, want)
	}
}

func TestReadConfigFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.env")
	content := strings.Join([]string{
		"# ask-human-telegram",
		TokenEnvVar + "= token-from-file ",
		ChatIDEnvVar + "= -100123",
		"IGNORED=value",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := ReadConfigFile(configPath)
	if err != nil {
		t.Fatalf("ReadConfigFile returned error: %v", err)
	}

	if config.TelegramBotToken != "token-from-file" {
		t.Fatalf("TelegramBotToken = %q, want token-from-file", config.TelegramBotToken)
	}
	if config.TelegramChatID != "-100123" {
		t.Fatalf("TelegramChatID = %q, want -100123", config.TelegramChatID)
	}
}

func TestDefaultTelegramBotTokenUsesConfigFile(t *testing.T) {
	configPath := useEmptyConfig(t)
	t.Setenv(TokenEnvVar, "")
	if err := WriteConfigFile(configPath, Config{TelegramBotToken: "token-from-config", TelegramChatID: "1234"}); err != nil {
		t.Fatalf("WriteConfigFile returned error: %v", err)
	}

	token, err := DefaultTelegramBotToken()
	if err != nil {
		t.Fatalf("DefaultTelegramBotToken returned error: %v", err)
	}

	if token != "token-from-config" {
		t.Fatalf("token = %q, want token-from-config", token)
	}
}

func TestDefaultTelegramBotTokenEnvVarOverridesConfigFile(t *testing.T) {
	configPath := useEmptyConfig(t)
	t.Setenv(TokenEnvVar, "token-from-env")
	if err := WriteConfigFile(configPath, Config{TelegramBotToken: "token-from-config", TelegramChatID: "1234"}); err != nil {
		t.Fatalf("WriteConfigFile returned error: %v", err)
	}

	token, err := DefaultTelegramBotToken()
	if err != nil {
		t.Fatalf("DefaultTelegramBotToken returned error: %v", err)
	}

	if token != "token-from-env" {
		t.Fatalf("token = %q, want token-from-env", token)
	}
}

func TestDefaultTelegramChatIDUsesConfigFile(t *testing.T) {
	configPath := useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "")
	if err := WriteConfigFile(configPath, Config{TelegramBotToken: "token-from-config", TelegramChatID: "-100123"}); err != nil {
		t.Fatalf("WriteConfigFile returned error: %v", err)
	}

	chatID, err := DefaultTelegramChatID()
	if err != nil {
		t.Fatalf("DefaultTelegramChatID returned error: %v", err)
	}

	if chatID != -100123 {
		t.Fatalf("chatID = %d, want -100123", chatID)
	}
}

func TestDefaultTelegramChatIDEnvVarOverridesConfigFile(t *testing.T) {
	configPath := useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "4321")
	if err := WriteConfigFile(configPath, Config{TelegramBotToken: "token-from-config", TelegramChatID: "1234"}); err != nil {
		t.Fatalf("WriteConfigFile returned error: %v", err)
	}

	chatID, err := DefaultTelegramChatID()
	if err != nil {
		t.Fatalf("DefaultTelegramChatID returned error: %v", err)
	}

	if chatID != 4321 {
		t.Fatalf("chatID = %d, want 4321", chatID)
	}
}

func TestParseAskCLIFlagOverridesEnvVarAndConfigFile(t *testing.T) {
	configPath := useEmptyConfig(t)
	t.Setenv(ChatIDEnvVar, "4321")
	if err := WriteConfigFile(configPath, Config{TelegramBotToken: "token-from-config", TelegramChatID: "5678"}); err != nil {
		t.Fatalf("WriteConfigFile returned error: %v", err)
	}

	cli, err := ParseAskCLI([]string{"--telegram-chat", "1234", "What time is it?"})
	if err != nil {
		t.Fatalf("ParseAskCLI returned error: %v", err)
	}

	if cli.TelegramChat != 1234 {
		t.Fatalf("TelegramChat = %d, want 1234", cli.TelegramChat)
	}
}

func TestRunConfigWritesConfigFile(t *testing.T) {
	configPath := useEmptyConfig(t)

	var stdout strings.Builder
	err := RunConfig([]string{"--telegram-bot-token", "token-from-flag", "--telegram-chat", "-100123"}, strings.NewReader(""), &stdout)
	if err != nil {
		t.Fatalf("RunConfig returned error: %v", err)
	}

	config, err := ReadConfigFile(configPath)
	if err != nil {
		t.Fatalf("ReadConfigFile returned error: %v", err)
	}
	if config.TelegramBotToken != "token-from-flag" {
		t.Fatalf("TelegramBotToken = %q, want token-from-flag", config.TelegramBotToken)
	}
	if config.TelegramChatID != "-100123" {
		t.Fatalf("TelegramChatID = %q, want -100123", config.TelegramChatID)
	}
	if !strings.Contains(stdout.String(), configPath) {
		t.Fatalf("stdout = %q, want config path %q", stdout.String(), configPath)
	}

	fileInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0600 {
		t.Fatalf("config file mode = %v, want 0600", got)
	}

	dirInfo, err := os.Stat(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("config dir mode = %v, want 0700", got)
	}
}

func TestRunConfigPromptsForMissingValues(t *testing.T) {
	configPath := useEmptyConfig(t)

	var stdout strings.Builder
	err := RunConfig(nil, strings.NewReader("token-from-prompt\n1234\n"), &stdout)
	if err != nil {
		t.Fatalf("RunConfig returned error: %v", err)
	}

	config, err := ReadConfigFile(configPath)
	if err != nil {
		t.Fatalf("ReadConfigFile returned error: %v", err)
	}
	if config.TelegramBotToken != "token-from-prompt" {
		t.Fatalf("TelegramBotToken = %q, want token-from-prompt", config.TelegramBotToken)
	}
	if config.TelegramChatID != "1234" {
		t.Fatalf("TelegramChatID = %q, want 1234", config.TelegramChatID)
	}
	if !strings.Contains(stdout.String(), "Telegram bot token: ") {
		t.Fatalf("stdout = %q, want bot token prompt", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Telegram chat ID: ") {
		t.Fatalf("stdout = %q, want chat ID prompt", stdout.String())
	}
}

func TestRunConfigRejectsInvalidTelegramChat(t *testing.T) {
	useEmptyConfig(t)

	err := RunConfig([]string{"--telegram-bot-token", "token-from-flag", "--telegram-chat", "not-an-integer"}, strings.NewReader(""), io.Discard)
	if err == nil {
		t.Fatal("RunConfig returned nil error, want invalid chat ID error")
	}
	if !strings.Contains(err.Error(), ChatIDEnvVar) {
		t.Fatalf("error = %q, want mention of %s", err.Error(), ChatIDEnvVar)
	}
}

func TestExtractReplyAcceptsDirectReply(t *testing.T) {
	sent := Message{
		MessageID: 100,
		Date:      1000,
		Chat:      chat{ID: 77},
	}

	reply, ok := extractReply(update{
		UpdateID: 5,
		Message: &Message{
			MessageID:      101,
			Date:           1000,
			Text:           "answer",
			Chat:           chat{ID: 77},
			From:           &user{IsBot: false},
			ReplyToMessage: &messageHeader{MessageID: 100},
		},
	}, 77, sent)

	if !ok {
		t.Fatal("expected reply to be accepted")
	}
	if reply != "answer" {
		t.Fatalf("reply = %q, want %q", reply, "answer")
	}
}
