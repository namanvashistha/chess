import SwiftUI
import UIKit

/// The Stone palette. Every colour has a light and a dark value: in dark mode
/// the stone dims the way it would in the evening and surfaces turn to dark
/// glass.
enum Palette {
    // board
    // A step further apart than the first cut, so pieces of either colour
    // separate from both squares.
    static let squareLight = Color.dynamic(light: 0xDDD8CB, dark: 0xC1BBAE)
    static let squareDark = Color.dynamic(light: 0x84918A, dark: 0x67736B)
    static let coordinate = Color.dynamic(light: 0x6F7A70, dark: 0x4F5A52)
    /// A soft honey that reads on both limestone and slate.
    static let lastMove = Color.dynamic(light: 0xE8C468, dark: 0xD9B556)

    // The accent is evergreen: a deeper cut of the slate square, so the whole
    // app reads as one family of stone and moss. It marks what you can act on.
    static let accent = Color.dynamic(light: 0x2E6150, dark: 0x7FB79F)
    static let accentTop = Color.dynamic(light: 0x3D7562, dark: 0x93C6AF)
    static let accentEdge = Color.dynamic(light: 0x1F4538, dark: 0x4E7D69)
    static let onAccent = Color.dynamic(light: 0xF3F1EA, dark: 0x0F1A15)
    static let accentText = Color.dynamic(light: 0x2E6150, dark: 0x8DC4AC)
    /// Lines of light on the board: a brighter, softer green than the accent,
    /// so they glow on pale stone instead of reading as dark ink.
    static let ray = Color.dynamic(light: 0x4FA887, dark: 0x8DCBB0)
    static let thinking = Color.dynamic(light: 0x4F7FC9, dark: 0x7DA6E6)
    static let danger = Color.dynamic(light: 0xC23B3B, dark: 0xFF8A8A)

    // app surfaces and text
    static let ink = Color.dynamic(light: 0x22272C, dark: 0xECEEF0)
    static let mist = Color.dynamic(light: 0x687076, dark: 0x9AA2A8)
    static let backgroundTop = Color.dynamic(light: 0xF1EFE9, dark: 0x16191D)
    static let backgroundBottom = Color.dynamic(light: 0xE6E3DC, dark: 0x0F1114)
    static let glow = Color.dynamic(light: 0xFFFAF0, dark: 0x262A2F)
    static let statFill = Color.dynamic(light: 0x22272C, dark: 0xFFFFFF).opacity(0.06)

    // coach verdicts: a text colour and a badge fill each
    static let great = Color.dynamic(light: 0x2F9E5C, dark: 0x6FD69A)
    static let greatFill = Color.dynamic(light: 0xBFE9CF, dark: 0x1F4A33)
    static let goodFill = Color.dynamic(light: 0xDFE7F0, dark: 0x2A3440)
    static let inaccuracy = Color.dynamic(light: 0x9A7206, dark: 0xF0CC64)
    static let inaccuracyFill = Color.dynamic(light: 0xF6E3A6, dark: 0x4A3D14)
    static let mistake = Color.dynamic(light: 0xB8601B, dark: 0xF5A565)
    static let mistakeFill = Color.dynamic(light: 0xF7CFAA, dark: 0x4D2F17)
    // "Oops" is a warm coral, not an alarm red; red stays for check.
    static let blunder = Color.dynamic(light: 0xC2654C, dark: 0xF29C86)
    static let blunderFill = Color.dynamic(light: 0xF6D3C8, dark: 0x4D2C23)
}

extension Color {
    init(hex: UInt32) {
        self.init(UIColor(hex: hex))
    }

    static func dynamic(light: UInt32, dark: UInt32) -> Color {
        Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(hex: dark) : UIColor(hex: light) })
    }
}

extension UIColor {
    convenience init(hex: UInt32) {
        self.init(red: CGFloat((hex >> 16) & 0xFF) / 255,
                  green: CGFloat((hex >> 8) & 0xFF) / 255,
                  blue: CGFloat(hex & 0xFF) / 255,
                  alpha: 1)
    }
}

/// Type: a characterful rounded grotesque for display, the system face for
/// everything else.
enum Typeface {
    static func display(_ size: CGFloat, _ weight: Font.Weight = .heavy) -> Font {
        .system(size: size, weight: weight, design: .rounded)
    }
}

/// The app's background: warm daylight, or evening.
struct AppBackground: View {
    var body: some View {
        LinearGradient(colors: [Palette.backgroundTop, Palette.backgroundBottom], startPoint: .top, endPoint: .bottom)
            .overlay(alignment: .top) {
                RadialGradient(colors: [Palette.glow, .clear], center: .top, startRadius: 0, endRadius: 380)
                    .frame(height: 380)
            }
            .ignoresSafeArea()
    }
}

/// The primary button: evergreen with a pressed-in edge.
struct PrimaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        let pressed = configuration.isPressed
        configuration.label
            .font(Typeface.display(19))
            .foregroundStyle(Palette.onAccent)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 18)
            .background(
                RoundedRectangle(cornerRadius: 20, style: .continuous)
                    .fill(LinearGradient(colors: [Palette.accentTop, Palette.accent], startPoint: .top, endPoint: .bottom))
                    .shadow(color: Palette.accentEdge, radius: 0, y: pressed ? 2 : 6)
                    .shadow(color: Palette.accent.opacity(0.35), radius: 14, y: pressed ? 6 : 12)
            )
            .offset(y: pressed ? 4 : 0)
            .scaleEffect(pressed ? 0.985 : 1)
            .animation(.spring(response: 0.2, dampingFraction: 0.6), value: pressed)
            .onChange(of: pressed) { _, down in if down { Haptics.tap(.medium, intensity: 0.6) } }
    }
}

/// Cards and chips: they give a little under the finger and spring back.
struct PressableStyle: ButtonStyle {
    var scale: CGFloat = 0.965

    func makeBody(configuration: Configuration) -> some View {
        let pressed = configuration.isPressed
        configuration.label
            .scaleEffect(pressed ? scale : 1)
            .brightness(pressed ? -0.02 : 0)
            .animation(.spring(response: 0.25, dampingFraction: 0.55), value: pressed)
            .onChange(of: pressed) { _, down in if down { Haptics.tap(.soft, intensity: 0.8) } }
    }
}
