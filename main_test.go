package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseCLI(t *testing.T) {
	cli, err := parseCLI([]string{"--telegram-chat", "1234", "--timeout", "45", "What", "time", "is", "it?"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}

	if cli.telegramChat != 1234 {
		t.Fatalf("telegramChat = %d, want 1234", cli.telegramChat)
	}
	if cli.timeout != 45*time.Second {
		t.Fatalf("timeout = %s, want 45s", cli.timeout)
	}
	if cli.prompt != "What time is it?" {
		t.Fatalf("prompt = %q, want %q", cli.prompt, "What time is it?")
	}
}

func TestParseCLIDefaultTimeout(t *testing.T) {
	cli, err := parseCLI([]string{"--telegram-chat", "1234", "What time is it?"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}

	if cli.timeout != 10*time.Minute {
		t.Fatalf("timeout = %s, want 10m0s", cli.timeout)
	}
}

func TestParseCLIUsesTelegramChatEnvVar(t *testing.T) {
	t.Setenv(chatIDEnvVar, "4321")

	cli, err := parseCLI([]string{"What time is it?"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}

	if cli.telegramChat != 4321 {
		t.Fatalf("telegramChat = %d, want 4321", cli.telegramChat)
	}
}

func TestParseCLIRejectsInvalidTelegramChatEnvVar(t *testing.T) {
	t.Setenv(chatIDEnvVar, "abc")

	_, err := parseCLI([]string{"What time is it?"})
	if err == nil {
		t.Fatal("parseCLI returned nil error, want invalid env var error")
	}
	if !strings.Contains(err.Error(), chatIDEnvVar) {
		t.Fatalf("error = %q, want mention of %s", err.Error(), chatIDEnvVar)
	}
}

func TestParseCLIHelp(t *testing.T) {
	_, err := parseCLI([]string{"--help"})
	if !errors.Is(err, errUsageRequested) {
		t.Fatalf("parseCLI error = %v, want %v", err, errUsageRequested)
	}
}

func TestRunHelpWritesUsage(t *testing.T) {
	var stdout strings.Builder
	originalChatIDEnvValue, hadChatIDEnv := os.LookupEnv(chatIDEnvVar)
	if hadChatIDEnv {
		defer func() {
			_ = os.Setenv(chatIDEnvVar, originalChatIDEnvValue)
		}()
	} else {
		defer func() {
			_ = os.Unsetenv(chatIDEnvVar)
		}()
	}
	_ = os.Unsetenv(chatIDEnvVar)

	if err := run(context.Background(), []string{"--help"}, &stdout); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Usage: ask-human") {
		t.Fatalf("help output = %q, want usage header", output)
	}
	if !strings.Contains(output, "--telegram-chat") {
		t.Fatalf("help output = %q, want telegram-chat flag", output)
	}
	if !strings.Contains(output, chatIDEnvVar) {
		t.Fatalf("help output = %q, want env var hint", output)
	}
	if !strings.Contains(output, "default: 600") {
		t.Fatalf("help output = %q, want updated default timeout", output)
	}
}

func TestExtractReplyAcceptsDirectReply(t *testing.T) {
	sent := message{
		MessageID: 100,
		Date:      1000,
		Chat:      chat{ID: 77},
	}

	reply, ok := extractReply(update{
		UpdateID: 5,
		Message: &message{
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
