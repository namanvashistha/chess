import Chesscore
import SwiftUI

extension EnvironmentValues {
    /// Shared by the home board and the game board, so going into a game the
    /// board itself moves into place instead of the page sliding.
    @Entry var boardSpace: Namespace.ID?
}

extension View {
    /// Everything but the board on a screen that is arriving: it gathers
    /// around the board, which carries across fully drawn. Things above the
    /// board drift down into place, things below rise, each a beat apart.
    func chrome(_ shown: Bool, from edge: VerticalEdge = .bottom, delay: Double = 0.14) -> some View {
        opacity(shown ? 1 : 0)
            .offset(y: shown ? 0 : (edge == .top ? -10 : 14))
            .animation(.spring(response: 0.5, dampingFraction: 0.85).delay(delay), value: shown)
    }

    /// Marks the board that carries across between home and the game. Only
    /// its position travels: animating its size would redraw every glass
    /// piece on every frame, and the two boards are nearly the same width.
    @ViewBuilder func heroBoard(_ space: Namespace.ID?) -> some View {
        if let space { matchedGeometryEffect(id: "board", in: space, properties: .position) } else { self }
    }
}

/// The app: home, and the game it starts or resumes.
struct RootView: View {
    @State private var model = GameModel()
    @Namespace private var boardSpace
    @State private var playing = false
    @State private var homeID = 0 // refreshes home's saved-game and history views on return
    /// The first-open welcome has been played (or skipped). Settings can reset it.
    @AppStorage("welcome.seen") private var welcomed = false
    @State private var welcoming = false
    @State private var learnAfterWelcome = false

    /// The board carrying across between home and a game.
    /// One smooth glide, no bounce: just the board changing place.
    static let carry = Animation.smooth(duration: 0.45)
    /// A screen arrives without fading (its board must stay fully drawn while
    /// it carries across; the rest fades in on its own) and leaves by fading.
    /// The arriving side changes nothing but must not be .identity, which
    /// would drop the board before it can carry across.
    static let screen = AnyTransition.asymmetric(insertion: .scale(scale: 0.999), removal: .opacity)

    var body: some View {
        ZStack {
            AppBackground()
            if playing {
                GameScreen(model: model) {
                    homeID += 1
                    withAnimation(Self.carry) { playing = false }
                }
                .transition(Self.screen)
            } else {
                HomeScreen(
                    start: { mode, opponent in
                        model.start(mode, opponent: opponent)
                        withAnimation(Self.carry) { playing = true }
                    },
                    resume: {
                        if model.resume() { withAnimation(Self.carry) { playing = true } }
                    }
                )
                .id(homeID)
                .transition(Self.screen)
            }
        }
        .environment(\.boardSpace, boardSpace)
        .fullScreenCover(isPresented: $welcoming, onDismiss: { welcomed = true }) {
            WelcomeView(enter: {
                // The welcome has already dropped away, so the cover itself
                // must not animate: the curtain is the only movement.
                var t = Transaction()
                t.disablesAnimations = true
                withTransaction(t) { welcoming = false }
            }, learn: { learnAfterWelcome = true })
            .presentationBackground(.clear)
        }
        .fullScreenCover(isPresented: $learnAfterWelcome) {
            LearnView {
                model.start(.engine, opponent: .easy)
                withAnimation(Self.carry) { playing = true }
            }
        }
        .onAppear {
            // Only a true first run; someone who has already played doesn't need it.
            if !welcomed && (!Stats.recent.isEmpty || SavedGame.load() != nil) { welcomed = true }
            welcoming = !welcomed || ProcessInfo.processInfo.arguments.contains("-welcome")
        }
        // Settings' "How it works" clears the flag to play it again.
        .onChange(of: welcomed) { _, seen in if !seen { welcoming = true } }
        .sensoryFeedback(trigger: model.feedback) { _, new in Sound.hapticsOn ? new?.kind.haptic : nil }
        .task {
            GameCenter.shared.model = model
            GameCenter.shared.showGame = { withAnimation(Self.carry) { playing = true } }
            GameCenter.shared.authenticate()
            SharePlay.shared.model = model
            SharePlay.shared.showGame = { withAnimation(Self.carry) { playing = true } }
            SharePlay.shared.listen()
        }
        #if DEBUG
        .task {
            if ProcessInfo.processInfo.arguments.contains("-friendTest") {
                DebugLink.shared.start(model)
                playing = true
            }
            let args = ProcessInfo.processInfo.arguments
            if let i = args.firstIndex(of: "-finish"), i + 1 < args.count,
               let outcome = ["won": Finish.Outcome.won, "lost": .lost, "draw": .draw][args[i + 1]] {
                playing = true
                model.previewFinish(outcome)
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
    @Environment(\.boardSpace) private var boardSpace
    /// Fades in everything around the board as home arrives.
    @State private var shown = false
    @AppStorage("opponent") private var opponentRaw = Opponent.easy.rawValue
    @State private var settings = false
    @State private var history = false
    @State private var learn = false
    @AppStorage("learn.done") private var learned = false
    @AppStorage("learn.cardClosed") private var cardClosed = false
    @State private var seat: Opponent = .easy
    @State private var swipe: CGFloat = 0
    /// The swipe hint shows until the first swipe, then never again.
    @AppStorage("home.swiped") private var swiped = false
    /// How many times the board has peeked to show it can be swiped.
    @AppStorage("home.peeks") private var peeks = 0
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
                bar.chrome(shown, from: .top, delay: 0.1)
                if !learned && !cardClosed && saved == nil {
                    learnCard
                        .padding(.top, 16)
                        .chrome(shown, from: .top, delay: 0.12)
                }
                table
                    .padding(.top, 24)
                cta
                    .padding(.top, 24)
                    .chrome(shown, delay: 0.18)
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
                    .padding(.top, 24)
                    .chrome(shown, delay: 0.22)
                historyLine
                    .padding(.top, 16)
                    .chrome(shown, delay: 0.26)
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
        .task { await peek() }
        .onAppear { shown = true }
        .sheet(isPresented: $settings) {
            SettingsSheet(learn: {
                settings = false
                Task { try? await Task.sleep(for: .seconds(0.4)); learn = true }
            }, welcome: {
                settings = false
                // After the sheet has fully gone, so the cover appears once.
                Task { try? await Task.sleep(for: .seconds(0.7)); UserDefaults.standard.set(false, forKey: "welcome.seen") }
            })
            .presentationDetents([.height(390)])
        }
        .fullScreenCover(isPresented: $learn) { LearnView { start(.engine, .easy) } }
        .sheet(isPresented: $history) {
            ScrollView { RecentGames(records: recent, limit: 20).padding(24) }
                .presentationDetents([.medium, .large])
                .presentationBackground(Palette.surface)
                .presentationDragIndicator(.visible)
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

    /// The greeting, with settings tucked beside it: one line above the board.
    private var bar: some View {
        HStack(alignment: .firstTextBaseline) {
            Text(greeting)
                .font(Typeface.display(26)).tracking(-0.4)
            Spacer()
            Button { Haptics.tap(); settings = true } label: {
                Image(systemName: "gearshape").font(.system(size: 18, weight: .medium))
                    .foregroundStyle(Palette.mist)
                    .frame(width: 44, height: 44)
                    .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Settings")
        }
        .padding(.top, 8)
    }

    // MARK: The table

    private var table: some View {
        VStack(spacing: 12) {
            // Across the table: who you'd be playing.
            HStack(spacing: 12) {
                OpponentAvatar(opponent: seat, size: 42)
                    .opacity(seat.isUnlocked ? 1 : 0.5)
                // "vs", so the name never reads as a greeting under "Good
                // evening"; the title itself says how strong they are.
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text("vs").font(.system(size: 15, weight: .medium)).foregroundStyle(Palette.mist)
                    Text(seat.name).font(Typeface.display(19))
                }
                Spacer()
                // One dot per opponent. A tap anywhere on them moves on to the
                // next, like a swipe, and wraps back to the first.
                Button { Task { await slideToNext() } } label: {
                    HStack(spacing: 5) {
                        ForEach(Opponent.allCases) { o in
                            Capsule().fill(o == seat ? Palette.accent : Palette.mist.opacity(0.3))
                                .frame(width: o == seat ? 14 : 6, height: 6)
                        }
                    }
                    .padding(.horizontal, 8).padding(.vertical, 13)
                    .contentShape(.rect)
                }
                .buttonStyle(.plain)
                .padding(.trailing, -8)
                .accessibilityHidden(true)
            }
            .id(seat)
            .transition(.opacity)
            .chrome(shown, from: .top, delay: 0.14)

            ZStack {
                StaticBoard(fen: String((savedHere?.fen ?? Self.setup).split(separator: " ").first ?? ""))
                    .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                    .heroBoard(boardSpace)
                    .shadow(color: .black.opacity(0.22), radius: 16, y: 12)
                    .saturation(seat.isUnlocked ? 1 : 0.2)
                    .opacity(seat.isUnlocked ? 1 : 0.55)
                if !seat.isUnlocked, let prev = seat.previous {
                    // One line that counts down, and the wins so far as a bar.
                    VStack(spacing: 8) {
                        Image(systemName: "lock.fill").font(.system(size: 22, weight: .semibold))
                        Text(winsLeft == Opponent.winsToUnlock ? L("Beat the \(prev.name) \(winsLeft) times")
                             : winsLeft == 1 ? L("Beat the \(prev.name) once more")
                             : L("Beat the \(prev.name) \(winsLeft) more times"))
                            .font(Typeface.display(16))
                        // A short bar in three parts: progress, not pages.
                        HStack(spacing: 3) {
                            ForEach(0..<Opponent.winsToUnlock, id: \.self) { k in
                                Capsule().fill(k < Opponent.winsToUnlock - winsLeft ? Palette.accent : Palette.mist.opacity(0.3))
                                    .frame(width: 22, height: 4)
                            }
                        }
                        .accessibilityHidden(true)
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
                    withAnimation(.spring(response: 0.4, dampingFraction: 0.8)) { swipe = 0 }
                    choose(all[next])
                })
            .onTapGesture { if seat.isUnlocked { go() } }
            .accessibilityElement()
            .accessibilityLabel("Board, playing the \(seat.name)")
            .accessibilityAdjustableAction { dir in
                let all = Opponent.allCases
                let i = all.firstIndex(of: seat) ?? 0
                seat = all[dir == .increment ? min(i + 1, all.count - 1) : max(i - 1, 0)]
            }

            // Until the first swipe, a quiet word on how to change opponents.
            if !swiped && seat.isUnlocked {
                Text("Swipe for another opponent").font(.system(size: 12)).foregroundStyle(Palette.mist.opacity(0.8))
                    .transition(.opacity)
                    .chrome(shown, delay: 0.2)
            }
        }
        .animation(.easeInOut(duration: 0.25), value: seat)
    }

    /// Sits across from another opponent, by swipe or by tapping the dots.
    private func choose(_ o: Opponent) {
        guard o != seat else { return }
        withAnimation(.spring(response: 0.4, dampingFraction: 0.8)) { seat = o }
        swiped = true
        Sound.shared.play(.select)
        if o.isUnlocked { opponentRaw = o.rawValue }
    }

    /// The dots' tap, played as a swipe: the board slides out to the left and
    /// the next opponent's slides in from the right.
    private func slideToNext() async {
        let all = Opponent.allCases
        let next = all[((all.firstIndex(of: seat) ?? 0) + 1) % all.count]
        withAnimation(.easeIn(duration: 0.14)) { swipe = -48 }
        try? await Task.sleep(for: .seconds(0.14))
        choose(next)
        var t = Transaction(); t.disablesAnimations = true
        withTransaction(t) { swipe = 48 }
        withAnimation(.spring(response: 0.38, dampingFraction: 0.8)) { swipe = 0 }
    }

    /// On the first few visits, until the first swipe, the board slides a
    /// little toward the next opponent and springs back: it can be swiped.
    private func peek() async {
        guard !swiped, peeks < 3 else { return }
        try? await Task.sleep(for: .seconds(0.9))
        guard !swiped, swipe == 0 else { return }
        let last = seat == Opponent.allCases.last
        withAnimation(.easeInOut(duration: 0.32)) { swipe = last ? 26 : -26 }
        try? await Task.sleep(for: .seconds(0.32))
        withAnimation(.spring(response: 0.5, dampingFraction: 0.55)) { swipe = 0 }
        peeks += 1
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
            }
            // The quieter ways in: plain words side by side, each a full-size target.
            HStack(spacing: 0) {
                if savedHere != nil {
                    Button { Haptics.tap(); start(.engine, seat) } label: {
                        Label("New game", systemImage: "plus")
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .contentShape(.rect)
                    }
                    .buttonStyle(.plain)
                }
                friendLink
            }
            .font(.system(size: 15, weight: .semibold))
            .foregroundStyle(Palette.ink)
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
            // A quiet line, not a card: your games are there when you want them.
            Button { Haptics.tap(); history = true } label: {
                // How it's going as a shape, not a number: the same line the
                // sheet opens with.
                HStack(spacing: 10) {
                    let trend = recent.accuracyTrend
                    if trend.count >= 3 {
                        AccuracyTrend(values: trend, compact: true).frame(width: 36, height: 14)
                    }
                    Text("Your games").font(.system(size: 14, weight: .semibold)).foregroundStyle(Palette.ink)
                    Image(systemName: "chevron.right").font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(Palette.mist)
                }
                .frame(minHeight: 44)
                .contentShape(.rect)
            }
            .buttonStyle(PressableStyle(scale: 0.98))
        }
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

/// Your games as a quiet journal: how your accuracy is trending, then each
/// game by day, with a small mark for how it ended.
private struct RecentGames: View {
    let records: [GameRecord]
    var limit = 3

    /// Games that were played, not abandoned on the first move.
    private var shown: [GameRecord] { Array(records.filter { $0.moves >= 2 }.prefix(limit)) }

    private var trend: [Int] { records.accuracyTrend }


    /// The games grouped by the day they were played, newest first.
    private var days: [(day: Date, games: [GameRecord])] {
        let cal = Calendar.current
        var out: [(day: Date, games: [GameRecord])] = []
        for r in shown {
            let d = cal.startOfDay(for: r.date)
            if out.last?.day == d { out[out.count - 1].games.append(r) } else { out.append((d, [r])) }
        }
        return out
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Your games").font(Typeface.display(24)).tracking(-0.4)
            // Two figures, each with what it means, then the line they come from.
            if trend.count >= 2 {
                HStack(alignment: .firstTextBaseline) {
                    figure("\(trend.reduce(0, +) / trend.count)%", L("Accuracy · last \(trend.count) games"))
                    Spacer()
                    if Stats.greatMoves > 0 {
                        figure("\(Stats.greatMoves)", L("Great moves"), trailing: true)
                    }
                }
                .padding(.top, 16)
            }
            if trend.count >= 3 {
                AccuracyTrend(values: trend)
                    .frame(height: 44)
                    .padding(.top, 16)
            }
            if shown.isEmpty {
                Text("Your finished games will show up here, with how accurately you played.")
                    .font(.system(size: 14)).foregroundStyle(Palette.mist)
                    .padding(.top, 16)
            }
            ForEach(days, id: \.day) { group in
                Text(dayLabel(group.day))
                    .font(.system(size: 12, weight: .semibold)).foregroundStyle(Palette.mist)
                    .padding(.top, 24).padding(.bottom, 4)
                ForEach(group.games) { r in row(r) }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func figure(_ value: String, _ label: String, trailing: Bool = false) -> some View {
        VStack(alignment: trailing ? .trailing : .leading, spacing: 2) {
            Text(value).font(Typeface.display(28, .bold)).monospacedDigit()
            Text(label).font(.system(size: 12.5)).foregroundStyle(Palette.mist)
        }
    }

    private func row(_ r: GameRecord) -> some View {
        let twoPlayers = r.opponent == "Pass & play"
        return HStack(spacing: 14) {
            ResultMark(outcome: twoPlayers ? nil : r.outcome)
            VStack(alignment: .leading, spacing: 2) {
                Text(twoPlayers ? L("Pass & play") : r.opponentName)
                    .font(.system(size: 16, weight: .semibold))
                Text(detail(r, twoPlayers: twoPlayers))
                    .font(.system(size: 13)).foregroundStyle(Palette.mist)
            }
            Spacer()
            if !twoPlayers, let a = r.accuracy {
                Text("\(a)%").font(Typeface.display(17)).monospacedDigit()
            }
        }
        .padding(.vertical, 10)
        .accessibilityElement(children: .combine)
    }

    private func detail(_ r: GameRecord, twoPlayers: Bool) -> String {
        let moves = r.moves == 1 ? L("1 move") : L("\(r.moves) moves")
        guard !twoPlayers else { return moves }
        let result = ["won": L("Won"), "lost": L("Lost"), "draw": L("Draw")][r.outcome] ?? ""
        return result.isEmpty ? moves : "\(result) · \(moves)"
    }

    private func dayLabel(_ day: Date) -> String {
        let cal = Calendar.current
        if cal.isDateInToday(day) { return L("Today") }
        if cal.isDateInYesterday(day) { return L("Yesterday") }
        return day.formatted(.dateTime.weekday(.wide).day().month(.abbreviated))
    }
}

/// How a game ended, as a small mark: filled green for a win, a ring for a
/// loss, a quiet dot for a draw, a hollow pair for a game for two.
private struct ResultMark: View {
    let outcome: String?

    var body: some View {
        Group {
            switch outcome {
            case "won": Circle().fill(Palette.great)
            case "lost": Circle().strokeBorder(Palette.mist.opacity(0.7), lineWidth: 1.5)
            case "draw": Circle().fill(Palette.mist.opacity(0.5))
            default: Circle().strokeBorder(Palette.mist.opacity(0.35), style: StrokeStyle(lineWidth: 1.5, dash: [2, 2]))
            }
        }
        .frame(width: 10, height: 10)
        .accessibilityHidden(true)
    }
}

extension [GameRecord] {
    /// Accuracy in the last ten games against an opponent, oldest first.
    var accuracyTrend: [Int] {
        [Int](filter { $0.opponent != "Pass & play" }.compactMap(\.accuracy).prefix(10).reversed())
    }
}

/// Accuracy over the last games as one soft line, the latest marked: the same
/// hand as the game's arc on the finish card.
private struct AccuracyTrend: View {
    let values: [Int]
    /// Small, inline: a thinner line and no marker.
    var compact = false

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width, h = geo.size.height
            let lo = Double(max(0, (values.min() ?? 0) - 8)), hi = Double(min(100, (values.max() ?? 100) + 8))
            let pts = values.indices.map { i in
                CGPoint(x: w * CGFloat(i) / CGFloat(max(1, values.count - 1)),
                        y: h * (1 - CGFloat((Double(values[i]) - lo) / max(1, hi - lo))))
            }
            ZStack(alignment: .topLeading) {
                GameArc.smooth(pts)
                    .stroke(Palette.accent, style: StrokeStyle(lineWidth: compact ? 1.5 : 1.75, lineCap: .round, lineJoin: .round))
                if !compact, let last = pts.last {
                    Circle().fill(Palette.accent)
                        .frame(width: 8, height: 8)
                        .overlay(Circle().strokeBorder(Palette.surface, lineWidth: 2))
                        .position(last)
                }
            }
        }
        .accessibilityHidden(true)
    }
}

private struct SettingsSheet: View {
    var learn: () -> Void = {}
    var welcome: () -> Void = {}
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
            Button { welcome() } label: {
                HStack {
                    Label("How it works", systemImage: "sparkles")
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
    @Environment(\.boardSpace) private var boardSpace
    /// Fades in everything around the board as the game arrives.
    @State private var shown = false
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
                            CapturedRow(pieces: model.captured(by: "b"), lead: model.materialLead(for: "b")).chrome(shown, from: .top, delay: 0.16)
                            BoardView(model: model).heroBoard(boardSpace)
                            CapturedRow(pieces: model.captured(by: "w"), lead: model.materialLead(for: "w")).chrome(shown, delay: 0.18)
                        }
                        .frame(width: min(geo.size.height - 70, geo.size.width - 420))
                        VStack(spacing: 10) {
                            TopBar(model: model, back: back)
                            coach
                            Spacer(minLength: 0)
                            Dock(model: model)
                        }
                        .chrome(shown, delay: 0.2)
                        .frame(width: 360)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .padding(.vertical, 12)
                } else {
                    VStack(spacing: 6) {
                        TopBar(model: model, back: back).chrome(shown, from: .top, delay: 0.12)
                        CapturedRow(pieces: model.captured(by: "b"), lead: model.materialLead(for: "b")).chrome(shown, from: .top, delay: 0.16)
                        BoardView(model: model).heroBoard(boardSpace)
                        CapturedRow(pieces: model.captured(by: "w"), lead: model.materialLead(for: "w")).chrome(shown, delay: 0.18)
                        coach.chrome(shown, delay: 0.22)
                        Spacer(minLength: 0)
                        Dock(model: model).chrome(shown, delay: 0.28)
                    }
                    .frame(maxWidth: min(760, geo.size.height * 0.6))
                    .frame(maxWidth: .infinity)
                }
            }
            .padding(.horizontal, 14)
            .padding(.bottom, 8)
            .foregroundStyle(Palette.ink)

            if let finish = model.finish {
                FinishCard(finish: finish, king: model.myColor == "b" ? "k" : "K", share: model.finishMoment,
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
        .onAppear { shown = true }
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
            // Just the name, with dots beside it while they think: the board's
            // light and the coach already say when it's your move.
            HStack(spacing: 8) {
                Text(model.mode == .engine ? model.opponent.name : model.mode == .friend ? model.friendName : "Pass & play")
                    .font(Typeface.display(17, .bold))
                if model.thinking || (model.mode == .friend && !model.isOver && model.turn != model.myColor) {
                    ThinkingDots().transition(.opacity)
                }
            }
            .animation(.easeOut(duration: 0.2), value: model.thinking)
            Spacer()
            // No live score: accuracy waits for the game-over card, so play feels unjudged.
        }
        .padding(.top, 4)
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
struct CoachView: View {
    let card: CoachCard?
    let opponent: Opponent
    var friend: String?
    /// The intro keeps its second lines quiet, under a much larger heading.
    var compact = false

    var body: some View {
        ZStack {
            if let card {
                HStack(spacing: 12) {
                    badge(card)
                    // Claims the full width, so a line only wraps when it reaches
                    // the card's edge (a Spacer beside it made the layout split
                    // the room and wrap the sentence early).
                    VStack(alignment: .leading, spacing: 2) {
                        Text(card.title).font(Typeface.display(17)).foregroundStyle(titleColor(card.kind))
                        if !card.text.isEmpty {
                            Text(card.text).font(.system(size: compact ? 12 : 13.5)).foregroundStyle(Palette.mist)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
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

    private var canHint: Bool {
        model.findingHint || (model.playerToMove && (model.hintsLeft > 0 || model.hintIsFree))
    }

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
                    .disabled(model.moves.isEmpty)
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
                    // Against a friend there are no hints, and the top bar says
                    // whose turn it is: the dock keeps only its menu.
                    Spacer()
                } else {
                Button {
                    guard !model.findingHint else { return }
                    Haptics.tap()
                    hints += 1
                    model.hint()
                } label: {
                    HStack(spacing: 6) {
                        Image(systemName: "sparkles").symbolEffect(.bounce, value: hints)
                        Text(model.hintIsFree && !model.hintRevealed ? "Show the move" : model.hintsLeft > 0 || model.findingHint ? "Hint" : "No hints")
                            .contentTransition(.opacity)
                        if model.hintsLeft > 0 && !(model.hintIsFree && !model.hintRevealed) && !model.findingHint {
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
                .animation(.easeInOut(duration: 0.25), value: model.hintIsFree)
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
    /// The player's own king, for the emblem.
    let king: Character
    let share: ShareMoment?
    let again: () -> Void
    /// Starts a game against the opponent this win just opened.
    let tryNext: (() -> Void)?
    let home: () -> Void

    var body: some View {
        ZStack {
            // A calm wash in the app's own evening: the board recedes without
            // smearing into a blur.
            LinearGradient(colors: [Palette.backgroundTop.opacity(0.88), Palette.backgroundBottom.opacity(0.96)],
                           startPoint: .top, endPoint: .bottom)
                .ignoresSafeArea()
            VStack(alignment: .leading, spacing: 0) {
                // What happened, in a line.
                HStack(spacing: 14) {
                    FinishEmblem(outcome: finish.outcome, king: king)
                    VStack(alignment: .leading, spacing: 3) {
                        Text(finish.title).font(Typeface.display(24)).tracking(-0.4)
                            .lineLimit(2).minimumScaleFactor(0.8)
                        Text(finish.firstWin ? "Your first win · \(finish.text)" : finish.text)
                            .font(.system(size: 14)).foregroundStyle(Palette.mist)
                            .lineLimit(2)
                    }
                    Spacer(minLength: 0)
                }

                // How it went: the arc of the game, and what it says.
                if !finish.arc.isEmpty {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Your chances").font(.system(size: 12, weight: .semibold)).foregroundStyle(Palette.mist)
                        GameArc(arc: finish.arc,
                                best: finish.best.map { $0.ply + 1 },
                                turn: finish.turnPly.map { $0 + 1 })
                            .frame(height: 78)
                        HStack(spacing: 14) {
                            if let best = finish.best {
                                legend(Palette.great, best.san.isEmpty ? L("Best · move \(best.moveNumber)")
                                                                       : "\(L("Best · move \(best.moveNumber)")) · \(best.san)")
                            }
                            if let turn = finish.turnPly {
                                legend(Palette.blunder, L("Turned · move \(turn / 2 + 1)"))
                            }
                        }
                    }
                    .padding(16)
                    .background(Palette.statFill, in: .rect(cornerRadius: 16, style: .continuous))
                    .padding(.top, 16)

                    if let insight = finish.insight {
                        Text(insight).font(.system(size: 15, weight: .semibold))
                            .fixedSize(horizontal: false, vertical: true)
                            .padding(.top, 16)
                    }
                }

                // The numbers, as tiles you can read at a glance.
                if !stats.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        if finish.sides != nil {
                            Text("Great moves").font(.system(size: 12, weight: .semibold)).foregroundStyle(Palette.mist)
                        }
                        HStack(spacing: 8) {
                            ForEach(stats, id: \.label) { stat in
                                VStack(spacing: 1) {
                                    Text(stat.value).font(Typeface.display(20, .bold)).monospacedDigit()
                                    Text(stat.label).font(.system(size: 11.5, weight: .medium)).foregroundStyle(Palette.mist)
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                                .background(Palette.statFill, in: .rect(cornerRadius: 14, style: .continuous))
                            }
                        }
                    }
                    .padding(.top, 16)
                }

                // What comes next.
                Group {
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
                                .foregroundStyle(Palette.mist)
                        }
                        .font(.system(size: 13.5, weight: .medium))
                    }
                }
                .frame(maxWidth: .infinity)
                .padding(.top, 16)

                // The way on: one clear next step, then Home and Share beside
                // each other, quiet.
                VStack(spacing: 8) {
                    if let next = finish.unlocked, let tryNext {
                        Button("Play the \(next.name)") { tryNext() }.buttonStyle(PrimaryButtonStyle())
                    } else {
                        Button("Play again", action: again).buttonStyle(PrimaryButtonStyle())
                    }
                    HStack(spacing: 10) {
                        Button { Haptics.tap(); home() } label: {
                            Label("Home", systemImage: "house")
                                .frame(maxWidth: .infinity, minHeight: 48)
                                .contentShape(.capsule)
                        }
                        .buttonStyle(QuietButtonStyle())
                        if let share {
                            ShareMomentButton(moment: share, label: "Share")
                                .buttonStyle(QuietButtonStyle())
                        }
                    }
                    .font(.system(size: 15, weight: .semibold))
                }
                .padding(.top, 24)
            }
            .padding(24)
            .frame(maxWidth: 440)
            .background {
                let card = RoundedRectangle(cornerRadius: 30, style: .continuous)
                card.fill(Palette.surface)
                    // A faint light of the outcome's colour from the emblem's corner.
                    .overlay {
                        RadialGradient(colors: [FinishEmblem.tint(finish.outcome).opacity(0.16), .clear],
                                       center: .topLeading, startRadius: 0, endRadius: 300)
                            .clipShape(card)
                    }
                    .overlay(card.strokeBorder(Palette.hairline, lineWidth: 1))
                    .shadow(color: .black.opacity(0.28), radius: 30, y: 18)
            }
            .overlay(alignment: .topLeading) {
                if finish.outcome == .won { Celebration().offset(x: 54, y: 54) }
            }
            .padding(16)
        }
        .foregroundStyle(Palette.ink)
    }

    private func legend(_ color: Color, _ text: String) -> some View {
        HStack(spacing: 6) {
            Circle().fill(color).frame(width: 7, height: 7)
            Text(text).font(.system(size: 12, weight: .medium)).foregroundStyle(Palette.mist).lineLimit(1)
        }
    }

    /// Accuracy, great moves and the game's length. Pass and play shows each
    /// side's great moves instead.
    private var stats: [(value: String, label: String)] {
        if let s = finish.sides {
            return s.white + s.black == 0 ? [] : [("\(s.white)", L("White")), ("\(s.black)", L("Black"))]
        }
        var out: [(value: String, label: String)] = []
        if let a = finish.accuracy { out.append(("\(a)%", L("Accuracy"))) }
        out.append(("\(finish.great)", L("Great")))
        if finish.plies > 0 { out.append(("\((finish.plies + 1) / 2)", L("Moves"))) }
        return out
    }
}

/// The game as one line: the player's chance to win from the first move to
/// the last, green above even and coral below, with the best move and the
/// turning point marked. It draws itself in once.
private struct GameArc: View {
    let arc: [Double]
    /// Indexes into `arc` for the two markers.
    let best: Int?
    let turn: Int?
    @State private var drawn = false

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width, h = geo.size.height
            let pts = arc.indices.map { i in
                CGPoint(x: w * CGFloat(i) / CGFloat(max(1, arc.count - 1)), y: h * (1 - CGFloat(arc[i])))
            }
            let line = Self.smooth(pts)
            let area = Path { p in
                p.addPath(line)
                p.addLine(to: CGPoint(x: w, y: h / 2))
                p.addLine(to: CGPoint(x: 0, y: h / 2))
                p.closeSubpath()
            }
            ZStack(alignment: .topLeading) {
                Path { p in p.move(to: CGPoint(x: 0, y: h / 2)); p.addLine(to: CGPoint(x: w, y: h / 2)) }
                    .stroke(Palette.mist.opacity(0.35), style: StrokeStyle(lineWidth: 1, dash: [3, 4]))
                area.fill(Palette.great.opacity(0.24))
                    .mask(Rectangle().frame(height: h / 2).frame(maxHeight: .infinity, alignment: .top))
                area.fill(Palette.blunder.opacity(0.24))
                    .mask(Rectangle().frame(height: h / 2).frame(maxHeight: .infinity, alignment: .bottom))
                    .opacity(drawn ? 1 : 0)
                line.trim(from: 0, to: drawn ? 1 : 0)
                    .stroke(Palette.ink.opacity(0.8), style: StrokeStyle(lineWidth: 1.75, lineCap: .round, lineJoin: .round))
                ForEach([(best, Palette.great), (turn, Palette.blunder)].compactMap { m in m.0.map { ($0, m.1) } }, id: \.0) { i, color in
                    if pts.indices.contains(i) {
                        Circle().fill(color)
                            .frame(width: 9, height: 9)
                            .overlay(Circle().strokeBorder(Palette.surface, lineWidth: 2))
                            .frame(width: 13, height: 13)
                            .position(pts[i])
                            .scaleEffect(drawn ? 1 : 0.01, anchor: UnitPoint(x: pts[i].x / w, y: pts[i].y / h))
                            .animation(.spring(response: 0.35, dampingFraction: 0.55).delay(0.25 + 1.0 * Double(i) / Double(arc.count)), value: drawn)
                    }
                }
            }
            .animation(.easeInOut(duration: 1.1).delay(0.25), value: drawn)
        }
        .onAppear { drawn = true }
        .accessibilityHidden(true)
    }

    /// A gentle curve through the points: each segment bends through the
    /// midpoints, so the line reads as a flow, not a zigzag.
    static func smooth(_ pts: [CGPoint]) -> Path {
        Path { p in
            guard let first = pts.first else { return }
            p.move(to: first)
            for i in 1..<pts.count {
                let a = pts[i - 1], b = pts[i]
                let mid = CGPoint(x: (a.x + b.x) / 2, y: (a.y + b.y) / 2)
                p.addQuadCurve(to: mid, control: a)
                if i == pts.count - 1 { p.addQuadCurve(to: b, control: b) }
            }
        }
    }
}

/// The outcome at a glance: your king on a disc, standing in honey for a win,
/// tipped over on limestone for a loss (the way a game is resigned), beside
/// its rival on slate for a draw.
private struct FinishEmblem: View {
    let outcome: Finish.Outcome
    let king: Character

    static func tint(_ outcome: Finish.Outcome) -> Color {
        switch outcome {
        case .won: Palette.lastMove
        case .lost: Palette.squareLight
        case .draw: Palette.squareDark
        }
    }

    var body: some View {
        let tint = Self.tint(outcome)
        ZStack {
            Circle()
                .fill(LinearGradient(colors: [tint.mix(with: .white, by: 0.25), tint],
                                     startPoint: .topLeading, endPoint: .bottomTrailing))
            Circle().strokeBorder(.white.opacity(0.3), lineWidth: 1)
            switch outcome {
            case .won:
                GlassPiece(code: king).frame(width: 40, height: 40).offset(y: -1)
            case .lost:
                GlassPiece(code: king, grounded: false).frame(width: 40, height: 40)
                    .rotationEffect(.degrees(-55), anchor: UnitPoint(x: 0.5, y: 0.86))
                    .offset(x: 10, y: -3)
            case .draw:
                HStack(spacing: -13) {
                    GlassPiece(code: king).frame(width: 34, height: 34)
                    GlassPiece(code: king == "K" ? "k" : "K").frame(width: 34, height: 34)
                }
                .offset(y: -1)
            }
        }
        .frame(width: 60, height: 60)
        .shadow(color: tint.opacity(0.4), radius: 12, y: 5)
        .accessibilityHidden(true)
    }
}

/// A secondary action: plain words that dim under the finger, no frame.
private struct QuietButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .foregroundStyle(Palette.ink)
            .opacity(configuration.isPressed ? 0.5 : 1)
            .animation(.easeOut(duration: 0.15), value: configuration.isPressed)
    }
}

#Preview {
    RootView()
}

/// A win: glass flecks in the app's colours rise and fall once.
struct Celebration: View {
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


/// Picks a primitive button style at runtime, so a button can switch between
/// prominent and plain glass.
struct AnyPrimitiveButtonStyle: PrimitiveButtonStyle {
    private let make: (Configuration) -> AnyView
    init<S: PrimitiveButtonStyle>(_ style: S) { make = { AnyView(style.makeBody(configuration: $0)) } }
    func makeBody(configuration: Configuration) -> some View { make(configuration) }
}
