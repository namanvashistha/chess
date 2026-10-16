import Chesscore
import SwiftUI

/// The app: home, and the game it starts.
struct RootView: View {
    @State private var model = GameModel()
    @State private var playing = false

    var body: some View {
        ZStack {
            AppBackground()
            if playing {
                GameScreen(model: model) { withAnimation(.smooth(duration: 0.4)) { playing = false } }
                    .transition(.move(edge: .trailing).combined(with: .opacity))
            } else {
                HomeScreen { mode, opponent in
                    model.start(mode, opponent: opponent)
                    withAnimation(.smooth(duration: 0.4)) { playing = true }
                }
                .transition(.move(edge: .leading).combined(with: .opacity))
            }
        }
        .sensoryFeedback(trigger: model.feedback) { _, new in new?.kind.haptic }
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
    @AppStorage("opponent") private var opponentRaw = Opponent.flint.rawValue

    private var opponent: Opponent { Opponent(rawValue: opponentRaw) ?? .flint }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(greeting)
                    Text("Ready for a game?").foregroundStyle(Palette.amberText)
                }
                .font(Typeface.display(30))
                .tracking(-0.5)
                .padding(.top, 8)

                HStack(spacing: 8) {
                    Chip(text: Stats.streak > 0 ? "\(Stats.streak)-day streak" : "Start a streak", symbol: "flame.fill", tint: .orange)
                    Chip(text: Stats.averageAccuracy.map { "\($0)% accuracy" } ?? "No games yet", symbol: "scope", tint: Palette.mist)
                }

                HeroBoard()

                HStack(spacing: 8) {
                    ForEach(Opponent.allCases) { o in
                        OpponentButton(opponent: o, selected: o == opponent) {
                            opponentRaw = o.rawValue
                            Sound.shared.play(.select)
                        }
                    }
                }

                Button { start(.engine, opponent) } label: {
                    Label("Play", systemImage: "arrow.right").labelStyle(TrailingIcon())
                }
                .buttonStyle(AmberButtonStyle())

                HStack(spacing: 8) {
                    Button { start(.passAndPlay, opponent) } label: {
                        Text("Pass & play").frame(maxWidth: .infinity, minHeight: 40)
                    }
                    Button {} label: {
                        VStack(spacing: 0) {
                            Text("Play a friend")
                            Text("Soon").font(.caption2).foregroundStyle(Palette.mist)
                        }
                        .frame(maxWidth: .infinity, minHeight: 40)
                    }
                    .disabled(true)
                }
                .font(Typeface.display(15, .bold))
                .buttonStyle(.glass)
                .controlSize(.large)
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 24)
        }
        .scrollBounceBehavior(.basedOnSize)
        .foregroundStyle(Palette.ink)
    }

    private var greeting: String {
        switch Calendar.current.component(.hour, from: .now) {
        case 5..<12: "Good morning."
        case 12..<17: "Good afternoon."
        default: "Good evening."
        }
    }
}

private struct TrailingIcon: LabelStyle {
    func makeBody(configuration: Configuration) -> some View {
        HStack(spacing: 10) { configuration.title; configuration.icon }
    }
}

private struct Chip: View {
    let text: String
    let symbol: String
    let tint: Color

    var body: some View {
        Label { Text(text) } icon: { Image(systemName: symbol).foregroundStyle(tint) }
            .font(.system(size: 13, weight: .semibold))
            .padding(.horizontal, 11).padding(.vertical, 7)
            .glassEffect(.regular, in: .capsule)
    }
}

private struct OpponentButton: View {
    let opponent: Opponent
    let selected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 4) {
                Image(systemName: ["circle.fill", "diamond.fill", "star.fill"][opponent.rawValue])
                    .font(.system(size: 22))
                    .foregroundStyle([Palette.great, Palette.amber, Palette.blunder][opponent.rawValue])
                Text(opponent.name).font(Typeface.display(15, .bold))
                Text(opponent.subtitle).font(.system(size: 11.5)).foregroundStyle(Palette.mist)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 12)
            .contentShape(.rect)
        }
        .buttonStyle(.plain)
        .glassEffect(selected ? .regular.tint(Palette.amber.opacity(0.25)).interactive() : .regular.interactive(),
                     in: .rect(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).strokeBorder(Palette.amber, lineWidth: selected ? 1.5 : 0)
        }
        .animation(.easeOut(duration: 0.15), value: selected)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }
}

/// The home screen's board: an opening position, seen at an angle, fading out.
private struct HeroBoard: View {
    private static let fen = "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R"

    var body: some View {
        ZStack(alignment: .bottomLeading) {
            GeometryReader { geo in
                StaticBoard(fen: Self.fen)
                    .frame(width: geo.size.width * 0.92)
                    .rotation3DEffect(.degrees(52), axis: (x: 1, y: 0, z: 0), anchor: .top, perspective: 0.55)
                    .offset(x: geo.size.width * 0.04, y: geo.size.width * 0.03)
            }
            LinearGradient(stops: [.init(color: .clear, location: 0.35),
                                   .init(color: Palette.backgroundTop.opacity(0.88), location: 0.72),
                                   .init(color: Palette.backgroundTop, location: 1)],
                           startPoint: .top, endPoint: .bottom)
            VStack(alignment: .leading, spacing: 2) {
                Text("Play the engine").font(Typeface.display(22))
                Text("It explains every move as you go.").font(.system(size: 13.5)).foregroundStyle(Palette.mist)
            }
            .padding(18)
        }
        .aspectRatio(1.5, contentMode: .fit)
        .clipShape(.rect(cornerRadius: 22))
        .glassEffect(.regular, in: .rect(cornerRadius: 22))
        .accessibilityElement(children: .combine)
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
                ForEach(0..<64, id: \.self) { i in
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
                CoachView(card: model.coach)
                    .padding(.top, 4)
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
                Image(systemName: "chevron.left").font(.system(size: 17, weight: .semibold)).frame(width: 40, height: 40)
            }
            .buttonStyle(.glass)
            .buttonBorderShape(.circle)
            .accessibilityLabel("Home")

            VStack(alignment: .leading, spacing: 1) {
                Text(model.mode == .engine ? model.opponent.name : "Pass & play").font(Typeface.display(17, .bold))
                HStack(spacing: 6) {
                    Text(stateText).font(.system(size: 12.5)).foregroundStyle(Palette.mist)
                    if model.thinking { ThinkingDots() }
                }
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 1) {
                Text(model.accuracy.map { "\($0)%" } ?? "–").font(Typeface.display(17)).monospacedDigit()
                    .contentTransition(.numericText())
                    .animation(.snappy, value: model.accuracy)
                Text("ACCURACY").font(.system(size: 10, weight: .semibold)).tracking(0.6).foregroundStyle(Palette.mist)
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
                Circle().fill(Palette.amberText).frame(width: 4, height: 4)
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
/// sentence.
private struct CoachView: View {
    let card: CoachCard?

    var body: some View {
        ZStack {
            if let card {
                HStack(spacing: 12) {
                    Text(card.badge)
                        .font(Typeface.display(20))
                        .foregroundStyle(card.kind == .coach ? Palette.amberInk : Palette.ink)
                        .frame(width: 46, height: 46)
                        .background(badgeFill(card.kind), in: .rect(cornerRadius: 14))
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
                .transition(.asymmetric(insertion: .scale(scale: 0.92, anchor: .bottom).combined(with: .opacity),
                                        removal: .opacity))
                .accessibilityElement(children: .combine)
            }
        }
        .frame(minHeight: 74)
        .animation(.spring(response: 0.45, dampingFraction: 0.68), value: card?.id)
    }

    private func badgeFill(_ k: CoachCard.Kind) -> AnyShapeStyle {
        switch k {
        case .great: AnyShapeStyle(Palette.greatFill)
        case .good: AnyShapeStyle(Palette.goodFill)
        case .inaccuracy: AnyShapeStyle(Palette.inaccuracyFill)
        case .mistake, .warning: AnyShapeStyle(Palette.mistakeFill)
        case .blunder: AnyShapeStyle(Palette.blunderFill)
        case .coach: AnyShapeStyle(LinearGradient(colors: [Palette.amberTop, Palette.amber], startPoint: .top, endPoint: .bottom))
        }
    }

    private func titleColor(_ k: CoachCard.Kind) -> Color {
        switch k {
        case .great: Palette.great
        case .inaccuracy: Palette.inaccuracy
        case .mistake, .warning: Palette.mistake
        case .blunder: Palette.blunder
        case .good, .coach: Palette.ink
        }
    }
}

private struct Dock: View {
    let model: GameModel

    var body: some View {
        GlassEffectContainer(spacing: 8) {
            HStack(spacing: 8) {
                Button(action: model.undo) { Label("Undo", systemImage: "arrow.uturn.backward").frame(maxWidth: .infinity) }
                    .buttonStyle(.glass)
                    .disabled(model.busy || model.losses.isEmpty)
                Button { model.start(model.mode, opponent: model.opponent) } label: {
                    Label("New", systemImage: "arrow.clockwise").frame(maxWidth: .infinity)
                }
                .buttonStyle(.glass)
                .disabled(model.busy)
                Button(action: model.hint) { Label("Show me", systemImage: "sparkles").frame(maxWidth: .infinity) }
                    .buttonStyle(.glassProminent)
                    .tint(Color(hex: 0x2F9E5C))
                    .disabled(!model.playerToMove)
            }
            .font(Typeface.display(14, .bold))
            .lineLimit(1)
            .minimumScaleFactor(0.85)
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
                Button("Play again", action: again).buttonStyle(AmberButtonStyle())
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
