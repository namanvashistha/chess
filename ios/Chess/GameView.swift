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
    @AppStorage("opponent") private var opponentRaw = Opponent.easy.rawValue
    @State private var settings = false
    private let saved = SavedGame.load()
    private let recent = Stats.recent

    /// The chosen opponent, falling back to Easy if the choice isn't open yet.
    private var opponent: Opponent {
        let o = Opponent(rawValue: opponentRaw) ?? .easy
        return o.isUnlocked ? o : .easy
    }
    @State private var nudged: Opponent?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // One quiet bar, one headline, one supporting line.
                HStack(spacing: 8) {
                    BrandMark()
                    Text("Cozy Chess").font(Typeface.display(16, .bold))
                    Spacer()
                    Button { Haptics.tap(); settings = true } label: {
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
                            OpponentButton(opponent: o, selected: o == opponent, nudged: nudged == o) {
                                if o.isUnlocked {
                                    opponentRaw = o.rawValue
                                    Sound.shared.play(.select)
                                } else {
                                    Haptics.tap(.rigid, intensity: 0.5)
                                    nudged = o
                                    Task { try? await Task.sleep(for: .seconds(0.5)); nudged = nil }
                                }
                            }
                        }
                    }
                }

                Button { start(.engine, opponent) } label: {
                    Label(opponent.startLabel, systemImage: "arrow.right").labelStyle(TrailingIcon())
                }
                .buttonStyle(PrimaryButtonStyle())

                Button { Haptics.tap(); start(.passAndPlay, opponent) } label: {
                    Label("Pass & play on this phone", systemImage: "person.2").frame(maxWidth: .infinity, minHeight: 40)
                }
                .font(Typeface.display(15, .bold))
                .buttonStyle(.glass)
                .controlSize(.large)

                ProgressCard(records: recent)
                    .padding(.top, 6)

                RecentGames(records: Array(recent.prefix(5)))
                    .padding(.top, 6)
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 24)
        }
        .scrollBounceBehavior(.basedOnSize)
        .foregroundStyle(Palette.ink)
        .sheet(isPresented: $settings) { SettingsSheet().presentationDetents([.height(260)]) }
    }

    /// The tagline, or a nudge back to the game in progress. Numbers stay in
    /// Recent games: the header welcomes, it doesn't keep score.
    private var supportingLine: String {
        saved == nil ? "See the game, one calm move at a time." : "Your game is where you left it."
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
                    Text(saved.passAndPlay ? "Pass & play" : "\(Opponent(rawValue: saved.opponent)?.name ?? "Easy") game")
                        .font(Typeface.display(21))
                        .lineLimit(1).minimumScaleFactor(0.8)
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
        .buttonStyle(PressableStyle(scale: 0.98))
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
    /// Briefly true after tapping a locked opponent: the tile shakes.
    var nudged = false
    let action: () -> Void

    var body: some View {
        let open = opponent.isUnlocked
        Button(action: action) {
            VStack(spacing: 6) {
                ZStack(alignment: .bottomTrailing) {
                    OpponentAvatar(opponent: opponent, size: 34)
                        .scaleEffect(selected ? 1.08 : 1)
                        .animation(.spring(response: 0.35, dampingFraction: 0.5), value: selected)
                        .opacity(open ? 1 : 0.45)
                    if !open {
                        Image(systemName: "lock.fill")
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(Palette.mist)
                            .padding(4)
                            .background(.regularMaterial, in: .circle)
                            .offset(x: 6, y: 4)
                    }
                }
                Text(opponent.name).font(Typeface.display(15, .bold))
                    .foregroundStyle(open ? Palette.ink : Palette.mist)
                if open {
                    Text(opponent.subtitle).font(.system(size: 11.5)).foregroundStyle(Palette.mist)
                } else if let prev = opponent.previous {
                    // How close it is: one dot per win needed.
                    VStack(spacing: 3) {
                        HStack(spacing: 3) {
                            ForEach(0..<Opponent.winsToUnlock, id: \.self) { k in
                                Circle().fill(k < Stats.wins(against: prev) ? Palette.accent : Palette.mist.opacity(0.3))
                                    .frame(width: 5, height: 5)
                            }
                        }
                        Text("Win 3 \(prev.name)").font(.system(size: 11)).foregroundStyle(Palette.mist)
                    }
                }
            }
            .offset(x: nudged ? 4 : 0)
            .animation(nudged ? .spring(response: 0.08, dampingFraction: 0.2).repeatCount(3, autoreverses: true) : .default, value: nudged)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 12)
            .contentShape(.rect)
        }
        .buttonStyle(PressableStyle())
        .glassEffect(selected ? .regular.tint(Palette.accent.opacity(0.25)).interactive() : .regular.interactive(),
                     in: .rect(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).strokeBorder(Palette.accent, lineWidth: selected ? 1.5 : 0)
        }
        .animation(.easeOut(duration: 0.15), value: selected)
        .accessibilityAddTraits(selected ? .isSelected : [])
        .accessibilityHint(open ? "" : "Locked. Win \(Opponent.winsToUnlock) \(opponent.previous?.name ?? "") games to open it.")
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

/// How the player is doing over their last ten games: accuracy as bars, oldest
/// first, and the great moves they've found. It only says "improving" when
/// that's true.
private struct ProgressCard: View {
    let records: [GameRecord]

    private var games: [Int] { Array(records.compactMap(\.accuracy).prefix(10).reversed()) }

    private var improving: Bool {
        let g = games
        guard g.count >= 4 else { return false }
        let half = g.count / 2
        return Self.mean(g.suffix(half)) > Self.mean(g.prefix(half)) + 1
    }

    private static func mean(_ xs: ArraySlice<Int>) -> Double {
        let total: Int = xs.reduce(0, +)
        return Double(total) / Double(max(1, xs.count))
    }

    var body: some View {
        let g = games
        if g.count >= 2 {
            VStack(alignment: .leading, spacing: 8) {
                SectionLabel(text: "Your progress")
                VStack(alignment: .leading, spacing: 12) {
                    HStack(alignment: .firstTextBaseline) {
                        Text(improving ? "You're improving" : "Accuracy, last \(g.count) games")
                            .font(Typeface.display(17))
                        Spacer()
                        Text("\(Int(Self.mean(g[...]).rounded()))% avg").font(.system(size: 13, weight: .semibold)).foregroundStyle(Palette.mist)
                    }
                    // Scaled to this player's own range, so a few points of change show.
                    let lo = Double((g.min() ?? 0) - 8), hi = Double(max((g.max() ?? 100), (g.min() ?? 0) + 10))
                    HStack(alignment: .bottom, spacing: 6) {
                        ForEach(Array(g.enumerated()), id: \.offset) { k, a in
                            VStack(spacing: 4) {
                                RoundedRectangle(cornerRadius: 4, style: .continuous)
                                    .fill(k == g.count - 1 ? Palette.accent : Palette.accent.opacity(0.35))
                                    .frame(width: 14, height: 8 + CGFloat((Double(a) - lo) / (hi - lo)) * 52)
                                Text("\(a)").font(.system(size: 10, weight: .semibold)).monospacedDigit()
                                    .foregroundStyle(Palette.mist)
                            }
                            .frame(maxWidth: .infinity)
                        }
                    }
                    .frame(height: 82, alignment: .bottom)
                    HStack(spacing: 6) {
                        Text("!!").font(.system(size: 12, weight: .heavy)).foregroundStyle(Palette.great)
                        Text("\(Stats.greatMoves) great \(Stats.greatMoves == 1 ? "move" : "moves") found")
                            .font(.system(size: 13.5)).foregroundStyle(Palette.mist)
                    }
                }
                .padding(14)
                .glassEffect(.regular, in: .rect(cornerRadius: 18))
            }
            .accessibilityElement(children: .combine)
        }
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
                            Group {
                                if let o = r.level.flatMap(Opponent.init) ?? Opponent.allCases.first(where: { $0.name == r.opponentName }) {
                                    OpponentAvatar(opponent: o, size: 30)
                                } else {
                                    Image(systemName: "person.2.fill").font(.system(size: 13)).foregroundStyle(Palette.mist)
                                }
                            }
                            .frame(width: 32)
                            VStack(alignment: .leading, spacing: 1) {
                                // Neutral: the opponent, not the score. Only a win gets a word.
                                HStack(spacing: 6) {
                                    Text(r.opponent == "Pass & play" ? "Pass & play" : "vs \(r.opponentName)")
                                        .font(.system(size: 15, weight: .semibold))
                                    if r.outcome == "won" {
                                        Text("Won").font(.system(size: 11, weight: .bold))
                                            .foregroundStyle(Palette.great)
                                            .padding(.horizontal, 6).padding(.vertical, 1)
                                            .background(Palette.greatFill, in: .capsule)
                                    }
                                }
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
    /// How far the screen has been pulled from the left edge, on the way home.
    @State private var pull: CGFloat = 0

    var body: some View {
        ZStack {
            VStack(spacing: 6) {
                TopBar(model: model, back: back)
                CapturedRow(pieces: model.captured(by: "b"), lead: model.materialLead(for: "b"))
                BoardView(model: model)
                CapturedRow(pieces: model.captured(by: "w"), lead: model.materialLead(for: "w"))
                CoachView(card: model.coach, opponent: model.opponent)
                    .overlay(alignment: .topTrailing) {
                        if let entry = model.shareableEntry {
                            ShareMomentButton(moment: model.moment(for: entry))
                                .buttonStyle(.glass)
                                .buttonBorderShape(.circle)
                                .padding(8)
                                .transition(.scale.combined(with: .opacity))
                        }
                    }
                    .padding(.top, 4)
                    .contentShape(.rect)
                    // Swipe right for the move before, left for the move after.
                    .gesture(DragGesture(minimumDistance: 24).onEnded { v in
                        guard abs(v.translation.width) > abs(v.translation.height), abs(v.translation.width) > 40 else { return }
                        withAnimation(.smooth(duration: 0.3)) { model.step(v.translation.width > 0 ? -1 : 1) }
                    })
                MoveTrail(model: model)
                Spacer(minLength: 0)
                Dock(model: model)
            }
            .padding(.horizontal, 14)
            .padding(.bottom, 8)
            .foregroundStyle(Palette.ink)

            if let finish = model.finish {
                FinishCard(finish: finish, share: model.winMoment,
                           again: { model.start(model.mode, opponent: model.opponent) },
                           tryNext: finish.unlocked.map { next in {
                               UserDefaults.standard.set(next.rawValue, forKey: "opponent")
                               model.start(.engine, opponent: next)
                           } },
                           home: back)
                    .transition(.opacity.combined(with: .scale(scale: 0.94)))
            }
        }
        .animation(.spring(response: 0.5, dampingFraction: 0.75), value: model.finish)
        .animation(.smooth(duration: 0.25), value: model.reviewing)
        .offset(x: pull)
        // The iOS back gesture: pull from the left edge to go home.
        .overlay(alignment: .leading) {
            Color.clear
                .frame(width: 16)
                .contentShape(.rect)
                .padding(.top, 80)
                .gesture(DragGesture(minimumDistance: 8, coordinateSpace: .global)
                    .onChanged { v in pull = max(0, v.translation.width) }
                    .onEnded { v in
                        if v.translation.width > 110 || v.predictedEndTranslation.width > 260 {
                            Haptics.tap()
                            back()
                        } else {
                            withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) { pull = 0 }
                        }
                    })
        }
    }
}

private struct TopBar: View {
    let model: GameModel
    let back: () -> Void

    var body: some View {
        HStack(spacing: 10) {
            Button { Haptics.tap(); back() } label: {
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
                                .buttonStyle(PressableStyle(scale: 0.92))
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
                    .onChange(of: model.reviewing) { _, id in
                        if let id { withAnimation { proxy.scrollTo(id, anchor: .center) } }
                    }
                }
                if model.reviewing != nil {
                    HStack {
                        Label("Swipe the card to step through", systemImage: "hand.draw")
                            .font(.system(size: 12.5)).foregroundStyle(Palette.mist)
                        Spacer()
                        Button("Back to game") { Haptics.tap(); model.endReview() }
                            .font(.system(size: 13, weight: .semibold))
                            .buttonStyle(.glass)
                            .buttonBorderShape(.capsule)
                    }
                    .padding(.horizontal, 2)
                    .transition(.opacity)
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
    @State private var undos = 0
    @State private var hints = 0
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
                Button {
                    Haptics.tap()
                    undos += 1
                    model.undo()
                } label: {
                    Image(systemName: "arrow.uturn.backward")
                        .symbolEffect(.rotate.counterClockwise.byLayer, options: .speed(2), value: undos)
                        .frame(width: 30, height: Self.height)
                }
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
                        Haptics.tap(.medium)
                        if c == .newGame { model.start(model.mode, opponent: model.opponent) } else { model.resign() }
                    }
                } message: { c in
                    Text(c.message)
                }
                Button {
                    Haptics.tap()
                    hints += 1
                    model.hint()
                } label: {
                    HStack(spacing: 6) {
                        Image(systemName: "sparkles").symbolEffect(.bounce, value: hints)
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
    let share: ShareMoment?
    let again: () -> Void
    /// Starts a game against the opponent this win just opened.
    let tryNext: (() -> Void)?
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
                if let next = finish.unlocked {
                    HStack(spacing: 10) {
                        Image(systemName: "lock.open.fill").foregroundStyle(Palette.great)
                        VStack(alignment: .leading, spacing: 1) {
                            Text("\(next.name) is open").font(Typeface.display(16))
                            Text(next.subtitle).font(.system(size: 12.5)).foregroundStyle(Palette.mist)
                        }
                        Spacer()
                    }
                    .padding(12)
                    .background(Palette.greatFill.opacity(0.6), in: .rect(cornerRadius: 16))
                    .padding(.bottom, 12)
                }
                VStack(spacing: 14) {
                    if let next = finish.unlocked, let tryNext {
                        Button("Play \(next.name.lowercased()) next") { tryNext() }.buttonStyle(PrimaryButtonStyle())
                    } else {
                        Button("Play again", action: again).buttonStyle(PrimaryButtonStyle())
                    }
                    // Below the primary button's pressed-in edge, as real secondary actions.
                    HStack(spacing: 10) {
                        Button { Haptics.tap(); home() } label: { Text("Home").frame(maxWidth: .infinity, minHeight: 30) }
                        if let share {
                            ShareMomentButton(moment: share, label: "Share")
                        }
                    }
                    .font(Typeface.display(16, .bold))
                    .buttonStyle(.glass)
                    .buttonBorderShape(.capsule)
                    .controlSize(.large)
                    .padding(.top, 6)
                }
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
