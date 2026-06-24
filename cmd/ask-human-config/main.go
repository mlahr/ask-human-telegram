package main

import (
	"fmt"
	"os"

	"github.com/mlahr/ask-human-telegram/internal/humantelegram"
)

func main() {
	if err := humantelegram.RunConfig(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
