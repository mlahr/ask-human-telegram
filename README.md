## ask-human-telegram

`ask-human-telegram` provides two small Telegram CLIs:

- `ask-human` sends a question to a Telegram chat, waits for the next human reply, and prints that reply to `STDOUT`.
- `notify-human` sends a message to a Telegram chat and exits without waiting for a reply.
- `ask-human-config` saves the Telegram bot token and default chat ID in a per-user config file.

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
4. Copy the HTTP API token that `@BotFather` returns. Use that exact token as the value for `TELEGRAM_BOT_TOKEN`.

Treat the bot token as a secret. Anyone with the token can call the Telegram Bot API as that bot.

### 2. Add the bot to the target chat

For a private one-to-one chat, open the bot in Telegram and press **Start**.

For a group or supergroup:

1. Add the bot as a member of the group.
2. Send at least one message in the group after the bot has joined.
3. If the bot cannot send messages, grant it permission to send messages in the group settings.

For a channel, add the bot as an administrator with permission to post messages. `ask-human` is intended for chats where a human can reply with a normal message, so channels are usually useful only for `notify-human`.

### 3. Find the Telegram chat ID

In the commands below, values inside angle brackets are placeholders. Replace `<BOT_TOKEN_FROM_BOTFATHER>` with the full token that `@BotFather` gave you. Telegram bot tokens usually look like `123456789:AA...`, but the exact value is different for every bot.

Set your real bot token temporarily in your shell:

```bash
export TELEGRAM_BOT_TOKEN="<BOT_TOKEN_FROM_BOTFATHER>"
```

Send a message in the target chat, then call `getUpdates`:

```bash
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getUpdates"
```

Find the `message.chat.id` value in the JSON response. That integer is the chat ID to pass with `--telegram-chat` or store in `ASK_HUMAN_TELEGRAM_CHAT_ID`.

Chat ID shapes:

- Private chat IDs are usually positive integers, for example `<PRIVATE_CHAT_ID>`.
- Group and supergroup chat IDs are usually negative integers, for example `<GROUP_OR_SUPERGROUP_CHAT_ID>`.
- Use the exact integer returned by Telegram, including the leading `-` when present.

If `getUpdates` returns an empty `result` array, send a new message in the chat and run the `curl` command again. For groups, confirm the bot is already a member before sending that message.

## Configuration

Set the Telegram bot token in the `TELEGRAM_BOT_TOKEN` environment variable.
Optionally set `ASK_HUMAN_TELEGRAM_CHAT_ID` so you do not need to pass `--telegram-chat` on every command:

```bash
export TELEGRAM_BOT_TOKEN="<BOT_TOKEN_FROM_BOTFATHER>"
export ASK_HUMAN_TELEGRAM_CHAT_ID="<CHAT_ID_FROM_GETUPDATES>"
```

Replace `<BOT_TOKEN_FROM_BOTFATHER>` with the token returned by `@BotFather`. Replace `<CHAT_ID_FROM_GETUPDATES>` with the integer from `message.chat.id` in the `getUpdates` response.

To persist those values for future runs, use `ask-human-config` after installing or building the commands:

```bash
ask-human-config
```

The command prompts for the bot token and chat ID, then writes them to the per-user config file:

- `${XDG_CONFIG_HOME}/ask-human-telegram/config.env` when `XDG_CONFIG_HOME` is set
- `${HOME}/.config/ask-human-telegram/config.env` otherwise

For scripts, pass both values as flags:

```bash
ask-human-config --telegram-bot-token "<BOT_TOKEN_FROM_BOTFATHER>" --telegram-chat "<CHAT_ID_FROM_GETUPDATES>"
```

The config file is a fallback. `TELEGRAM_BOT_TOKEN` overrides the stored token. `ASK_HUMAN_TELEGRAM_CHAT_ID` overrides the stored chat ID, and `--telegram-chat` overrides both.

## Install

On Debian-based Linux amd64 or arm64 systems, install the latest released Debian package:

```bash
curl -fsSL https://raw.githubusercontent.com/mlahr/ask-human-telegram/main/install.sh | bash
```

The installer downloads the latest GitHub Release `.deb`, verifies it against the release `checksums.txt`, and installs `ask-human`, `notify-human`, and `ask-human-config`.

## Build from source

Build all commands into `./bin`:

```bash
make build
```

Install all commands into `$GOBIN`, or `$GOPATH/bin` when `$GOBIN` is unset:

```bash
make install
```

Plain `go build` and `go install` operate on one package at a time, so use the Makefile when you want all binaries with the names `ask-human`, `notify-human`, and `ask-human-config`.

## Usage

```bash
./ask-human --telegram-chat <CHAT_ID> --timeout <SECONDS> "Your question here"
./notify-human --telegram-chat <CHAT_ID> "Your notification here"
```

### Arguments

- `--telegram-chat`: Telegram chat ID. Required unless `ASK_HUMAN_TELEGRAM_CHAT_ID` or persisted config provides a default.
- `--timeout`: Timeout in seconds for `ask-human`, defaults to `600` (10 minutes)
- Prompt or message text: Required trailing positional text that will be sent to the chat

### Example

```bash
./ask-human --telegram-chat <CHAT_ID_FROM_GETUPDATES> --timeout 600 "What time is it?"
```

If a human replies before the timeout, the program prints the reply:

```bash
14:37
```

Send a notification without waiting for a reply:

```bash
./notify-human --telegram-chat <CHAT_ID_FROM_GETUPDATES> "Deploy finished"
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
export TELEGRAM_BOT_TOKEN="<BOT_TOKEN_FROM_BOTFATHER>"
export ASK_HUMAN_TELEGRAM_CHAT_ID="<CHAT_ID_FROM_GETUPDATES>"
```

Load it before running the commands:

```bash
source .env.local
ask-human "What should this job do next?"
```

Do not commit files that contain `TELEGRAM_BOT_TOKEN`.

## How it works

1. Reads CLI arguments
2. Reads `TELEGRAM_BOT_TOKEN` from the environment or persisted user config
3. Checks the latest Telegram update offset
4. Sends the prompt message to the requested chat
5. Polls Telegram for new messages
6. Accepts the first non-bot message in the same chat that arrives after the prompt
7. Prints the message text or caption to `STDOUT`

Direct replies to the sent prompt are also accepted when they have the same Telegram timestamp as the prompt.

`notify-human` only reads configuration, sends the message, and exits after Telegram accepts the API request.

## Exit behavior

The program exits with a non-zero status if:

- `TELEGRAM_BOT_TOKEN` is missing and no persisted bot token is configured
- `--telegram-chat` is missing and no default chat ID is available from `ASK_HUMAN_TELEGRAM_CHAT_ID` or persisted config
- The prompt or message text is missing
- `ask-human --timeout` is zero or negative
- The Telegram API returns an error
- No human reply is received before the `ask-human` timeout

## Test

```bash
make test
```

## License

MIT
