# Calm Chess · Google Play

Everything needed to publish on Google Play: the steps, the listing text and the
graphics. The listing mirrors the App Store's (`ios/appstore/README.md`).

## Steps

1. **Developer account.** Sign up at play.google.com/console (a one-time $25) and
   verify your identity. New personal accounts must run a **closed test with at
   least 12 testers for 14 days** before the app can go public, so start
   gathering testers (their Gmail addresses) early.
2. **Upload key.** Create it once, keep it safe and backed up, and never commit it.
   From `android/`:

   ```bash
   keytool -genkeypair -v -keystore upload.jks -alias upload -keyalg RSA -keysize 4096 -validity 10000
   ```

   Then create `android/keystore.properties` (also ignored by git):

   ```properties
   storeFile=upload.jks
   storePassword=…
   keyAlias=upload
   keyPassword=…
   ```

   Google Play keeps the app signing key itself (Play App Signing); this key only
   uploads. If it is ever lost, Play support can reset it.
3. **Build.** From the repository root: `make android-release`. The bundle is at
   `android/app/build/outputs/bundle/release/app-release.aab`.
4. **Create the app** in Play Console: name Calm Chess, default language English
   (United States), App, Free. Fill in the listing and the forms below.
5. **Closed testing.** Testing → Closed testing → create a track, add your testers,
   upload the bundle, and send them the opt-in link. After 14 days with 12 or more
   testers opted in, apply for production access.
6. **Production.** Upload the bundle (or promote the tested release) and submit
   for review, which usually takes a few days.

For each new upload, raise `versionCode` in `android/app/build.gradle.kts`; for each
new version, raise `versionName` too, alongside the iPhone app's version.

## Graphics

- **App icon:** `icon-512.png` (512 × 512), the same art as the iPhone icon.
- **Feature graphic:** `feature-graphic.png` (1024 × 500).
- **Phone screenshots:** `screenshots/` (1080 × 2160), the same set as the App Store:
  1. `1-home.png`: home, with the game in progress
  2. `2-coach.png`: "Great move" from the coach while the opponent thinks
  3. `3-long-press.png`: holding a piece shows what guards and attacks it
  4. `4-hint.png`: a hint showing the move
  5. `5-dark.png`: dark mode, with a knight's moves

The icon layers and the feature graphic are drawn by the app itself: a debug build
launched with `adb shell am start -n com.namanvashistha.calmchess/.MainActivity
--ez renderIcon true` writes them to the app's `files/icon/` folder.

## Store listing

- **App name** (30): Calm Chess
  (if Play objects that the name is in use, Calm Chess: Kind Coach)
- **Short description** (80): Learn chess without the noise. A kind coach explains every move.
- **Category:** Games → Board
- **Tags:** Chess, Board, Strategy, Puzzle, Casual
- **Contact email:** your support email (shown publicly)
- **Website:** https://namanvashistha.com/chess/cozychess/
- **Privacy policy:** https://namanvashistha.com/chess/cozychess/privacy.html

## Full description (4000)

Calm Chess is chess without the pressure.

Start against the Rookie, a true beginner you can beat from your very first game. Win three to face the Master, then the Grandmaster, or pass the phone to a friend. After every move a kind coach tells you, in plain words, whether it was great, good or a slip, and why.

SEE THE GAME
• Lines of light show where a piece can go
• Watch the engine think: its candidate moves appear before it plays
• The board brightens on the side whose move it is
• Hold any piece to see what guards it and what attacks it

LEARN WITHOUT PRESSURE
• A move-by-move coach that explains, never scolds
• Learn the moves: ten tiny lessons, from the rook to checkmate
• Watch your accuracy climb game by game
• Share your great moves and wins as a picture
• Hints when you are stuck
• Undo any move and try again
• Step back through your game to see where it turned

MADE TO FEEL GOOD
• Glass pieces on a stone board, in light and dark
• Soft sounds and gentle haptics
• Drag or tap to move
• In English, Hindi and Spanish

PRIVATE BY DESIGN
• No account, no ads, no tracking
• Works fully offline: the engine runs on your phone
• Your game is saved on your device, right where you left it

## Forms in Play Console

- **App access:** all functionality is available without special access.
- **Ads:** No, the app contains no ads.
- **Content rating:** take the IARC questionnaire as a game with no violence,
  no user interaction or sharing between users, no purchases. Expect Everyone / PEGI 3.
- **Target audience:** 13 and over keeps the Designed for Families rules out of
  scope; choosing younger ages is possible, since the app collects nothing and has
  no ads, but adds a Families review.
- **Data safety:** No data collected, no data shared. The app has no internet
  permission at all; it asks only for vibration.
- **Government apps / financial features / health:** No.

## What's new (version 1.1)

The first version of Calm Chess on Android.
