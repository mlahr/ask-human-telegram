# ask-human-telegram

CLI commands for asking or notifying a human through Telegram from scripts.

- `ask-human` sends a prompt, waits for the next accepted human reply in the same chat, and prints the reply to `STDOUT`.
- `notify-human` sends a Telegram message and exits without writing normal output on success.
- `ask-human-config` stores the bot token and default chat ID in a per-user config file.

## Quick start

Install or build the commands, then configure Telegram:

```bash
ask-human-config
```

Send a test notification:

```bash
notify-human "ask-human-telegram is configured"
```

Ask a question and wait for a reply:

```bash
ask-human --timeout 600 "Reply with anything"
```

Run `<command> --help` for command-specific flags.

## Install

On Debian-based Linux amd64 or arm64 systems, install the latest released `.deb` package:

```bash
curl -fsSL https://raw.githubusercontent.com/mlahr/ask-human-telegram/main/install.sh | bash
```

The installer downloads the latest GitHub Release package, verifies it against `checksums.txt`, and installs `ask-human`, `notify-human`, and `ask-human-config`.

## Configure Telegram

You need:

- A Telegram bot token.
- A Telegram chat ID for the private chat, group, supergroup, or channel that should receive messages.
- A bot that can send messages to that chat.

For the detailed bot and chat ID setup, see [docs/telegram-setup.md](docs/telegram-setup.md).

To persist configuration locally:

```bash
ask-human-config --telegram-bot-token "<BOT_TOKEN>" --telegram-chat "<CHAT_ID>"
```

Without flags, `ask-human-config` prompts for both values.

The config file is written with restrictive permissions:

- `${XDG_CONFIG_HOME}/ask-human-telegram/config.env` when `XDG_CONFIG_HOME` is set.
- `${HOME}/.config/ask-human-telegram/config.env` otherwise.

Configuration precedence:

- `TELEGRAM_BOT_TOKEN` overrides the persisted bot token.
- `--telegram-chat` overrides `ASK_HUMAN_TELEGRAM_CHAT_ID`.
- `ASK_HUMAN_TELEGRAM_CHAT_ID` overrides the persisted chat ID.

## Usage

Ask a human and print the reply:

```bash
ask-human --telegram-chat "<CHAT_ID>" --timeout 600 "Approve deploy?"
```

If the chat ID is configured, omit `--telegram-chat`:

```bash
ask-human --timeout 600 "Approve deploy?"
```

Send a notification without waiting for a reply:

```bash
notify-human --telegram-chat "<CHAT_ID>" "Deploy finished"
```

Required values:

- `TELEGRAM_BOT_TOKEN`, either from the environment or persisted config.
- A Telegram chat ID, either from `--telegram-chat`, `ASK_HUMAN_TELEGRAM_CHAT_ID`, or persisted config.
- Prompt or message text as trailing positional arguments.

`ask-human --timeout` is in seconds and defaults to `600`.

## Build from source

Go `1.22` or newer is required.

Build all commands into `./bin`:

```bash
make build
```

Run built binaries directly:

```bash
./bin/ask-human --help
./bin/notify-human --help
./bin/ask-human-config --help
```

Install all commands into `$GOBIN`, or `$GOPATH/bin` when `$GOBIN` is unset:

```bash
make install
```

## Test

```bash
make test
```

## License

MIT
