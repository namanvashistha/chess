import SwiftUI

@main
struct ChessApp: App {
    #if DEBUG
    init() { IconRenderer.renderIfAsked() }
    #endif

    var body: some Scene {
        WindowGroup {
            RootView()
        }
    }
}
