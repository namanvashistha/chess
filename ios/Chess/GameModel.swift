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
            case .great: (L("Great move"), "★")
            case .good: (L("Good move"), "✓")
            case .inaccuracy: (L("Almost"), "~")
            case .mistake: (L("Not quite"), "?")
            case .blunder: (L("Oops"), "↩")
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
        let old = ["Pebble": 0, "Flint": 1, "Obsidian": 2, "Club Player": 0, "Rookie": 0, "Master": 1, "Grandmaster": 2,
                   "Easy": 0, "Medium": 1, "Hard": 2][opponent]
        return old.flatMap(Opponent.init)?.name ?? opponent
    }

    /// The opponent's level, for records saved before levels were stored.
    var resolvedLevel: Int? {
        level ?? ["Pebble": 0, "Flint": 1, "Obsidian": 2, "Club Player": 0, "Rookie": 0, "Master": 1, "Grandmaster": 2,
                  "Easy": 0, "Medium": 1, "Hard": 2][opponent]
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
    /// Whose move it was, in pass and play; nil against the engine.
    var side: String?
    let before: [Character]
    let after: [Character]
    let move: BoardMove
    let moveNumber: Int
    let kind: CoachCard.Kind
    let text: String
    /// The half-move it was, for its marker on the finish card's arc.
    var ply = 0
    var san = ""
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
    /// Pass and play: great moves for each side, in place of accuracy.
    var sides: (white: Int, black: Int)?
    /// The opponent this win opened up, if it was the one that did.
    var unlocked: Opponent?
    /// The first win ever against this opponent.
    var firstWin = false
    /// Wins so far toward opening the next opponent, when it is still closed.
    var towardNext: (next: Opponent, wins: Int)?
    /// Half-moves played.
    var plies = 0
    /// The player's chance to win, 0 to 1, from the start and after every
    /// half-move: the story of the game, drawn on the finish card.
    var arc: [Double] = []
    /// The half-move where the game turned against the player, if one did.
    var turnPly: Int?
    /// The one thing the arc says about the game.
    var insight: String?

    /// Reads the arc: where it turned on the player's own move, and the one
    /// sentence worth saying about it. `arc` starts before the first move.
    func reading(_ arc: [Double], playerParity: Int, twoPlayers: Bool) -> Finish {
        var f = self
        guard arc.count >= 5 else { return f }
        f.arc = arc
        guard !twoPlayers else { return f }
        // The biggest drop that came on the player's own move.
        var worst: (ply: Int, drop: Double)?
        for i in 1..<arc.count where (i - 1) % 2 == playerParity {
            let drop = arc[i - 1] - arc[i]
            if drop > (worst?.drop ?? 0.15) { worst = (i - 1, drop) }
        }
        f.turnPly = outcome == .won ? nil : worst?.ply
        func move(_ i: Int) -> Int { (i - 1) / 2 + 1 }
        func pct(_ v: Double) -> Int { Int((v * 100).rounded()) }
        let played = Array(arc.indices.dropFirst())
        let peak = played.max { arc[$0] < arc[$1] }!
        let low = played.min { arc[$0] < arc[$1] }!
        switch outcome {
        case .won:
            if arc[low] <= 0.35 { f.insight = L("Came back from \(pct(arc[low]))% on move \(move(low)).") }
            else if played.allSatisfy({ arc[$0] >= 0.45 }) { f.insight = L("You never fell behind.") }
        case .lost, .draw:
            if arc[peak] >= 0.6 { f.insight = L("You were winning: \(pct(arc[peak]))% on move \(move(peak)).") }
            else if let w = worst { f.insight = L("It turned on move \(w.ply / 2 + 1).") }
        }
        return f
    }

    static func == (a: Finish, b: Finish) -> Bool {
        a.outcome == b.outcome && a.title == b.title && a.text == b.text && a.accuracy == b.accuracy
            && a.unlocked == b.unlocked && a.firstWin == b.firstWin
    }
}

enum Opponent: Int, CaseIterable, Identifiable {
    case easy, medium, hard
    var id: Int { rawValue }
    /// Chess titles, so each win reads as a step up.
    var name: String { [L("Rookie"), L("Master"), L("Grandmaster")][rawValue] }
    var subtitle: String { [L("Beginner"), L("Intermediate"), L("Expert")][rawValue] }
    /// Easy is a true beginner; each level above it is the engine's next step up.
    var engineLevel: String { ["beginner", "easy", "medium"][rawValue] }
    var startLabel: String { L("Play the \(name)") }

    /// Hints at the start of a game, before they're matched to the player.
    var hints: Int { [5, 4, 3][rawValue] }

    /// Hints to start a new game with, matched to how the player has been
    /// doing against this opponent lately: more when it's been hard, a few
    /// fewer when it's been going well. Only ever the starting count; a hint
    /// once given is never taken back.
    var startingHints: Int {
        let games = Stats.recent.filter { $0.resolvedLevel == rawValue }.prefix(5)
        let accuracy = games.compactMap(\.accuracy)
        guard accuracy.count >= 2 else { return hints }
        let avg = accuracy.reduce(0, +) / accuracy.count
        var extra = avg < 60 ? 2 : avg < 72 ? 1 : avg >= 88 ? -1 : 0
        // Three losses in a row: a little more help, whatever the accuracy.
        if games.count >= 3, games.prefix(3).allSatisfy({ $0.outcome == "lost" }) { extra += 1 }
        return min(7, max(2, hints + extra))
    }
    /// Extra hints handed out when the player falls behind, at most this many a game.
    var comebackHints: Int { [4, 2, 0][rawValue] }
    /// How far behind, in centipawns, before an extra hint is offered.
    var behindBy: Int { [100, 150, 0][rawValue] }

    /// Wins against the level below that open this one.
    static let winsToUnlock = 3
    var previous: Opponent? { Opponent(rawValue: rawValue - 1) }
    var isUnlocked: Bool { previous.map { Stats.wins(against: $0) >= Self.winsToUnlock } ?? true }
}

enum Mode { case engine, passAndPlay, friend }

/// How a game against a friend on another phone travels: Game Center, Messages
/// or SharePlay. The model tells it about this player's moves; it calls
/// `receive` with the whole game when the friend's arrive.
@MainActor
protocol MatchLink: AnyObject {
    /// This player moved (or the game ended on their move); `moves` is the whole game.
    func didMove(_ model: GameModel)
    /// This player resigned.
    func didResign(_ model: GameModel)
    /// This player asked for another game with the same friend.
    func rematch(_ model: GameModel)
}

/// The game screen's state. The Go engine owns the rules and the coaching; this
/// keeps what the view needs on top and runs the engine off the main thread.
@MainActor
@Observable
final class GameModel {
    private(set) var mode: Mode = .engine
    /// The colour this player has: always White against the engine; either in a
    /// friend game. Pass and play has no single player.
    private(set) var myColor = "w"
    /// The friend's name, in a friend game.
    private(set) var friendName = ""
    /// How the friend game travels between phones.
    @ObservationIgnored weak var link: MatchLink?
    /// A friend game where this player is Black: the board turns so their pieces
    /// sit at the bottom.
    var flipped: Bool { mode == .friend && myColor == "b" }
    /// The friend resigned.
    @ObservationIgnored private var friendResigned = false
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
    /// Opponent turns with nothing to point out, for spacing out tips.
    @ObservationIgnored private var quietTurns = 0
    /// Which general idea the coach shares next.
    @ObservationIgnored private var ideaTurn = 0
    /// When the verdict on the player's last move appeared.
    @ObservationIgnored private var verdictShownAt: Date?
    /// The verdict on the last move, pinned to the piece that made it until the
    /// next move: the trail entry and the square the piece stands on.
    private(set) var mark: (entry: TrailEntry, square: Square)?
    /// A great move's sparkle: bumps once per great move, with where it landed.
    private(set) var sparkle: (id: Int, square: Square)?
    @ObservationIgnored private var legal: [String] = []
    @ObservationIgnored private var ids: [Int?] = Array(repeating: nil, count: 64)
    @ObservationIgnored private var nextID = 0
    @ObservationIgnored private var history: [(ids: [Int?], last: BoardMove?)] = []
    @ObservationIgnored private var serial = 0
    /// Bumps whenever pending work must be dropped: a new game, or an undo
    /// while the engine is still thinking. Background results check it.
    @ObservationIgnored private var epoch = 0
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
    var playerToMove: Bool { !busy && !isOver && (mode == .passAndPlay || turn == myColor) }
    /// Accuracy against the engine, on Lichess's curve: each move is scored by
    /// how much it lowers the chance of winning, so a slip costs some points
    /// rather than zeroing the move. Meaningless when two people share the
    /// board, so pass and play has none.
    var accuracy: Int? {
        guard mode != .passAndPlay, losses.count >= Self.accuracyAfter else { return nil }
        func win(_ cp: Double) -> Double { 50 + 50 * (2 / (1 + exp(-0.00368208 * cp)) - 1) }
        let scores = losses.map { loss -> Double in
            let drop = 50 - win(-Double(loss))
            return max(0, min(100, 103.1668 * exp(-0.04354 * drop) - 3.1669))
        }
        return Int((scores.reduce(0, +) / Double(scores.count)).rounded())
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
        epoch += 1
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
        // Hints are the engine's help: fair against the engine, not against a friend.
        hintsLeft = mode == .engine ? opponent.startingHints : mode == .friend ? 0 : 3
        if mode != .friend { myColor = "w"; friendName = ""; link = nil }
        friendResigned = false
        comebacks = 0
        reviewing = nil
        hintShown = nil
        nudge = nil
        mark = nil
        hintRevealed = false
        // A friend game lives with its link; it never replaces the saved game.
        if !keepingSave && mode != .friend { SavedGame.clear() }
        refresh()
        moveSerial += 1
        say(.coach, "✦", L("Your move"), mode == .engine
            ? L("You play White. Tap a piece to see where it can go.")
            : L("White starts. Pass the phone after each move."))
    }

    /// Opens a game against a friend: this player's colour, the friend's name and
    /// the game so far. Called again whenever the link has newer moves than a
    /// plain `receive` can apply.
    func startFriend(name: String, color: String, moves remote: [String], link: MatchLink) {
        myColor = color
        friendName = name
        self.link = link
        start(.friend, opponent: .easy, keepingSave: true)
        myColor = color
        friendName = name
        self.link = link
        for uci in remote {
            guard game.play(uci) else { break }
            moves.append(uci)
        }
        ids = Array(repeating: nil, count: 64)
        refresh()
        if let last = moves.last { lastMove = BoardMove(uci: last, board: board) }
        moveSerial += 1
        if !endIfOver() { greetFriendTurn() }
    }

    /// The friend's moves, as the whole game so far. New moves glide in one by
    /// one; anything that doesn't follow on from this board starts it afresh.
    func receive(moves remote: [String], resigned: Bool = false) {
        guard mode == .friend, let link else { return }
        guard remote.count >= moves.count, Array(remote.prefix(moves.count)) == moves else {
            startFriend(name: friendName, color: myColor, moves: remote, link: link)
            return
        }
        let fresh = Array(remote.dropFirst(moves.count))
        for uci in fresh {
            let taken = BoardMove(uci: uci, board: board).flatMap { $0.capture ? board[$0.to] : nil }
            _ = apply(uci)
            if !isOver, uci == fresh.last, turn == myColor {
                let tip = watchFor(lastMove, taken: taken)
                say(tip.urgent ? .warning : .opponent, game.lastSAN(), tip.title, tip.text)
            }
        }
        if resigned, !isOver {
            friendResigned = true
            status = "resigned"
        }
        _ = endIfOver()
    }

    /// The friend's name became known (they accepted the invitation).
    func renameFriend(_ name: String) {
        guard mode == .friend, name != friendName else { return }
        friendName = name
    }

    /// Another game with the same friend, through the same link.
    func rematch() {
        guard mode == .friend else { return }
        link?.rematch(self)
    }

    /// The friend's name mid-sentence: "your friend" when there's no name.
    var friendInSentence: String { friendName == Self.unnamedFriend ? L("your friend") : friendName }

    /// The name for a friend whose name the link doesn't know.
    static var unnamedFriend: String { L("Your friend") }

    private func greetFriendTurn() {
        if turn == myColor {
            say(.coach, "✦", L("Your move"), myColor == "w" ? L("You play White against \(friendInSentence).")
                                                              : L("You play Black against \(friendInSentence)."))
        } else {
            say(.opponent, "…", L("\(friendName)'s move"), L("You'll get a notification when they've played."))
        }
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
        // Say whose move it is, since that's the first thing you need back.
        say(.coach, "✦", L("Welcome back"), mode == .engine && turn == "b" && !isOver
            ? L("Your game is where you left it. They're thinking.")
            : L("Your game is where you left it. Your move."))
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
            let measured = out
            await MainActor.run {
                guard game === self.game else { return }
                let out = measured
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
        guard mode != .friend else { return }
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

    /// A private copy of the game for work off the main thread, so an undo
    /// can change the real one meanwhile. Replayed from the moves, so it keeps
    /// the history repetition needs.
    private func snapshot() -> ChesscoreGame {
        guard let g = ChesscoreNewGame() else { return game }
        for m in moves { _ = g.play(m) }
        return g
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

    /// The game's arc from the player's side (White's, in pass and play): the
    /// chance to win after every half-move, ending at the result on a mate.
    private func withArc(_ f: Finish) -> Finish {
        var f = f
        f.plies = moves.count
        let mine = mode == .passAndPlay ? "w" : myColor
        var last = 0.5
        var arc = [0.5] + moves.indices.map { i -> Double in
            if i < evals.count, let cp = evals[i] {
                let w = abs(cp) >= 9000 ? (cp > 0 ? 1.0 : 0.0) : 1 / (1 + exp(-Double(cp) / 220))
                last = mine == "w" ? w : 1 - w
            }
            return last
        }
        if status == ChesscoreStatusCheckmate { arc[arc.count - 1] = f.outcome == .won ? 1 : 0 }
        return f.reading(arc, playerParity: mine == "w" ? 0 : 1, twoPlayers: mode == .passAndPlay)
    }

    #if DEBUG
    /// A finish card for a made-up game, to look at the card without playing
    /// one: launched with `-finish won`, `lost` or `draw`.
    func previewFinish(_ outcome: Finish.Outcome) {
        let keys: [(Int, Double)] = switch outcome {
        case .won: [(0, 0.5), (8, 0.56), (15, 0.27), (22, 0.48), (30, 0.74), (40, 0.9), (47, 1)]
        case .lost: [(0, 0.5), (9, 0.6), (17, 0.73), (20, 0.7), (21, 0.31), (30, 0.2), (37, 0.08)]
        case .draw: [(0, 0.5), (10, 0.62), (18, 0.44), (28, 0.58), (40, 0.5)]
        }
        var arc: [Double] = []
        for (a, b) in zip(keys, keys.dropFirst()) {
            for i in a.0..<b.0 {
                let t = Double(i - a.0) / Double(b.0 - a.0)
                arc.append(a.1 + (b.1 - a.1) * t + sin(Double(i) * 1.7) * 0.015)
            }
        }
        arc.append(keys.last!.1)
        let won = outcome == .won
        var f = Finish(outcome: outcome,
                       title: won ? L("You beat the \(opponent.name)!") : outcome == .draw ? L("A draw") : L("Good game"),
                       text: won ? L("Checkmate in \(24) moves.") : outcome == .draw ? L("The same position came up three times.")
                           : L("Resigned on move \(19)"),
                       accuracy: won ? 86 : 71, great: won ? 3 : 1, blunders: won ? 0 : 2)
        f.plies = arc.count - 1
        f.best = BestMove(before: Array(repeating: ".", count: 64), after: Array(repeating: ".", count: 64),
                          move: BoardMove(from: 62, to: 45), moveNumber: won ? 16 : 8, kind: .great,
                          text: "", ply: won ? 31 : 14, san: won ? "Rxe7" : "Nf3")
        if won { f.towardNext = (.medium, 1) }
        finish = f.reading(arc, playerParity: 0, twoPlayers: false)
    }
    #endif

    /// The player's best move: the strongest verdict, the latest among equals.
    /// The finish card leads with it, so every game ends on something good.
    private func bestMove() -> BestMove? {
        let rank: [CoachCard.Kind: Int] = [.great: 4, .good: 3, .inaccuracy: 2, .mistake: 1, .blunder: 0]
        let mine = trail
        guard let e = mine.max(by: { (rank[$0.card.kind] ?? 0, $0.ply) < (rank[$1.card.kind] ?? 0, $1.ply) }),
              (rank[e.card.kind] ?? 0) >= 3,
              let after = position(afterPly: e.ply), let move = after.move,
              let before = position(afterPly: e.ply - 1) else { return nil }
        return BestMove(side: mode == .passAndPlay ? (e.ply % 2 == 0 ? L("White") : L("Black")) : nil,
                        before: before.board, after: after.board, move: move, moveNumber: e.ply / 2 + 1,
                        kind: e.card.kind, text: e.card.text, ply: e.ply, san: e.san)
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
                           detail: entry.card.text, footer: L("Found with Cozy Chess"))
    }

    /// A win against the engine, ready to share.
    /// What the game-over card shares: a win against the engine, or otherwise
    /// the game's best move, so every game has something to show.
    var finishMoment: ShareMoment? {
        guard let finish else { return nil }
        if mode != .passAndPlay, finish.outcome == .won {
            let accuracy = finish.accuracy.map { " · \($0)% accuracy" } ?? ""
            let who = mode == .friend ? friendName : "the \(opponent.name)"
            return ShareMoment(board: board, move: lastMove, title: "I beat \(who)",
                               detail: finish.text + accuracy, footer: L("Played on Cozy Chess"))
        }
        // Not a win: share warmly, about what went well, never the loss.
        let against = mode == .engine ? " against the \(opponent.name)" : mode == .friend ? " against \(friendName)" : ""
        guard let best = finish.best else {
            let title = mode == .engine ? "I played the \(opponent.name)" : mode == .friend ? "I played \(friendName)" : finish.title
            return ShareMoment(board: board, move: lastMove, title: title,
                               detail: L("Every game teaches me something."), footer: L("Learning with Cozy Chess"))
        }
        let title = best.side.map { "\($0)'s best move" } ?? "A move I'm proud of"
        return ShareMoment(board: best.after, move: best.move, title: title,
                           detail: L("Move \(best.moveNumber)\(against) · \(best.text)"), footer: L("Learning with Cozy Chess"))
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
        let me = mode == .passAndPlay ? turn : myColor, them = me == "w" ? "b" : "w"
        func name(_ sq: Square) -> String { L("\(Self.pieceName(board[sq])) on \(Squares.name(sq))") }
        func squares(_ s: String) -> [Square] { s.split(separator: " ").compactMap { Squares.parse($0) } }
        func isMine(_ sq: Square) -> Bool { board[sq] != "." && (board[sq].isUppercase ? "w" : "b") == me }

        if inCheck {
            return (L("Your king is in check"), L("Move it, block the line, or take the checking piece."), true)
        }
        if let m {
            let mover = Self.pieceName(board[m.to])
            // What the moved piece now hits, biggest first.
            let hit = (0..<64).filter { isMine($0) && board[$0] != "K" && squares(game.attackers(Squares.name($0))).contains(m.to) }
                .sorted { (Self.value[Character(board[$0].lowercased())] ?? 0) > (Self.value[Character(board[$1].lowercased())] ?? 0) }
            if let sq = hit.first(where: { game.guards(Squares.name($0)).isEmpty }) {
                return (L("Watch your \(name(sq))"), L("Their \(mover) now attacks it, and nothing guards it."), true)
            }
            if let taken {
                let back = !game.attackers(Squares.name(m.to)).isEmpty
                return (L("They took your \(Self.pieceName(taken))"),
                        back ? L("You can take back on \(Squares.name(m.to)).") : L("Look for a way to win it back."), false)
            }
            if let sq = hit.first, (Self.value[Character(board[sq].lowercased())] ?? 0) > (Self.value[Character(board[m.to].lowercased())] ?? 0) {
                return (L("Their \(mover) eyes your \(Self.pieceName(board[sq]))"),
                        L("It's guarded, but their \(mover) is worth less, so a trade would cost you."), true)
            }
        }
        if let sq = loose.sorted().first {
            return (L("Your \(name(sq)) is loose"), L("It's attacked and nothing guards it."), true)
        }
        if let sq = squares(game.loose(them)).first {
            return (L("A chance"), L("Their \(name(sq)) is under attack and unguarded."), false)
        }
        // Nothing pressing: a quiet "Your move", with a tip only now and then.
        // Every other quiet turn, one idea worth carrying into the game.
        quietTurns += 1
        return (L("Your move"), quietTurns % 2 == 1 ? tip() : "", false)
    }

    /// A tip that fits the position, or one of the app's gestures, each shown once.
    private func tip() -> String {
        let castling = game.fen().split(separator: " ").dropFirst(2).first.map(String.init) ?? ""
        let white = (mode == .passAndPlay ? turn : myColor) == "w"
        let home: [(Square, Character)] = white ? [(57, "N"), (62, "N"), (58, "B"), (61, "B")]  // b1 g1 c1 f1
                                                : [(1, "n"), (6, "n"), (2, "b"), (5, "b")]      // b8 g8 c8 f8
        let undeveloped = home.filter { board[$0.0] == $0.1 }.count
        if moves.count < 16 && undeveloped >= 2 {
            return L("Bring a knight or bishop toward the centre.")
        }
        if moves.count < 30 && board[white ? 60 : 4] == (white ? "K" : "k")
            && castling.contains(where: { white ? ($0 == "K" || $0 == "Q") : ($0 == "k" || $0 == "q") }) {
            return L("Castling tucks your king away and brings a rook into play.")
        }
        let gestures = [("tip.hold", L("Hold any piece to see what guards it and what attacks it.")),
                        ("tip.drag", L("You can drag pieces as well as tap them.")),
                        ("tip.swipe", L("Swipe this card to look back at your earlier moves."))]
        if let (key, text) = gestures.first(where: { !UserDefaults.standard.bool(forKey: $0.0) }) {
            UserDefaults.standard.set(true, forKey: key)
            return text
        }
        // Ideas, never moves: something to look for, not something to play.
        let ideas = [L("Before you move, ask what their last move wants."),
                     L("Pieces in the centre reach the most squares."),
                     L("Look for checks, captures and threats, for both sides."),
                     L("A knight on the edge reaches only half its squares."),
                     L("Rooks love open lines with no pawns in the way."),
                     L("Ahead in pieces? Trading down makes it simpler."),
                     L("Pawns can't move back, so push them with a purpose."),
                     L("Every piece helping beats one piece attacking alone.")]
        ideaTurn += 1
        return ideas[(ideaTurn - 1) % ideas.count]
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
        let mineColor = (piece.isUppercase ? "w" : "b") == myColor
        let what = Self.pieceName(piece)
        let title = switch mode {
        case .engine: mineColor ? L("Your \(what) on \(name)") : L("Your opponent's \(what) on \(name)")
        case .friend: mineColor ? L("Your \(what) on \(name)") : L("\(friendName)'s \(what) on \(name)")
        case .passAndPlay: piece.isUppercase ? L("White's \(what) on \(name)") : L("Black's \(what) on \(name)")
        }
        let g = guards.count, a = attackers.count
        let text: String = switch (g, a) {
        case (0, 0): L("Nothing attacks it, and nothing guards it.")
        case (_, 0): L("Nothing attacks it, and it's guarded by \(g) of your pieces.")
        case (0, _): L("Attacked by \(a) of their pieces, and nothing guards it.")
        default: L("Attacked by \(a), guarded by \(g).")
        }
        say(.coach, "◎", title, text)
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

    /// True when "Hint" would repeat a hint already used in this position.
    var hintIsFree: Bool { hintShown?.ply == moves.count }
    /// The engine is looking for a hint. The game is busy meanwhile, but the
    /// Hint button stays as it is, so it doesn't flash grey and back.
    private(set) var findingHint = false

    /// Hints come in two steps. The first points at the piece to move; asking
    /// again in the same position shows the move and why, at no extra cost.
    func hint() {
        guard playerToMove else { return }
        selected = nil
        if let shown = hintShown, shown.ply == moves.count, let m = BoardMove(uci: shown.uci, board: board) {
            nudge = nil
            hintRevealed = true
            rays = makeRays(.hint, [m])
            say(.great, "✦", L("Try this"), shown.text)
            emit(.good)
            return
        }
        guard hintsLeft > 0 else { return }
        hintsLeft -= 1
        save()
        rays = nil
        busy = true
        findingHint = true
        let game = snapshot(), ply = moves.count, token = epoch
        Task.detached(priority: .userInitiated) {
            let best = game.search(3, moveTimeMs: 0)?.bestMove ?? ""
            let san = best.isEmpty ? "" : game.san(best)
            let why = best.isEmpty ? "" : EngineText.localize(game.explain(best))
            await MainActor.run {
                self.findingHint = false
                guard token == self.epoch else { return }
                self.busy = false
                guard !best.isEmpty, let m = BoardMove(uci: best, board: self.board) else { return }
                let text = "\(san): \(why)"
                self.hintShown = (ply, best, text)
                self.hintRevealed = false
                self.nudge = m.from
                let piece = Self.pieceName(self.board[m.from])
                self.say(.coach, "✦", L("Look at your \(piece)"),
                         L("Your \(piece) on \(Squares.name(m.from)) has a good move. Find it, or tap Show the move."))
                self.emit(.select)
            }
        }
    }

    func undo() {
        mark = nil
        nudge = nil
        hintRevealed = false
        guard mode != .friend else { return } // a sent move can't be taken back
        var steps = mode == .engine ? 2 : 1
        if mode == .engine && turn == "b" { steps = 1 } // the engine has not replied yet
        guard status != "resigned", moves.count >= steps else { return }
        // Undo works at any moment: whatever the engine was weighing is dropped.
        epoch += 1
        busy = false
        thinking = false
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
        // Drop the verdicts on the moves taken back; one still being judged
        // has none yet, and nothing earlier is touched.
        let dropped = trail.filter { $0.ply >= moves.count }.count
        trail.removeAll { $0.ply >= moves.count }
        losses.removeLast(min(dropped, losses.count))
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
        say(.coach, "↶", L("Rewound"), L("Your move again. Take another look."))
        save()
    }

    /// Ends the game as a loss.
    func resign() {
        guard mode != .passAndPlay, !isOver, !busy else { return }
        status = "resigned"
        selected = nil
        rays = nil
        loose = []
        if let k = board.firstIndex(of: myColor == "w" ? "K" : "k") { toppled = ids[k] }
        if mode == .friend { link?.didResign(self) }
        let count = (moves.count + 1) / 2
        let f = Finish(outcome: .lost, title: L("Good game"), text: L("Resigned on move \(count)"),
                       accuracy: accuracy, great: great, blunders: blunders)
        Task {
            try? await Task.sleep(for: .seconds(0.9))
            var shown = f
            shown.goodMoves = trail.filter { $0.card.kind == .good || $0.card.kind == .great }.count
            shown.best = bestMove()
            finish = withArc(shown)
            emit(.lose)
            if mode == .engine {
                Stats.record(accuracy: f.accuracy, great: f.great)
                SavedGame.clear()
            }
            Stats.addRecord(GameRecord(date: .now, opponent: mode == .friend ? friendName : opponent.name, outcome: "lost",
                                       accuracy: f.accuracy, moves: count,
                                       level: mode == .engine ? opponent.rawValue : nil, great: f.great))
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
        // A friend game sends the move at once; the verdict is only for this phone.
        if mode == .friend { link?.didMove(self) }

        // Judge the move against the position before it, off the main thread.
        let game = snapshot(), token = epoch
        Task.detached(priority: .userInitiated) {
            let verdict = ChesscoreNewGameFromFEN(fenBefore)?.assess(uci)
            let eval = game.eval(2)
            await MainActor.run {
                guard token == self.epoch else { return }
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
        let words = kind.verdict ?? (L("Good move"), "✓")
        let nudge = kind == .blunder ? " " + L("Undo and try again.") : ""
        // A calm coach: a good move is the norm, so it gets the badge and a
        // word, no explanation and no pause. Great moves and slips are worth
        // a sentence and the time to read it.
        let reason = EngineText.localize(v.text)
        say(kind, words.badge, words.title, kind == .good ? "" : reason + nudge)
        // The trail keeps the verdict without "Undo and try again", which only
        // makes sense right after the move.
        verdictShownAt = kind == .good ? nil : .now
        if let card = coach {
            let kept = CoachCard(id: card.id, kind: card.kind, badge: card.badge, title: card.title, text: reason)
            let entry = TrailEntry(id: bump(), ply: ply, san: san, card: kept)
            trail.append(entry)
            if let to = lastMove?.to {
                mark = (entry, to)
                if kind == .great { sparkle = (entry.id, to) }
            }
        }
        reviewing = nil
        emit(kind == .great ? .great : kind == .good ? .good : .bad)
        save()
    }

    /// The engine shows the moves it is weighing, one by one, then commits.
    private func engineTurn() {
        busy = true
        thinking = true
        let game = snapshot(), level = opponent.engineLevel, token = epoch
        Task.detached(priority: .userInitiated) {
            let candidates = game.candidates(4).split(separator: " ").map(String.init)
            let choice = game.botMove(level)
            let line = choice.isEmpty ? "" : EngineText.localize(game.describe(choice))
            await MainActor.run {
                guard token == self.epoch else { return }
                self.showThinking(candidates, choice, line, token)
            }
        }
    }

    private func showThinking(_ candidates: [String], _ choice: String, _ line: String, _ token: Int) {
        guard !choice.isEmpty else { busy = false; thinking = false; return }
        var shown = candidates
        if !shown.contains(choice) {
            if shown.isEmpty { shown = [choice] } else { shown[shown.count - 1] = choice }
        }
        Task {
            for k in 1...shown.count {
                guard token == epoch else { return }
                rays = makeRays(.thinking, shown.prefix(k).compactMap { BoardMove(uci: $0, board: board) })
                try? await Task.sleep(for: .milliseconds(260))
            }
            guard token == epoch else { return }
            if let m = BoardMove(uci: choice, board: board) { rays = makeRays(.chosen, [m]) }
            try? await Task.sleep(for: .milliseconds(420))
            // Let the verdict on the player's move be read before it's replaced.
            if let shown = verdictShownAt {
                let left = Self.readingTime(trail.last?.card) - Date.now.timeIntervalSince(shown)
                if left > 0 { try? await Task.sleep(for: .seconds(left)) }
            }
            guard token == epoch else { return }
            rays = nil
            thinking = false
            let taken = BoardMove(uci: choice, board: board).flatMap { $0.capture ? board[$0.to] : nil }
            let san = apply(choice)
            busy = false
            if endIfOver() {
                say(.opponent, san, L("Your opponent \(line)."), L("That's the game."))
                return
            }
            let after = snapshot()
            let eval = await Task.detached { after.eval(2) }.value
            guard token == epoch else { return }
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
        if mode == .passAndPlay || turn == myColor { mark = nil }
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
        if friendResigned {
            f = Finish(outcome: .won, title: L("You beat \(friendName)!"), text: L("\(friendName) resigned after \(moves) moves."),
                       accuracy: accuracy, great: great, blunders: blunders)
        } else if mate, mode == .friend {
            let iWon = turn != myColor
            f = Finish(outcome: iWon ? .won : .lost, title: iWon ? L("You beat \(friendName)!") : L("Good fight"),
                       text: iWon ? L("Checkmate in \(moves) moves.") : L("Checkmate after \(moves) moves. Next time is yours."),
                       accuracy: accuracy, great: great, blunders: blunders)
        } else if mate {
            let whiteWon = turn == "b"
            if mode == .passAndPlay {
                f = Finish(outcome: .won, title: whiteWon ? L("White wins") : L("Black wins"),
                           text: L("Checkmate in \(moves) moves."), accuracy: accuracy, great: great, blunders: blunders)
            } else {
                f = Finish(outcome: whiteWon ? .won : .lost, title: whiteWon ? L("You won!") : L("Good fight"),
                           text: whiteWon ? L("Checkmate in \(moves) moves.") : L("Checkmate after \(moves) moves. Next time is yours."), accuracy: accuracy, great: great, blunders: blunders)
            }
        } else {
            let why = [ChesscoreStatusStalemate: L("Stalemate: no legal moves, and no check."),
                       ChesscoreStatusRepetition: L("The same position came up three times."),
                       ChesscoreStatusFiftyMove: L("Fifty moves without a capture or a pawn move."),
                       ChesscoreStatusInsufficient: L("Neither side has enough left to mate.")][status] ?? L("The game is drawn.")
            f = Finish(outcome: .draw, title: L("A draw"), text: why, accuracy: accuracy, great: great, blunders: blunders)
        }
        var final = f
        final.goodMoves = trail.filter { $0.card.kind == .good || $0.card.kind == .great }.count
        if mode == .engine, f.outcome == .won {
            let next = Opponent(rawValue: opponent.rawValue + 1)
            let wasLocked = next.map { !$0.isUnlocked } ?? false
            final.firstWin = Stats.wins(against: opponent) == 0
            Stats.addWin(against: opponent)
            final = Finish(outcome: .won, title: L("You beat the \(opponent.name)!"), text: f.text,
                           accuracy: f.accuracy, great: f.great, blunders: f.blunders, goodMoves: final.goodMoves,
                           firstWin: final.firstWin)
            if let next, wasLocked {
                if next.isUnlocked { final.unlocked = next }
                else { final.towardNext = (next, Stats.wins(against: opponent)) }
            }
        }
        final.best = bestMove()
        final = withArc(final)
        if mode == .passAndPlay {
            final.sides = (trail.filter { $0.ply % 2 == 0 && $0.card.kind == .great }.count,
                           trail.filter { $0.ply % 2 == 1 && $0.card.kind == .great }.count)
        }
        Task {
            try? await Task.sleep(for: .seconds(1.1))
            finish = final
            emit(final.outcome == .won ? .win : .lose)
            if mode == .engine { Stats.record(accuracy: final.accuracy, great: final.great) }
            Stats.addRecord(GameRecord(date: .now, opponent: mode == .engine ? opponent.name : mode == .friend ? friendName : "Pass & play",
                                       outcome: ["won", "lost", "draw"][[.won, .lost, .draw].firstIndex(of: final.outcome)!],
                                       accuracy: final.accuracy, moves: moves,
                                       level: mode == .engine ? opponent.rawValue : nil, great: final.great))
            if self.mode != .friend { SavedGame.clear() }
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
        let mine = mode == .passAndPlay ? turn : myColor
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
        localizedPiece(["p": "pawn", "n": "knight", "b": "bishop", "r": "rook", "q": "queen", "k": "king"][Character(c.lowercased())] ?? "piece")
    }

    /// A piece's English name ("knight") in the player's language.
    nonisolated static func localizedPiece(_ english: String) -> String {
        switch english {
        case "pawn": L("pawn")
        case "knight": L("knight")
        case "bishop": L("bishop")
        case "rook": L("rook")
        case "queen": L("queen")
        case "king": L("king")
        default: L("piece")
        }
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
