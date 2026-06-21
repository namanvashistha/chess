package constant

// Reserved identity for the built-in computer opponent. The bot is a real user
// row so it can occupy a seat and move through the normal pipeline; it's
// recognised by name (games load id+name) and authenticates moves with this
// fixed token.
const (
	BotName  = "chess-bot"
	BotToken = "chess-bot-engine-v1"
)
