import SwiftUI

/// "Learn the moves": ten tiny moments for someone who has never played. Each
/// is one idea, one short line and one thing to do, on the game's own board.
/// Solve it and the next one comes on its own: no counters, no buttons in
/// between, no way to fail.
struct LearnView: View {
    /// Finished and wants a first game.
    let play: () -> Void
    @Environment(\.dismiss) private var dismiss

    /// Where you are, kept so you can leave and come back.
    @AppStorage("learn.step") private var step = 0
    @State private var lesson = LessonModel()
    @State private var solved = false
    @State private var busy = false
    @State private var taken = 0
    @State private var wrong = 0
    @State private var done = 0

    private var moment: Moment? { Moment.all.indices.contains(step) ? Moment.all[step] : nil }
    private var progress: Double { Double(step + (solved ? 1 : 0)) / Double(Moment.all.count) }

    var body: some View {
        ZStack {
            AppBackground()
            VStack(spacing: 0) {
                header
                if let moment {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(moment.title).font(Typeface.display(30)).tracking(-0.5)
                        Text(moment.text).font(.system(size: 17)).foregroundStyle(Palette.mist)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.top, 28)
                    .id(step)
                    .transition(.opacity)

                    BoardView(model: lesson)
                        .overlay(alignment: .top) {
                            if moment.goal.isMate && solved { Celebration().offset(y: 80) }
                        }
                        .padding(.top, 24)
                    Spacer(minLength: 16)
                } else {
                    finale
                }
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 16)
            .frame(maxWidth: 560)
            .foregroundStyle(Palette.ink)
        }
        .onAppear {
            lesson.onMove = { from, to in move(from, to) }
            load()
        }
        .sensoryFeedback(.impact(weight: .medium), trigger: taken)
        .sensoryFeedback(.impact(flexibility: .soft, intensity: 0.6), trigger: wrong)
        .sensoryFeedback(.success, trigger: done)
        .animation(.smooth(duration: 0.4), value: step)
    }

    /// Close, and a thin line that fills as you go.
    private var header: some View {
        HStack(spacing: 16) {
            Button {
                Haptics.tap()
                if moment == nil { step = 0 } // finished: next time starts fresh
                dismiss()
            } label: {
                Image(systemName: "xmark").font(.system(size: 14, weight: .semibold)).frame(width: 30, height: 30)
            }
            .buttonStyle(.glass)
            .buttonBorderShape(.circle)
            .accessibilityLabel("Close")
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Capsule().fill(Palette.mist.opacity(0.2))
                    Capsule().fill(Palette.accent)
                        .frame(width: max(6, geo.size.width * progress))
                }
            }
            .frame(height: 6)
            .animation(.spring(response: 0.5, dampingFraction: 0.8), value: progress)
            .accessibilityHidden(true)
            Color.clear.frame(width: 30, height: 30)
        }
        .padding(.top, 8)
    }

    /// The payoff: an empty board fills with every piece you just learned, in
    /// the order you learned them, until a whole game is set up. That's chess,
    /// and you know every piece on it.
    private var finale: some View {
        VStack(spacing: 0) {
            VStack(alignment: .leading, spacing: 6) {
                Text("You know every piece").font(Typeface.display(30)).tracking(-0.5)
                Text("That's the whole game. Time to play.").font(.system(size: 17)).foregroundStyle(Palette.mist)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.top, 28)
            FinaleBoard { done += 1; Sound.shared.play(.win) }
                .padding(.top, 24)
            Spacer(minLength: 16)
            VStack(spacing: 20) {
                Button("Play the Rookie") {
                    Haptics.tap()
                    UserDefaults.standard.set(true, forKey: "learn.done")
                    step = 0
                    dismiss()
                    play()
                }
                .buttonStyle(PrimaryButtonStyle())
                Button("Back to home") {
                    UserDefaults.standard.set(true, forKey: "learn.done")
                    step = 0
                    dismiss()
                }
                .font(.system(size: 15, weight: .semibold)).foregroundStyle(Palette.mist)
                .frame(minHeight: 44)
                .buttonStyle(.plain)
            }
            .padding(.bottom, 4)
        }
    }

    private func load() {
        guard let moment else { return }
        solved = false
        busy = false
        lesson.load(moment)
        if case let .escape(_, from) = moment.goal {
            lesson.showOptions(danger: BoardMove(from: from, to: moment.start, capture: true))
        } else {
            lesson.showOptions()
        }
    }

    private func move(_ from: Square, _ to: Square) {
        guard let moment, !busy else { return }
        busy = true
        let shown = step
        // Exactly as in a game: the piece glides, and what it takes goes at once.
        if lesson.slide(to: to) {
            taken += 1
            Sound.shared.play(.capture)
        } else {
            Sound.shared.play(.move)
        }
        Task {
            try? await Task.sleep(for: .seconds(0.3))
            guard shown == step else { return }
            switch moment.goal {
            case .takeAll:
                if !lesson.others.values.contains(where: \.isLowercase) { finish(at: to) } else { carryOn() }
            case .promote:
                if to < 8 {
                    lesson.promote()
                    finish(at: to)
                } else { carryOn() }
            case let .check(king):
                if lesson.attacks(king) {
                    lesson.showCheck(from: to, on: king)
                    finish(at: nil)
                } else { await miss(back: from) }
            case let .escape(safe, _):
                if safe.contains(to) { finish(at: to) } else { await miss(back: from) }
            case let .mate(squares, king):
                if squares.contains(to) {
                    lesson.showCheck(from: to, on: king)
                    try? await Task.sleep(for: .seconds(0.3))
                    lesson.topple(at: king)
                    finish(at: nil, last: true)
                } else { await miss(back: from) }
            }
        }
    }

    private func carryOn() {
        busy = false
        lesson.showOptions()
    }

    /// Not this one: a soft bump, and the piece goes back where it was.
    private func miss(back from: Square) async {
        wrong += 1
        try? await Task.sleep(for: .seconds(0.25))
        lesson.slide(to: from)
        try? await Task.sleep(for: .seconds(0.3))
        busy = false
        if let moment, case let .escape(_, attacker) = moment.goal {
            lesson.showOptions(danger: BoardMove(from: attacker, to: from, capture: true))
        } else {
            lesson.showOptions()
        }
    }

    /// Solved: the line fills, a moment to enjoy it, then the next one.
    private func finish(at square: Square?, last: Bool = false) {
        solved = true
        done += 1
        if let square { lesson.celebrate(at: square) }
        Sound.shared.play(last ? .win : .great)
        let shown = step
        Task {
            try? await Task.sleep(for: .seconds(last ? 2.2 : 1.1))
            guard shown == step else { return }
            withAnimation(.smooth(duration: 0.4)) { step += 1 }
            load()
        }
    }
}

/// The finale's board: empty stone, then the pieces arrive one after another,
/// in the order they were learned, white then black, and settle onto their
/// starting squares.
private struct FinaleBoard: View {
    /// Called once the last piece has landed.
    let landed: () -> Void
    @State private var shown = false

    /// The starting position, in the order the pieces were learned.
    private static let arrivals: [(code: Character, square: Square)] = {
        let order: [Character] = ["R", "B", "Q", "N", "K", "P"]
        let back: [Character] = ["R", "N", "B", "Q", "K", "B", "N", "R"]
        var white: [(Character, Square)] = (0..<8).map { (back[$0], 56 + $0) } + (0..<8).map { ("P", 48 + $0) }
        white.sort { (order.firstIndex(of: $0.0) ?? 0) < (order.firstIndex(of: $1.0) ?? 0) }
        let black = white.map { (Character($0.0.lowercased()), $0.1 % 8 + (7 - $0.1 / 8) * 8) }
        return (white + black).map { (code: $0.0, square: $0.1) }
    }()

    var body: some View {
        GeometryReader { geo in
            let size = geo.size.width, q = size / 8
            ZStack(alignment: .topLeading) {
                StaticBoard(fen: "8/8/8/8/8/8/8/8")
                ForEach(Self.arrivals.indices, id: \.self) { i in
                    let a = Self.arrivals[i]
                    let home = center(a.square, size)
                    GlassPiece(code: a.code)
                        .frame(width: q, height: q)
                        .scaleEffect(shown ? 1 : 0.4)
                        .opacity(shown ? 1 : 0)
                        // Each one drops in from a little above its square.
                        .position(x: home.x, y: home.y - (shown ? 0 : q * 1.2))
                        .animation(.spring(response: 0.45, dampingFraction: 0.68).delay(0.3 + Double(i) * 0.05), value: shown)
                }
            }
            .frame(width: size, height: size)
            .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
            .shadow(color: .black.opacity(0.28), radius: 18, y: 14)
        }
        .aspectRatio(1, contentMode: .fit)
        .task {
            shown = true
            try? await Task.sleep(for: .seconds(0.3 + Double(Self.arrivals.count) * 0.05 + 0.3))
            landed()
        }
        .accessibilityLabel("A full chess board, set up")
    }
}

/// A Learn moment as the game board sees it: the one piece you move, what else
/// stands on the board, and the lines from your piece to where it can go. It
/// speaks the same language as a game, so it plays on the very same board.
@Observable
final class LessonModel: BoardSource {
    private(set) var shownPieces: [PieceItem] = []
    let flipped = false
    private(set) var selected: Square?
    private(set) var rays: Rays?
    private(set) var bursts: [Burst] = []
    private(set) var toppled: Int?
    private(set) var movingPiece: Int?
    private(set) var moveSerial = 0
    /// You always move in a lesson, so the light rests on your side.
    let turn = "w"
    let reviewing: Int? = nil
    let shownLastMove: BoardMove? = nil
    private(set) var sparkle: (id: Int, square: Square)?
    let nudge: Square? = nil
    let loose: Set<Square> = []
    private(set) var checkSquare: Square?
    var shownMark: (entry: TrailEntry, square: Square)? { nil }

    /// A move the player made, from and to; Learn decides what it means.
    @ObservationIgnored var onMove: ((Square, Square) -> Void)?
    @ObservationIgnored private var serial = 0
    private static let moverID = 0

    private var mover: PieceItem? { shownPieces.first { $0.id == Self.moverID } }

    /// Everything but your piece: black pieces to take, white ones in the way.
    var others: [Square: Character] {
        Dictionary(shownPieces.filter { $0.id != Self.moverID }.map { ($0.square, $0.code) }, uniquingKeysWith: { a, _ in a })
    }

    private var targets: [Square] {
        guard let m = mover else { return [] }
        return Moment.moves(m.code, from: m.square, on: others)
    }

    func load(_ moment: Moment) {
        var items = [PieceItem(id: Self.moverID, code: moment.piece, square: moment.start)]
        for (k, entry) in moment.others.sorted(by: { $0.key < $1.key }).enumerated() {
            items.append(PieceItem(id: k + 1, code: entry.value, square: entry.key))
        }
        shownPieces = items
        selected = nil
        rays = nil
        bursts = []
        toppled = nil
        checkSquare = nil
        movingPiece = nil
        sparkle = nil
    }

    /// The lines from your piece to where it can go, and a line of danger to
    /// your king if it is in check.
    func showOptions(danger: BoardMove? = nil) {
        guard let m = mover else { return }
        var moves = targets.map { BoardMove(from: m.square, to: $0, capture: others[$0]?.isLowercase == true) }
        var red: Set<BoardMove> = []
        if let danger { moves.append(danger); red = [danger] }
        serial += 1
        rays = Rays(id: serial, kind: .options, moves: moves, knight: m.code == "N" ? Set(moves.filter { $0 != danger }) : [],
                    danger: red)
        checkSquare = danger?.to
    }

    /// Your move lit up red at the king, and the king's square glowing.
    func showCheck(from: Square, on king: Square) {
        let line = BoardMove(from: from, to: king, capture: true)
        serial += 1
        rays = Rays(id: serial, kind: .chosen, moves: [line], knight: [], danger: [line])
        checkSquare = king
    }

    /// Glides your piece to a square with the game board's motion, taking a
    /// black piece there at once, in glass, as a game does. True if it took one.
    @discardableResult
    func slide(to square: Square) -> Bool {
        rays = nil
        checkSquare = nil
        selected = nil
        var took = false
        if let k = shownPieces.firstIndex(where: { $0.id != Self.moverID && $0.square == square && $0.code.isLowercase }) {
            shownPieces.remove(at: k)
            serial += 1
            let burst = Burst(id: serial, square: square, dark: true)
            bursts.append(burst)
            took = true
            Task { @MainActor in
                try? await Task.sleep(for: .seconds(0.9))
                self.bursts.removeAll { $0 == burst }
            }
        }
        guard let i = shownPieces.firstIndex(where: { $0.id == Self.moverID }) else { return took }
        movingPiece = Self.moverID
        shownPieces[i].square = square
        moveSerial += 1
        return took
    }

    /// The board's own sparkle for a moment solved.
    func celebrate(at square: Square) {
        serial += 1
        sparkle = (serial, square)
    }

    func promote() {
        guard let i = shownPieces.firstIndex(where: { $0.id == Self.moverID }) else { return }
        withAnimation(.spring(response: 0.4, dampingFraction: 0.55)) { shownPieces[i].code = "Q" }
    }

    /// Does your piece, where it stands, attack this square?
    func attacks(_ square: Square) -> Bool { targets.contains(square) }

    func topple(at square: Square) {
        toppled = shownPieces.first { $0.square == square && $0.id != Self.moverID }?.id
    }

    // The board's finger: tap where a line ends, or carry the piece there.
    func tap(_ square: Square) {
        guard let m = mover, rays?.kind == .options, targets.contains(square) else { return }
        onMove?(m.square, square)
    }

    func pickUp(_ square: Square) -> Bool {
        guard rays?.kind == .options, square == mover?.square else { return false }
        selected = square
        return true
    }

    func drop(on square: Square?) {
        selected = nil
        guard let square, let m = mover, rays?.kind == .options, targets.contains(square) else { return }
        onMove?(m.square, square)
    }

    func inspect(_ square: Square) -> Bool { false }
    func endInspect() {}
    func revealMark() {}
}

/// One moment: a piece to move, what else is on the board, and what solves it.
struct Moment {
    enum Goal {
        /// Take every black piece.
        case takeAll
        /// Walk the pawn to the far side, where it becomes a queen.
        case promote
        /// Attack the king on this square.
        case check(Square)
        /// Your king is attacked from `from`; any of `safe` gets it out.
        case escape(Set<Square>, from: Square)
        /// The move on one of these squares checkmates the king.
        case mate(Set<Square>, king: Square)

        var isMate: Bool { if case .mate = self { true } else { false } }
    }

    let piece: Character
    let start: Square
    var others: [Square: Character] = [:]
    let goal: Goal
    let title: String
    let text: String

    /// A square from its name, like "e4".
    static func sq(_ name: String) -> Square {
        let f = Int(name.unicodeScalars.first!.value) - 97, r = Int(String(name.last!))!
        return (8 - r) * 8 + f
    }

    private static func board(_ list: [String: Character]) -> [Square: Character] {
        Dictionary(uniqueKeysWithValues: list.map { (sq($0.key), $0.value) })
    }

    static let all: [Moment] = [
        Moment(piece: "R", start: sq("a1"), others: board(["a5": "p", "e5": "p", "e8": "p"]), goal: .takeAll,
               title: L("The rook"), text: L("Take the pawns.")),
        Moment(piece: "B", start: sq("c1"), others: board(["e3": "p", "g5": "p", "d8": "p"]), goal: .takeAll,
               title: L("The bishop"), text: L("Take the pawns.")),
        Moment(piece: "Q", start: sq("d1"), others: board(["d5": "p", "h5": "p", "e8": "p"]), goal: .takeAll,
               title: L("The queen"), text: L("Take the pawns.")),
        Moment(piece: "N", start: sq("g1"), others: board(["f2": "P", "g2": "P", "h2": "P", "f3": "p", "e5": "p", "d7": "p"]),
               goal: .takeAll, title: L("The knight"), text: L("Take the pawns.")),
        Moment(piece: "K", start: sq("e1"), others: board(["e2": "p", "f3": "p", "f4": "p"]), goal: .takeAll,
               title: L("The king"), text: L("Take the pawns.")),
        Moment(piece: "P", start: sq("e2"), others: board(["d5": "p"]), goal: .takeAll,
               title: L("The pawn"), text: L("Take the pawn.")),
        Moment(piece: "P", start: sq("e7"), goal: .promote,
               title: L("Promotion"), text: L("Reach the far side.")),
        Moment(piece: "R", start: sq("a1"), others: board(["e8": "k"]), goal: .check(sq("e8")),
               title: L("Check"), text: L("Attack the king.")),
        Moment(piece: "K", start: sq("e1"), others: board(["e8": "r"]),
               goal: .escape([sq("d1"), sq("d2"), sq("f1"), sq("f2")], from: sq("e8")),
               title: L("Escape"), text: L("Step out of check.")),
        Moment(piece: "R", start: sq("a1"), others: board(["g8": "k", "f7": "p", "g7": "p", "h7": "p"]),
               goal: .mate([sq("a8")], king: sq("g8")),
               title: L("Checkmate"), text: L("Trap the king.")),
    ]

    /// Where a white piece can go: slides stop at the first piece, taking it
    /// if it's black; knights jump; pawns step forward and take diagonally.
    static func moves(_ code: Character, from s: Square, on board: [Square: Character]) -> [Square] {
        let r = s / 8, c = s % 8
        var out: [Square] = []
        func at(_ rr: Int, _ cc: Int) -> Square? { (0..<8).contains(rr) && (0..<8).contains(cc) ? rr * 8 + cc : nil }
        func slide(_ dirs: [(Int, Int)]) {
            for (dr, dc) in dirs {
                var rr = r + dr, cc = c + dc
                while let t = at(rr, cc) {
                    if let p = board[t] { if p.isLowercase { out.append(t) }; break }
                    out.append(t)
                    rr += dr; cc += dc
                }
            }
        }
        func step(_ offs: [(Int, Int)]) {
            for (dr, dc) in offs {
                if let t = at(r + dr, c + dc), board[t]?.isUppercase != true { out.append(t) }
            }
        }
        let orth = [(1, 0), (-1, 0), (0, 1), (0, -1)], diag = [(1, 1), (1, -1), (-1, 1), (-1, -1)]
        switch code {
        case "R": slide(orth)
        case "B": slide(diag)
        case "Q": slide(orth + diag)
        case "K": step(orth + diag)
        case "N": step([(2, 1), (2, -1), (-2, 1), (-2, -1), (1, 2), (1, -2), (-1, 2), (-1, -2)])
        case "P":
            if let t = at(r - 1, c), board[t] == nil {
                out.append(t)
                if r == 6, let t2 = at(r - 2, c), board[t2] == nil { out.append(t2) }
            }
            for dc in [-1, 1] { if let t = at(r - 1, c + dc), board[t]?.isLowercase == true { out.append(t) } }
        default: break
        }
        return out
    }
}
