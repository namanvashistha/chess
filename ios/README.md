# Chess for iPhone

A native SwiftUI app. The Go engine in `app/engine` runs on the phone, so the app needs no server.

## Layout

- `Chess.xcodeproj`: the Xcode project. Files in `Chess/` are picked up automatically, so new Swift files need no project edits.
- `Chess/`: SwiftUI sources and the asset catalog. The piece images are copied from `app/static/images`.
- `Frameworks/Chesscore.xcframework`: the engine, built from `mobile/chesscore`. It is not committed.

## Build

1. Build the engine framework from the repository root:

   ```bash
   make ios-engine
   ```

   The Makefile shows how to install the pinned `gomobile`.

2. Open `Chess.xcodeproj` in Xcode and run the `Chess` scheme. Xcode needs the iOS platform that matches its SDK, which you install from Xcode > Settings > Components.

Run `make ios-engine` again after any change to the engine.
