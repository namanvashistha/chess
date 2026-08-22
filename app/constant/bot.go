package constant

// BotName is the reserved identity for the built-in computer opponent. The bot
// is a real user row so it can occupy a seat and move through the normal
// pipeline; it's recognised by name (games load id+name).
//
// There used to be a companion BotToken constant holding a fixed credential in
// source. Bot moves are now applied directly as a resolved user instead of
// re-authenticating with a shared secret, so it is gone.
const BotName = "chess-bot"
