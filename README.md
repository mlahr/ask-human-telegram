## ask-human-telegram

`ask-human-telegram` provides two small Telegram CLIs:

- `ask-human` sends a question to a Telegram chat, waits for the next human reply, and prints that reply to `STDOUT`.
- `notify-human` sends a message to a Telegram chat and exits without waiting for a reply.

It is intended for workflows where a script or automation needs to pause and ask a real person for input.

## Features

- Sends a prompt to a specific Telegram chat using a bot
- Waits for the next non-bot reply in that chat
- Prints the received reply to `STDOUT`
- Sends fire-and-forget notifications without waiting for a reply
- Exits with an error if the timeout is reached
- Exits with an error if the Telegram API request fails

## Requirements

- Go `1.22` or newer to build the program
- A Telegram bot token
- The bot must be able to send messages to the target chat

## Configuration

Set the Telegram bot token in the `TELEGRAM_BOT_TOKEN` environment variable:

```bash
export TELEGRAM_BOT_TOKEN="123456:your-bot-token"
```

## Build

Build both commands into `./bin`:

```bash
./scripts/build
```

Install both commands into `$GOBIN`, or `$GOPATH/bin` when `$GOBIN` is unset:

```bash
./scripts/install
```

Plain `go build` and `go install` operate on one package at a time, so use the scripts when you want both binaries with the names `ask-human` and `notify-human`.

## Usage

```bash
./ask-human --telegram-chat <CHAT_ID> --timeout <SECONDS> "Your question here"
./notify-human --telegram-chat <CHAT_ID> "Your notification here"
```

### Arguments

- `--telegram-chat`: Required Telegram chat ID
- `--timeout`: Timeout in seconds for `ask-human`, defaults to `600` (10 minutes)
- Prompt or message text: Required trailing positional text that will be sent to the chat

### Example

```bash
./ask-human --telegram-chat 123456789 --timeout 600 "What time is it?"
```

If a human replies before the timeout, the program prints the reply:

```bash
14:37
```

Send a notification without waiting for a reply:

```bash
./notify-human --telegram-chat 123456789 "Deploy finished"
```

## How it works

1. Reads CLI arguments
2. Reads `TELEGRAM_BOT_TOKEN` from the environment
3. Checks the latest Telegram update offset
4. Sends the prompt message to the requested chat
5. Polls Telegram for new messages
6. Accepts the first non-bot message in the same chat that arrives after the prompt
7. Prints the message text or caption to `STDOUT`

Direct replies to the sent prompt are also accepted when they have the same Telegram timestamp as the prompt.

`notify-human` only reads configuration, sends the message, and exits after Telegram accepts the API request.

## Exit behavior

The program exits with a non-zero status if:

- `TELEGRAM_BOT_TOKEN` is missing
- `--telegram-chat` is missing
- The prompt or message text is missing
- `ask-human --timeout` is zero or negative
- The Telegram API returns an error
- No human reply is received before the `ask-human` timeout

## Test

```bash
go test ./...
```

## License

MIT
