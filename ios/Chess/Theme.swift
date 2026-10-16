import SwiftUI
import UIKit

/// The Quiet Board palette from DESIGN.md. Neutrals follow the system
/// appearance; board and state colours are the same in both.
enum Palette {
    static let accent = Color(hex: 0x5B8DEF)
    static let onAccent = Color(hex: 0x0C1118)
    static let squareLight = Color(hex: 0xEBECD0)
    static let squareDark = Color(hex: 0x7D945D)
    static let lastMove = Color(hex: 0xFFC440)
    static let danger = Color(hex: 0xE0564F)
    static let good = Color(hex: 0x3AA564)

    static let background = Color.dynamic(light: 0xF4F5F7, dark: 0x0E1117)
    static let surface = Color.dynamic(light: 0xFFFFFF, dark: 0x1A212D)
    static let surfaceRaised = Color.dynamic(light: 0xE3E8EF, dark: 0x2A3545)
    static let text = Color.dynamic(light: 0x1C2430, dark: 0xE7ECF3)
    static let muted = Color.dynamic(light: 0x5F6A7A, dark: 0x98A3B3)
    static let faint = Color.dynamic(light: 0x8A94A3, dark: 0x6B7686)
    static let border = Color.dynamic(light: 0xE2E6EC, dark: 0x2A3342)
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

/// Uppercase eyebrow label, the system's connective tissue.
struct EyebrowLabel: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 10.5, weight: .bold))
            .tracking(1.4)
            .textCase(.uppercase)
            .foregroundStyle(Palette.faint)
    }
}
