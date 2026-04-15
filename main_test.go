package main

import (
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
