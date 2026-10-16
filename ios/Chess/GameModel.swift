import Chesscore
import Foundation
import Observation

/// A square index in display order: 0 is a8, 7 is h8, 63 is h1. This matches
/// the order of `ChesscoreGame.board()`.
typealias Square = Int

enum Squares {
    static func name(_ sq: Square) -> String {
        let file = Character(UnicodeScalar(UInt8(ascii: "a") + UInt8(sq % 8)))
        return "\(file)\(8 - sq / 8)"
    }

    static func parse<S: StringProtocol>(_ name: S) -> Square? {
        guard name.count == 2,
              let f = name.first?.asciiValue, let r = name.last?.wholeNumberValue,
              (97...104).contains(f), (1...8).contains(r) else { return nil }
        return (8 - r) * 8 + Int(f - 97)
    }

    static func file(_ sq: Square) -> Int { sq % 8 }
    static func row(_ sq: Square) -> Int { sq / 8 }
}

/// A piece on the board with a stable identity, so the view can animate it
/// from square to square instead of redrawing the board.
struct PieceItem: Identifiable, Equatable {
    let id: Int
    var code: Character
    var square: Square

    var isWhite: Bool { code.isUppercase }
    var imageName: String { (isWhite ? "w" : "b") + code.uppercased() }
}

/// A one-off event the view answers with a haptic.
struct Feedback: Equatable {
    enum Kind { case selection, move, capture, castle, check, mate }
    let serial: Int
    let kind: Kind
}

/// A capture's ring, drawn once where the captured piece stood.
struct Burst: Identifiable, Equatable {
    let id: Int
    let square: Square
}

/// Game state for the UI. The Go engine owns the rules; this tracks what the
/// view needs on top: piece identities, notation, highlights and events.
@MainActor
@Observable
final class GameModel {
    /// The engine plays this colour; the player has the other one.
    let engineSide = "b"

    private(set) var board: [Character] = []
    private(set) var pieces: [PieceItem] = []
    private(set) var turn = "w"
    private(set) var status = ""
    private(set) var inCheck = false
    private(set) var lastMove: (from: Square, to: Square)?
    private(set) var sans: [String] = []
    private(set) var thinking = false
    private(set) var feedback: Feedback?
    private(set) var bursts: [Burst] = []
    /// Bumped on every move, so the view can animate exactly those changes.
    private(set) var moveSerial = 0
    private(set) var movingPiece: Int?
    var selected: Square?

    @ObservationIgnored private var game: ChesscoreGame
    @ObservationIgnored private var legal: [String] = []
    @ObservationIgnored private var ids: [Int?] = Array(repeating: nil, count: 64)
    @ObservationIgnored private var nextID = 0
    @ObservationIgnored private var history: [(ids: [Int?], lastMove: (from: Square, to: Square)?)] = []
    @ObservationIgnored private var serial = 0

    init() {
        game = ChesscoreNewGame()!
        refresh()
        reconcileIDs()
    }

    var isOver: Bool { status != ChesscoreStatusPlaying }
    var playerToMove: Bool { turn != engineSide && !isOver && !thinking }

    /// Squares the selected piece can move to.
    var targets: [Square] {
        guard let selected else { return [] }
        let from = Squares.name(selected)
        var seen = Set<Square>()
        return legal.filter { $0.hasPrefix(from) }
            .compactMap { Squares.parse($0.dropFirst(2).prefix(2)) }
            .filter { seen.insert($0).inserted }
    }

    var checkSquare: Square? {
        inCheck ? board.firstIndex(of: turn == "w" ? "K" : "k") : nil
    }

    // MARK: - Actions

    func tap(_ square: Square) {
        guard playerToMove else { return }
        if let from = selected, targets.contains(square) {
            play(from: from, to: square)
            return
        }
        if isOwnPiece(board[square]), selected != square {
            selected = square
            emit(.selection)
        } else {
            selected = nil
        }
    }

    func newGame() {
        guard !thinking else { return }
        game = ChesscoreNewGame()!
        ids = Array(repeating: nil, count: 64)
        history = []
        sans = []
        lastMove = nil
        selected = nil
        movingPiece = nil
        refresh()
        reconcileIDs()
        moveSerial += 1
    }

    /// Takes back the player's last move and the engine's reply to it.
    func undo() {
        guard !thinking, !history.isEmpty else { return }
        let steps = (turn != engineSide && history.count >= 2) ? 2 : 1
        for _ in 0..<steps {
            game.undo()
            let snap = history.removeLast()
            ids = snap.ids
            lastMove = snap.lastMove
            sans.removeLast()
        }
        selected = nil
        movingPiece = nil
        refresh()
        reconcileIDs()
        moveSerial += 1
    }

    // MARK: - Moves

    private func isOwnPiece(_ c: Character) -> Bool {
        c != "." && (c.isUppercase ? "w" : "b") == turn
    }

    private func play(from: Square, to: Square) {
        var uci = Squares.name(from) + Squares.name(to)
        // Promotion choice comes later; for now a pawn always becomes a queen.
        if !legal.contains(uci), legal.contains(uci + "q") { uci += "q" }
        apply(uci)
    }

    private func apply(_ uci: String) {
        guard let from = Squares.parse(uci.prefix(2)),
              let to = Squares.parse(uci.dropFirst(2).prefix(2)) else { return }
        let before = board, legalBefore = legal
        do { try game.play(uci) } catch { return }

        history.append((ids, lastMove))
        let captured = moveIDs(from: from, to: to, before: before)
        lastMove = (from, to)
        selected = nil
        refresh()
        reconcileIDs()
        sans.append(Notation.san(uci: uci, before: before, legal: legalBefore,
                                 check: inCheck, mate: status == ChesscoreStatusCheckmate))
        moveSerial += 1

        let castle = before[from].lowercased() == "k" && abs(Squares.file(to) - Squares.file(from)) == 2
        if status == ChesscoreStatusCheckmate { emit(.mate) }
        else if inCheck { emit(.check) }
        else if captured != nil { emit(.capture) }
        else if castle { emit(.castle) }
        else { emit(.move) }
        if let captured { burst(at: captured) }

        if turn == engineSide, !isOver { engineReply() }
    }

    /// Moves piece identities to match a move, returning the square of any
    /// captured piece. Castling moves the rook; en passant removes a pawn that
    /// is not on the destination square.
    private func moveIDs(from: Square, to: Square, before: [Character]) -> Square? {
        let piece = before[from].lowercased()
        var captured: Square? = before[to] != "." ? to : nil
        if piece == "p", Squares.file(from) != Squares.file(to), before[to] == "." {
            let victim = Squares.row(from) * 8 + Squares.file(to)
            ids[victim] = nil
            captured = victim
        }
        if piece == "k", abs(Squares.file(to) - Squares.file(from)) == 2 {
            let row = Squares.row(from) * 8
            let (rookFrom, rookTo) = Squares.file(to) == 6 ? (row + 7, row + 5) : (row, row + 3)
            ids[rookTo] = ids[rookFrom]
            ids[rookFrom] = nil
        }
        movingPiece = ids[from]
        ids[to] = ids[from]
        ids[from] = nil
        return captured
    }

    /// Makes identities agree with the engine's board: empty squares lose
    /// theirs and any piece without one gets a fresh one.
    private func reconcileIDs() {
        var items: [PieceItem] = []
        for sq in 0..<64 {
            if board[sq] == "." {
                ids[sq] = nil
                continue
            }
            if ids[sq] == nil {
                ids[sq] = nextID
                nextID += 1
            }
            items.append(PieceItem(id: ids[sq]!, code: board[sq], square: sq))
        }
        pieces = items
    }

    private func engineReply() {
        thinking = true
        let game = self.game
        Task.detached(priority: .userInitiated) {
            let started = Date()
            let best = game.search(0, moveTimeMs: 700)?.bestMove ?? ""
            // Let the player's piece finish landing before the reply starts.
            let wait = 0.45 - Date().timeIntervalSince(started)
            if wait > 0 { try? await Task.sleep(for: .seconds(wait)) }
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

    // MARK: - Events

    private func emit(_ kind: Feedback.Kind) {
        serial += 1
        feedback = Feedback(serial: serial, kind: kind)
    }

    private func burst(at square: Square) {
        serial += 1
        let b = Burst(id: serial, square: square)
        bursts.append(b)
        Task {
            try? await Task.sleep(for: .seconds(0.6))
            bursts.removeAll { $0 == b }
        }
    }

    // MARK: - Material

    private static let values: [Character: Int] = ["p": 1, "n": 3, "b": 3, "r": 5, "q": 9]

    /// Pieces the given side has captured, most valuable first.
    func captured(by side: String) -> [Character] {
        let start: [Character: Int] = ["q": 1, "r": 2, "b": 2, "n": 2, "p": 8]
        let opponentIsWhite = side == "b"
        var have: [Character: Int] = [:]
        for c in board where c != "." && c.isUppercase == opponentIsWhite {
            have[Character(c.lowercased()), default: 0] += 1
        }
        var out: [Character] = []
        for kind in "qrbnp" {
            let missing = max(0, start[kind]! - have[kind, default: 0])
            let code = opponentIsWhite ? Character(kind.uppercased()) : kind
            out += Array(repeating: code, count: missing)
        }
        return out
    }

    /// Material lead in pawns for the given side; 0 or negative when behind.
    func materialLead(for side: String) -> Int {
        var score = 0
        for c in board where c != "." {
            score += (c.isUppercase ? 1 : -1) * Self.values[Character(c.lowercased()), default: 0]
        }
        return side == "w" ? score : -score
    }
}
