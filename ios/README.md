# Chess for iPhone

A native SwiftUI app. The Go engine in `app/engine` runs on the phone, so the app needs no server.

## Layout

- `Chess.xcodeproj`: the Xcode project. Files in `Chess/` are picked up automatically, so new Swift files need no project edits.
- `Chess/`: SwiftUI sources. The pieces are drawn in code (`PieceShape.swift`) and rendered in Liquid Glass, so there are no piece images.
- `Frameworks/Chesscore.xcframework`: the engine, built from `mobile/chesscore`. It is not committed.

## Build

1. Build the engine framework from the repository root:

   ```bash
   make ios-engine
   ```

   The Makefile shows how to install the pinned `gomobile`.

2. Open `Chess.xcodeproj` in Xcode and run the `Chess` scheme. The app needs iOS 26 (Liquid Glass), and Xcode needs the matching iOS platform from Xcode > Settings > Components.

Run `make ios-engine` again after any change to the engine.
