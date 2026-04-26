package humantelegram

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseAskCLI(t *testing.T) {
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
	t.Setenv(ChatIDEnvVar, "")

	_, err := ParseAskCLI([]string{"--help"})
	if !errors.Is(err, ErrUsageRequested) {
		t.Fatalf("ParseAskCLI error = %v, want %v", err, ErrUsageRequested)
	}
}

func TestRunAskHelpWritesUsage(t *testing.T) {
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
	t.Setenv(ChatIDEnvVar, "")

	_, err := ParseNotifyCLI([]string{"--help"})
	if !errors.Is(err, ErrUsageRequested) {
		t.Fatalf("ParseNotifyCLI error = %v, want %v", err, ErrUsageRequested)
	}
}

func TestRunNotifyHelpWritesUsage(t *testing.T) {
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
