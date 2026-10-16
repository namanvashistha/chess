import SwiftUI

/// The board in the Quiet Motion look: flat tournament squares, pieces that
/// glide on a spring, a lifted selected piece, and a board that leans with the
/// phone while a soft light moves across it.
///
/// The tilt changes every frame, so only this wrapper reads it. The squares
/// and pieces layers below read the game model alone and are not rebuilt as
/// the phone moves.
struct BoardView: View {
    let model: GameModel
    let tilt: Tilt

    /// Degrees of lean at full tilt.
    private let maxLean = 7.0

    var body: some View {
        GeometryReader { geo in
            let size = geo.size.width
            let light = CGPoint(x: 0.5 - tilt.x * 0.42, y: 0.3 - tilt.y * 0.38)
            ZStack {
                SquaresLayer(model: model, size: size)
                    .visualEffect { content, proxy in
                        content.colorEffect(ShaderLibrary.sheen(
                            .float2(proxy.size), .float2(light), .float(1)))
                    }
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                HighlightsLayer(model: model, size: size)
                PiecesLayer(model: model, size: size)
                    .shadow(color: .black.opacity(0.28), radius: 1.4,
                            x: 1 - tilt.x * 2.4, y: 2 - tilt.y * 2.2)
            }
            .frame(width: size, height: size)
            .rotation3DEffect(.degrees(-tilt.y * maxLean), axis: (x: 1, y: 0, z: 0), perspective: 0.5)
            .rotation3DEffect(.degrees(tilt.x * maxLean), axis: (x: 0, y: 1, z: 0), perspective: 0.5)
            .shadow(color: .black.opacity(0.22), radius: 18, x: tilt.x * -6, y: 14 - tilt.y * 6)
            .contentShape(Rectangle())
            .gesture(SpatialTapGesture().onEnded { tap in
                let q = size / 8
                let col = Int(tap.location.x / q), row = Int(tap.location.y / q)
                guard (0..<8).contains(col), (0..<8).contains(row) else { return }
                model.tap(row * 8 + col)
            })
        }
        .aspectRatio(1, contentMode: .fit)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("Chess board")
    }
}

/// Square centre in board coordinates.
private func center(_ sq: Square, _ size: CGFloat) -> CGPoint {
    let q = size / 8
    return CGPoint(x: (CGFloat(Squares.file(sq)) + 0.5) * q, y: (CGFloat(Squares.row(sq)) + 0.5) * q)
}

/// Squares, last-move tint and coordinates.
private struct SquaresLayer: View {
    let model: GameModel
    let size: CGFloat

    var body: some View {
        let q = size / 8
        let last = model.lastMove
        VStack(spacing: 0) {
            ForEach(0..<8, id: \.self) { row in
                HStack(spacing: 0) {
                    ForEach(0..<8, id: \.self) { col in
                        let sq = row * 8 + col
                        let dark = (row + col) % 2 == 1
                        ZStack {
                            Rectangle().fill(dark ? Palette.squareDark : Palette.squareLight)
                            if let last, last.from == sq || last.to == sq {
                                Rectangle().fill(Palette.lastMove.opacity(0.42))
                            }
                            coordinates(row: row, col: col, dark: dark, q: q)
                        }
                        .frame(width: q, height: q)
                    }
                }
            }
        }
        .animation(.easeOut(duration: 0.25), value: model.moveSerial)
    }

    @ViewBuilder
    private func coordinates(row: Int, col: Int, dark: Bool, q: CGFloat) -> some View {
        let color = dark ? Palette.squareLight : Palette.squareDark
        let font = Font.system(size: q * 0.2, weight: .bold, design: .rounded)
        if col == 0 {
            Text("\(8 - row)").font(font).foregroundStyle(color)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                .padding(q * 0.05)
        }
        if row == 7 {
            Text(String(Character(UnicodeScalar(UInt8(97 + col))))).font(font).foregroundStyle(color)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottomTrailing)
                .padding(q * 0.06)
        }
    }
}

/// The selected square's glow, legal-move markers, check and capture rings.
private struct HighlightsLayer: View {
    let model: GameModel
    let size: CGFloat

    var body: some View {
        let q = size / 8
        let targets = model.targets
        ZStack {
            if let check = model.checkSquare {
                RadialGradient(colors: [Palette.danger, Palette.danger.opacity(0.55), Palette.danger.opacity(0)],
                               center: .center, startRadius: 0, endRadius: q * 0.62)
                    .frame(width: q, height: q)
                    .phaseAnimator([0.75, 1.0]) { v, phase in v.opacity(phase) } animation: { _ in .easeInOut(duration: 0.55) }
                    .position(center(check, size))
                    .transition(.opacity)
            }
            if let sel = model.selected {
                SelectedGlow(q: q)
                    .position(center(sel, size))
                    .transition(.opacity.animation(.easeOut(duration: 0.15)))
                    .id(sel)
            }
            ForEach(Array(targets.enumerated()), id: \.element) { k, sq in
                TargetMarker(q: q, occupied: model.board[sq] != ".")
                    .position(center(sq, size))
                    .transition(.scale(scale: 0.1).combined(with: .opacity)
                        .animation(.spring(response: 0.3, dampingFraction: 0.55).delay(Double(k) * 0.016)))
            }
            ForEach(model.bursts) { b in
                BurstRing(q: q).position(center(b.square, size))
            }
        }
        .frame(width: size, height: size)
        .allowsHitTesting(false)
        .animation(.easeOut(duration: 0.15), value: model.selected)
    }
}

private struct SelectedGlow: View {
    let q: CGFloat

    var body: some View {
        Rectangle()
            .fill(Palette.accent.opacity(0.5))
            .overlay(Rectangle().strokeBorder(.white.opacity(0.55), lineWidth: 2))
            .frame(width: q, height: q)
            .shadow(color: Palette.accent.opacity(0.8), radius: q * 0.28)
            .phaseAnimator([0.75, 1.0]) { v, phase in v.opacity(phase) } animation: { _ in .easeInOut(duration: 0.9) }
    }
}

private struct TargetMarker: View {
    let q: CGFloat
    let occupied: Bool

    var body: some View {
        if occupied {
            Circle().strokeBorder(.black.opacity(0.28), lineWidth: q * 0.08)
                .frame(width: q * 0.9, height: q * 0.9)
        } else {
            Circle().fill(.black.opacity(0.28))
                .frame(width: q * 0.28, height: q * 0.28)
        }
    }
}

private struct BurstRing: View {
    let q: CGFloat
    @State private var done = false

    var body: some View {
        Circle()
            .strokeBorder(.white.opacity(0.95), lineWidth: 2)
            .frame(width: q * 0.8, height: q * 0.8)
            .scaleEffect(done ? 1.5 : 0.3)
            .opacity(done ? 0 : 1)
            .onAppear { withAnimation(.easeOut(duration: 0.5)) { done = true } }
    }
}

/// The pieces, each keyed by identity so a move animates as a glide.
private struct PiecesLayer: View {
    let model: GameModel
    let size: CGFloat

    var body: some View {
        let q = size / 8
        ZStack {
            ForEach(model.pieces) { piece in
                let lifted = piece.square == model.selected
                Image(piece.imageName)
                    .resizable()
                    .interpolation(.high)
                    .frame(width: q, height: q)
                    .scaleEffect(lifted ? 1.14 : 1)
                    .offset(y: lifted ? -q * 0.06 : 0)
                    .shadow(color: .black.opacity(lifted ? 0.22 : 0), radius: 3, x: 1.5, y: q * 0.08)
                    .animation(.spring(response: 0.26, dampingFraction: 0.6), value: lifted)
                    .position(center(piece.square, size))
                    .zIndex(piece.id == model.movingPiece ? 1 : 0)
                    .transition(.scale(scale: 0.5).combined(with: .opacity))
            }
        }
        .frame(width: size, height: size)
        .allowsHitTesting(false)
        .animation(.spring(response: 0.34, dampingFraction: 0.72), value: model.moveSerial)
    }
}
