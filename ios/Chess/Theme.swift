import SwiftUI
import UIKit

/// The Stone palette. Every colour has a light and a dark value: in dark mode
/// the stone dims the way it would in the evening and surfaces turn to dark
/// glass.
enum Palette {
    // board
    static let squareLight = Color.dynamic(light: 0xD9D4C8, dark: 0xBDB7AA)
    static let squareDark = Color.dynamic(light: 0x8E9A90, dark: 0x717D73)
    static let coordinate = Color.dynamic(light: 0x6F7A70, dark: 0x4F5A52)

    // light and accents
    static let amber = Color(hex: 0xF0A83A)
    static let amberTop = Color(hex: 0xFFC867)
    static let amberEdge = Color(hex: 0xC47F1D)
    static let amberInk = Color(hex: 0x2A1A02)
    static let amberText = Color.dynamic(light: 0xA8640A, dark: 0xF3B44C)
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
    static let blunder = Color.dynamic(light: 0xC23B3B, dark: 0xFF8A8A)
    static let blunderFill = Color.dynamic(light: 0xF6C1C1, dark: 0x4F2222)
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

/// The big amber button with a pressed-in edge.
struct AmberButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        let pressed = configuration.isPressed
        configuration.label
            .font(Typeface.display(19))
            .foregroundStyle(Palette.amberInk)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 18)
            .background(
                RoundedRectangle(cornerRadius: 20, style: .continuous)
                    .fill(LinearGradient(colors: [Palette.amberTop, Palette.amber], startPoint: .top, endPoint: .bottom))
                    .shadow(color: Palette.amberEdge, radius: 0, y: pressed ? 2 : 6)
                    .shadow(color: Palette.amber.opacity(0.45), radius: 14, y: pressed ? 6 : 12)
            )
            .offset(y: pressed ? 4 : 0)
            .animation(.spring(response: 0.2, dampingFraction: 0.6), value: pressed)
    }
}
