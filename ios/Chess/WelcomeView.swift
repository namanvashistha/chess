import Chesscore
import SwiftUI

/// The first time the app opens: a short story, then you play it out.
///
/// The lore, in one sentence: chess is something you look at, most apps made it
/// something you're scored on, and this table is calm, with a friend at your
/// side. Every beat is one move of that sentence, and the board does the acting,
/// so "See the game" at the end pays it off.
///
/// It opens calm: the table laying itself, piece by piece. Then, briefly, the
/// reason it exists: the noise chess usually comes with (a clock, a rating,
/// a verdict) drifts over the board and fades, and the quiet comes back for
/// good. Then you meet the coach, before anything is asked of you. Then it's
/// yours: pick up a piece and hold it as long as you like while the coach
/// says what it sees on each square, and put it down. It ends on the promise
/// the app keeps, and you ask for it yourself: see the game, one calm move at
/// a time.
///
/// Nothing moves on by itself. Every beat waits for you, by the Next button
/// or a tap anywhere, so this is a thing you go through rather than a
/// slideshow being presented at you. The one exception is the table laying
/// itself at the start, which is the screen arriving, not the story running.
struct WelcomeView: View {
    /// Done: fade away and leave home, board and all, in place.
    let enter: () -> Void
    /// Done, but new to chess: learn the moves first.
    let learn: () -> Void
    /// Fading away (Skip, or off to Learn).
    @State private var fading = false

    enum Stage: Int { case calm, noise, hush, coach, pick, verdict, settle }
    @State private var stage = Stage.calm
    @State private var board = WelcomeModel()
    /// On its way out: the whole screen falls, like a curtain dropping, and
    /// home is behind it.
    @State private var leaving = false
    /// How far it has to fall to be gone. Measured, with room for the edges.
    @State private var drop: CGFloat = 1200
    /// When a beat was last asked for, so a single tap can never count twice.
    /// Short enough that tapping quickly through still works.
    @State private var asked = Date.distantPast

    var body: some View {
        ZStack(alignment: .top) {
            AppBackground()
            VStack(spacing: 0) {
                // Room for Skip, which lives above the tap-to-go-on layer.
                Color.clear.frame(height: 44)

                // One line at a time, in a space of its own height so the board
                // never shifts. The old line leaves before the new one arrives.
                ZStack(alignment: .topLeading) {
                    // Show, don't tell: a few words, and the board does the rest.
                    VStack(alignment: .leading, spacing: 6) {
                        if let title {
                            Text(title).font(Typeface.display(30)).tracking(-0.5)
                                .lineLimit(1).minimumScaleFactor(0.7)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .id(lines)
                    .transition(.asymmetric(
                        insertion: .opacity.combined(with: .offset(y: 8)).animation(.easeOut(duration: 0.5).delay(0.2)),
                        removal: .opacity.animation(.easeIn(duration: 0.2))))
                }
                .frame(height: 70, alignment: .topLeading)
                .padding(.top, 12)

                // The noise drifts over the board, which dims and softens under
                // it, then clears; the board itself never leaves.
                ZStack {
                    // Only dimmed: blurring the whole glass board, frame by
                    // frame, is what made this beat stutter.
                    BoardView(model: board)
                        .opacity(stage == .noise ? 0.25 : 1)
                    if stage == .noise {
                        Noise().transition(.opacity.animation(.easeInOut(duration: 0.9)))
                    }
                    if stage == .pick && board.inviting {
                        Shimmer(pass: board.invitePass).transition(.opacity)
                        TouchRipple(square: board.invitePawn, pass: board.touchPass).transition(.opacity)
                    }
                }
                .padding(.top, 24)
                .animation(.easeInOut(duration: 1.0), value: stage == .noise)

                // Under the board: whoever is speaking, then the way on.
                ZStack(alignment: .top) {
                    VStack(spacing: 10) {
                        if stage == .coach || stage == .pick || stage == .verdict, let card = board.coach {
                            CoachView(card: card, opponent: .easy, compact: true)
                                .transition(.asymmetric(insertion: .opacity, removal: .opacity.animation(.easeIn(duration: 0.15))))
                        }
                        // The way on, given a shape so it reads as something
                        // to press rather than a line of grey text. It sits
                        // bottom right, under Skip, so the two corners are the
                        // two ways out and neither competes with the board.
                        if advancing {
                            HStack {
                                Spacer()
                                Button { goOn() } label: {
                                    HStack(spacing: 5) {
                                        Text("Next")
                                        Image(systemName: "chevron.right")
                                            .font(.system(size: 12, weight: .bold))
                                            .symbolEffect(.wiggle.right, options: .speed(1.6), value: stage)
                                    }
                                    .font(.system(size: 15, weight: .semibold))
                                    .foregroundStyle(Palette.ink)
                                    .padding(.horizontal, 18)
                                    .padding(.vertical, 11)
                                    .contentShape(.capsule)
                                }
                                .buttonStyle(NextButtonStyle())
                            }
                            .frame(minHeight: 44)
                            .transition(.opacity)
                        }
                    }
                    if stage == .settle {
                        VStack(spacing: 18) {
                            // "See the game." above, "Show me" here: the line
                            // and the answer to it, rather than the same
                            // sentence twice. It sits where home's own button
                            // sits, so the curtain drops onto a match.
                            Button { begin() } label: { Text("Show me") }
                                .buttonStyle(PrimaryButtonStyle())
                            Button("New to chess? Learn the moves") { Haptics.tap(); finish(then: learn) }
                                .font(.system(size: 15, weight: .semibold)).foregroundStyle(Palette.mist)
                                .frame(minHeight: 44)
                                .buttonStyle(.plain)
                        }
                        .transition(.opacity.combined(with: .offset(y: 12)).animation(.easeOut(duration: 0.5).delay(0.25)))
                    }
                }
                .padding(.top, 20)
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 16)
            .frame(maxWidth: 560)
            .foregroundStyle(Palette.ink)

            // Skip stays above it, so the way out is never swallowed.
            if stage != .settle {
                HStack {
                    Spacer()
                    Button("Skip") { Haptics.tap(); finish() }
                        .font(.system(size: 15, weight: .semibold)).foregroundStyle(Palette.mist)
                        .frame(minWidth: 44, minHeight: 44)
                        .buttonStyle(.plain)
                }
                .frame(maxWidth: 560)
                .padding(.horizontal, 20)
                .transition(.opacity)
            }
        }
        .background {
            GeometryReader { g in
                Color.clear.onAppear { drop = g.size.height + 120 }
            }
        }
        // The curtain: it falls away, gathering speed, and home is behind it.
        .offset(y: leaving ? drop : 0)
        .opacity(fading ? 0 : 1)
        .animation(.spring(response: 0.5, dampingFraction: 0.85), value: stage)
        .task { await tell() }
    }

    /// Which lines are showing: the verdict keeps the pick-up lines.
    private var lines: Int { stage == .verdict ? Stage.pick.rawValue : stage.rawValue }

    /// The beats you step through. The board itself carries the pick beat, and
    /// the ending has a button of its own.
    private var advancing: Bool { stage != .pick && stage != .settle }


    /// The narrator's line, until the coach takes over. From the moment you
    /// look at the card, a heading above it is a second voice saying the same
    /// sort of thing, so there isn't one: the coach has you until the end.
    private var title: String? {
        switch stage {
        case .calm: L("Take a seat.")
        case .noise: L("Most chess is loud.")
        case .hush: L("This table is calm.")
        case .coach: L("A friend at your side.")
        case .pick, .verdict: nil
        case .settle: L("See the game.")
        }
    }

    /// The story's opening: the calm table already laid, the noise it
    /// leaves out, then the quiet again, and the board is handed over. It
    /// picks up from wherever it is, so if the view is shown again (as a cover
    /// can be while it settles) the story carries on instead of rushing.
    private func tell() async {
        board.onMove = { moved() }
        if stage == .calm { await board.setUp() }
    }

    /// On you go. Every beat waits here until you ask for the next one, by the
    /// button or by a tap anywhere: a screen that moves by itself is a
    /// slideshow being presented at you, however calm the slides are.
    private func goOn() {
        let now = Date()
        guard now.timeIntervalSince(asked) > 0.2 else { return }
        asked = now
        Haptics.tap(.light, intensity: 0.6)
        switch stage {
        case .calm:
            board.showAll()
            withAnimation { stage = .noise }
        case .noise:
            withAnimation { stage = .hush }
            windDown()
        case .hush:
            withAnimation { stage = .coach }
            board.greet()
        case .coach:
            withAnimation { stage = .pick }
            board.open = true
            board.hint()
            Task { await board.invite() }
        case .verdict:
            settle()
        case .pick, .settle:
            break
        }
    }

    /// The clock you felt in the loud beat doesn't stop dead: a few last
    /// taps, slower and softer each time, and then one slow breath.
    private func windDown() {
        Task {
            for (gap, level) in [(0.5, 0.25), (0.6, 0.17), (0.75, 0.1)] {
                try? await Task.sleep(for: .seconds(gap))
                guard stage == .hush else { return }
                Haptics.tap(.rigid, intensity: level)
            }
            try? await Task.sleep(for: .seconds(0.5))
            guard stage == .hush else { return }
            Haptics.breath()
        }
    }

    /// The piece is down and the coach answers it. Read it for as long as you
    /// like; the last beat comes when you ask for it.
    private func moved() {
        withAnimation { stage = .verdict }
    }

    /// The last beat: the pawn glides quietly back, so the board is the one
    /// home is about to show, under the button home itself shows.
    private func settle() {
        guard stage == .verdict else { return }
        withAnimation { stage = .settle }
        board.takeBack()
    }

    /// Asked for: the whole screen drops away and home is behind it. It is
    /// eased in, not out, so it falls the way a curtain does, gathering speed
    /// rather than coasting to a stop.
    private func begin() {
        Haptics.tap(.medium, intensity: 0.8)
        Sound.shared.play(.great)
        UserDefaults.standard.set(true, forKey: "welcome.seen")
        withAnimation(.easeIn(duration: 0.5)) { leaving = true }
        Task {
            try? await Task.sleep(for: .seconds(0.5))
            enter()
        }
    }

    private func finish(then next: (() -> Void)? = nil) {
        UserDefaults.standard.set(true, forKey: "welcome.seen")
        withAnimation(.easeOut(duration: 0.3)) { fading = true }
        Task {
            try? await Task.sleep(for: .seconds(0.3))
            enter()
            next?()
        }
    }
}


/// The noise the app leaves out: a clock running down, a rating slipping, a
/// harsh verdict, drifting half out of focus. Shown once, then gone.
private struct Noise: View {
    @State private var seconds = 12

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width, h = geo.size.height
            ZStack {
                fragment(String(format: "0:%02d", seconds), size: 46, mono: true, color: Palette.danger, blur: 0.6)
                    .contentTransition(.numericText(countsDown: true))
                    .position(x: w * 0.3, y: h * 0.3)
                fragment("1452  ▾ −12", size: 24, mono: true, color: Palette.mist, blur: 1.2)
                    .position(x: w * 0.68, y: h * 0.52)
                fragment("?? " + L("Blunder"), size: 22, color: Palette.danger, blur: 1.8)
                    .phaseAnimator([0.45, 1.0]) { v, o in v.opacity(o) } animation: { _ in
                        .easeInOut(duration: 1.1)
                    }
                    .position(x: w * 0.36, y: h * 0.72)
                fragment(L("Rated · 3 min"), size: 16, color: Palette.mist, blur: 2.4)
                    .position(x: w * 0.74, y: h * 0.2)
                fragment(L("Time's up"), size: 18, color: Palette.mist, blur: 2.8)
                    .position(x: w * 0.7, y: h * 0.86)
            }
            // This beat waits for you like the others, so the noise must never
            // settle while you sit on it: a frozen clock is just a picture.
            .phaseAnimator([6.0, -10.0]) { v, y in v.offset(y: y) } animation: { _ in
                .easeInOut(duration: 3)
            }
            // Drawn as one layer, so the blurred text isn't recomputed per frame.
            .drawingGroup()
        }
        .aspectRatio(1, contentMode: .fit)
        .allowsHitTesting(false)
        .accessibilityHidden(true)
        .task {
            // Round and round: it runs out, and another one starts.
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(0.45))
                if Task.isCancelled { break }
                withAnimation(.snappy(duration: 0.25)) { seconds = seconds > 0 ? seconds - 1 : 12 }
                // The clock you can feel: a crisp, light tap each second.
                Haptics.tap(.rigid, intensity: 0.35)
            }
        }
    }

    private func fragment(_ text: String, size: CGFloat, mono: Bool = false, color: Color, blur: CGFloat) -> some View {
        Text(text)
            .font(.system(size: size, weight: .bold, design: mono ? .monospaced : .rounded))
            .monospacedDigit()
            .foregroundStyle(color.opacity(0.75))
            .blur(radius: blur)
    }
}

/// Next: a soft capsule that breathes slowly, its fill and shadow rising and
/// falling, so it reads as the thing to press without calling out. It presses
/// in a little and darkens under the finger.
private struct NextButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        let pressed = configuration.isPressed
        configuration.label
            .background {
                // A raised capsule: an opaque surface whose shadow lifts and
                // settles as it breathes, with a faint tint that deepens with it.
                Capsule().fill(Palette.surface)
                    .overlay(Capsule().strokeBorder(Palette.hairline, lineWidth: 1))
                    .phaseAnimator([0.0, 1.0]) { cap, t in
                        cap.overlay(Capsule().fill(Palette.ink.opacity(pressed ? 0.14 : 0.03 + t * 0.07)))
                            .shadow(color: .black.opacity(pressed ? 0.08 : 0.10 + t * 0.14),
                                    radius: pressed ? 3 : 5 + t * 9, y: pressed ? 1 : 2 + t * 5)
                    } animation: { _ in .easeInOut(duration: 1.6) }
            }
            .scaleEffect(pressed ? 0.95 : 1)
            .animation(.spring(response: 0.25, dampingFraction: 0.6), value: pressed)
    }
}

/// The invitation to touch: a soft band of light sweeping across your two
/// ranks, as if the pieces were waking up. One pass per `pass`.
private struct Shimmer: View {
    let pass: Int
    @State private var x: CGFloat = -0.3

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width, q = w / 8
            LinearGradient(colors: [.clear, .white.opacity(0.8), .clear], startPoint: .leading, endPoint: .trailing)
                .frame(width: q * 2.4, height: q * 2)
                .rotationEffect(.degrees(12))
                .position(x: w * x, y: q * 7)
                .blendMode(.softLight)
                // Only over your pieces.
                .mask(Rectangle().frame(height: q * 2).frame(maxHeight: .infinity, alignment: .bottom))
        }
        .aspectRatio(1, contentMode: .fit)
        .allowsHitTesting(false)
        .accessibilityHidden(true)
        .onChange(of: pass, initial: true) {
            x = -0.3
            withAnimation(.easeInOut(duration: 1.3)) { x = 1.3 }
        }
    }
}

/// A touch, shown rather than drawn as a hand: a soft ring of light that
/// presses in on a square and spreads out, the way a fingertip would.
private struct TouchRipple: View {
    let square: Square
    let pass: Int
    @State private var spread = false
    @State private var shown = false

    var body: some View {
        GeometryReader { geo in
            let q = geo.size.width / 8
            ZStack {
                Circle().fill(Color.white.opacity(0.35))
                    .frame(width: q * 0.5, height: q * 0.5)
                    .scaleEffect(spread ? 1 : 0.6)
                Circle().strokeBorder(Palette.accent, lineWidth: 2.5)
                    .frame(width: q * 0.9, height: q * 0.9)
                    .scaleEffect(spread ? 1.6 : 0.7)
            }
            .opacity(shown ? (spread ? 0 : 1) : 0)
            .position(center(square, geo.size.width))
        }
        .aspectRatio(1, contentMode: .fit)
        .allowsHitTesting(false)
        .accessibilityHidden(true)
        .onChange(of: pass) {
            spread = false
            shown = true
            withAnimation(.easeOut(duration: 0.9)) { spread = true }
        }
    }
}

/// The welcome board: a real game at the start, where you play one move for
/// White, which then glides quietly back. It speaks the board's language, so
/// it is the very board you'll play on. Its coach reads every square you
/// could go to as soon as the board is set, so while you hold a piece it can
/// tell you, live, what it thinks of the square under your finger.
@Observable
final class WelcomeModel: BoardSource {
    private(set) var shownPieces: [PieceItem] = []
    let flipped = false
    private(set) var selected: Square?
    private(set) var rays: Rays?
    let bursts: [Burst] = []
    let toppled: Int? = nil
    private(set) var movingPiece: Int?
    private(set) var moveSerial = 0
    /// Whose move it is: after yours the light drifts to their side, and
    /// comes back when you take it back.
    private(set) var turn = "w"
    let reviewing: Int? = nil
    private(set) var shownLastMove: BoardMove?
    let sparkle: (id: Int, square: Square)? = nil
    let nudge: Square? = nil
    let loose: Set<Square> = []
    let checkSquare: Square? = nil
    var shownMark: (entry: TrailEntry, square: Square)? { nil }
    /// What the coach is saying right now.
    private(set) var coach: CoachCard?

    /// The board is handed over: pieces can be picked up.
    var open = false
    /// Back rank out, then pawns, then theirs: the order the table is laid in.
    /// The order a board is set by hand: every rank from the middle out (king
    /// and queen, then bishops, knights and rooks; the pawns the same way), so
    /// each row ripples outward from the centre. White, then Black.
    private static let setUpOrder: [Square] = {
        let out = [4, 3, 5, 2, 6, 1, 7, 0] // e d f c g b h a
        let white = out.map { 56 + $0 } + out.map { 48 + $0 }
        let black = out.map { $0 } + out.map { 8 + $0 }
        return white + black
    }()
    /// Set down by hand, not faded in.
    let placesPieces = true
    /// You moved on, so the slow laying stands down.
    @ObservationIgnored private var hurried = false
    /// Still waiting for a first touch: the board invites one.
    private(set) var inviting = false
    /// Bumps for each sweep of light across your pieces.
    private(set) var invitePass = 0
    /// Bumps for each shown touch on the inviting pawn.
    private(set) var touchPass = 0
    /// The pawn that shows how a touch works.
    let invitePawn: Square = 52 // e2
    @ObservationIgnored private var touched = false
    /// The move has been played.
    @ObservationIgnored var onMove: (() -> Void)?
    @ObservationIgnored private let game = ChesscoreNewGame()!
    @ObservationIgnored private var legal: [String] = []
    @ObservationIgnored private var played: (id: Int, from: Square, to: Square)?
    /// The coach's read of every first move, worked out while you look.
    @ObservationIgnored private var reads: [String: (kind: CoachCard.Kind, text: String)] = [:]
    @ObservationIgnored private var hovered: Square?
    @ObservationIgnored private var serial = 0

    init() {
        legal = game.legalMoves().split(separator: " ").map(String.init)
        let fen = game.fen(), moves = legal
        Task { [weak self] in
            let out = await Task.detached(priority: .userInitiated) {
                var out: [String: (kind: CoachCard.Kind, text: String)] = [:]
                for m in moves {
                    guard let v = ChesscoreNewGameFromFEN(fen)?.assess(m) else { continue }
                    out[m] = (CoachCard.Kind(rawValue: v.kind) ?? .good, EngineText.localize(v.text))
                }
                return out
            }.value
            self?.reads = out
        }
    }

    /// Sets the pieces on the board one by one: back rank, then pawns, White
    /// then Black, the way you'd set up a real board.
    /// The table laying itself, piece by piece. The one thing here that moves
    /// on its own, and it is the screen arriving rather than the story running.
    func setUp() async {
        let board = Array(game.board())
        // Wait for the screen to finish arriving: pieces set on a board that
        // is still sliding into place look like they fly in from the side.
        if shownPieces.isEmpty { try? await Task.sleep(for: .seconds(0.7)) }
        for (k, sq) in Self.setUpOrder.enumerated() where board[sq] != "." && !shownPieces.contains(where: { $0.id == sq }) {
            if Task.isCancelled || hurried { return }
            // Each piece comes down onto its square and settles.
            withAnimation(.spring(response: 0.42, dampingFraction: 0.62)) {
                shownPieces.append(PieceItem(id: sq, code: board[sq], square: sq))
            }
            // A light tap as each piece is set down: a little firmer for the
            // back rank, softer for the pawns.
            Haptics.tap(.soft, intensity: k % 16 < 8 ? 0.5 : 0.3)
            // The back rank is placed with care, the pawns go down quicker.
            try? await Task.sleep(for: .milliseconds(k % 16 < 8 ? 70 : 38))
        }
    }

    /// You moved on while it was still laying: the rest arrives at once.
    func showAll() {
        hurried = true
        let board = Array(game.board())
        withAnimation(.easeOut(duration: 0.25)) {
            for sq in Self.setUpOrder where board[sq] != "." && !shownPieces.contains(where: { $0.id == sq }) {
                shownPieces.append(PieceItem(id: sq, code: board[sq], square: sq))
            }
        }
    }

    /// Its first word, said before anything is asked of you: it is here to
    /// think along, and it keeps no score. "Second" is the chess word for the
    /// person who sits with a player and analyses, so it names the role without
    /// any classroom in it, and the line below defines it by use.
    func greet() {
        guard coach == nil else { return }
        say(.coach, L("Your second"), L("I'll tell you what I see."))
    }

    /// The board is yours: the one thing to do, and who is waiting while you
    /// do it. No rule about the piece, which would make this a lesson, but not
    /// a bare instruction either: said in the first person, so the someone of
    /// "Someone looks with you" is still there the moment you start playing.
    /// It stands until you touch anything, and then the next word is about
    /// where the piece you picked up can go.
    func hint() {
        guard !touched else { return }
        say(.coach, L("Pick up any piece"), L("I'll follow along."))
    }

    /// Until the first touch: a sweep of light across your pieces, then the
    /// king's pawn lifts and settles, as a piece does when picked up. Every
    /// few seconds, gently, and never again once you've touched the board.
    /// The whole gesture in miniature, until the first touch: light sweeps
    /// your pieces, a touch ripples on the king's pawn, it lifts and its
    /// lines draw out to where it can go, then it all settles. Every few
    /// seconds, and never again once you've touched the board.
    func invite() async {
        inviting = true
        let pawn = invitePawn
        try? await Task.sleep(for: .seconds(0.6))
        while !touched, !Task.isCancelled {
            invitePass += 1
            try? await Task.sleep(for: .seconds(0.9))
            guard !touched else { break }
            touchPass += 1
            try? await Task.sleep(for: .seconds(0.15))
            guard !touched else { break }
            selected = pawn
            let moves = targets(pawn)
            serial += 1
            rays = Rays(id: serial, kind: .options, moves: moves, knight: [])
            try? await Task.sleep(for: .seconds(1.3))
            if !touched {
                selected = nil
                rays = nil
            }
            try? await Task.sleep(for: .seconds(2.2))
        }
        withAnimation(.easeOut(duration: 0.3)) { inviting = false }
    }

    private func targets(_ from: Square) -> [BoardMove] {
        let name = Squares.name(from)
        var seen = Set<Square>()
        return legal.filter { $0.hasPrefix(name) }
            .compactMap { m in Squares.parse(m.dropFirst(2).prefix(2)).map { BoardMove(from: from, to: $0) } }
            .filter { seen.insert($0.to).inserted }
    }

    private var canPlay: Bool { open && played == nil }

    private func say(_ kind: CoachCard.Kind, _ title: String, _ text: String, keep: Bool = true) {
        // One id while you're thinking, so the card's words change in place
        // instead of popping a new card for every square.
        if !keep { serial += 1 }
        coach = CoachCard(id: serial, kind: kind, badge: kind.verdict?.badge ?? "✦", title: title, text: text)
    }

    private func select(_ square: Square) {
        touched = true
        // The same cue the real board gives, because this is the real board.
        if selected != square { Sound.shared.play(.select) }
        selected = square
        hovered = nil
        let moves = targets(square)
        serial += 1
        let code = shownPieces.first { $0.square == square }?.code ?? "P"
        rays = Rays(id: serial, kind: .options, moves: moves, knight: code == "N" ? Set(moves) : [])
        // The lines already show where it can go, so the words are only the
        // two ways of saying it. Naming the piece would teach; this doesn't.
        say(.coach, L("Tap a square, or drag"), "", keep: false)
    }

    /// The coach, live, on the square under the carried piece.
    func hover(_ square: Square?) {
        guard canPlay, let from = selected, square != hovered else { return }
        hovered = square
        guard let square, let uci = legal.first(where: { $0.hasPrefix(Squares.name(from) + Squares.name(square)) }) else {
            if square == nil || square == from { select(from) }
            return
        }
        guard let read = reads[uci] else {
            say(.coach, L("Let me look…"), "")
            return
        }
        let title: String = switch read.kind {
        case .great: L("Ooh, a strong square")
        case .good: L("A good square")
        case .inaccuracy: L("Hmm, maybe")
        default: L("Careful there")
        }
        say(read.kind == .great || read.kind == .good ? read.kind : .coach, title,
            read.text.isEmpty ? L("Nothing wrong with it. Put it down if you like it.") : read.text)
    }

    private func play(_ from: Square, _ to: Square) {
        guard let i = shownPieces.firstIndex(where: { $0.square == from }) else { return }
        let uci = Squares.name(from) + Squares.name(to)
        guard legal.contains(uci) else { return }
        let read = reads[uci]
        _ = game.play(uci)
        played = (shownPieces[i].id, from, to)
        selected = nil
        hovered = nil
        rays = nil
        movingPiece = shownPieces[i].id
        shownPieces[i].square = to
        shownLastMove = BoardMove(from: from, to: to)
        turn = "b"
        moveSerial += 1
        Sound.shared.play(.move)
        let kind = read?.kind ?? .good
        let words = kind.verdict ?? (L("Good move"), "✓")
        say(kind, words.title, (read?.text).flatMap { $0.isEmpty ? nil : $0 } ?? L("A calm, steady start."), keep: false)
        Sound.shared.play(kind == .great ? .great : .good)
        onMove?()
    }

    /// The move glides back, leaving the board as home keeps it.
    func takeBack() {
        guard let p = played, let i = shownPieces.firstIndex(where: { $0.id == p.id }) else { return }
        Sound.shared.play(.move)
        _ = game.undo()
        movingPiece = p.id
        shownPieces[i].square = p.from
        shownLastMove = nil
        turn = "w"
        moveSerial += 1
    }

    func tap(_ square: Square) {
        guard canPlay else { return }
        // Before the first touch, a lifted pawn is only the board's demo:
        // the first tap picks up what you tapped, never puts the demo down.
        if !touched { selected = nil; rays = nil }
        touched = true
        if let selected, rays?.moves.contains(where: { $0.to == square }) == true {
            play(selected, square)
        } else if selected != square, shownPieces.contains(where: { $0.square == square && $0.code.isUppercase }),
                  !targets(square).isEmpty {
            select(square)
        } else {
            // Put back down: the quieter half of picking it up.
            if selected != nil {
                Sound.shared.play(.putDown)
                Haptics.tap(.light, intensity: 0.5)
            }
            selected = nil
            rays = nil
        }
    }

    func pickUp(_ square: Square) -> Bool {
        if !touched { selected = nil; rays = nil }
        touched = true
        guard canPlay, shownPieces.contains(where: { $0.square == square && $0.code.isUppercase }),
              !targets(square).isEmpty else { return false }
        if selected != square { select(square) }
        return true
    }

    func drop(on square: Square?) {
        guard canPlay, let selected else { return }
        if let square, rays?.moves.contains(where: { $0.to == square }) == true {
            play(selected, square)
        } else {
            select(selected)
        }
    }

    /// Nothing here is held down.
    func inspect(_ square: Square) -> Bool { false }

    func endInspect() {}
    func revealMark() {}
}
