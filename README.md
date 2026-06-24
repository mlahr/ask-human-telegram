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
- A Telegram chat ID for the private chat, group, supergroup, or channel that should receive messages
- The bot must be able to send messages to the target chat

## Telegram setup

### 1. Create a Telegram bot

1. Open Telegram and start a chat with `@BotFather`.
2. Send `/newbot`.
3. Follow the prompts for the bot display name and bot username.
4. Copy the HTTP API token that `@BotFather` returns. This is the value for `TELEGRAM_BOT_TOKEN`.

Treat the bot token as a secret. Anyone with the token can call the Telegram Bot API as that bot.

### 2. Add the bot to the target chat

For a private one-to-one chat, open the bot in Telegram and press **Start**.

For a group or supergroup:

1. Add the bot as a member of the group.
2. Send at least one message in the group after the bot has joined.
3. If the bot cannot send messages, grant it permission to send messages in the group settings.

For a channel, add the bot as an administrator with permission to post messages. `ask-human` is intended for chats where a human can reply with a normal message, so channels are usually useful only for `notify-human`.

### 3. Find the Telegram chat ID

Set the token temporarily in your shell:

```bash
export TELEGRAM_BOT_TOKEN="123456:your-bot-token"
```

Send a message in the target chat, then call `getUpdates`:

```bash
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getUpdates"
```

Find the `message.chat.id` value in the JSON response. That integer is the chat ID to pass with `--telegram-chat` or store in `ASK_HUMAN_TELEGRAM_CHAT_ID`.

Chat ID shapes:

- Private chat IDs are usually positive integers, for example `123456789`.
- Group and supergroup chat IDs are usually negative integers, for example `-1001234567890`.
- Use the exact integer returned by Telegram, including the leading `-` when present.

If `getUpdates` returns an empty `result` array, send a new message in the chat and run the `curl` command again. For groups, confirm the bot is already a member before sending that message.

## Configuration

Set the Telegram bot token in the `TELEGRAM_BOT_TOKEN` environment variable.
Optionally set `ASK_HUMAN_TELEGRAM_CHAT_ID` so you do not need to pass `--telegram-chat` on every command:

```bash
export TELEGRAM_BOT_TOKEN="123456:your-bot-token"
export ASK_HUMAN_TELEGRAM_CHAT_ID="123456789"
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

- `--telegram-chat`: Telegram chat ID. Required unless `ASK_HUMAN_TELEGRAM_CHAT_ID` is set.
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

If `ASK_HUMAN_TELEGRAM_CHAT_ID` is set, omit `--telegram-chat`:

```bash
./ask-human --timeout 600 "Approve deploy to production?"
./notify-human "Deploy finished"
```

## Integration examples

### Shell script approval gate

Use `ask-human` when automation must pause until a person answers:

```bash
#!/usr/bin/env bash
set -euo pipefail

answer="$(ask-human --timeout 900 "Deploy ${GIT_SHA} to production? Reply yes or no.")"

case "${answer}" in
  yes|YES|Yes)
    ./deploy-production
    notify-human "Production deploy for ${GIT_SHA} completed"
    ;;
  *)
    notify-human "Production deploy for ${GIT_SHA} skipped after reply: ${answer}"
    exit 1
    ;;
esac
```

`ask-human` prints only the accepted human reply to `STDOUT`, so command substitution can capture it without parsing Telegram response JSON.

### CI or cron notification

Use `notify-human` when automation should send a status message and continue immediately:

```bash
notify-human "Nightly backup completed on $(hostname)"
```

### Environment file

For local use, store configuration in a shell-specific environment file that is not committed to the repository:

```bash
export TELEGRAM_BOT_TOKEN="123456:your-bot-token"
export ASK_HUMAN_TELEGRAM_CHAT_ID="-1001234567890"
```

Load it before running the commands:

```bash
source .env.local
ask-human "What should this job do next?"
```

Do not commit files that contain `TELEGRAM_BOT_TOKEN`.

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
