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
        .task {
            GameCenter.shared.model = model
            GameCenter.shared.showGame = { withAnimation(.smooth(duration: 0.4)) { playing = true } }
            GameCenter.shared.authenticate()
            SharePlay.shared.model = model
            SharePlay.shared.showGame = { withAnimation(.smooth(duration: 0.4)) { playing = true } }
            SharePlay.shared.listen()
        }
        #if DEBUG
        .task {
            if ProcessInfo.processInfo.arguments.contains("-friendTest") {
                DebugLink.shared.start(model)
                playing = true
            }
        }
        #endif
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

/// Home is a seat at the table. The board is the hero: your game as you left
/// it, or the pieces set up and waiting. Your opponent sits across it; swipe
/// the board to change who. One button sits below: Continue, or Begin.
private struct HomeScreen: View {
    let start: (Mode, Opponent) -> Void
    let resume: () -> Void
    @AppStorage("opponent") private var opponentRaw = Opponent.easy.rawValue
    @State private var settings = false
    @State private var history = false
    @State private var learn = false
    @AppStorage("learn.done") private var learned = false
    @AppStorage("learn.cardClosed") private var cardClosed = false
    @State private var seat: Opponent = .easy
    @State private var swipe: CGFloat = 0
    private let saved = SavedGame.load()
    private let recent = Stats.recent

    private static let setup = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"

    /// The saved game against the engine, if it's with whoever is in the seat.
    private var savedHere: SavedGame? {
        guard let saved, !saved.passAndPlay, saved.opponent == seat.rawValue else { return nil }
        return saved
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                bar
                Text(greeting)
                    .font(Typeface.display(26)).tracking(-0.4)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.top, 18)
                if !learned && !cardClosed && saved == nil {
                    learnCard
                        .padding(.top, 16)
                }
                table
                    .padding(.top, 20)
                cta
                    .padding(.top, 22)
                if let saved, saved.passAndPlay {
                    Button { Haptics.tap(); resume() } label: {
                        Label("Continue your game on this phone", systemImage: "iphone")
                    }
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(Palette.mist)
                    .buttonStyle(.plain)
                    .padding(.top, 10)
                }
                FriendGames()
                    .padding(.top, 22)
                historyLine
                    .padding(.top, 22)
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 24)
            .frame(maxWidth: 560)
            .frame(maxWidth: .infinity)
        }
        .scrollBounceBehavior(.basedOnSize)
        .foregroundStyle(Palette.ink)
        .onAppear {
            if let saved, !saved.passAndPlay, let o = Opponent(rawValue: saved.opponent) { seat = o }
            else { seat = Opponent(rawValue: opponentRaw).flatMap { $0.isUnlocked ? $0 : nil } ?? .easy }
        }
        .sheet(isPresented: $settings) {
            SettingsSheet(learn: {
                settings = false
                Task { try? await Task.sleep(for: .seconds(0.4)); learn = true }
            })
            .presentationDetents([.height(340)])
        }
        .fullScreenCover(isPresented: $learn) { LearnView { start(.engine, .easy) } }
        .sheet(isPresented: $history) {
            ScrollView { RecentGames(records: recent, limit: 20).padding(20) }
                .presentationDetents([.medium, .large])
                .presentationBackground(.regularMaterial)
        }
    }

    // MARK: New to chess

    private var learnCard: some View {
        HStack(spacing: 12) {
            GlassPiece(code: "N").frame(width: 40, height: 40)
            VStack(alignment: .leading, spacing: 1) {
                Text("New to chess?").font(.system(size: 15.5, weight: .semibold))
                Text("Learn how the pieces move · 2 minutes").font(.system(size: 13)).foregroundStyle(Palette.mist)
            }
            Spacer(minLength: 0)
            Button { Haptics.tap(); withAnimation { cardClosed = true } } label: {
                Image(systemName: "xmark").font(.system(size: 11, weight: .bold)).foregroundStyle(Palette.mist)
                    .frame(width: 28, height: 28).contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Not now")
        }
        .padding(.leading, 12).padding(.trailing, 6).padding(.vertical, 10)
        .contentShape(.rect)
        .onTapGesture { Haptics.tap(); learn = true }
        .glassEffect(.regular.interactive(), in: .rect(cornerRadius: 18))
        .accessibilityAddTraits(.isButton)
    }

    // MARK: Bar

    private var bar: some View {
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
    }

    // MARK: The table

    private var table: some View {
        VStack(spacing: 12) {
            // Across the table: who you'd be playing.
            HStack(spacing: 12) {
                OpponentAvatar(opponent: seat, size: 42)
                    .opacity(seat.isUnlocked ? 1 : 0.5)
                VStack(alignment: .leading, spacing: 1) {
                    // Labelled, so the name never reads as a greeting to the player.
                    Text("Your opponent").font(.system(size: 10.5, weight: .bold)).tracking(1).textCase(.uppercase)
                        .foregroundStyle(Palette.mist)
                    Text(seat.name).font(Typeface.display(19))
                    Text(seatLine).font(.system(size: 13)).foregroundStyle(Palette.mist)
                }
                Spacer()
                HStack(spacing: 5) {
                    ForEach(Opponent.allCases) { o in
                        Capsule().fill(o == seat ? Palette.accent : Palette.mist.opacity(0.3))
                            .frame(width: o == seat ? 14 : 6, height: 6)
                    }
                }
                .accessibilityHidden(true)
            }
            .id(seat)
            .transition(.opacity)

            ZStack {
                StaticBoard(fen: String((savedHere?.fen ?? Self.setup).split(separator: " ").first ?? ""))
                    .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                    .shadow(color: .black.opacity(0.22), radius: 16, y: 12)
                    .saturation(seat.isUnlocked ? 1 : 0.2)
                    .opacity(seat.isUnlocked ? 1 : 0.55)
                if !seat.isUnlocked, let prev = seat.previous {
                    VStack(spacing: 6) {
                        Image(systemName: "lock.fill").font(.system(size: 22, weight: .semibold))
                        Text("Beat the \(prev.name) \(Opponent.winsToUnlock) times")
                            .font(Typeface.display(16))
                        Text("\(winsLeft) to go").font(.system(size: 13)).foregroundStyle(Palette.mist)
                    }
                    .padding(.horizontal, 20).padding(.vertical, 16)
                    .glassEffect(.regular, in: .rect(cornerRadius: 20))
                }
            }
            .offset(x: swipe)
            .contentShape(Rectangle())
            .gesture(DragGesture(minimumDistance: 12)
                .onChanged { v in swipe = v.translation.width * 0.35 }
                .onEnded { v in
                    let all = Opponent.allCases
                    let i = all.firstIndex(of: seat) ?? 0
                    let next = v.translation.width < -50 ? min(i + 1, all.count - 1)
                             : v.translation.width > 50 ? max(i - 1, 0) : i
                    withAnimation(.spring(response: 0.4, dampingFraction: 0.8)) {
                        swipe = 0
                        if next != i { seat = all[next] }
                    }
                    if next != i {
                        Sound.shared.play(.select)
                        if all[next].isUnlocked { opponentRaw = all[next].rawValue }
                    }
                })
            .onTapGesture { if seat.isUnlocked { go() } }
            .accessibilityElement()
            .accessibilityLabel("Board, playing the \(seat.name)")
            .accessibilityAdjustableAction { dir in
                let all = Opponent.allCases
                let i = all.firstIndex(of: seat) ?? 0
                seat = all[dir == .increment ? min(i + 1, all.count - 1) : max(i - 1, 0)]
            }

            // Your side of the table.
            HStack(spacing: 6) {
                Circle().fill(Palette.squareLight).frame(width: 8, height: 8)
                    .overlay(Circle().strokeBorder(Palette.mist.opacity(0.5), lineWidth: 0.5))
                Text("You, playing White").font(.system(size: 13)).foregroundStyle(Palette.mist)
                Spacer()
                if savedHere == nil && seat.isUnlocked {
                    Text("Swipe for another opponent").font(.system(size: 12)).foregroundStyle(Palette.mist.opacity(0.8))
                }
            }
        }
        .animation(.easeInOut(duration: 0.25), value: seat)
    }

    private var seatLine: String {
        if !seat.isUnlocked { return L("\(seat.subtitle) · locked") }
        if let s = savedHere { return L("Your game · move \(s.moves.count / 2 + 1)") }
        return seat.subtitle
    }

    private var winsLeft: Int {
        guard let prev = seat.previous else { return 0 }
        return max(0, Opponent.winsToUnlock - Stats.wins(against: prev))
    }

    // MARK: The one button

    @ViewBuilder
    private var cta: some View {
        VStack(spacing: 18) {
            if seat.isUnlocked {
                Button { go() } label: {
                    Text(savedHere == nil ? "Begin" : "Continue")
                }
                .buttonStyle(PrimaryButtonStyle())
            } else {
                Text("Win \(winsLeft) more against the \(seat.previous?.name ?? "") to sit down here")
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(Palette.mist)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: .infinity, minHeight: 56)
            }
            // The quieter ways in, side by side in one pill, each a full-size target.
            HStack(spacing: 0) {
                if savedHere != nil {
                    Button { Haptics.tap(); start(.engine, seat) } label: {
                        Label("New game", systemImage: "plus")
                            .frame(maxWidth: .infinity, minHeight: 48)
                            .contentShape(.rect)
                    }
                    .buttonStyle(.plain)
                    Rectangle().fill(Palette.mist.opacity(0.3)).frame(width: 0.5, height: 22)
                }
                friendLink
            }
            .font(.system(size: 15, weight: .semibold))
            .foregroundStyle(Palette.ink)
            .glassEffect(.regular.interactive(), in: .capsule)
        }
    }

    private func go() {
        Haptics.tap()
        if savedHere != nil { resume() } else { start(.engine, seat) }
    }

    // MARK: Friends and history

    private var friendLink: some View {
        Menu {
            if SharePlay.shared.available {
                Button("Play on this FaceTime call", systemImage: "shareplay") { SharePlay.shared.start() }
            }
            Button("Invite with Game Center", systemImage: "person.crop.circle.badge.plus") {
                GameCenter.shared.invite()
            }
            Button("Play on this phone", systemImage: "iphone") { start(.passAndPlay, seat) }
        } label: {
            Label("Play a friend", systemImage: "person.2")
                .frame(maxWidth: .infinity, minHeight: 48)
                .contentShape(.rect)
        }
    }

    @ViewBuilder
    private var historyLine: some View {
        if !recent.isEmpty {
            Button { Haptics.tap(); history = true } label: {
                HStack(spacing: 8) {
                    Image(systemName: "clock.arrow.circlepath").foregroundStyle(Palette.mist)
                    Text("Your games").font(.system(size: 14.5, weight: .semibold))
                    if let s = historySummary {
                        Text(s).font(.system(size: 13.5)).foregroundStyle(Palette.mist)
                    }
                    Spacer()
                    Image(systemName: "chevron.right").font(.system(size: 12, weight: .semibold)).foregroundStyle(Palette.mist)
                }
                .padding(.horizontal, 14).padding(.vertical, 13)
                .contentShape(.rect)
            }
            .buttonStyle(PressableStyle(scale: 0.98))
            .glassEffect(.regular.interactive(), in: .capsule)
        }
    }

    private var historySummary: String? {
        let a = recent.filter { $0.opponent != "Pass & play" }.compactMap(\.accuracy).prefix(10)
        guard a.count >= 2 else { return L("\(recent.count) played") }
        return L("\(a.reduce(0, +) / a.count)% lately")
    }

    private var greeting: String {
        switch Calendar.current.component(.hour, from: .now) {
        case 5..<12: L("Good morning")
        case 12..<17: L("Good afternoon")
        default: L("Good evening")
        }
    }
}

private struct SectionLabel: View {
    let text: LocalizedStringKey
    var body: some View {
        Text(text).textCase(.uppercase)
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
                    Text("Move \(saved.moves.count / 2 + 1)")
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
                        Text("\(Opponent.winsToUnlock) wins to open").font(.system(size: 11)).foregroundStyle(Palette.mist)
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
        .accessibilityHint(open ? "" : "Locked. Beat the \(opponent.previous?.name ?? "") \(Opponent.winsToUnlock) times to open it.")
    }
}

/// A friend: their initial on a soft glass circle.
struct FriendAvatar: View {
    let name: String
    let size: CGFloat

    var body: some View {
        Text(String(name.first ?? "?").uppercased())
            .font(Typeface.display(size * 0.42))
            .foregroundStyle(Palette.accentText)
            .frame(width: size, height: size)
            .glassEffect(.regular.tint(Palette.accent.opacity(0.15)), in: .circle)
            .accessibilityHidden(true)
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

/// Games with friends that are still going, from Game Center: a strip of small
/// boards, each as it stands, with whose move it is.
private struct FriendGames: View {
    private var center: GameCenter { GameCenter.shared }

    var body: some View {
        if !center.matches.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                SectionLabel(text: "With friends")
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 12) {
                        ForEach(center.matches, id: \.matchID) { m in
                            let mine = GameCenter.isMyTurn(m)
                            Button { Haptics.tap(); center.open(m) } label: {
                                VStack(alignment: .leading, spacing: 6) {
                                    StaticBoard(fen: GameCenter.boardFEN(GameCenter.moves(in: m)))
                                        .frame(width: 104, height: 104)
                                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                                    Text(GameCenter.friendName(in: m)).font(.system(size: 14, weight: .semibold))
                                        .lineLimit(1)
                                    HStack(spacing: 5) {
                                        Circle().fill(mine ? Palette.great : Palette.mist.opacity(0.5)).frame(width: 6, height: 6)
                                        Text(mine ? "Your move" : "Their move").font(.system(size: 12))
                                            .foregroundStyle(mine ? Palette.great : Palette.mist)
                                    }
                                }
                                .padding(10)
                                .contentShape(.rect)
                            }
                            .buttonStyle(PressableStyle())
                            .glassEffect(.regular.interactive(), in: .rect(cornerRadius: 18))
                        }
                    }
                    .padding(.vertical, 4)
                }
                .scrollClipDisabled()
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .task { await center.reload() }
        }
    }
}

private struct RecentGames: View {
    let records: [GameRecord]

    var limit = 3
    /// The last few games: enough to remember, not a ledger.
    private var shown: [GameRecord] { Array(records.prefix(limit)) }

    /// One quiet line in place of a chart: how it's going lately.
    private var summary: String? {
        let recent = records.filter { $0.opponent != "Pass & play" }.compactMap(\.accuracy).prefix(10)
        guard recent.count >= 2 else { return nil }
        let avg = recent.reduce(0, +) / recent.count
        let great = Stats.greatMoves
        return great > 0 ? L("\(avg)% lately · \(great) great moves") : L("\(avg)% lately")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline) {
                SectionLabel(text: "Your games")
                Spacer()
                if let summary { Text(summary).font(.system(size: 12.5)).foregroundStyle(Palette.mist) }
            }
            if records.isEmpty {
                Text("Your finished games will show up here, with how accurately you played.")
                    .font(.system(size: 14)).foregroundStyle(Palette.mist)
                    .padding(.vertical, 6)
            } else {
                VStack(spacing: 0) {
                    ForEach(Array(shown.enumerated()), id: \.element.id) { k, r in
                        HStack(spacing: 12) {
                            Group {
                                if let o = r.resolvedLevel.flatMap(Opponent.init) {
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
                            if r.opponent != "Pass & play", let a = r.accuracy {
                                Text("\(a)%").font(Typeface.display(17)).monospacedDigit()
                            }
                        }
                        .padding(.vertical, 10)
                        .padding(.horizontal, 12)
                        if k < shown.count - 1 { Divider().padding(.leading, 56) }
                    }
                }
                .glassEffect(.regular, in: .rect(cornerRadius: 18))
            }
        }
    }
}

private struct SettingsSheet: View {
    var learn: () -> Void = {}
    @AppStorage("sound") private var sound = true
    @AppStorage("haptics") private var haptics = true
    @Environment(\.openURL) private var openURL

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("Settings").font(Typeface.display(22))
            Toggle("Sound", isOn: $sound)
            Toggle("Haptics", isOn: $haptics)
            Button { learn() } label: {
                HStack {
                    Label("Learn the moves", systemImage: "graduationcap")
                    Spacer()
                    Image(systemName: "chevron.right").font(.system(size: 13, weight: .semibold)).foregroundStyle(Palette.mist)
                }
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            // iOS keeps each app's language in Settings; this opens that page.
            Button { openURL(URL(string: UIApplication.openSettingsURLString)!) } label: {
                HStack {
                    Label("Language", systemImage: "globe")
                    Spacer()
                    Text(Locale.current.localizedString(forLanguageCode: Locale.current.language.languageCode?.identifier ?? "en") ?? "")
                        .foregroundStyle(Palette.mist)
                    Image(systemName: "arrow.up.forward").font(.system(size: 12, weight: .semibold)).foregroundStyle(Palette.mist)
                }
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
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

struct GameScreen: View {
    let model: GameModel
    let back: () -> Void
    /// How far the screen has been pulled from the left edge, on the way home.
    @State private var pull: CGFloat = 0

    /// The coach card, with its swipe to step through past moves, and a way
    /// back to the game while looking back.
    private var coach: some View {
        VStack(spacing: 6) {
            CoachView(card: model.coach, opponent: model.opponent, friend: model.mode == .friend ? model.friendName : nil)
                .padding(.top, 4)
                .contentShape(.rect)
                // Swipe right for the move before, left for the move after.
                .gesture(DragGesture(minimumDistance: 24).onEnded { v in
                    guard abs(v.translation.width) > abs(v.translation.height), abs(v.translation.width) > 40 else { return }
                    withAnimation(.smooth(duration: 0.3)) { model.step(v.translation.width > 0 ? -1 : 1) }
                })
            if model.reviewing != nil {
                Button { Haptics.tap(); model.endReview() } label: {
                    Label("Back to game", systemImage: "arrow.uturn.forward")
                }
                .font(.system(size: 13.5, weight: .semibold))
                .buttonStyle(.glass)
                .buttonBorderShape(.capsule)
                .transition(.opacity)
            }
        }
    }

    var body: some View {
        ZStack {
            GeometryReader { geo in
                // A wide screen (iPad in landscape): the board on the left, the
                // coach and controls in a column beside it. Otherwise one
                // column, kept to a comfortable width on iPad.
                if geo.size.width > 700 && geo.size.width > geo.size.height {
                    HStack(alignment: .center, spacing: 28) {
                        VStack(spacing: 6) {
                            CapturedRow(pieces: model.captured(by: "b"), lead: model.materialLead(for: "b"))
                            BoardView(model: model)
                            CapturedRow(pieces: model.captured(by: "w"), lead: model.materialLead(for: "w"))
                        }
                        .frame(width: min(geo.size.height - 70, geo.size.width - 420))
                        VStack(spacing: 10) {
                            TopBar(model: model, back: back)
                            coach
                            Spacer(minLength: 0)
                            Dock(model: model)
                        }
                        .frame(width: 360)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .padding(.vertical, 12)
                } else {
                    VStack(spacing: 6) {
                        TopBar(model: model, back: back)
                        CapturedRow(pieces: model.captured(by: "b"), lead: model.materialLead(for: "b"))
                        BoardView(model: model)
                        CapturedRow(pieces: model.captured(by: "w"), lead: model.materialLead(for: "w"))
                        coach
                        Spacer(minLength: 0)
                        Dock(model: model)
                    }
                    .frame(maxWidth: min(760, geo.size.height * 0.6))
                    .frame(maxWidth: .infinity)
                }
            }
            .padding(.horizontal, 14)
            .padding(.bottom, 8)
            .foregroundStyle(Palette.ink)

            if let finish = model.finish {
                FinishCard(finish: finish, share: model.finishMoment,
                           again: { model.mode == .friend ? model.rematch() : model.start(model.mode, opponent: model.opponent) },
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
            } else if model.mode == .friend {
                FriendAvatar(name: model.friendName, size: 36)
            }
            VStack(alignment: .leading, spacing: 1) {
                Text(model.mode == .engine ? model.opponent.name : model.mode == .friend ? model.friendName : "Pass & play")
                    .font(Typeface.display(17, .bold))
                HStack(spacing: 6) {
                    if model.playerToMove { Circle().fill(Palette.great).frame(width: 6, height: 6) }
                    Text(stateText).font(.system(size: 12.5)).foregroundStyle(Palette.mist)
                    if model.thinking { ThinkingDots() }
                }
            }
            Spacer()
            // No live score: accuracy waits for the game-over card, so play feels unjudged.
        }
        .padding(.top, 4)
    }

    private var stateText: String {
        if model.isOver { return L("Game over") }
        if model.thinking { return L("Thinking") }
        if model.mode == .passAndPlay { return model.turn == "w" ? L("White to move") : L("Black to move") }
        if model.mode == .friend { return model.turn == model.myColor ? L("Your move") : L("Their move") }
        return model.turn == "w" ? L("Your move") : ""
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
    var friend: String?

    var body: some View {
        ZStack {
            if let card {
                HStack(spacing: 12) {
                    badge(card)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(card.title).font(Typeface.display(17)).foregroundStyle(titleColor(card.kind))
                        if !card.text.isEmpty {
                            Text(card.text).font(.system(size: 13.5)).foregroundStyle(Palette.mist)
                                .fixedSize(horizontal: false, vertical: true)
                        }
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
                if let friend { FriendAvatar(name: friend, size: 46) } else { OpponentAvatar(opponent: opponent, size: 46) }
                Text(card.badge)
                    .font(.system(size: 11, weight: .bold)).monospacedDigit()
                    .lineLimit(1).minimumScaleFactor(0.6)
                    .padding(.horizontal, 5).padding(.vertical, 2)
                    .background(card.kind == .warning ? Palette.mistakeFill : Palette.goodFill, in: .capsule)
                    .offset(x: 6, y: 4)
            }
            .frame(width: 46, height: 46)
        } else {
            Group {
                if let symbol = card.kind.symbol {
                    Image(systemName: symbol).font(.system(size: 19, weight: .bold))
                } else {
                    Text(card.badge).font(Typeface.display(20))
                }
            }
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

    private var canHint: Bool { model.playerToMove && (model.hintsLeft > 0 || model.hintIsFree) }

    private var confirming: Binding<Bool> {
        Binding(get: { confirm != nil }, set: { if !$0 { confirm = nil } })
    }

    var body: some View {
        GlassEffectContainer(spacing: 8) {
            HStack(spacing: 8) {
                if model.mode != .friend {
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
                }
                Menu {
                    if model.mode != .friend {
                        Button("New game", systemImage: "plus") { confirm = .newGame }
                    }
                    if model.mode != .passAndPlay && !model.isOver {
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
                if model.mode == .friend {
                    // Against a friend there are no hints: just whose turn it is.
                    Text(model.isOver ? "Game over" : model.turn == model.myColor ? "Your move" : "Waiting for \(model.friendInSentence)")
                        .foregroundStyle(Palette.ink)
                        .frame(maxWidth: .infinity, minHeight: Self.height, maxHeight: Self.height)
                        .padding(.vertical, 7)
                        .glassEffect(.regular, in: .capsule)
                } else {
                Button {
                    Haptics.tap()
                    hints += 1
                    model.hint()
                } label: {
                    HStack(spacing: 6) {
                        Image(systemName: "sparkles").symbolEffect(.bounce, value: hints)
                        Text(model.hintIsFree && !model.hintRevealed ? "Show the move" : model.hintsLeft > 0 ? "Show me" : "No hints")
                        if model.hintsLeft > 0 && !(model.hintIsFree && !model.hintRevealed) {
                            Text("\(model.hintsLeft)").font(.system(size: 12, weight: .bold)).monospacedDigit()
                                .contentTransition(.numericText())
                                .animation(.snappy, value: model.hintsLeft)
                                .padding(.horizontal, 6).padding(.vertical, 1)
                                .background(canHint ? Color.white.opacity(0.25) : Palette.mist.opacity(0.15), in: .capsule)
                        }
                    }
                    .frame(maxWidth: .infinity, minHeight: Self.height, maxHeight: Self.height)
                    // Disabled, the green fades to white: switch to quiet glass with
                    // grey text so the label stays readable in light mode.
                    .foregroundStyle(canHint ? Palette.onAccent : Palette.mist)
                }
                .buttonStyle(canHint ? AnyPrimitiveButtonStyle(.glassProminent) : AnyPrimitiveButtonStyle(.glass))
                .buttonBorderShape(.capsule)
                .tint(canHint ? Palette.accent : nil)
                .disabled(!canHint)
                }
            }
            .font(Typeface.display(14, .bold))
            .lineLimit(1)
            .minimumScaleFactor(0.8)
            .controlSize(.large)
        }

    }
}

/// The end of a game, in one calm column: what happened, your best move played
/// again, one line of numbers, what's next, and the way on.
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
            VStack(spacing: 20) {
                // What happened.
                VStack(spacing: 4) {
                    Text(finish.title).font(Typeface.display(28)).tracking(-0.5)
                        .multilineTextAlignment(.center)
                    Text(finish.firstWin ? "Your first win · \(finish.text)" : finish.text)
                        .font(.system(size: 14.5)).foregroundStyle(Palette.mist)
                        .multilineTextAlignment(.center)
                }

                // The best move, played again.
                if let best = finish.best {
                    VStack(spacing: 10) {
                        BestMoveReplay(best: best)
                            .frame(width: 196, height: 196)
                        VStack(spacing: 2) {
                            HStack(spacing: 5) {
                                if let symbol = best.kind.symbol { Image(systemName: symbol).font(.system(size: 11, weight: .bold)) }
                                Text(best.side.map { "Best move · \($0), move \(best.moveNumber)" }
                                     ?? "Your best move · move \(best.moveNumber)")
                            }
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(verdictColor(best.kind))
                            Text(best.text).font(.system(size: 13)).foregroundStyle(Palette.mist)
                                .multilineTextAlignment(.center).lineLimit(2)
                        }
                    }
                }

                // One line of numbers, and what comes next.
                VStack(spacing: 8) {
                    Text(numbers).font(.system(size: 13, weight: .medium)).monospacedDigit()
                        .foregroundStyle(Palette.mist)
                    if let next = finish.unlocked {
                        Label("The \(next.name) is open", systemImage: "lock.open.fill")
                            .font(.system(size: 13.5, weight: .semibold)).foregroundStyle(Palette.great)
                    } else if let toward = finish.towardNext {
                        HStack(spacing: 8) {
                            HStack(spacing: 3) {
                                ForEach(0..<Opponent.winsToUnlock, id: \.self) { k in
                                    Circle().fill(k < toward.wins ? Palette.accent : Palette.mist.opacity(0.3)).frame(width: 6, height: 6)
                                }
                            }
                            let left = Opponent.winsToUnlock - toward.wins
                            Text(left == 1 ? L("1 more win to face the \(toward.next.name)")
                                           : L("\(left) more wins to face the \(toward.next.name)"))
                        }
                        .font(.system(size: 13.5, weight: .medium))
                    }
                }

                // The way on: one clear next step, then Home and Share as one
                // quiet pill below it.
                VStack(spacing: 16) {
                    if let next = finish.unlocked, let tryNext {
                        Button("Play the \(next.name)") { tryNext() }.buttonStyle(PrimaryButtonStyle())
                    } else {
                        Button("Play again", action: again).buttonStyle(PrimaryButtonStyle())
                    }
                    HStack(spacing: 0) {
                        Button { Haptics.tap(); home() } label: {
                            Label("Home", systemImage: "house")
                                .frame(maxWidth: .infinity, minHeight: 46)
                                .contentShape(.rect)
                        }
                        if let share {
                            Rectangle().fill(Palette.mist.opacity(0.3)).frame(width: 0.5, height: 22)
                            ShareMomentButton(moment: share, label: "Share")
                                .frame(maxWidth: .infinity, minHeight: 46)
                                .contentShape(.rect)
                        }
                    }
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(Palette.ink)
                    .buttonStyle(.plain)
                    .glassEffect(.regular.interactive(), in: .capsule)
                }
                .padding(.top, 4)
            }
            .padding(.horizontal, 24).padding(.top, 28).padding(.bottom, 26)
            .frame(maxWidth: 460)
            .glassEffect(.regular, in: .rect(cornerRadius: 32))
            .overlay(alignment: .top) {
                if finish.outcome == .won { Celebration().offset(y: 40) }
            }
            .padding(22)
        }
        .foregroundStyle(Palette.ink)
    }

    /// "94% accuracy · 2 great · 3 good", leaving out what's zero or unknown.
    private var numbers: String {
        if let s = finish.sides {
            return s.white + s.black == 0 ? L("A game for two") : L("White \(s.white) great · Black \(s.black) great")
        }
        var parts: [String] = []
        if let a = finish.accuracy { parts.append(L("\(a)% accuracy")) }
        if finish.great > 0 { parts.append(L("\(finish.great) great")) }
        let good = finish.goodMoves - finish.great
        if good > 0 { parts.append(L("\(good) good")) }
        return parts.isEmpty ? L("Every game teaches something") : parts.joined(separator: " · ")
    }
}

#Preview {
    RootView()
}

/// A win: glass flecks in the app's colours rise and fall once.
private struct Celebration: View {
    @State private var flown = false
    private let flecks: [(dx: CGFloat, dy: CGFloat, rot: Double, delay: Double, color: Int, size: CGFloat)] = (0..<28).map { _ in
        (CGFloat.random(in: -170...170), CGFloat.random(in: -190 ... -60), Double.random(in: -360...360),
         Double.random(in: 0...0.15), Int.random(in: 0..<3), CGFloat.random(in: 6...11))
    }

    var body: some View {
        ZStack {
            ForEach(flecks.indices, id: \.self) { i in
                let f = flecks[i]
                RoundedRectangle(cornerRadius: 2)
                    .fill([Palette.great, Palette.accent, Palette.inaccuracy][f.color])
                    .frame(width: f.size, height: f.size * 0.6)
                    .rotationEffect(.degrees(flown ? f.rot : 0))
                    .offset(x: flown ? f.dx : 0, y: flown ? f.dy : 0)
                    .opacity(flown ? 0 : 1)
                    .animation(.timingCurve(0.1, 0.8, 0.3, 1, duration: 1.4).delay(f.delay), value: flown)
            }
        }
        .allowsHitTesting(false)
        .accessibilityHidden(true)
        .onAppear { flown = true }
    }
}

/// The finish card's replay: the board before the move, then the piece glides
/// to where it went and anything it took fades. Tap to see it again.
private struct BestMoveReplay: View {
    let best: BestMove
    @State private var played = false

    var body: some View {
        GeometryReader { geo in
            let size = geo.size.width, q = size / 8
            let m = best.move
            let lift = CGPoint(x: CGFloat(Squares.file(m.from) - Squares.file(m.to)) * q,
                               y: CGFloat(Squares.row(m.from) - Squares.row(m.to)) * q)
            ZStack(alignment: .topLeading) {
                VStack(spacing: 0) {
                    ForEach(0..<8, id: \.self) { r in
                        HStack(spacing: 0) {
                            ForEach(0..<8, id: \.self) { c in
                                let sq = r * 8 + c
                                ZStack {
                                    Rectangle().fill((r + c) % 2 == 1 ? Palette.squareDark : Palette.squareLight)
                                    if sq == m.from || sq == m.to {
                                        Rectangle().fill(Palette.lastMove.opacity(sq == m.to ? (played ? 0.45 : 0.15) : 0.26))
                                    }
                                }
                                .frame(width: q, height: q)
                            }
                        }
                    }
                }
                // What it took, fading as the piece arrives.
                if best.before[m.to] != "." {
                    GlassPiece(code: best.before[m.to])
                        .frame(width: q, height: q)
                        .position(center(m.to, size))
                        .opacity(played ? 0 : 1)
                }
                ForEach(0..<64, id: \.self) { i in
                    if best.after[i] != "." {
                        GlassPiece(code: best.after[i], lifted: i == m.to && !played)
                            .frame(width: q, height: q)
                            .position(center(i, size))
                            .offset(x: i == m.to && !played ? lift.x : 0, y: i == m.to && !played ? lift.y : 0)
                            .zIndex(i == m.to ? 20 : Double(i / 8))
                    }
                }
            }
        }
        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        .shadow(color: .black.opacity(0.18), radius: 8, y: 5)
        .contentShape(Rectangle())
        .onTapGesture { replay() }
        .task { replay() }
        .accessibilityLabel("Your best move, move \(best.moveNumber)")
        .accessibilityAddTraits(.isButton)
    }

    private func replay() {
        played = false
        Task {
            try? await Task.sleep(for: .seconds(0.7))
            withAnimation(.spring(response: 0.6, dampingFraction: 0.7)) { played = true }
            Haptics.tap(.soft, intensity: 0.6)
        }
    }
}

/// Picks a primitive button style at runtime, so a button can switch between
/// prominent and plain glass.
struct AnyPrimitiveButtonStyle: PrimitiveButtonStyle {
    private let make: (Configuration) -> AnyView
    init<S: PrimitiveButtonStyle>(_ style: S) { make = { AnyView(style.makeBody(configuration: $0)) } }
    func makeBody(configuration: Configuration) -> some View { make(configuration) }
}
