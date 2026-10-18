# Calm Chess for Android

The Android app: the same calm chess as the iPhone app, in Kotlin and Jetpack
Compose. The Go engine in `app/engine` runs on the phone (bound with gomobile
from `mobile/chesscore`), so the app needs no server.

The iPhone app is the reference. Files mirror its structure and names
(`GameModel`, `BoardView`, `GlassPiece`, `PieceArt`, `Palette`), and the pieces
are drawn from the same path data, so the two stay in step.

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

Runs on Android 12 and up (rich haptics, real blur and the system splash
screen everywhere), built against Android 17. Native code is aligned for
16 KB memory pages, as Google Play requires.
