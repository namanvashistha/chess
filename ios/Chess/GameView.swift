import Chesscore
import SwiftUI

/// The app: home, and the game it starts or resumes.
struct RootView: View {
    @State private var model = GameModel()
    @State private var playing = false
    @State private var homeID = 0 // refreshes home's saved-game and history views on return

    var body: some View {
        ZStack {
            AppBackground()
            if playing {
                GameScreen(model: model) {
                    homeID += 1
                    withAnimation(.smooth(duration: 0.4)) { playing = false }
                }
                .transition(.move(edge: .trailing).combined(with: .opacity))
            } else {
                HomeScreen(
                    start: { mode, opponent in
                        model.start(mode, opponent: opponent)
                        withAnimation(.smooth(duration: 0.4)) { playing = true }
                    },
                    resume: {
                        if model.resume() { withAnimation(.smooth(duration: 0.4)) { playing = true } }
                    }
                )
                .id(homeID)
                .transition(.move(edge: .leading).combined(with: .opacity))
            }
        }
        .sensoryFeedback(trigger: model.feedback) { _, new in Sound.hapticsOn ? new?.kind.haptic : nil }
    }
}

private extension Feedback.Kind {
    var haptic: SensoryFeedback? {
        switch self {
        case .select: .selection
        case .move: .impact(weight: .light)
        case .capture: .impact(weight: .medium)
        case .check: .warning
        case .great, .win: .success
        case .good: nil
        case .bad, .lose: .impact(flexibility: .soft, intensity: 0.7)
        }
    }
}

// MARK: - Home

private struct HomeScreen: View {
    let start: (Mode, Opponent) -> Void
    let resume: () -> Void
    @AppStorage("opponent") private var opponentRaw = Opponent.flint.rawValue
    @State private var settings = false
    private let saved = SavedGame.load()
    private let recent = Stats.recent

    private var opponent: Opponent { Opponent(rawValue: opponentRaw) ?? .flint }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // One quiet bar, one headline, one supporting line.
                HStack(spacing: 8) {
                    BrandMark()
                    Text("Cozy Chess").font(Typeface.display(16, .bold))
                    Spacer()
                    Button { settings = true } label: {
                        Image(systemName: "gearshape").font(.system(size: 15, weight: .medium)).frame(width: 30, height: 30)
                    }
                    .buttonStyle(.glass)
                    .buttonBorderShape(.circle)
                    .accessibilityLabel("Settings")
                }
                .padding(.top, 4)

                VStack(alignment: .leading, spacing: 6) {
                    Text(greeting)
                        .font(Typeface.display(34))
                        .tracking(-0.6)
                    Text(supportingLine)
                        .font(.system(size: 15))
                        .foregroundStyle(Palette.mist)
                }
                .padding(.top, 10)
                .padding(.bottom, 4)

                if let saved {
                    ContinueCard(saved: saved, action: resume)
                }

                VStack(alignment: .leading, spacing: 8) {
                    SectionLabel(text: saved == nil ? "Choose your opponent" : "Or start a new game")
                    HStack(spacing: 8) {
                        ForEach(Opponent.allCases) { o in
                            OpponentButton(opponent: o, selected: o == opponent) {
                                opponentRaw = o.rawValue
                                Sound.shared.play(.select)
                            }
                        }
                    }
                }

                Button { start(.engine, opponent) } label: {
                    Label("Play \(opponent.name)", systemImage: "arrow.right").labelStyle(TrailingIcon())
                }
                .buttonStyle(PrimaryButtonStyle())

                Button { start(.passAndPlay, opponent) } label: {
                    Label("Pass & play on this phone", systemImage: "person.2").frame(maxWidth: .infinity, minHeight: 40)
                }
                .font(Typeface.display(15, .bold))
                .buttonStyle(.glass)
                .controlSize(.large)

                RecentGames(records: recent)
                    .padding(.top, 6)
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 24)
        }
        .scrollBounceBehavior(.basedOnSize)
        .foregroundStyle(Palette.ink)
        .sheet(isPresented: $settings) { SettingsSheet().presentationDetents([.height(260)]) }
    }

    /// The tagline for a new player; their numbers once they have some.
    private var supportingLine: String {
        var parts: [String] = []
        if Stats.streak > 1 { parts.append("\(Stats.streak)-day streak") }
        if let a = Stats.averageAccuracy { parts.append("\(a)% accuracy") }
        if saved != nil && parts.isEmpty { return "Your game is where you left it." }
        return parts.isEmpty ? "See the game, one calm move at a time." : parts.joined(separator: "  ·  ")
    }

    private var greeting: String {
        switch Calendar.current.component(.hour, from: .now) {
        case 5..<12: "Good morning."
        case 12..<17: "Good afternoon."
        default: "Good evening."
        }
    }
}

private struct SectionLabel: View {
    let text: String
    var body: some View {
        Text(text.uppercased())
            .font(.system(size: 11, weight: .bold)).tracking(1.2)
            .foregroundStyle(Palette.mist)
    }
}

private struct TrailingIcon: LabelStyle {
    func makeBody(configuration: Configuration) -> some View {
        HStack(spacing: 10) { configuration.title; configuration.icon }
    }
}

/// The app's mark: the glass knight on a corner of the stone board.
private struct BrandMark: View {
    var body: some View {
        ZStack {
            VStack(spacing: 0) {
                HStack(spacing: 0) { Palette.squareLight; Palette.squareDark }
                HStack(spacing: 0) { Palette.squareDark; Palette.squareLight }
            }
            GlassPiece(code: "N").padding(3)
        }
        .frame(width: 26, height: 26)
        .clipShape(.rect(cornerRadius: 7, style: .continuous))
        .accessibilityHidden(true)
    }
}

/// The saved game, shown as its position with a way back in.
private struct ContinueCard: View {
    let saved: SavedGame
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 14) {
                StaticBoard(fen: String(saved.fen.split(separator: " ").first ?? ""))
                    .frame(width: 104, height: 104)
                    .clipShape(.rect(cornerRadius: 10))
                VStack(alignment: .leading, spacing: 4) {
                    Text("Continue").font(.system(size: 12, weight: .bold)).tracking(1).textCase(.uppercase)
                        .foregroundStyle(Palette.accentText)
                    Text(saved.passAndPlay ? "Pass & play" : "vs \(Opponent(rawValue: saved.opponent)?.name ?? "Flint")")
                        .font(Typeface.display(21))
                    Text("Move \(saved.moves.count / 2 + 1)" + (saved.trail.isEmpty ? "" : " · \(saved.trail.count) rated"))
                        .font(.system(size: 13.5)).foregroundStyle(Palette.mist)
                    TrailDots(trail: Array(saved.trail.suffix(12)))
                        .padding(.top, 2)
                }
                Spacer(minLength: 0)
                Image(systemName: "chevron.right").foregroundStyle(Palette.mist)
            }
            .padding(12)
            .contentShape(.rect)
        }
        .buttonStyle(.plain)
        .glassEffect(.regular.interactive(), in: .rect(cornerRadius: 22))
    }
}

/// A verdict's colour, shared by the trail and the coach card.
func verdictColor(_ k: CoachCard.Kind) -> Color {
    switch k {
    case .great: Palette.great
    case .good: Palette.mist
    case .inaccuracy: Palette.inaccuracy
    case .mistake, .warning: Palette.mistake
    case .blunder: Palette.blunder
    case .coach, .opponent: Palette.ink
    }
}

private struct TrailDots: View {
    let trail: [TrailEntry]
    var body: some View {
        HStack(spacing: 4) {
            ForEach(trail) { e in Circle().fill(verdictColor(e.card.kind)).frame(width: 7, height: 7) }
        }
    }
}

private struct OpponentButton: View {
    let opponent: Opponent
    let selected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 6) {
                OpponentAvatar(opponent: opponent, size: 34)
                Text(opponent.name).font(Typeface.display(15, .bold))
                Text(opponent.subtitle).font(.system(size: 11.5)).foregroundStyle(Palette.mist)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 12)
            .contentShape(.rect)
        }
        .buttonStyle(.plain)
        .glassEffect(selected ? .regular.tint(Palette.accent.opacity(0.25)).interactive() : .regular.interactive(),
                     in: .rect(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).strokeBorder(Palette.accent, lineWidth: selected ? 1.5 : 0)
        }
        .animation(.easeOut(duration: 0.15), value: selected)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }
}

/// Each opponent is a stone: a glass pebble, a cut flint, an obsidian star.
struct OpponentAvatar: View {
    let opponent: Opponent
    let size: CGFloat

    var body: some View {
        // real stone colours: a pale river pebble, a tan flint, black obsidian
        let tint: Color = [Color.dynamic(light: 0x8FA39A, dark: 0xA9BDB3),
                           Color.dynamic(light: 0xA2855C, dark: 0xC9AE85),
                           Color.dynamic(light: 0x3A3F4A, dark: 0xA9B2C3)][opponent.rawValue]
        let symbol = ["circle.fill", "diamond.fill", "star.fill"][opponent.rawValue]
        Image(systemName: symbol)
            .font(.system(size: size * 0.5))
            .foregroundStyle(LinearGradient(colors: [tint.opacity(0.75), tint], startPoint: .topLeading, endPoint: .bottomTrailing))
            .frame(width: size, height: size)
            .glassEffect(.regular.tint(tint.opacity(0.18)), in: .circle)
            .accessibilityHidden(true)
    }
}

private struct RecentGames: View {
    let records: [GameRecord]

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionLabel(text: "Recent games")
            if records.isEmpty {
                Text("Your finished games will show up here, with how accurately you played.")
                    .font(.system(size: 14)).foregroundStyle(Palette.mist)
                    .padding(.vertical, 6)
            } else {
                VStack(spacing: 0) {
                    ForEach(Array(records.enumerated()), id: \.element.id) { k, r in
                        HStack(spacing: 12) {
                            Text(r.outcome == "won" ? "♔" : r.outcome == "lost" ? "♚" : "½")
                                .font(Typeface.display(20))
                                .frame(width: 32)
                            VStack(alignment: .leading, spacing: 1) {
                                Text(r.outcome == "won" ? "Won vs \(r.opponent)" : r.outcome == "lost" ? "Lost to \(r.opponent)" : "Drew with \(r.opponent)")
                                    .font(.system(size: 15, weight: .semibold))
                                Text("\(r.moves) moves · \(r.date.formatted(.relative(presentation: .named)))")
                                    .font(.system(size: 12.5)).foregroundStyle(Palette.mist)
                            }
                            Spacer()
                            Text(r.accuracy.map { "\($0)%" } ?? "–").font(Typeface.display(17)).monospacedDigit()
                        }
                        .padding(.vertical, 10)
                        .padding(.horizontal, 12)
                        if k < records.count - 1 { Divider().padding(.leading, 56) }
                    }
                }
                .glassEffect(.regular, in: .rect(cornerRadius: 18))
            }
        }
    }
}

private struct SettingsSheet: View {
    @AppStorage("sound") private var sound = true
    @AppStorage("haptics") private var haptics = true

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("Settings").font(Typeface.display(22))
            Toggle("Sound", isOn: $sound)
            Toggle("Haptics", isOn: $haptics)
            Text("Light and dark follow your iPhone's appearance.")
                .font(.system(size: 13)).foregroundStyle(Palette.mist)
        }
        .font(.system(size: 16, weight: .medium))
        .tint(Palette.accent)
        .padding(24)
        .frame(maxHeight: .infinity, alignment: .top)
    }
}

/// A board drawn from a FEN placement, without a game behind it.
struct StaticBoard: View {
    let fen: String

    var body: some View {
        GeometryReader { geo in
            let q = geo.size.width / 8
            let board = Self.expand(fen)
            ZStack(alignment: .topLeading) {
                VStack(spacing: 0) {
                    ForEach(0..<8, id: \.self) { r in
                        HStack(spacing: 0) {
                            ForEach(0..<8, id: \.self) { c in
                                Rectangle().fill((r + c) % 2 == 1 ? Palette.squareDark : Palette.squareLight)
                                    .frame(width: q, height: q)
                            }
                        }
                    }
                }
                ForEach(0..<min(64, board.count), id: \.self) { i in
                    if board[i] != "." {
                        GlassPiece(code: board[i])
                            .frame(width: q, height: q)
                            .position(center(i, geo.size.width))
                            .zIndex(Double(i / 8))
                    }
                }
            }
        }
        .aspectRatio(1, contentMode: .fit)
        .accessibilityHidden(true)
    }

    static func expand(_ placement: String) -> [Character] {
        var out: [Character] = []
        for ch in placement where ch != "/" {
            if let n = ch.wholeNumberValue { out += Array(repeating: ".", count: n) } else { out.append(ch) }
        }
        return out
    }
}

// MARK: - Game

private struct GameScreen: View {
    let model: GameModel
    let back: () -> Void

    var body: some View {
        ZStack {
            VStack(spacing: 6) {
                TopBar(model: model, back: back)
                CapturedRow(pieces: model.captured(by: "b"), lead: model.materialLead(for: "b"))
                BoardView(model: model)
                CapturedRow(pieces: model.captured(by: "w"), lead: model.materialLead(for: "w"))
                CoachView(card: model.coach, opponent: model.opponent)
                    .padding(.top, 4)
                MoveTrail(model: model)
                Spacer(minLength: 0)
                Dock(model: model)
            }
            .padding(.horizontal, 14)
            .padding(.bottom, 8)
            .foregroundStyle(Palette.ink)

            if let finish = model.finish {
                FinishCard(finish: finish, again: { model.start(model.mode, opponent: model.opponent) }, home: back)
                    .transition(.opacity.combined(with: .scale(scale: 0.94)))
            }
        }
        .animation(.spring(response: 0.5, dampingFraction: 0.75), value: model.finish)
    }
}

private struct TopBar: View {
    let model: GameModel
    let back: () -> Void

    var body: some View {
        HStack(spacing: 10) {
            Button(action: back) {
                Image(systemName: "chevron.left").font(.system(size: 15, weight: .semibold)).frame(width: 30, height: 30)
            }
            .buttonStyle(.glass)
            .buttonBorderShape(.circle)
            .accessibilityLabel("Home")

            if model.mode == .engine {
                OpponentAvatar(opponent: model.opponent, size: 36)
            }
            VStack(alignment: .leading, spacing: 1) {
                Text(model.mode == .engine ? model.opponent.name : "Pass & play").font(Typeface.display(17, .bold))
                HStack(spacing: 6) {
                    if model.playerToMove { Circle().fill(Palette.great).frame(width: 6, height: 6) }
                    Text(stateText).font(.system(size: 12.5)).foregroundStyle(Palette.mist)
                    if model.thinking { ThinkingDots() }
                }
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 1) {
                Text(model.accuracy.map { "\($0)%" } ?? "–").font(Typeface.display(17)).monospacedDigit()
                    .contentTransition(.numericText())
                    .animation(.snappy, value: model.accuracy)
                Text(model.accuracy == nil ? "AFTER \(GameModel.accuracyAfter) MOVES" : "ACCURACY")
                    .font(.system(size: 9.5, weight: .semibold)).tracking(0.6).foregroundStyle(Palette.mist)
            }
            .padding(.horizontal, 12).padding(.vertical, 7)
            .glassEffect(.regular, in: .rect(cornerRadius: 14))
        }
        .padding(.top, 4)
    }

    private var stateText: String {
        if model.isOver { return "Game over" }
        if model.thinking { return "Thinking" }
        if model.mode == .passAndPlay { return model.turn == "w" ? "White to move" : "Black to move" }
        return model.turn == "w" ? "Your move" : ""
    }
}

private struct ThinkingDots: View {
    var body: some View {
        HStack(spacing: 3) {
            ForEach(0..<3) { i in
                Circle().fill(Palette.accentText).frame(width: 4, height: 4)
                    .phaseAnimator([0.2, 1]) { v, p in v.opacity(p) } animation: { _ in
                        .easeInOut(duration: 0.5).delay(Double(i) * 0.15)
                    }
            }
        }
    }
}

private struct CapturedRow: View {
    let pieces: [Character]
    let lead: Int

    var body: some View {
        HStack(spacing: -5) {
            ForEach(Array(pieces.enumerated()), id: \.offset) { _, p in
                GlassPiece(code: p).frame(width: 20, height: 20)
            }
            if lead > 0 {
                Text("+\(lead)").font(.system(size: 12, weight: .bold)).monospacedDigit()
                    .foregroundStyle(Palette.mist).padding(.leading, 12)
            }
            Spacer(minLength: 0)
        }
        .frame(height: 22)
        .padding(.horizontal, 6)
    }
}

/// The coach's card: a verdict badge, a title in the verdict's colour, and one
/// sentence. The opponent's moves are narrated here too, with its avatar.
private struct CoachView: View {
    let card: CoachCard?
    let opponent: Opponent

    var body: some View {
        ZStack {
            if let card {
                HStack(spacing: 12) {
                    badge(card)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(card.title).font(Typeface.display(17)).foregroundStyle(titleColor(card.kind))
                        Text(card.text).font(.system(size: 13.5)).foregroundStyle(Palette.mist)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                    Spacer(minLength: 0)
                }
                .padding(12)
                .glassEffect(.regular, in: .rect(cornerRadius: 20))
                .id(card.id)
                .transition(.asymmetric(insertion: .scale(scale: 0.94, anchor: .bottom).combined(with: .opacity),
                                        removal: .opacity))
                .accessibilityElement(children: .combine)
            }
        }
        .frame(minHeight: 74)
        .animation(.spring(response: 0.45, dampingFraction: 0.7), value: card?.id)
    }

    @ViewBuilder
    private func badge(_ card: CoachCard) -> some View {
        if card.kind == .opponent || card.kind == .warning {
            ZStack(alignment: .bottomTrailing) {
                OpponentAvatar(opponent: opponent, size: 46)
                Text(card.badge)
                    .font(.system(size: 11, weight: .bold)).monospacedDigit()
                    .lineLimit(1).minimumScaleFactor(0.6)
                    .padding(.horizontal, 5).padding(.vertical, 2)
                    .background(card.kind == .warning ? Palette.mistakeFill : Palette.goodFill, in: .capsule)
                    .offset(x: 6, y: 4)
            }
            .frame(width: 46, height: 46)
        } else {
            Text(card.badge)
                .font(Typeface.display(20))
                .foregroundStyle(card.kind == .coach ? Palette.onAccent : Palette.ink)
                .frame(width: 46, height: 46)
                .background(badgeFill(card.kind), in: .rect(cornerRadius: 14))
        }
    }

    private func badgeFill(_ k: CoachCard.Kind) -> AnyShapeStyle {
        switch k {
        case .great: AnyShapeStyle(Palette.greatFill)
        case .good, .opponent: AnyShapeStyle(Palette.goodFill)
        case .inaccuracy: AnyShapeStyle(Palette.inaccuracyFill)
        case .mistake, .warning: AnyShapeStyle(Palette.mistakeFill)
        case .blunder: AnyShapeStyle(Palette.blunderFill)
        case .coach: AnyShapeStyle(LinearGradient(colors: [Palette.accentTop, Palette.accent], startPoint: .top, endPoint: .bottom))
        }
    }

    private func titleColor(_ k: CoachCard.Kind) -> Color {
        switch k {
        case .good, .coach, .opponent: Palette.ink
        default: verdictColor(k)
        }
    }
}

/// Every rated move so far, as chips coloured by verdict. Tapping one shows its
/// verdict again.
private struct MoveTrail: View {
    let model: GameModel

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            if model.trail.isEmpty {
                Text("Your moves and how good they were will collect here.")
                    .font(.system(size: 12.5)).foregroundStyle(Palette.mist)
                    .padding(.horizontal, 4)
            } else {
                ScrollViewReader { proxy in
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 6) {
                            ForEach(model.trail) { e in
                                Button { model.review(e) } label: {
                                    HStack(spacing: 5) {
                                        Circle().fill(verdictColor(e.card.kind)).frame(width: 7, height: 7)
                                        Text(e.san).font(.system(size: 13, weight: .semibold)).monospacedDigit()
                                    }
                                    .padding(.horizontal, 10).padding(.vertical, 7)
                                    .contentShape(.capsule)
                                }
                                .buttonStyle(.plain)
                                .glassEffect(model.reviewing == e.id ? .regular.tint(Palette.accent.opacity(0.3)) : .regular,
                                             in: .capsule)
                                .id(e.id)
                                .accessibilityLabel("\(e.san), \(e.card.title)")
                            }
                        }
                        .padding(.vertical, 6)
                        .padding(.horizontal, 2)
                    }
                    .scrollEdgeEffectHidden(true, for: .all)
                    // let the chips' glass shadows draw past the row instead of being cut into a band
                    .scrollClipDisabled()
                    .onChange(of: model.trail.count) { _, _ in
                        if let last = model.trail.last { withAnimation { proxy.scrollTo(last.id, anchor: .trailing) } }
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.top, 2)
    }
}

private struct Dock: View {
    let model: GameModel
    @State private var confirm: Confirm?
    /// One content height for every dock button, so the row lines up.
    private static let height: CGFloat = 26

    enum Confirm { case newGame, resign
        var title: String { self == .newGame ? "Start a new game?" : "Resign this game?" }
        var action: String { self == .newGame ? "New game" : "Resign" }
        var message: String { self == .newGame ? "This game will be lost." : "It counts as a loss in your recent games." }
    }

    private var confirming: Binding<Bool> {
        Binding(get: { confirm != nil }, set: { if !$0 { confirm = nil } })
    }

    var body: some View {
        GlassEffectContainer(spacing: 8) {
            HStack(spacing: 8) {
                Button(action: model.undo) { Image(systemName: "arrow.uturn.backward").frame(width: 30, height: Self.height) }
                    .buttonStyle(.glass)
                    .buttonBorderShape(.capsule)
                    .disabled(model.busy || model.trail.isEmpty)
                    .accessibilityLabel("Undo")
                Menu {
                    Button("New game", systemImage: "plus") { confirm = .newGame }
                    if model.mode == .engine && !model.isOver {
                        Button("Resign", systemImage: "flag", role: .destructive) { confirm = .resign }
                    }
                } label: {
                    Image(systemName: "ellipsis").frame(width: 30, height: Self.height)
                }
                .buttonStyle(.glass)
                .buttonBorderShape(.capsule)
                .disabled(model.busy)
                .accessibilityLabel("Game options")
                .confirmationDialog(confirm?.title ?? "", isPresented: confirming, titleVisibility: .visible,
                                    presenting: confirm) { c in
                    Button(c.action, role: .destructive) {
                        if c == .newGame { model.start(model.mode, opponent: model.opponent) } else { model.resign() }
                    }
                } message: { c in
                    Text(c.message)
                }
                Button(action: model.hint) {
                    HStack(spacing: 6) {
                        Image(systemName: "sparkles")
                        Text(model.hintsLeft > 0 ? "Show me" : "No hints")
                        if model.hintsLeft > 0 {
                            Text("\(model.hintsLeft)").font(.system(size: 12, weight: .bold)).monospacedDigit()
                                .padding(.horizontal, 6).padding(.vertical, 1)
                                .background(.white.opacity(0.25), in: .capsule)
                        }
                    }
                    .frame(maxWidth: .infinity, minHeight: Self.height, maxHeight: Self.height)
                }
                .buttonStyle(.glassProminent)
                .buttonBorderShape(.capsule)
                .tint(Palette.accent)
                .foregroundStyle(Palette.onAccent)
                .disabled(!model.playerToMove || (model.hintsLeft == 0 && !model.hintIsFree))
            }
            .font(Typeface.display(14, .bold))
            .lineLimit(1)
            .minimumScaleFactor(0.8)
            .controlSize(.large)
        }

    }
}

private struct FinishCard: View {
    let finish: Finish
    let again: () -> Void
    let home: () -> Void

    var body: some View {
        ZStack {
            Rectangle().fill(.ultraThinMaterial).ignoresSafeArea()
            VStack(spacing: 6) {
                Text(finish.outcome == .won ? "♔" : finish.outcome == .lost ? "♚" : "½")
                    .font(Typeface.display(54))
                Text(finish.title).font(Typeface.display(30)).tracking(-0.5)
                Text(finish.text).font(.system(size: 15)).foregroundStyle(Palette.mist).multilineTextAlignment(.center)
                    .padding(.bottom, 10)
                HStack(spacing: 8) {
                    Stat(value: finish.accuracy.map { "\($0)%" } ?? "–", label: "Accuracy")
                    Stat(value: "\(finish.great)", label: "Great")
                    Stat(value: "\(finish.blunders)", label: "Blunders")
                }
                .padding(.bottom, 12)
                Button("Play again", action: again).buttonStyle(PrimaryButtonStyle())
                Button("Home", action: home)
                    .font(Typeface.display(15, .bold))
                    .buttonStyle(.plain)
                    .foregroundStyle(Palette.mist)
                    .padding(.top, 8)
            }
            .padding(.horizontal, 22).padding(.vertical, 26)
            .glassEffect(.regular, in: .rect(cornerRadius: 30))
            .padding(24)
        }
        .foregroundStyle(Palette.ink)
    }
}

private struct Stat: View {
    let value: String
    let label: String

    var body: some View {
        VStack(spacing: 2) {
            Text(value).font(Typeface.display(22)).monospacedDigit()
            Text(label.uppercased()).font(.system(size: 10.5, weight: .semibold)).tracking(0.5).foregroundStyle(Palette.mist)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 12)
        .background(Palette.statFill, in: .rect(cornerRadius: 16))
    }
}

#Preview {
    RootView()
}
