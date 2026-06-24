package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mlahr/ask-human-telegram/internal/humantelegram"
)

func main() {
	if err := humantelegram.RunAsk(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
