# Telegram setup

Use this guide to get the two values required by `ask-human-telegram`:

- `TELEGRAM_BOT_TOKEN`
- `ASK_HUMAN_TELEGRAM_CHAT_ID`

## Create a bot token

1. Open Telegram and start a chat with `@BotFather`.
2. Send `/newbot`.
3. Follow the prompts for the bot display name and bot username.
4. Copy the HTTP API token returned by `@BotFather`.

Treat the bot token as a secret. Anyone with the token can call the Telegram Bot API as that bot.

## Add the bot to the target chat

For a private one-to-one chat, open the bot in Telegram and press **Start**.

For a group or supergroup:

1. Add the bot as a member.
2. Send at least one message after the bot joins.
3. If sending fails, grant the bot permission to send messages in the group settings.

For a channel, add the bot as an administrator with permission to post messages. Channels are usually useful for `notify-human`; `ask-human` needs a chat where a human can reply with a normal message.

## Find the chat ID

Set the token temporarily in your shell:

```bash
export TELEGRAM_BOT_TOKEN="<BOT_TOKEN_FROM_BOTFATHER>"
```

Send a message in the target chat, then call `getUpdates`:

```bash
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getUpdates"
```

Find `message.chat.id` in the JSON response. Use that exact integer as the chat ID:

```bash
ask-human-config --telegram-bot-token "${TELEGRAM_BOT_TOKEN}" --telegram-chat "<CHAT_ID_FROM_GETUPDATES>"
```

Private chat IDs are usually positive integers. Group and supergroup chat IDs are usually negative integers. Keep the leading `-` when Telegram returns one.

If `getUpdates` returns an empty `result` array, send a new message in the target chat and run the `curl` command again. For groups, confirm the bot was already a member before that message was sent.
