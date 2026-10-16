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

/// A move between two squares, as the board draws it.
struct BoardMove: Equatable, Hashable {
    let from: Square
    let to: Square
    var capture = false

    init(from: Square, to: Square, capture: Bool = false) {
        self.from = from; self.to = to; self.capture = capture
    }

    init?(uci: String, board: [Character]) {
        guard let f = Squares.parse(uci.prefix(2)), let t = Squares.parse(uci.dropFirst(2).prefix(2)) else { return nil }
        from = f; to = t
        capture = board.indices.contains(t) && board[t] != "."
    }
}

/// A piece on the board with a stable identity, so it can glide between squares.
struct PieceItem: Identifiable, Equatable {
    let id: Int
    var code: Character
    var square: Square
}

/// Lines of light drawn on the board.
struct Rays: Equatable {
    enum Kind { case options, thinking, chosen, hint }
    let id: Int
    let kind: Kind
    let moves: [BoardMove]
    /// Moves drawn as a knight's L rather than a straight line.
    let knight: Set<BoardMove>
}

/// What the coach is saying.
struct CoachCard: Equatable {
    enum Kind: String { case great, good, inaccuracy, mistake, blunder, coach, warning }
    let id: Int
    let kind: Kind
    let badge: String
    let title: String
    let text: String
}

/// A capture's glass shards.
struct Burst: Identifiable, Equatable {
    let id: Int
    let square: Square
    let dark: Bool
}

/// A one-off event for haptics and sound.
struct Feedback: Equatable {
    enum Kind { case select, move, capture, check, great, good, bad, win, lose }
    let id: Int
    let kind: Kind
}

/// How the game ended, for the finish card.
struct Finish: Equatable {
    enum Outcome { case won, lost, draw }
    let outcome: Outcome
    let title: String
    let text: String
    let accuracy: Int?
    let great: Int
    let blunders: Int
}

enum Opponent: Int, CaseIterable, Identifiable {
    case pebble, flint, obsidian
    var id: Int { rawValue }
    var name: String { ["Pebble", "Flint", "Obsidian"][rawValue] }
    var subtitle: String { ["Learning", "Club player", "No mercy"][rawValue] }
    var engineLevel: String { ["easy", "medium", "hard"][rawValue] }
}

enum Mode { case engine, passAndPlay }

/// The game screen's state. The Go engine owns the rules and the coaching; this
/// keeps what the view needs on top and runs the engine off the main thread.
@MainActor
@Observable
final class GameModel {
    private(set) var mode: Mode = .engine
    private(set) var opponent: Opponent = .flint

    private(set) var board: [Character] = []
    private(set) var pieces: [PieceItem] = []
    private(set) var turn = "w"
    private(set) var status = ""
    private(set) var inCheck = false
    private(set) var lastMove: BoardMove?
    private(set) var loose: Set<Square> = []
    private(set) var selected: Square?
    private(set) var rays: Rays?
    private(set) var coach: CoachCard?
    private(set) var bursts: [Burst] = []
    private(set) var toppled: Int?
    private(set) var finish: Finish?
    private(set) var feedback: Feedback?
    private(set) var movingPiece: Int?
    private(set) var moveSerial = 0
    /// White's share of the position, 0 to 1. The board's warm light sits on
    /// the side that is ahead.
    private(set) var whiteShare = 0.5
    private(set) var busy = false
    private(set) var thinking = false
    private(set) var losses: [Int] = []

    @ObservationIgnored private var game: ChesscoreGame
    @ObservationIgnored private var legal: [String] = []
    @ObservationIgnored private var ids: [Int?] = Array(repeating: nil, count: 64)
    @ObservationIgnored private var nextID = 0
    @ObservationIgnored private var history: [(ids: [Int?], last: BoardMove?)] = []
    @ObservationIgnored private var serial = 0
    @ObservationIgnored private var great = 0
    @ObservationIgnored private var blunders = 0

    init() {
        game = ChesscoreNewGame()!
        refresh()
    }

    var isOver: Bool { !status.isEmpty }
    var playerToMove: Bool { !busy && !isOver && (mode == .passAndPlay || turn == "w") }
    var accuracy: Int? {
        guard !losses.isEmpty else { return nil }
        let total = losses.map { max(0, 100 - Double($0) / 3.2) }.reduce(0, +)
        return Int((total / Double(losses.count)).rounded())
    }

    /// Squares the selected piece can move to.
    var targets: [BoardMove] {
        guard let selected else { return [] }
        let from = Squares.name(selected)
        var seen = Set<Square>()
        return legal.filter { $0.hasPrefix(from) }
            .compactMap { BoardMove(uci: $0, board: board) }
            .filter { seen.insert($0.to).inserted }
    }

    var checkSquare: Square? {
        inCheck ? board.firstIndex(of: turn == "w" ? "K" : "k") : nil
    }

    // MARK: - Starting

    func start(_ mode: Mode, opponent: Opponent) {
        self.mode = mode
        self.opponent = opponent
        game = ChesscoreNewGame()!
        ids = Array(repeating: nil, count: 64)
        history = []
        lastMove = nil
        selected = nil
        rays = nil
        bursts = []
        toppled = nil
        finish = nil
        losses = []
        great = 0
        blunders = 0
        busy = false
        thinking = false
        whiteShare = 0.5
        refresh()
        moveSerial += 1
        say(.coach, "✦", "Your move", mode == .engine
            ? "You play White. Tap a piece to see where it can go."
            : "White starts. Pass the phone after each move.")
    }

    // MARK: - Taps

    func tap(_ square: Square) {
        guard playerToMove else { return }
        if let from = selected, targets.contains(where: { $0.to == square }) {
            play(from: from, to: square)
            return
        }
        let piece = board[square]
        let mine = piece != "." && (piece.isUppercase ? "w" : "b") == turn
        if mine && selected != square {
            selected = square
            rays = makeRays(.options, targets)
            emit(.select)
        } else {
            selected = nil
            rays = nil
        }
    }

    func hint() {
        guard playerToMove else { return }
        selected = nil
        rays = nil
        busy = true
        let game = self.game
        Task.detached(priority: .userInitiated) {
            let best = game.search(3, moveTimeMs: 0)?.bestMove ?? ""
            let san = best.isEmpty ? "" : game.san(best)
            let why = best.isEmpty ? "" : game.explain(best)
            await MainActor.run {
                self.busy = false
                guard !best.isEmpty, game === self.game, let m = BoardMove(uci: best, board: self.board) else { return }
                self.rays = self.makeRays(.hint, [m])
                self.say(.great, "✦", "Try this", "\(san): \(why)")
                self.emit(.good)
            }
        }
    }

    func undo() {
        let steps = mode == .engine ? 2 : 1
        guard !busy, history.count >= steps else { return }
        for _ in 0..<steps {
            game.undo()
            let snap = history.removeLast()
            ids = snap.ids
            lastMove = snap.last
        }
        if !losses.isEmpty { losses.removeLast() }
        selected = nil
        rays = nil
        toppled = nil
        finish = nil
        movingPiece = nil
        refresh()
        moveSerial += 1
        say(.coach, "↶", "Rewound", "Your move again. Take another look.")
    }

    // MARK: - Moves

    private func play(from: Square, to: Square) {
        var uci = Squares.name(from) + Squares.name(to)
        if !legal.contains(uci), legal.contains(uci + "q") { uci += "q" }
        let fenBefore = game.fen()
        apply(uci)
        busy = true

        // Judge the move against the position before it, off the main thread.
        let game = self.game
        Task.detached(priority: .userInitiated) {
            let verdict = try? ChesscoreNewGameFromFEN(fenBefore, nil)?.assess(uci)
            let eval = game.eval(2)
            await MainActor.run {
                guard game === self.game else { return }
                if let verdict { self.judged(verdict) }
                self.setEval(eval)
                self.busy = false
                if self.endIfOver() { return }
                if self.mode == .engine { self.engineTurn() }
            }
        }
    }

    private func judged(_ v: ChesscoreAssessment) {
        losses.append(v.loss)
        let kind = CoachCard.Kind(rawValue: v.kind) ?? .good
        if kind == .great { great += 1 }
        if kind == .blunder { blunders += 1 }
        let badge = ["great": "!!", "good": "✓", "inaccuracy": "?!", "mistake": "?", "blunder": "??"][v.kind] ?? "✓"
        say(kind, badge, v.title, v.text)
        emit(kind == .great ? .great : kind == .good ? .good : .bad)
    }

    /// The engine shows the moves it is weighing, one by one, then commits.
    private func engineTurn() {
        busy = true
        thinking = true
        let game = self.game, level = opponent.engineLevel
        Task.detached(priority: .userInitiated) {
            let candidates = game.candidates(4).split(separator: " ").map(String.init)
            let choice = game.botMove(level)
            await MainActor.run { self.showThinking(candidates, choice, game) }
        }
    }

    private func showThinking(_ candidates: [String], _ choice: String, _ game: ChesscoreGame) {
        guard game === self.game, !choice.isEmpty else { busy = false; thinking = false; return }
        var shown = candidates
        if !shown.contains(choice) {
            if shown.isEmpty { shown = [choice] } else { shown[shown.count - 1] = choice }
        }
        Task {
            for k in 1...shown.count {
                rays = makeRays(.thinking, shown.prefix(k).compactMap { BoardMove(uci: $0, board: board) })
                try? await Task.sleep(for: .milliseconds(260))
            }
            if let m = BoardMove(uci: choice, board: board) { rays = makeRays(.chosen, [m]) }
            try? await Task.sleep(for: .milliseconds(420))
            guard game === self.game else { return }
            rays = nil
            thinking = false
            apply(choice)
            busy = false
            if endIfOver() { return }
            let eval = await Task.detached { game.eval(2) }.value
            setEval(eval)
            if inCheck {
                say(.warning, "+", "Check!", "Your king is attacked. Move it, block, or capture.")
            } else if let sq = loose.sorted().first {
                say(.inaccuracy, "!", "Heads up",
                    "Your \(Self.pieceName(board[sq])) on \(Squares.name(sq)) is under attack and undefended.")
            }
        }
    }

    private func apply(_ uci: String) {
        guard let from = Squares.parse(uci.prefix(2)), let to = Squares.parse(uci.dropFirst(2).prefix(2)) else { return }
        let before = board
        var err: NSError?
        _ = game.play(uci, error: &err)
        if err != nil { return }

        history.append((ids, lastMove))
        let captured = moveIDs(from: from, to: to, before: before)
        lastMove = BoardMove(from: from, to: to)
        selected = nil
        rays = nil
        refresh()
        moveSerial += 1

        if let sq = captured {
            let burst = Burst(id: bump(), square: sq, dark: !before[sq].isUppercase)
            bursts.append(burst)
            Task { try? await Task.sleep(for: .seconds(0.9)); bursts.removeAll { $0 == burst } }
            emit(.capture)
        } else {
            emit(.move)
        }
        if inCheck { emit(.check) }
    }

    private func endIfOver() -> Bool {
        guard isOver else { return false }
        busy = false
        let mate = status == ChesscoreStatusCheckmate
        if mate, let k = checkSquare { toppled = ids[k] }
        let moves = (history.count + 1) / 2
        let f: Finish
        if mate {
            let whiteWon = turn == "b"
            if mode == .passAndPlay {
                f = Finish(outcome: .won, title: "\(whiteWon ? "White" : "Black") wins",
                           text: "Checkmate in \(moves) moves.", accuracy: accuracy, great: great, blunders: blunders)
            } else {
                f = Finish(outcome: whiteWon ? .won : .lost, title: whiteWon ? "You won!" : "Good fight",
                           text: "Checkmate in \(moves) moves.", accuracy: accuracy, great: great, blunders: blunders)
            }
        } else {
            let why = [ChesscoreStatusStalemate: "Stalemate: no legal moves, and no check.",
                       ChesscoreStatusRepetition: "The same position came up three times.",
                       ChesscoreStatusFiftyMove: "Fifty moves without a capture or a pawn move.",
                       ChesscoreStatusInsufficient: "Neither side has enough left to mate."][status] ?? "The game is drawn."
            f = Finish(outcome: .draw, title: "A draw", text: why, accuracy: accuracy, great: great, blunders: blunders)
        }
        Task {
            try? await Task.sleep(for: .seconds(1.1))
            finish = f
            emit(f.outcome == .won ? .win : .lose)
            Stats.record(accuracy: f.accuracy)
        }
        return true
    }

    // MARK: - Bookkeeping

    /// Moves piece identities to match a move and returns the captured square.
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
            let (rf, rt) = Squares.file(to) == 6 ? (row + 7, row + 5) : (row, row + 3)
            ids[rt] = ids[rf]
            ids[rf] = nil
        }
        movingPiece = ids[from]
        ids[to] = ids[from]
        ids[from] = nil
        return captured
    }

    private func refresh() {
        board = Array(game.board())
        turn = game.turn()
        status = game.status()
        inCheck = game.inCheck()
        legal = game.legalMoves().split(separator: " ").map(String.init)
        let mine = mode == .engine ? "w" : turn
        loose = isOver ? [] : Set(game.loose(mine).split(separator: " ").compactMap { Squares.parse($0) })
        var items: [PieceItem] = []
        for sq in 0..<64 {
            if board[sq] == "." { ids[sq] = nil; continue }
            if ids[sq] == nil { ids[sq] = nextID; nextID += 1 }
            items.append(PieceItem(id: ids[sq]!, code: board[sq], square: sq))
        }
        pieces = items
    }

    private func makeRays(_ kind: Rays.Kind, _ moves: [BoardMove]) -> Rays {
        let knight = Set(moves.filter { board[$0.from].lowercased() == "n" })
        return Rays(id: bump(), kind: kind, moves: moves, knight: knight)
    }

    private func setEval(_ cp: Int) {
        whiteShare = abs(cp) >= 9000 ? (cp > 0 ? 1 : 0) : 1 / (1 + exp(-Double(cp) / 220))
    }

    private func say(_ kind: CoachCard.Kind, _ badge: String, _ title: String, _ text: String) {
        coach = CoachCard(id: bump(), kind: kind, badge: badge, title: title, text: text)
    }

    private func emit(_ kind: Feedback.Kind) {
        feedback = Feedback(id: bump(), kind: kind)
        Sound.shared.play(kind)
    }

    @discardableResult
    private func bump() -> Int {
        serial += 1
        return serial
    }

    static func pieceName(_ c: Character) -> String {
        ["p": "pawn", "n": "knight", "b": "bishop", "r": "rook", "q": "queen", "k": "king"][Character(c.lowercased())] ?? "piece"
    }

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
            out += Array(repeating: opponentIsWhite ? Character(kind.uppercased()) : kind, count: missing)
        }
        return out
    }

    /// Material lead in pawns for the given side; 0 or negative when behind.
    func materialLead(for side: String) -> Int {
        let values: [Character: Int] = ["p": 1, "n": 3, "b": 3, "r": 5, "q": 9]
        var score = 0
        for c in board where c != "." {
            score += (c.isUppercase ? 1 : -1) * values[Character(c.lowercased()), default: 0]
        }
        return side == "w" ? score : -score
    }
}

/// The player's running numbers for the home screen: a day streak and average
/// accuracy over recent games.
enum Stats {
    private static var defaults: UserDefaults { .standard }

    static var streak: Int { defaults.integer(forKey: "streak") }
    static var averageAccuracy: Int? {
        defaults.integer(forKey: "games") == 0 ? nil : Int(defaults.double(forKey: "accuracy").rounded())
    }

    static func record(accuracy: Int?) {
        let cal = Calendar.current
        let today = cal.startOfDay(for: .now)
        let last = defaults.object(forKey: "lastDay") as? Date
        if last.map({ cal.isDate($0, inSameDayAs: today) }) != true {
            let yesterday = cal.date(byAdding: .day, value: -1, to: today)!
            let continues = last.map { cal.isDate($0, inSameDayAs: yesterday) } ?? false
            defaults.set(continues ? streak + 1 : 1, forKey: "streak")
            defaults.set(today, forKey: "lastDay")
        }
        if let accuracy {
            let n = defaults.integer(forKey: "games")
            let avg = defaults.double(forKey: "accuracy")
            // a moving average that leans toward recent games
            defaults.set(n == 0 ? Double(accuracy) : avg * 0.8 + Double(accuracy) * 0.2, forKey: "accuracy")
            defaults.set(n + 1, forKey: "games")
        }
    }
}
