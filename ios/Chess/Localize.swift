import Foundation

/// A sentence in the player's language. Interpolated values become the
/// translation's placeholders, so word order can change per language.
func L(_ value: String.LocalizationValue) -> String { String(localized: value) }

/// The engine writes its coaching in English, from a few fixed templates. This
/// matches each template and rebuilds the sentence in the player's language.
enum EngineText {
    static func localize(_ english: String) -> String {
        // In every language: name the piece, not the move. "Qxd3 would have
        // been even better" sends you hunting for a square; "your queen had a
        // better move" is enough, and Undo is right there.
        if let r = try? Regex(#"(\S+) would have been even better\."#), let m = english.wholeMatch(of: r),
           let san = m[1].substring {
            let kind: String = switch san.first {
            case "K", "O": "king"
            case "Q": "queen"
            case "R": "rook"
            case "B": "bishop"
            case "N": "knight"
            default: "pawn"
            }
            return L("Your \(GameModel.localizedPiece(kind)) had a better move here.")
        }
        guard !english.isEmpty, Locale.current.language.languageCode?.identifier != "en" else { return english }
        let s = english
        func piece(_ name: Substring) -> String { GameModel.localizedPiece(String(name)) }
        func match(_ pattern: String) -> [Substring]? {
            guard let r = try? Regex(pattern), let m = s.wholeMatch(of: r) else { return nil }
            return (1..<m.count).compactMap { m[$0].substring }
        }
        switch s {
        case "Checkmate.": return L("Checkmate.")
        case "King tucked away and the rook brought into play.": return L("King tucked away and the rook brought into play.")
        case "Check, and it keeps the pressure on.": return L("Check, and it keeps the pressure on.")
        case "Claims the centre.": return L("Claims the centre.")
        case "Exactly what the engine would play.": return L("Exactly what the engine would play.")
        case "A solid, sensible move.": return L("A solid, sensible move.")
        case "There was an even better move here.": return L("There was an even better move here.")
        case "it is checkmate.": return L("it is checkmate.")
        case "castle to keep your king safe.": return L("castle to keep your king safe.")
        case "it gives check.": return L("it gives check.")
        case "it improves your position the most.": return L("it improves your position the most.")
        default: break
        }
        if let g = match(#"You win the (\w+)\. Clean\."#) { return L("You win the \(piece(g[0])). Clean.") }
        if let g = match(#"Develops the (\w+) toward the centre\."#) { return L("Develops the \(piece(g[0])) toward the centre.") }
        if let g = match(#"Careful: your (\w+) on ([a-h][1-8]) can be taken now\."#) {
            return L("Careful: your \(piece(g[0])) on \(String(g[1])) can be taken now.")
        }
        if let g = match(#"it wins the (\w+)\."#) { return L("it wins the \(piece(g[0])).") }
        if let g = match(#"takes your (\w+) on ([a-h][1-8])(\. Checkmate|, with check)?"#) {
            let base = L("takes your \(piece(g[0])) on \(String(g[1]))")
            return base + suffix(english)
        }
        if let g = match(#"moves the (\w+) to ([a-h][1-8])(\. Checkmate|, with check)?"#) {
            let base = L("moves the \(piece(g[0])) to \(String(g[1]))")
            return base + suffix(english)
        }
        return english
    }

    private static func suffix(_ s: String) -> String {
        if s.hasSuffix(". Checkmate") { return L(". Checkmate") }
        if s.hasSuffix(", with check") { return L(", with check") }
        return ""
    }
}
