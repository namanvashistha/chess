import Chesscore
import SwiftUI

struct GameView: View {
    @State private var model = GameModel()
    @State private var tilt = Tilt()
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        ZStack {
            Background()
            VStack(spacing: 10) {
                Header()
                PlayerBar(model: model, side: "b", name: "Engine",
                          detail: "Thinks for under a second")
                BoardView(model: model, tilt: tilt)
                PlayerBar(model: model, side: "w", name: "You", detail: "Playing White")
                MoveStrip(model: model)
                Spacer(minLength: 0)
                Toolbar(model: model)
            }
            .padding(.horizontal, 16)
            if model.isOver {
                GameOverCard(model: model)
                    .transition(.opacity.combined(with: .scale(scale: 0.96)))
            }
        }
        .animation(.easeOut(duration: 0.3).delay(0.4), value: model.isOver)
        .sensoryFeedback(trigger: model.feedback) { _, new in new?.kind.haptic }
        .onChange(of: scenePhase, initial: true) { _, phase in
            if phase == .active { tilt.start() } else { tilt.stop() }
        }
    }
}

private extension Feedback.Kind {
    var haptic: SensoryFeedback {
        switch self {
        case .selection: .selection
        case .move: .impact(weight: .light)
        case .capture: .impact(weight: .medium)
        case .castle: .impact(flexibility: .rigid)
        case .check: .warning
        case .mate: .success
        }
    }
}

private struct Background: View {
    var body: some View {
        Palette.background
            .overlay(alignment: .top) {
                RadialGradient(colors: [Palette.accent.opacity(0.16), .clear],
                               center: .top, startRadius: 0, endRadius: 420)
                    .frame(height: 420)
                    .offset(y: -80)
            }
            .ignoresSafeArea()
    }
}

private struct Header: View {
    var body: some View {
        HStack(spacing: 10) {
            Image("wN")
                .resizable()
                .frame(width: 22, height: 22)
                .padding(5)
                .background(LinearGradient(colors: [Palette.accent, Color(hex: 0x4577E0)],
                                           startPoint: .topLeading, endPoint: .bottomTrailing),
                            in: RoundedRectangle(cornerRadius: 9, style: .continuous))
            Text("Chess").font(.system(size: 18, weight: .heavy)).tracking(-0.3)
            Spacer()
            Text("Vs Engine")
                .font(.system(size: 12.5, weight: .semibold))
                .foregroundStyle(Palette.muted)
                .padding(.horizontal, 10).padding(.vertical, 5)
                .background(Palette.surfaceRaised.opacity(0.8), in: Capsule())
        }
        .padding(.top, 4)
        .padding(.bottom, 4)
    }
}

private struct PlayerBar: View {
    let model: GameModel
    let side: String
    let name: String
    let detail: String

    var body: some View {
        let active = model.turn == side && !model.isOver
        let thinking = model.thinking && side == model.engineSide
        HStack(spacing: 10) {
            Image(side == "w" ? "wK" : "bK")
                .resizable()
                .frame(width: 24, height: 24)
                .frame(width: 32, height: 32)
                .background(Palette.surfaceRaised, in: Circle())
            VStack(alignment: .leading, spacing: 1) {
                HStack(spacing: 6) {
                    Text(name).font(.system(size: 15, weight: .semibold))
                    if active { TurnPill(thinking: thinking) }
                }
                Text(detail).font(.system(size: 12)).foregroundStyle(Palette.muted)
            }
            Spacer(minLength: 8)
            CapturedRow(pieces: model.captured(by: side), lead: model.materialLead(for: side))
        }
        .frame(minHeight: 44)
        .animation(.easeOut(duration: 0.2), value: active)
    }
}

private struct TurnPill: View {
    let thinking: Bool

    var body: some View {
        HStack(spacing: 4) {
            Circle().fill(Palette.good).frame(width: 6, height: 6)
                .phaseAnimator([0.45, 1.0]) { v, p in v.opacity(p) } animation: { _ in .easeInOut(duration: 0.9) }
            Text(thinking ? "Thinking" : "To move")
        }
        .font(.system(size: 10, weight: .bold))
        .tracking(0.8)
        .textCase(.uppercase)
        .foregroundStyle(Palette.good)
        .transition(.opacity)
    }
}

private struct CapturedRow: View {
    let pieces: [Character]
    let lead: Int

    var body: some View {
        HStack(spacing: 6) {
            HStack(spacing: -7) {
                ForEach(Array(pieces.enumerated()), id: \.offset) { _, p in
                    Image((p.isUppercase ? "w" : "b") + p.uppercased())
                        .resizable().frame(width: 17, height: 17)
                }
            }
            if lead > 0 {
                Text("+\(lead)").font(.system(size: 12, weight: .medium)).monospacedDigit()
                    .foregroundStyle(Palette.muted)
            }
        }
    }
}

private struct MoveStrip: View {
    let model: GameModel

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                EyebrowLabel(text: "Opening")
                Text(openingText)
                    .font(.system(size: 13, weight: .semibold))
                    .contentTransition(.opacity)
                    .animation(.easeOut(duration: 0.2), value: openingText)
            }
            ScrollViewReader { proxy in
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 2) {
                        if model.sans.isEmpty {
                            Text("Tap a white piece to start").foregroundStyle(Palette.faint)
                        }
                        ForEach(Array(model.sans.enumerated()), id: \.offset) { k, san in
                            if k % 2 == 0 {
                                Text("\(k / 2 + 1).").foregroundStyle(Palette.faint).padding(.leading, 6)
                            }
                            Text(san)
                                .fontWeight(k == model.sans.count - 1 ? .semibold : .regular)
                                .padding(.horizontal, 6).padding(.vertical, 3)
                                .background(k == model.sans.count - 1 ? Palette.surfaceRaised : .clear,
                                            in: RoundedRectangle(cornerRadius: 7, style: .continuous))
                                .id(k)
                        }
                    }
                    .font(.system(size: 14))
                    .monospacedDigit()
                }
                .onChange(of: model.sans.count) { _, n in
                    withAnimation(.easeOut(duration: 0.25)) { proxy.scrollTo(n - 1, anchor: .trailing) }
                }
            }
        }
        .padding(.top, 6)
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var openingText: String {
        if model.sans.isEmpty { return "Starting position" }
        return Notation.opening(model.sans) ?? "Out of the book"
    }
}

private struct Toolbar: View {
    let model: GameModel

    var body: some View {
        HStack {
            Spacer()
            ToolButton(title: "Undo", symbol: "arrow.uturn.backward", action: model.undo)
                .disabled(model.sans.isEmpty || model.thinking)
            Spacer()
            ToolButton(title: "New game", symbol: "plus", action: model.newGame)
                .disabled(model.thinking)
            Spacer()
        }
        .padding(.top, 8)
        .overlay(alignment: .top) { Rectangle().fill(Palette.border).frame(height: 1) }
    }
}

private struct ToolButton: View {
    let title: String
    let symbol: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 3) {
                Image(systemName: symbol).font(.system(size: 18, weight: .medium))
                Text(title).font(.system(size: 11, weight: .semibold))
            }
            .frame(minWidth: 64, minHeight: 44)
        }
        .buttonStyle(.plain)
        .foregroundStyle(Palette.muted)
    }
}

private struct GameOverCard: View {
    let model: GameModel

    var body: some View {
        ZStack {
            Palette.background.opacity(0.45).background(.ultraThinMaterial).ignoresSafeArea()
            VStack(spacing: 6) {
                EyebrowLabel(text: "Game over")
                Text(title).font(.system(size: 20, weight: .heavy)).tracking(-0.4)
                Text(detail).font(.system(size: 14)).foregroundStyle(Palette.muted)
                    .multilineTextAlignment(.center)
                    .padding(.bottom, 10)
                Button(action: model.newGame) {
                    Text("New game")
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(Palette.onAccent)
                        .padding(.horizontal, 22).padding(.vertical, 12)
                        .background(Palette.accent, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                }
                .buttonStyle(.plain)
            }
            .padding(24)
            .background(Palette.surface, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).strokeBorder(Palette.border))
            .shadow(color: .black.opacity(0.12), radius: 20, y: 8)
            .padding(32)
        }
    }

    private var title: String {
        switch model.status {
        case ChesscoreStatusCheckmate: model.turn == model.engineSide ? "You win" : "The engine wins"
        case ChesscoreStatusStalemate: "Stalemate"
        default: "Draw"
        }
    }

    private var detail: String {
        let moves = (model.sans.count + 1) / 2
        switch model.status {
        case ChesscoreStatusCheckmate: return "Checkmate in \(moves) moves."
        case ChesscoreStatusStalemate: return "No legal moves and no check, so the game is drawn."
        case ChesscoreStatusRepetition: return "The same position came up three times."
        case ChesscoreStatusInsufficient: return "Neither side has enough pieces left to mate."
        default: return "Fifty moves passed without a capture or a pawn move."
        }
    }
}

#Preview {
    GameView()
}
