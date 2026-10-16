import Chesscore
import Observation

/// A square index in display order: 0 is a8, 7 is h8, 63 is h1. This matches
/// the order of `ChesscoreGame.board()`.
typealias Square = Int

extension Square {
    /// Algebraic name, e.g. 52 -> "e2".
    var name: String {
        let file = Character(UnicodeScalar(UInt8(ascii: "a") + UInt8(self % 8)))
        return "\(file)\(8 - self / 8)"
    }

    init?(name: Substring) {
        guard name.count == 2,
              let f = name.first?.asciiValue, let r = name.last?.wholeNumberValue,
              (97...104).contains(f), (1...8).contains(r) else { return nil }
        self = (8 - r) * 8 + Int(f - 97)
    }
}

/// Game state for the UI. Wraps the Go engine, which owns the rules.
@MainActor
@Observable
final class GameModel {
    /// The engine plays this colour; the player has the other one.
    let engineSide = "b"

    private(set) var board: [Character] = []
    private(set) var turn = "w"
    private(set) var status = ""
    private(set) var inCheck = false
    private(set) var lastMove: (from: Square, to: Square)?
    private(set) var thinking = false
    var selected: Square?

    private var game: ChesscoreGame
    private var legal: [String] = []

    init() {
        game = ChesscoreNewGame()!
        refresh()
    }

    var isOver: Bool { status != ChesscoreStatusPlaying }

    /// Squares the selected piece can move to.
    var targets: Set<Square> {
        guard let selected else { return [] }
        let from = selected.name
        return Set(legal.filter { $0.hasPrefix(from) }.compactMap { Square(name: $0.dropFirst(2).prefix(2)) })
    }

    func tap(_ square: Square) {
        guard !thinking, !isOver, turn != engineSide else { return }
        if let from = selected, targets.contains(square) {
            play(from: from, to: square)
            return
        }
        selected = isOwnPiece(board[square]) && selected != square ? square : nil
    }

    func newGame() {
        guard !thinking else { return }
        game = ChesscoreNewGame()!
        lastMove = nil
        selected = nil
        refresh()
    }

    /// Takes back the player's last move and the engine's reply to it.
    func undo() {
        guard !thinking else { return }
        if turn != engineSide, game.moveCount() >= 2 { game.undo() }
        game.undo()
        lastMove = nil
        selected = nil
        refresh()
    }

    private func isOwnPiece(_ c: Character) -> Bool {
        c != "." && (c.isUppercase ? "w" : "b") == turn
    }

    private func play(from: Square, to: Square) {
        var uci = from.name + to.name
        // Promotion choice comes later; for now a pawn always becomes a queen.
        if !legal.contains(uci), legal.contains(uci + "q") { uci += "q" }
        apply(uci)
    }

    private func apply(_ uci: String) {
        do { try game.play(uci) } catch { return }
        let chars = Array(uci)
        if let f = Square(name: Substring(String(chars[0...1]))), let t = Square(name: Substring(String(chars[2...3]))) {
            lastMove = (f, t)
        }
        selected = nil
        refresh()
        if turn == engineSide, !isOver { engineReply() }
    }

    private func engineReply() {
        thinking = true
        let game = self.game
        Task.detached(priority: .userInitiated) {
            let best = game.search(0, moveTimeMs: 800)?.bestMove ?? ""
            await MainActor.run {
                self.thinking = false
                if !best.isEmpty, game === self.game { self.apply(best) }
            }
        }
    }

    private func refresh() {
        board = Array(game.board())
        turn = game.turn()
        status = game.status()
        inCheck = game.inCheck()
        legal = game.legalMoves().split(separator: " ").map(String.init)
    }
}
