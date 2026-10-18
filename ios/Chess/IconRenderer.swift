#if DEBUG
import SwiftUI

/// Renders the app icon from the app's own knight and stone, so the icon
/// always matches the pieces. Launch a debug build with `-renderIcon`; the
/// 1024-point PNG is written to the app's Documents folder as AppIcon.png.
@MainActor
enum IconRenderer {
    static func renderIfAsked() {
        guard ProcessInfo.processInfo.arguments.contains("-renderIcon") else { return }
        let renderer = ImageRenderer(content: AppIconArt().frame(width: 1024, height: 1024)
            .environment(\.colorScheme, .light))
        renderer.scale = 1
        renderer.isOpaque = true // the App Store refuses icons with an alpha channel
        guard let image = renderer.uiImage, let png = image.pngData(),
              let dir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first else { return }
        try? png.write(to: dir.appendingPathComponent("AppIcon.png"))
    }
}

/// The icon: four squares of the board, limestone and slate, with the white
/// knight standing across them, lit from the top left like the board.
private struct AppIconArt: View {
    var body: some View {
        ZStack {
            VStack(spacing: 0) {
                HStack(spacing: 0) { Palette.squareLight; Palette.squareDark }
                HStack(spacing: 0) { Palette.squareDark; Palette.squareLight }
            }
            Grain().blendMode(.multiply).opacity(0.35)
            RadialGradient(colors: [Color.white.opacity(0.45), .clear],
                           center: UnitPoint(x: 0.3, y: 0.25), startRadius: 0, endRadius: 700)
                .blendMode(.softLight)
            GlassPiece(code: "N")
                .frame(width: 800, height: 800)
                .offset(y: 14)
        }
        .frame(width: 1024, height: 1024)
    }
}
#endif
