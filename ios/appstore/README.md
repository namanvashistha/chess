# Cozy Chess · App Store

Everything needed to publish: the steps, the listing text and the screenshots.

## Steps

1. **Create the app.** In App Store Connect choose Apps → + → New App. Pick iOS, name it
   Cozy Chess, choose English (U.S.) as the primary language, bundle ID
   `com.namanvashistha.cozychess`, and SKU `cozychess-ios`.
2. **Upload a build.** Run `make ios-upload` from the repo root, or in Xcode use
   Product → Archive → Distribute App → App Store Connect. Processing takes 5–30 minutes.
3. **TestFlight.** Add yourself as an internal tester, install on your iPhone and play one full game.
4. **Fill in the listing** below, upload `screenshots/` under the 6.9" iPhone size, and
   under App Privacy choose "Data Not Collected".
5. **Submit for review.** Select the build, answer that it uses no encryption beyond
   Apple's (this is already set in the build), and submit. Review usually takes 1–2 days.

For each new upload, raise `CURRENT_PROJECT_VERSION` (the build number) in the project.
For each new App Store version, raise `MARKETING_VERSION` as well.

## Screenshots

`screenshots/` is 6.9" (1320 × 2868); `screenshots-6.3/` is the same set at 1206 × 2622 for the
"iPhone with Dynamic Island (medium display)" slot.

1. `1-home.jpg`: home, with the game in progress
2. `2-coach.jpg`: "Great move" from the coach while the opponent thinks
3. `3-long-press.jpg`: long-press shows what attacks and guards a piece
4. `4-hint.jpg`: Show me
5. `5-dark.jpg`: dark mode, with a knight's moves


## App information

- **Name** (30): Cozy Chess
- **Subtitle** (30): Calm chess with a kind coach
- **Bundle ID:** com.namanvashistha.cozychess
- **SKU:** cozychess-ios
- **Primary category:** Games → Board · **Secondary:** Education
- **Age rating:** 4+ (answer "None" to every question)
- **Price:** Free
- **Support URL:** https://namanvashistha.com/chess/cozychess/
- **Privacy policy URL:** https://namanvashistha.com/chess/cozychess/privacy.html
- **App privacy:** Data Not Collected
- **Copyright:** 2026 Naman Vashistha

## Promotional text (170)

Learn chess the cozy way. A coach explains every move in plain words, lines of light show what each piece can do, and nothing ever rushes you.

## Description (4000)

Cozy Chess is chess without the pressure.

Start against Easy, a true beginner you can beat from your very first game. Win three to open Medium, then Hard — or pass the phone to a friend. After every move a calm coach tells you, in plain words, whether it was great, good or a slip, and why.

SEE THE GAME
• Lines of light show where a piece can go
• Watch the engine think: its candidate moves appear before it plays
• The board brightens on the side that is ahead
• Long-press any piece to see what guards it and what attacks it

LEARN WITHOUT PRESSURE
• A move-by-move coach that explains, never scolds
• Watch your accuracy climb game by game
• Share your great moves and wins as a picture
• "Show me" hints when you are stuck
• Undo any move and try again
• Step back through your game to see where it turned

MADE TO FEEL GOOD
• Glass pieces on a stone board, in light and dark
• Soft sounds and gentle haptics
• Drag or tap to move

PRIVATE BY DESIGN
• No account, no ads, no tracking
• Works fully offline — the engine runs on your iPhone
• Your game is saved on your device, right where you left it

## Keywords (100)

chess,learn,coach,beginner,board,strategy,offline,relaxing,tutor,engine,training,hint,calm,classic

## What's new (version 1.0)

The first version of Cozy Chess.

## Notes for App Review

No sign-in is needed. Tap an opponent, then Play. Pieces move by tap-tap or drag; long-press a piece to see its guards and attackers. The app makes no network requests.
