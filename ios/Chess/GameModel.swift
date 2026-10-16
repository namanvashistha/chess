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
    /// Moves drawn in the danger colour: attackers, when inspecting a piece.
    var danger: Set<BoardMove> = []
}

/// What the coach is saying.
struct CoachCard: Equatable, Codable {
    enum Kind: String, Codable {
        case great, good, inaccuracy, mistake, blunder, coach, warning, opponent

        /// The app's own words for a verdict: kind, never harsh. The engine's
        /// chess terms (inaccuracy, mistake, blunder) stay out of sight.
        var verdict: (title: String, badge: String)? {
            switch self {
            case .great: ("Great move", "★")
            case .good: ("Good move", "✓")
            case .inaccuracy: ("Almost", "~")
            case .mistake: ("Not quite", "?")
            case .blunder: ("Oops", "↩")
            default: nil
            }
        }

        /// The everyday icon for a verdict: no chess notation.
        var symbol: String? {
            switch self {
            case .great: "star.fill"
            case .good: "checkmark"
            case .inaccuracy: "hand.thumbsup.fill"
            case .mistake: "questionmark"
            case .blunder: "arrow.uturn.backward"
            default: nil
            }
        }
    }
    let id: Int
    let kind: Kind
    let badge: String
    let title: String
    let text: String
}

/// One of the player's moves with the coach's verdict, for the move trail.
struct TrailEntry: Identifiable, Equatable, Codable {
    let id: Int
    let ply: Int
    let san: String
    let card: CoachCard
}

/// A finished game, for the home screen's list.
struct GameRecord: Identifiable, Equatable, Codable {
    var id: Date { date }
    let date: Date
    let opponent: String
    let outcome: String // "won", "lost", "draw"
    let accuracy: Int?
    let moves: Int
    /// The opponent's level; nil for pass and play and for records saved before levels.
    var level: Int?
    var great: Int?

    /// The name to show. Records saved before the rename say Pebble, Flint or Obsidian.
    var opponentName: String {
        if let level, let o = Opponent(rawValue: level) { return o.name }
        return ["Pebble": "Club Player", "Flint": "Master", "Obsidian": "Grandmaster",
                "Easy": "Club Player", "Medium": "Master", "Hard": "Grandmaster"][opponent] ?? opponent
    }
}

/// A game in progress, saved after every move so it survives quitting.
struct SavedGame: Codable {
    let passAndPlay: Bool
    let opponent: Int
    let moves: [String]
    let trail: [TrailEntry]
    let losses: [Int]
    let great: Int
    let blunders: Int
    let hintsLeft: Int
    var evals: [Int?]?
    /// Extra hints already given for falling behind; missing in older saves.
    var comebacks: Int?
    let fen: String

    private static let key = "savedGame"
    static func load() -> SavedGame? {
        UserDefaults.standard.data(forKey: key).flatMap { try? JSONDecoder().decode(SavedGame.self, from: $0) }
    }
    func store() { UserDefaults.standard.set(try? JSONEncoder().encode(self), forKey: Self.key) }
    static func clear() { UserDefaults.standard.removeObject(forKey: key) }
}

/// A move worth replaying: the boards either side of it and what the coach said.
struct BestMove {
    let before: [Character]
    let after: [Character]
    let move: BoardMove
    let moveNumber: Int
    let kind: CoachCard.Kind
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
    /// Good and great moves together, for the finish card.
    var goodMoves = 0
    /// The player's best move of the game, replayed at the top of the card.
    var best: BestMove?
    /// The opponent this win opened up, if it was the one that did.
    var unlocked: Opponent?
    /// The first win ever against this opponent.
    var firstWin = false
    /// Wins so far toward opening the next opponent, when it is still closed.
    var towardNext: (next: Opponent, wins: Int)?

    static func == (a: Finish, b: Finish) -> Bool {
        a.outcome == b.outcome && a.title == b.title && a.text == b.text && a.accuracy == b.accuracy
            && a.unlocked == b.unlocked && a.firstWin == b.firstWin
    }
}

enum Opponent: Int, CaseIterable, Identifiable {
    case easy, medium, hard
    var id: Int { rawValue }
    /// Chess titles, so each win reads as a step up.
    var name: String { ["Club Player", "Master", "Grandmaster"][rawValue] }
    var subtitle: String { ["Beginner", "Intermediate", "Expert"][rawValue] }
    /// Easy is a true beginner; each level above it is the engine's next step up.
    var engineLevel: String { ["beginner", "easy", "medium"][rawValue] }
    var startLabel: String { "Play the \(name)" }

    /// Hints at the start of a game: the gentler the opponent, the more help.
    var hints: Int { [5, 4, 3][rawValue] }
    /// Extra hints handed out when the player falls behind, at most this many a game.
    var comebackHints: Int { [4, 2, 0][rawValue] }
    /// How far behind, in centipawns, before an extra hint is offered.
    var behindBy: Int { [100, 150, 0][rawValue] }

    /// Wins against the level below that open this one.
    static let winsToUnlock = 3
    var previous: Opponent? { Opponent(rawValue: rawValue - 1) }
    var isUnlocked: Bool { previous.map { Stats.wins(against: $0) >= Self.winsToUnlock } ?? true }
}

enum Mode { case engine, passAndPlay }

/// The game screen's state. The Go engine owns the rules and the coaching; this
/// keeps what the view needs on top and runs the engine off the main thread.
@MainActor
@Observable
final class GameModel {
    private(set) var mode: Mode = .engine
    private(set) var opponent: Opponent = .easy

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
    /// The evaluation after each half-move, in centipawns from White's side,
    /// for the game line. A gap means it was never measured (older saves).
    private(set) var evals: [Int?] = []
    private(set) var busy = false
    private(set) var thinking = false
    private(set) var losses: [Int] = []
    private(set) var trail: [TrailEntry] = []
    private(set) var hintsLeft = 3
    /// Extra hints given this game because the player fell behind.
    @ObservationIgnored private var comebacks = 0
    /// The trail entry the coach card is showing, if the player tapped one.
    private(set) var reviewing: Int? {
        didSet { if reviewing == nil { reviewBoard = nil; reviewMove = nil } }
    }
    /// While reviewing, the board as it stood after that move.
    private(set) var reviewBoard: [Character]?
    private(set) var reviewMove: BoardMove?

    /// The pieces to draw: the live game's, or the reviewed position's.
    var shownPieces: [PieceItem] {
        guard let reviewBoard else { return pieces }
        return reviewBoard.indices.compactMap { sq in
            reviewBoard[sq] == "." ? nil : PieceItem(id: -1 - sq, code: reviewBoard[sq], square: sq)
        }
    }

    var shownLastMove: BoardMove? { reviewBoard == nil ? lastMove : reviewMove }

    @ObservationIgnored private var game: ChesscoreGame
    @ObservationIgnored private var beforeInspect: CoachCard?
    /// When the verdict on the player's last move appeared.
    @ObservationIgnored private var verdictShownAt: Date?
    /// The verdict on the last move, pinned to the piece that made it until the
    /// next move: the trail entry and the square the piece stands on.
    private(set) var mark: (entry: TrailEntry, square: Square)?
    @ObservationIgnored private var legal: [String] = []
    @ObservationIgnored private var ids: [Int?] = Array(repeating: nil, count: 64)
    @ObservationIgnored private var nextID = 0
    @ObservationIgnored private var history: [(ids: [Int?], last: BoardMove?)] = []
    @ObservationIgnored private var serial = 0
    @ObservationIgnored private var great = 0
    @ObservationIgnored private var blunders = 0
    /// The game so far in UCI; observed, because the game line grows with it.
    private(set) var moves: [String] = []
    /// The hint already paid for in this position, shown again for free.
    @ObservationIgnored private var hintShown: (ply: Int, uci: String, text: String)?
    /// The piece a hint is pointing at, before it shows the whole move.
    private(set) var nudge: Square?
    /// True once this position's hint has shown the whole move.
    private(set) var hintRevealed = false
    /// Accuracy is shown once there is enough to say something.
    static let accuracyAfter = 3

    init() {
        game = ChesscoreNewGame()!
        refresh()
    }

    var isOver: Bool { !status.isEmpty }
    var playerToMove: Bool { !busy && !isOver && (mode == .passAndPlay || turn == "w") }
    var accuracy: Int? {
        guard losses.count >= Self.accuracyAfter else { return nil }
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
        start(mode, opponent: opponent, keepingSave: false)
    }

    private func start(_ mode: Mode, opponent: Opponent, keepingSave: Bool) {
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
        evals = []
        trail = []
        moves = []
        hintsLeft = mode == .engine ? opponent.hints : 3
        comebacks = 0
        reviewing = nil
        hintShown = nil
        nudge = nil
        mark = nil
        hintRevealed = false
        if !keepingSave { SavedGame.clear() }
        refresh()
        moveSerial += 1
        say(.coach, "✦", "Your move", mode == .engine
            ? "You play White. Tap a piece to see where it can go."
            : "White starts. Pass the phone after each move.")
    }

    /// Picks up the saved game, if there is one.
    @discardableResult
    func resume() -> Bool {
        guard let saved = SavedGame.load() else { return false }
        start(saved.passAndPlay ? .passAndPlay : .engine, opponent: Opponent(rawValue: saved.opponent) ?? .easy,
              keepingSave: true)
        for uci in saved.moves {
            guard game.play(uci) else { break }
            moves.append(uci)
        }
        trail = saved.trail
        losses = saved.losses
        great = saved.great
        blunders = saved.blunders
        hintsLeft = saved.hintsLeft
        comebacks = saved.comebacks ?? 0
        evals = Array((saved.evals ?? []).prefix(moves.count))
        if let cp = evals.last ?? nil { whiteShare = abs(cp) >= 9000 ? (cp > 0 ? 1 : 0) : 1 / (1 + exp(-Double(cp) / 220)) }
        ids = Array(repeating: nil, count: 64)
        refresh()
        if let last = moves.last { lastMove = BoardMove(uci: last, board: board) }
        moveSerial += 1
        say(.coach, "✦", "Welcome back", "Your game is where you left it.")
        save()
        fillEvals()
        if mode == .engine, turn == "b", !isOver { engineTurn() }
        return true
    }

    /// Games saved before the game line existed have no evaluations. Measures
    /// the missing ones in the background so the line tells the whole story.
    private func fillEvals() {
        let moves = self.moves, have = evals
        guard have.count < moves.count || have.contains(where: { $0 == nil }) else { return }
        let game = self.game
        Task.detached(priority: .utility) {
            guard let g = ChesscoreNewGame() else { return }
            var out: [Int?] = []
            for (i, m) in moves.enumerated() {
                guard g.play(m) else { break }
                out.append(i < have.count && have[i] != nil ? have[i] : g.eval(2))
            }
            await MainActor.run {
                guard game === self.game else { return }
                var merged = out
                for i in self.evals.indices {
                    if i < merged.count { if let v = self.evals[i] { merged[i] = v } } else { merged.append(self.evals[i]) }
                }
                self.evals = Array(merged.prefix(self.moves.count))
                if self.moves.count == out.count, let cp = out.last ?? nil {
                    self.whiteShare = abs(cp) >= 9000 ? (cp > 0 ? 1 : 0) : 1 / (1 + exp(-Double(cp) / 220))
                }
                self.save()
            }
        }
    }

    private func save() {
        guard !isOver else { SavedGame.clear(); return }
        SavedGame(passAndPlay: mode == .passAndPlay, opponent: opponent.rawValue, moves: moves, trail: trail,
                  losses: losses, great: great, blunders: blunders, hintsLeft: hintsLeft, evals: evals, comebacks: comebacks,
                  fen: game.fen()).store()
    }

    /// Shows a past move's verdict again; tapping it twice returns to the latest.
    func review(_ entry: TrailEntry) {
        if reviewing == entry.id {
            reviewing = nil
            coach = trail.last?.card ?? coach
        } else {
            reviewing = entry.id
            showPosition(afterPly: entry.ply)
            coach = CoachCard(id: bump(), kind: entry.card.kind, badge: entry.card.badge,
                              title: "\((entry.ply / 2) + 1). \(entry.san) · \(entry.card.kind.verdict?.title ?? entry.card.title)",
                              text: entry.card.text.replacingOccurrences(of: " Undo and try again.", with: ""))
        }
    }

    /// Steps the review one rated move earlier (-1) or later (+1). Stepping
    /// past the latest move returns to the game.
    func step(_ delta: Int) {
        guard !trail.isEmpty else { return }
        let index = reviewing.flatMap { id in trail.firstIndex { $0.id == id } } ?? trail.count
        let next = index + delta
        if next >= trail.count {
            if reviewing != nil { endReview(); emit(.select) }
            return
        }
        guard next >= 0, next != index else { return }
        selected = nil
        rays = nil
        review(trail[next])
        emit(.select)
    }

    /// The mark to draw: hidden while reviewing, or once its piece is gone.
    var shownMark: (entry: TrailEntry, square: Square)? {
        guard reviewing == nil, let mark, board.indices.contains(mark.square) else { return nil }
        let piece = board[mark.square]
        let moverIsWhite = mark.entry.ply % 2 == 0
        guard piece != ".", piece.isUppercase == moverIsWhite else { return nil }
        return mark
    }

    /// Tapping the mark: the coach shows that move's verdict again.
    func revealMark() {
        guard let mark = shownMark else { return }
        coach = mark.entry.card
        emit(.select)
    }

    /// Shows a past move without toggling, for scrubbing along the game line.
    func show(_ entry: TrailEntry) {
        guard reviewing != entry.id else { return }
        selected = nil
        rays = nil
        nudge = nil
        review(entry)
    }

    func endReview() {
        guard reviewing != nil else { return }
        reviewing = nil
        coach = trail.last?.card ?? coach
    }

    private func showPosition(afterPly ply: Int) {
        guard let p = position(afterPly: ply) else { return }
        reviewBoard = p.board
        reviewMove = p.move
    }

    /// The board after a half-move, and that move.
    private func position(afterPly ply: Int) -> (board: [Character], move: BoardMove?)? {
        guard let g = ChesscoreNewGame() else { return nil }
        var before: [Character] = Array(g.board())
        for (i, uci) in moves.prefix(ply + 1).enumerated() {
            if i == ply { before = Array(g.board()) }
            _ = g.play(uci)
        }
        return (Array(g.board()), moves.indices.contains(ply) ? BoardMove(uci: moves[ply], board: before) : nil)
    }

    /// The player's best move: the strongest verdict, the latest among equals.
    /// The finish card leads with it, so every game ends on something good.
    private func bestMove() -> BestMove? {
        let rank: [CoachCard.Kind: Int] = [.great: 4, .good: 3, .inaccuracy: 2, .mistake: 1, .blunder: 0]
        let mine = mode == .engine ? trail.filter { $0.ply % 2 == 0 } : trail
        guard let e = mine.max(by: { (rank[$0.card.kind] ?? 0, $0.ply) < (rank[$1.card.kind] ?? 0, $1.ply) }),
              (rank[e.card.kind] ?? 0) >= 3,
              let after = position(afterPly: e.ply), let move = after.move,
              let before = position(afterPly: e.ply - 1) else { return nil }
        return BestMove(before: before.board, after: after.board, move: move, moveNumber: e.ply / 2 + 1,
                        kind: e.card.kind, text: e.card.text)
    }

    // MARK: - Sharing

    /// The great move the coach card is showing, live or in review.
    var shareableEntry: TrailEntry? {
        let entry = reviewing.flatMap { id in trail.first { $0.id == id } } ?? trail.last
        guard let entry, entry.card.kind == .great else { return nil }
        if reviewing == nil && coach?.id != entry.card.id { return nil }
        return entry
    }

    func moment(for entry: TrailEntry) -> ShareMoment {
        let p = position(afterPly: entry.ply)
        let number = entry.ply / 2 + 1
        let label = entry.ply % 2 == 0 ? "\(number). \(entry.san)" : "\(number)… \(entry.san)"
        return ShareMoment(board: p?.board ?? board, move: p?.move, title: "\(label) · Great move",
                           detail: entry.card.text, footer: "Found with Cozy Chess")
    }

    /// A win against the engine, ready to share.
    var winMoment: ShareMoment? {
        guard mode == .engine, let finish, finish.outcome == .won else { return nil }
        let accuracy = finish.accuracy.map { " · \($0)% accuracy" } ?? ""
        return ShareMoment(board: board, move: lastMove, title: "I beat \(opponent.name)",
                           detail: finish.text + accuracy, footer: "Played on Cozy Chess")
    }

    /// When the player is clearly behind and nearly out of hints, gives one
    /// more, up to the opponent's allowance for the game.
    private func offerComebackHint(eval: Int) -> Bool {
        guard mode == .engine, comebacks < opponent.comebackHints, hintsLeft <= 1,
              eval <= -opponent.behindBy else { return false }
        comebacks += 1
        hintsLeft += 1
        save()
        return true
    }

    // MARK: - What to watch

    private static let value: [Character: Int] = ["p": 1, "n": 3, "b": 3, "r": 5, "q": 9, "k": 0]

    /// After the opponent moves: what it means for the player, in a title and a
    /// line. Checks first, then what the move attacks, then anything left loose,
    /// then a chance to win something, then a tip for this stage of the game.
    private func watchFor(_ m: BoardMove?, taken: Character?) -> (title: String, text: String, urgent: Bool) {
        let me = "w", them = "b"
        func name(_ sq: Square) -> String { "\(Self.pieceName(board[sq])) on \(Squares.name(sq))" }
        func squares(_ s: String) -> [Square] { s.split(separator: " ").compactMap { Squares.parse($0) } }
        func isMine(_ sq: Square) -> Bool { board[sq] != "." && (board[sq].isUppercase ? "w" : "b") == me }

        if inCheck {
            return ("Your king is in check", "Move it, block the line, or take the checking piece.", true)
        }
        if let m {
            let mover = Self.pieceName(board[m.to])
            // What the moved piece now hits, biggest first.
            let hit = (0..<64).filter { isMine($0) && board[$0] != "K" && squares(game.attackers(Squares.name($0))).contains(m.to) }
                .sorted { (Self.value[Character(board[$0].lowercased())] ?? 0) > (Self.value[Character(board[$1].lowercased())] ?? 0) }
            if let sq = hit.first(where: { game.guards(Squares.name($0)).isEmpty }) {
                return ("Watch your \(name(sq))", "Their \(mover) now attacks it, and nothing guards it.", true)
            }
            if let taken {
                let back = !game.attackers(Squares.name(m.to)).isEmpty
                return ("They took your \(Self.pieceName(taken))",
                        back ? "You can take back on \(Squares.name(m.to))." : "Look for a way to win it back.", false)
            }
            if let sq = hit.first, (Self.value[Character(board[sq].lowercased())] ?? 0) > (Self.value[Character(board[m.to].lowercased())] ?? 0) {
                return ("Their \(mover) eyes your \(Self.pieceName(board[sq]))",
                        "It's guarded, but their \(mover) is worth less, so a trade would cost you.", true)
            }
        }
        if let sq = loose.sorted().first {
            return ("Your \(name(sq)) is loose", "It's attacked and nothing guards it.", true)
        }
        if let sq = squares(game.loose(them)).first {
            return ("A chance", "Their \(name(sq)) is under attack and unguarded.", false)
        }
        return (["Your move", "Your turn", "Over to you"][moves.count / 2 % 3], tip(), false)
    }

    /// A tip that fits the position, or one of the app's gestures, each shown once.
    private func tip() -> String {
        let castling = game.fen().split(separator: " ").dropFirst(2).first.map(String.init) ?? ""
        let home: [(Square, Character)] = [(57, "N"), (62, "N"), (58, "B"), (61, "B")] // b1 g1 c1 f1
        let undeveloped = home.filter { board[$0.0] == $0.1 }.count
        if moves.count < 16 && undeveloped >= 2 {
            return "Bring a knight or bishop toward the centre."
        }
        if moves.count < 30 && board[60] == "K" && castling.contains(where: { $0 == "K" || $0 == "Q" }) {
            return "Castling tucks your king away and brings a rook into play."
        }
        let gestures = [("tip.hold", "Hold any piece to see what guards it and what attacks it."),
                        ("tip.drag", "You can drag pieces as well as tap them."),
                        ("tip.line", "Drag along the line under the board to look back at your moves.")]
        if let (key, text) = gestures.first(where: { !UserDefaults.standard.bool(forKey: $0.0) }) {
            UserDefaults.standard.set(true, forKey: key)
            return text
        }
        return "Before you move, ask what their last move wants."
    }

    // MARK: - Touch

    /// Picks up one of the mover's pieces to drag. Returns false if it can't move.
    func pickUp(_ square: Square) -> Bool {
        guard playerToMove, reviewing == nil else { return false }
        let piece = board[square]
        guard piece != ".", (piece.isUppercase ? "w" : "b") == turn else { return false }
        if selected != square {
            selected = square
            rays = makeRays(.options, targets)
            emit(.select)
        }
        return true
    }

    /// Drops the dragged piece. A legal square plays the move; anywhere else
    /// it goes back and stays selected.
    func drop(on square: Square?) {
        guard let from = selected else { return }
        if let square, square != from, targets.contains(where: { $0.to == square }) {
            play(from: from, to: square)
        }
    }

    /// Long-press: lines from every piece guarding this one and every piece
    /// attacking it, and a line from the coach saying which is which.
    @discardableResult
    func inspect(_ square: Square) -> Bool {
        guard reviewing == nil, !busy, board.indices.contains(square), board[square] != "." else { return false }
        let name = Squares.name(square)
        func lines(_ list: String, danger: Bool) -> [BoardMove] {
            list.split(separator: " ").compactMap { Squares.parse($0) }.map { BoardMove(from: $0, to: square, capture: danger) }
        }
        let guards = lines(game.guards(name), danger: false)
        let attackers = lines(game.attackers(name), danger: true)
        selected = nil
        var r = makeRays(.options, guards + attackers)
        r.danger = Set(attackers)
        rays = r
        beforeInspect = coach
        let piece = board[square]
        let whose = mode == .engine ? (piece.isUppercase ? "Your" : "Your opponent's") : (piece.isUppercase ? "White's" : "Black's")
        func count(_ n: Int, _ word: String) -> String { n == 0 ? "no \(word)s" : n == 1 ? "1 \(word)" : "\(n) \(word)s" }
        let text: String = switch (guards.count, attackers.count) {
        case (_, 0): "Nothing attacks it" + (guards.isEmpty ? "." : ", and it's guarded by \(count(guards.count, "piece")).")
        case (0, _): "Attacked by \(count(attackers.count, "piece")) and nothing guards it."
        default: "Attacked by \(count(attackers.count, "piece")), guarded by \(count(guards.count, "piece"))."
        }
        say(.coach, "◎", "\(whose) \(Self.pieceName(piece)) on \(name)", text)
        emit(.select)
        return true
    }

    func endInspect() {
        rays = nil
        if let beforeInspect { coach = beforeInspect }
        beforeInspect = nil
    }

    // MARK: - Taps

    func tap(_ square: Square) {
        if reviewing != nil { endReview(); return }
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

    /// True when "Show me" would repeat a hint already used in this position.
    var hintIsFree: Bool { hintShown?.ply == moves.count }

    /// Hints come in two steps. The first points at the piece to move; asking
    /// again in the same position shows the move and why, at no extra cost.
    func hint() {
        guard playerToMove else { return }
        selected = nil
        if let shown = hintShown, shown.ply == moves.count, let m = BoardMove(uci: shown.uci, board: board) {
            nudge = nil
            hintRevealed = true
            rays = makeRays(.hint, [m])
            say(.great, "✦", "Try this", shown.text)
            emit(.good)
            return
        }
        guard hintsLeft > 0 else { return }
        hintsLeft -= 1
        save()
        rays = nil
        busy = true
        let game = self.game, ply = moves.count
        Task.detached(priority: .userInitiated) {
            let best = game.search(3, moveTimeMs: 0)?.bestMove ?? ""
            let san = best.isEmpty ? "" : game.san(best)
            let why = best.isEmpty ? "" : game.explain(best)
            await MainActor.run {
                self.busy = false
                guard !best.isEmpty, game === self.game, let m = BoardMove(uci: best, board: self.board) else { return }
                let text = "\(san): \(why)"
                self.hintShown = (ply, best, text)
                self.hintRevealed = false
                self.nudge = m.from
                let piece = Self.pieceName(self.board[m.from])
                self.say(.coach, "✦", "Look at your \(piece)",
                         "Your \(piece) on \(Squares.name(m.from)) has a good move. Find it, or tap Show the move.")
                self.emit(.select)
            }
        }
    }

    func undo() {
        mark = nil
        nudge = nil
        hintRevealed = false
        var steps = mode == .engine ? 2 : 1
        if mode == .engine && turn == "b" { steps = 1 } // the engine has not replied yet
        guard !busy, status != "resigned", moves.count >= steps else { return }
        for _ in 0..<steps {
            game.undo()
            moves.removeLast()
            if let snap = history.popLast() {
                ids = snap.ids
                lastMove = snap.last
            } else {
                // resumed game: no identities to restore, so pieces fade back in place
                ids = Array(repeating: nil, count: 64)
                lastMove = nil
            }
        }
        if !losses.isEmpty { losses.removeLast() }
        if !trail.isEmpty { trail.removeLast() }
        evals = Array(evals.prefix(moves.count))
        hintShown = nil
        reviewing = nil
        selected = nil
        rays = nil
        toppled = nil
        finish = nil
        movingPiece = nil
        refresh()
        moveSerial += 1
        say(.coach, "↶", "Rewound", "Your move again. Take another look.")
        save()
    }

    /// Ends the game as a loss.
    func resign() {
        guard mode == .engine, !isOver, !busy else { return }
        status = "resigned"
        selected = nil
        rays = nil
        loose = []
        if let k = board.firstIndex(of: "K") { toppled = ids[k] }
        let f = Finish(outcome: .lost, title: "Good game", text: "Every game teaches something. The next one is yours.",
                       accuracy: accuracy, great: great, blunders: blunders)
        let count = (moves.count + 1) / 2
        Task {
            try? await Task.sleep(for: .seconds(0.9))
            var shown = f
            shown.goodMoves = trail.filter { $0.card.kind == .good || $0.card.kind == .great }.count
            shown.best = bestMove()
            finish = shown
            emit(.lose)
            Stats.record(accuracy: f.accuracy, great: f.great)
            Stats.addRecord(GameRecord(date: .now, opponent: opponent.name, outcome: "lost", accuracy: f.accuracy, moves: count,
                                       level: opponent.rawValue, great: f.great))
            SavedGame.clear()
        }
    }

    // MARK: - Moves

    private func play(from: Square, to: Square) {
        var uci = Squares.name(from) + Squares.name(to)
        if !legal.contains(uci), legal.contains(uci + "q") { uci += "q" }
        let fenBefore = game.fen()
        let ply = moves.count
        let san = apply(uci)
        busy = true

        // Judge the move against the position before it, off the main thread.
        let game = self.game
        Task.detached(priority: .userInitiated) {
            let verdict = ChesscoreNewGameFromFEN(fenBefore)?.assess(uci)
            let eval = game.eval(2)
            await MainActor.run {
                guard game === self.game else { return }
                if let verdict { self.judged(verdict, ply: ply, san: san) }
                self.setEval(eval)
                self.busy = false
                if self.endIfOver() { return }
                if self.mode == .engine { self.engineTurn() }
            }
        }
    }

    private func judged(_ v: ChesscoreAssessment, ply: Int, san: String) {
        losses.append(v.loss)
        let kind = CoachCard.Kind(rawValue: v.kind) ?? .good
        if kind == .great { great += 1 }
        if kind == .blunder { blunders += 1 }
        let words = kind.verdict ?? ("Good move", "✓")
        let nudge = kind == .blunder ? " Undo and try again." : ""
        say(kind, words.badge, words.title, v.text + nudge)
        // The trail keeps the verdict without "Undo and try again", which only
        // makes sense right after the move.
        verdictShownAt = .now
        if let card = coach {
            let kept = CoachCard(id: card.id, kind: card.kind, badge: card.badge, title: card.title, text: v.text)
            let entry = TrailEntry(id: bump(), ply: ply, san: san, card: kept)
            trail.append(entry)
            if let to = lastMove?.to { mark = (entry, to) }
        }
        reviewing = nil
        emit(kind == .great ? .great : kind == .good ? .good : .bad)
        save()
    }

    /// The engine shows the moves it is weighing, one by one, then commits.
    private func engineTurn() {
        busy = true
        thinking = true
        let game = self.game, level = opponent.engineLevel
        Task.detached(priority: .userInitiated) {
            let candidates = game.candidates(4).split(separator: " ").map(String.init)
            let choice = game.botMove(level)
            let line = choice.isEmpty ? "" : game.describe(choice)
            await MainActor.run { self.showThinking(candidates, choice, line, game) }
        }
    }

    private func showThinking(_ candidates: [String], _ choice: String, _ line: String, _ game: ChesscoreGame) {
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
            // Let the verdict on the player's move be read before it's replaced.
            if let shown = verdictShownAt {
                let left = Self.readingTime(trail.last?.card) - Date.now.timeIntervalSince(shown)
                if left > 0 { try? await Task.sleep(for: .seconds(left)) }
            }
            guard game === self.game else { return }
            rays = nil
            thinking = false
            let taken = BoardMove(uci: choice, board: board).flatMap { $0.capture ? board[$0.to] : nil }
            let san = apply(choice)
            busy = false
            if endIfOver() {
                say(.opponent, san, "Your opponent \(line).", "That's the game.")
                return
            }
            let eval = await Task.detached { game.eval(2) }.value
            setEval(eval)
            let tip = watchFor(lastMove, taken: taken)
            let extra = offerComebackHint(eval: eval) ? " You're behind, so here's an extra hint." : ""
            say(tip.urgent ? .warning : .opponent, san, tip.title, tip.text + extra)
            verdictShownAt = nil
        }
    }

    @discardableResult
    private func apply(_ uci: String) -> String {
        guard let from = Squares.parse(uci.prefix(2)), let to = Squares.parse(uci.dropFirst(2).prefix(2)) else { return "" }
        let before = board
        guard game.play(uci) else { return "" }
        let san = game.lastSAN()
        moves.append(uci)

        history.append((ids, lastMove))
        // A new move by the player replaces the old mark; the opponent's reply
        // keeps it, unless it took that piece.
        if mode == .passAndPlay || turn == "w" { mark = nil }
        nudge = nil
        hintRevealed = false
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
        save()
        return san
    }

    private func endIfOver() -> Bool {
        guard isOver else { return false }
        busy = false
        let mate = status == ChesscoreStatusCheckmate
        if mate, let k = checkSquare { toppled = ids[k] }
        let moves = (self.moves.count + 1) / 2
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
        var final = f
        final.goodMoves = trail.filter { $0.card.kind == .good || $0.card.kind == .great }.count
        if mode == .engine, f.outcome == .won {
            let next = Opponent(rawValue: opponent.rawValue + 1)
            let wasLocked = next.map { !$0.isUnlocked } ?? false
            final.firstWin = Stats.wins(against: opponent) == 0
            Stats.addWin(against: opponent)
            final = Finish(outcome: .won, title: "You beat the \(opponent.name)!", text: f.text,
                           accuracy: f.accuracy, great: f.great, blunders: f.blunders, goodMoves: final.goodMoves,
                           firstWin: final.firstWin)
            if let next, wasLocked {
                if next.isUnlocked { final.unlocked = next }
                else { final.towardNext = (next, Stats.wins(against: opponent)) }
            }
        }
        final.best = bestMove()
        Task {
            try? await Task.sleep(for: .seconds(1.1))
            finish = final
            emit(final.outcome == .won ? .win : .lose)
            Stats.record(accuracy: final.accuracy, great: mode == .engine ? final.great : 0)
            Stats.addRecord(GameRecord(date: .now, opponent: mode == .engine ? opponent.name : "Pass & play",
                                       outcome: ["won", "lost", "draw"][[.won, .lost, .draw].firstIndex(of: final.outcome)!],
                                       accuracy: final.accuracy, moves: moves,
                                       level: mode == .engine ? opponent.rawValue : nil, great: final.great))
            SavedGame.clear()
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
        let i = moves.count - 1
        guard i >= 0 else { return }
        if evals.count <= i { evals += Array(repeating: nil, count: i + 1 - evals.count) }
        evals[i] = cp
    }

    private func say(_ kind: CoachCard.Kind, _ badge: String, _ title: String, _ text: String) {
        coach = CoachCard(id: bump(), kind: kind, badge: badge, title: title, text: text)
    }

    /// How long the coach's verdict needs on screen to be read: longer for
    /// longer lines, never under two seconds or over four.
    private static func readingTime(_ card: CoachCard?) -> Double {
        guard let card else { return 0 }
        return min(4, max(2, 1.2 + Double(card.title.count + card.text.count) * 0.035))
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

    static var recent: [GameRecord] {
        defaults.data(forKey: "recent").flatMap { try? JSONDecoder().decode([GameRecord].self, from: $0) } ?? []
    }

    /// Keeps the last 20 games: the home screen lists 5 and charts 10.
    static func addRecord(_ r: GameRecord) {
        let list = Array(([r] + recent).prefix(20))
        defaults.set(try? JSONEncoder().encode(list), forKey: "recent")
    }

    static func wins(against o: Opponent) -> Int { defaults.integer(forKey: "wins.\(o.rawValue)") }
    static func addWin(against o: Opponent) { defaults.set(wins(against: o) + 1, forKey: "wins.\(o.rawValue)") }

    /// Great moves found across every game against the engine.
    static var greatMoves: Int { defaults.integer(forKey: "greatMoves") }

    static func record(accuracy: Int?, great: Int = 0) {
        defaults.set(greatMoves + great, forKey: "greatMoves")
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
