import Chesscore
import SwiftUI

/// Quiet Board palette from DESIGN.md.
enum Palette {
    static let accent = Color(hex: 0x5B8DEF)
    static let squareLight = Color(hex: 0xEBECD0)
    static let squareDark = Color(hex: 0x7D945D)
    static let lastMove = Color(hex: 0xFFC440)
    static let danger = Color(hex: 0xE0564F)
    static let good = Color(hex: 0x3AA564)
}

extension Color {
    init(hex: UInt32) {
        self.init(red: Double((hex >> 16) & 0xFF) / 255,
                  green: Double((hex >> 8) & 0xFF) / 255,
                  blue: Double(hex & 0xFF) / 255)
    }
}

struct GameView: View {
    @State private var model = GameModel()

    var body: some View {
        VStack(spacing: 16) {
            PlayerBar(name: "Engine", detail: model.thinking ? "Thinking…" : "Black",
                      active: model.turn == "b" && !model.isOver)
            BoardView(model: model)
            PlayerBar(name: "You", detail: "White", active: model.turn == "w" && !model.isOver)
            Text(statusText)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(minHeight: 20)
            Spacer()
            HStack(spacing: 40) {
                Button("Undo", systemImage: "arrow.uturn.backward", action: model.undo)
                Button("New game", systemImage: "plus", action: model.newGame)
            }
            .labelStyle(.titleAndIcon)
            .font(.subheadline.weight(.semibold))
        }
        .padding()
    }

    private var statusText: String {
        switch model.status {
        case ChesscoreStatusCheckmate: return model.turn == "w" ? "Checkmate. Black wins." : "Checkmate. You win."
        case ChesscoreStatusStalemate: return "Stalemate. The game is drawn."
        case ChesscoreStatusRepetition: return "Draw by repetition."
        case ChesscoreStatusFiftyMove: return "Draw by the fifty-move rule."
        default: return model.inCheck ? "Check" : ""
        }
    }
}

struct PlayerBar: View {
    let name: String
    let detail: String
    let active: Bool

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(name).font(.headline)
                Text(detail).font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            if active {
                Label("To move", systemImage: "circle.fill")
                    .labelStyle(.titleAndIcon)
                    .font(.caption2.weight(.bold))
                    .textCase(.uppercase)
                    .foregroundStyle(Palette.good)
                    .imageScale(.small)
            }
        }
    }
}

struct BoardView: View {
    let model: GameModel

    var body: some View {
        let targets = model.targets
        let checkSquare = model.inCheck ? model.board.firstIndex(of: model.turn == "w" ? "K" : "k") : nil
        LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 0), count: 8), spacing: 0) {
            ForEach(0..<64, id: \.self) { sq in
                SquareView(square: sq,
                           piece: model.board.indices.contains(sq) ? model.board[sq] : ".",
                           selected: model.selected == sq,
                           target: targets.contains(sq),
                           last: model.lastMove.map { $0.from == sq || $0.to == sq } ?? false,
                           check: checkSquare == sq)
                    .onTapGesture { model.tap(sq) }
            }
        }
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }
}

struct SquareView: View {
    let square: Square
    let piece: Character
    let selected: Bool
    let target: Bool
    let last: Bool
    let check: Bool

    private var isDark: Bool { (square / 8 + square % 8) % 2 == 1 }

    var body: some View {
        ZStack {
            Rectangle().fill(isDark ? Palette.squareDark : Palette.squareLight)
            if last { Rectangle().fill(Palette.lastMove.opacity(0.42)) }
            if selected { Rectangle().fill(Palette.accent.opacity(0.45)) }
            if check {
                RadialGradient(colors: [Palette.danger, Palette.danger.opacity(0)], center: .center,
                               startRadius: 0, endRadius: 26)
            }
            if piece != "." {
                Image(Self.imageName(piece)).resizable().scaledToFit()
            }
            if target {
                if piece == "." {
                    Circle().fill(.black.opacity(0.28)).padding(15)
                } else {
                    Circle().strokeBorder(.black.opacity(0.28), lineWidth: 4).padding(2)
                }
            }
        }
        .aspectRatio(1, contentMode: .fit)
        .contentShape(Rectangle())
    }

    static func imageName(_ piece: Character) -> String {
        (piece.isUppercase ? "w" : "b") + piece.uppercased()
    }
}

#Preview {
    GameView()
}
