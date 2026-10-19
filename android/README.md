# Calm Chess for Android

The Android app: the same calm chess as the iPhone app, in Kotlin and Jetpack
Compose: the intro, Learn the moves, home, the game with its coach, hints and
finish card, Hindi and Spanish, and Share. The Go engine runs on the phone,
bound with gomobile from `mobile/chesscore` into `app/libs/chesscore.aar`, so
the app needs no server.

The iPhone app is the reference. Files mirror its structure and names
(`GameModel`, `BoardView`, `GlassPiece`, `PieceArt`, `Palette`), and the pieces
are drawn from the same path data, so the two stay in step. Translations come
from the iPhone's string catalog (`make android-strings`), and the sounds are
synthesized the same way, so the app ships no audio files.

Not on Android, because they are Apple's: Liquid Glass (a close look-alike is
drawn instead), Game Center invites, SharePlay and the iMessage game. Play a
friend offers pass and play on one phone.

## Build

Needs JDK 21 and the Android SDK (platform 37, build tools 37, NDK).

1. Build the engine library from the repository root. It is not committed.

   ```bash
   make android-engine
   ```

2. Build and install on a phone or emulator:

   ```bash
   cd android && ./gradlew installDebug
   ```

3. Publish: see `playstore/README.md` for the upload key, `make android-release`
   and the listing.

Runs on Android 12 and up (rich haptics, real blur and the system splash
screen everywhere), built against Android 17. Native code is aligned for
16 KB memory pages, as Google Play requires.
