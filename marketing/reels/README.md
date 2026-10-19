# Calm Chess reels

Reels made from the app's own art. The current set is built with Remotion
(`remotion/`): a 3D board in evening light with the app's glass pieces,
lines of light, kinetic captions, a synthesized soundtrack, and a brand card
held long enough to read. Each reel is a short script in `remotion/reels.py`,
whose every move is checked by a chess library; `remotion/render.sh` builds,
renders and masters them all into `out/`.

Built for casual viewers: a "wait, what?" in the first seconds, a payoff that
matches the promise, the brand last, and a line that names who to send it to.

| Reel | Hook | The "wait, what?" | Send line |
|---|---|---|---|
| `en-passant` | This capture is LEGAL. | A pawn takes a pawn that isn't in front of it, and the pawn vanishes. | Send this to the friend who'd call it cheating. |
| `stalemate` | He has a QUEEN and a ROOK. Watch him THROW it away. | A crushing move that's a draw; then the mate he missed. | Send this to the friend who always does this. |
| `knights-tour` | This knight will land on EVERY square. | 64 hops, faster and faster, no square twice. | Send this to someone who needs something satisfying. |
| `fools-mate` | You can lose at chess in 2 MOVES. | Checkmate in three seconds. | Send this to the friend who'd play f3. |
| `opera` | He gave away his QUEEN. On purpose. | The sacrifice, then mate two moves later (Morphy, 1858). | Send this to the friend who'd take the queen. |

The Opera Game also has two other hooks to test (`opera-best`, `opera-twoMoves`
in Remotion Studio).

### Captions

Lead with the words people search for, then the hook, then the send:

- **en-passant:** "chess rule most people don't know: en passant 🤯 this capture is completely legal. send this to the friend who'd call it cheating"
- **stalemate:** "how to throw away a winning chess game: stalemate. a queen and a rook up… and it's a draw. send this to the friend who always does this"
- **knights-tour:** "the knight's tour: a chess knight lands on all 64 squares, each exactly once. send this to someone who needs something satisfying"
- **fools-mate:** "fool's mate: the fastest checkmate in chess, in just 2 moves. send this to the friend who'd play f3"
- **opera:** "the most famous checkmate in chess history: Morphy's Opera Game, 1858. he gave away his queen on purpose. send this to the friend who'd take the queen"

Pinned comments ask for an opinion, not the act of commenting: "be honest, did you know this was legal?", "how many squares did you think it'd manage?", "what would you have played?".

### Making a new one

Write a function in `remotion/reels.py` (see the five there: a position, moves
with the frame each starts on, captions, rays, ghosts, the camera and sounds),
add it to the list at the bottom, then:

```bash
remotion/render.sh my-reel
```

Preview and scrub any reel live with `npm --prefix remotion run studio`.

## Earlier reels

The two below were the first attempt: footage recorded from the iPhone
simulator with captions laid over it by ffmpeg. They read as app ads, which
is why the set above replaced them.

Reels made from the real app. Nothing is faked: a debug-only reel studio in the
app plays a scripted scene on the iPhone simulator, which is recorded, and a
composer lays captions, sound and the end card over it.

## Mate in 1. Easy.

`out/mate_in_one.mp4` (1080 × 1920, 14.7 s, loops) and `out/mate_in_one.cover.jpg`.

The idea: a confident "Mate in 1. Easy. 😎" over the Scholar's Mate, where every
chess player sees Qxf7 is mate, and then the queen goes to h7 instead. That
mistake is the hook: people stop to watch, and comment the right move. What
other chess apps do next (red, BLUNDER, a rating drop, a buzzer) is shown for a
second, generically, with no real app imitated. Then the Calm Chess coach just
says "Oops. Undo and try again", the mate follows, and the end card says
"Chess that doesn't yell at you." It fades back into its first frame so it
plays twice before anyone notices.

What it is built for, from what Instagram ranks on:

- **The first three seconds:** a claim viewers want to check, then a visible mistake.
- **Comments:** the urge to correct "Qxh7" with "Qxf7 WAS MATE".
- **Sends:** "send this to the friend who hangs their queen", on screen while
  the win plays.
- **Rewatches:** the seamless loop, and a short running time.

### Posting

**Caption:**

> mate in 1. he played Qxh7. 💀
>
> other chess apps would've roasted him. this one just said "undo and try again" 🥹
>
> would YOU have found it? 👇
>
> Calm Chess · free, no ads, works offline · iPhone & Android

**Pinned comment (from your account):** "be honest, how long did it take you to see Qxf7? 👀"

**Hashtags (3–5 is enough):** #chess #chesspuzzle #learnchess #chesstok #chessforbeginners

**Cover:** `out/mate_in_one.cover.jpg`, the "Mate in 1. Easy. 😎" frame.

**Audio:** post with the reel's own sound (it's original, which Instagram favours).
To try a trending track instead, add it in Instagram at about 20% under the
original sound; only use Instagram's library, which is licensed for Reels.

**Reply to the first comments quickly.** Early replies keep the thread growing,
and every "Qxf7" comment is a chance to answer warmly ("you'd have made the
coach proud 🥹").

### Turning it into an ad

Post it organically first. If after 48 hours it's clearly ahead of your other
posts on shares and saves, boost that post (Ads Manager → Boost, or use it as an
existing post in an App promotion campaign) rather than uploading it as a new ad,
so its comments and likes come along. Start at ₹100–150 a day for a week.

## Only 3% see it in 3 seconds

`out/three_seconds.mp4` (1080 × 1920, 15.5 s, loops) and `out/three_seconds.cover.jpg`.

The A/B partner of Mate in 1. Where that one shows someone failing and lets
viewers correct them, this one challenges the viewer: a real trap (the Petrov,
where Nc6+ is a discovered check that wins the queen), a draining three-second
timer with a ticking clock, then "Time's up". The app's own hint circles the
knight, then shows the move; Nc6+, "DISCOVERED CHECK", the bishop blocks, "Too
late", and the knight takes the queen. It shows off a different strength
(hints that teach) and ends on the same card.

What it is built for: pausing to solve it (watch time), rewatching to see the
move again, and comments with how long it took. "send this to someone who'd
never see it 😏" invites the send.

**Caption:**

> only 3% find this in 3 seconds. did you? 👀
>
> (white to move, win the queen)
>
> Calm Chess shows you the move when you're stuck, then lets you find it yourself.
>
> Calm Chess · free, no ads, works offline · iPhone & Android

**Pinned comment:** "drop your time 👇 no cheating with the hint 😏"

**Hashtags:** #chess #chesspuzzle #chesstactics #learnchess #chessforbeginners

## Testing the two against each other

They are built to differ in one thing, the hook (someone else failing, versus
a challenge to you): the same length, end card, sounds and posting kit.

1. **Use Trial Reels** (Instagram's option when sharing a Reel: it's shown to
   non-followers first, not your followers). Post one, then the other, a day
   apart at the same hour, so neither steals the other's audience.
2. **After 48 hours, compare per view**, not raw totals: shares and sends per
   view, then average watch time, then comments per view. Sends matter most for
   reach beyond your followers.
3. **The winner gets the boost** (₹100–150 a day for a week), and its format
   gets the next three reels. If they're close, make one of each again with a
   new position before deciding: one pair of posts is a hint, not a result.

## Making another

1. Write the scene in `ios/Chess/Reel.swift` (and `Reel.kt` on Android): stage a
   position from a list of moves, then tap squares and pause between beats.
2. Build Debug and record it on the iPhone 17 Pro simulator:

   ```bash
   xcrun simctl status_bar booted override --time "9:41" --batteryState charged --batteryLevel 100
   xcrun simctl io booted recordVideo --codec=h264 --force raw.mov
   ```

   In another terminal, launch the scene, then stop the recording with Ctrl-C when it ends:

   ```bash
   xcrun simctl launch booted com.namanvashistha.cozychess -reel mate-in-one
   ```

3. Read the beat times off the frames (the first frame each shows; they shift by
   a few tenths between takes), and compose:

   ```bash
   python mate_in_one.py raw.mov out/mate_in_one.mp4 --start 5.3 --pick 7.0 --blunder 8.0 --undo 11.0 --pick2 12.3 --mate 13.3 --won 14.5
   ```

   ```bash
   python three_seconds.py raw.mov out/three_seconds.mp4 --start 5.6 --hint 9.6 --hint2 11.65 --pick 14.1 --move 14.85 --reply 17.8 --pick2 18.8 --take 19.5 --stop 20.35
   ```

   Both are built on `reelkit.py`: captions, the app's sounds, tap marks, the
   end card, and a timeline that cuts and freezes the recording on exact frames.
   A scene can script the opponent's replies (`scriptedReplies`) or stop it
   replying (`holdReplies`), so a take always goes the same way.

   The composer needs Pillow and ffmpeg, and uses macOS's SF Rounded and Apple
   Color Emoji fonts.
