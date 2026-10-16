/// Standard algebraic notation and opening names. The engine speaks UCI; these
/// are for people.
enum Notation {
    /// SAN for a UCI move, given the board and legal moves before it was played.
    static func san(uci: String, before: [Character], legal: [String], check: Bool, mate: Bool) -> String {
        guard let from = Squares.parse(uci.prefix(2)),
              let to = Squares.parse(uci.dropFirst(2).prefix(2)) else { return uci }
        let piece = before[from]
        let kind = Character(piece.uppercased())
        let dest = Squares.name(to)
        var s: String

        if kind == "K", abs(Squares.file(to) - Squares.file(from)) == 2 {
            s = Squares.file(to) == 6 ? "O-O" : "O-O-O"
        } else if kind == "P" {
            let capture = Squares.file(from) != Squares.file(to)
            s = (capture ? String(Squares.name(from).first!) + "x" : "") + dest
            if uci.count == 5 { s += "=" + uci.suffix(1).uppercased() }
        } else {
            // Another piece of the same kind that can reach the same square
            // forces a file or rank to tell them apart.
            let rivals = legal.compactMap { m -> Square? in
                guard let f = Squares.parse(m.prefix(2)), f != from,
                      Squares.parse(m.dropFirst(2).prefix(2)) == to, before[f] == piece else { return nil }
                return f
            }
            var hint = ""
            if !rivals.isEmpty {
                let name = Squares.name(from)
                if !rivals.contains(where: { Squares.file($0) == Squares.file(from) }) {
                    hint = String(name.first!)
                } else if !rivals.contains(where: { Squares.row($0) == Squares.row(from) }) {
                    hint = String(name.last!)
                } else {
                    hint = name
                }
            }
            s = String(kind) + hint + (before[to] != "." ? "x" : "") + dest
        }
        if mate { s += "#" } else if check { s += "+" }
        return s
    }

    /// The name of the opening the moves so far belong to, or nil once the game
    /// has left the book.
    static func opening(_ sans: [String]) -> String? {
        var name: String?
        for k in stride(from: 1, through: min(sans.count, 12), by: 1) {
            let key = sans.prefix(k).joined(separator: " ").replacingOccurrences(of: "+", with: "")
            if let n = openings[key] { name = n }
        }
        return name
    }

    private static let openings: [String: String] = [
        "e4": "King's Pawn",
        "e4 e5": "Open Game",
        "e4 e5 Nf3": "King's Knight Opening",
        "e4 e5 Nf3 Nc6": "King's Knight Opening",
        "e4 e5 Nf3 Nc6 Bc4": "Italian Game",
        "e4 e5 Nf3 Nc6 Bc4 Bc5": "Giuoco Piano",
        "e4 e5 Nf3 Nc6 Bc4 Nf6": "Two Knights Defence",
        "e4 e5 Nf3 Nc6 Bc4 Bc5 b4": "Evans Gambit",
        "e4 e5 Nf3 Nc6 Bb5": "Ruy Lopez",
        "e4 e5 Nf3 Nc6 Bb5 a6": "Ruy Lopez, Morphy Defence",
        "e4 e5 Nf3 Nc6 d4": "Scotch Game",
        "e4 e5 Nf3 Nf6": "Petrov's Defence",
        "e4 e5 Nf3 d6": "Philidor Defence",
        "e4 e5 f4": "King's Gambit",
        "e4 e5 Nc3": "Vienna Game",
        "e4 c5": "Sicilian Defence",
        "e4 c5 Nf3 d6": "Sicilian Defence",
        "e4 c5 Nf3 Nc6": "Sicilian Defence",
        "e4 c5 Nf3 e6": "Sicilian Defence",
        "e4 e6": "French Defence",
        "e4 e6 d4 d5": "French Defence",
        "e4 c6": "Caro-Kann Defence",
        "e4 c6 d4 d5": "Caro-Kann Defence",
        "e4 d5": "Scandinavian Defence",
        "e4 d6": "Pirc Defence",
        "e4 g6": "Modern Defence",
        "e4 Nf6": "Alekhine's Defence",
        "d4": "Queen's Pawn",
        "d4 d5": "Queen's Pawn Game",
        "d4 d5 c4": "Queen's Gambit",
        "d4 d5 c4 e6": "Queen's Gambit Declined",
        "d4 d5 c4 dxc4": "Queen's Gambit Accepted",
        "d4 d5 c4 c6": "Slav Defence",
        "d4 d5 Bf4": "London System",
        "d4 Nf6": "Indian Defence",
        "d4 Nf6 Bf4": "London System",
        "d4 Nf6 c4 g6": "King's Indian Defence",
        "d4 Nf6 c4 e6": "Indian Defence",
        "d4 Nf6 c4 e6 Nc3 Bb4": "Nimzo-Indian Defence",
        "d4 f5": "Dutch Defence",
        "c4": "English Opening",
        "Nf3": "Réti Opening",
        "g3": "King's Fianchetto",
        "b3": "Nimzo-Larsen Attack",
    ]
}
