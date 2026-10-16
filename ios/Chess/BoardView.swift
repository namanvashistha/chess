import SwiftUI

/// The Stone board: limestone and slate squares with a faint grain, glass
/// pieces that glide and settle, lines of light for moves, a warm light on the
/// side that is ahead, and glass shards when something is taken.
///
/// Nothing on it moves unless something happened.
struct BoardView: View {
    let model: GameModel
    /// Draws a still board (the home screen's hero) with no interaction.
    var still = false

    var body: some View {
        GeometryReader { geo in
            let size = geo.size.width
            ZStack {
                SquaresLayer(model: model, size: size)
                Grain().blendMode(.multiply).opacity(0.35).allowsHitTesting(false)
                ScoreLight(share: model.whiteShare)
                if !still {
                    RaysLayer(rays: model.rays, size: size)
                }
                PiecesLayer(model: model, size: size)
                ForEach(model.bursts) { b in
                    ShardBurst(dark: b.dark, size: size / 8)
                        .position(center(b.square, size))
                }
            }
            .frame(width: size, height: size)
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .overlay {
                RoundedRectangle(cornerRadius: 14, style: .continuous).strokeBorder(Color.black.opacity(0.12), lineWidth: 1)
            }
            .shadow(color: .black.opacity(0.28), radius: 18, y: 14)
            .contentShape(Rectangle())
            .gesture(SpatialTapGesture().onEnded { tap in
                guard !still else { return }
                let q = size / 8
                let col = Int(tap.location.x / q), row = Int(tap.location.y / q)
                guard (0..<8).contains(col), (0..<8).contains(row) else { return }
                model.tap(row * 8 + col)
            }, including: still ? .none : .all)
        }
        .aspectRatio(1, contentMode: .fit)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("Chess board")
    }
}

/// Square centre in board coordinates.
func center(_ sq: Square, _ size: CGFloat) -> CGPoint {
    let q = size / 8
    return CGPoint(x: (CGFloat(Squares.file(sq)) + 0.5) * q, y: (CGFloat(Squares.row(sq)) + 0.5) * q)
}

private struct SquaresLayer: View {
    let model: GameModel
    let size: CGFloat

    var body: some View {
        let q = size / 8
        let last = model.lastMove
        let check = model.checkSquare
        VStack(spacing: 0) {
            ForEach(0..<8, id: \.self) { row in
                HStack(spacing: 0) {
                    ForEach(0..<8, id: \.self) { col in
                        let sq = row * 8 + col
                        let dark = (row + col) % 2 == 1
                        ZStack {
                            Rectangle().fill(dark ? Palette.squareDark : Palette.squareLight)
                            if let last, last.from == sq || last.to == sq {
                                // neutral: amber is kept for what the player can act on
                                Rectangle().fill(Color.white.opacity(0.34))
                            }
                            if sq == check {
                                RadialGradient(colors: [Palette.danger.opacity(0.75), .clear], center: .center,
                                               startRadius: 0, endRadius: q * 0.6)
                            }
                            if model.loose.contains(sq) && model.rays == nil {
                                Circle().strokeBorder(Palette.danger.opacity(0.7), lineWidth: 2).padding(q * 0.06)
                            }
                            coordinates(row: row, col: col, q: q)
                        }
                        .frame(width: q, height: q)
                    }
                }
            }
        }
        .animation(.easeOut(duration: 0.25), value: model.moveSerial)
    }

    @ViewBuilder
    private func coordinates(row: Int, col: Int, q: CGFloat) -> some View {
        let font = Font.system(size: q * 0.15, weight: .bold, design: .rounded)
        // The board's rounded corners would clip the labels on a8 and h1, so
        // those two sit further in.
        if col == 0 {
            Text("\(8 - row)").font(font).foregroundStyle(Palette.coordinate.opacity(0.85))
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                .padding(q * (row == 0 ? 0.12 : 0.035))
        }
        if row == 7 {
            Text(String(Character(UnicodeScalar(UInt8(97 + col))))).font(font).foregroundStyle(Palette.coordinate.opacity(0.85))
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottomTrailing)
                .padding(q * (col == 7 ? 0.12 : 0.035))
        }
    }
}

/// A fine stone grain, the same every time.
private struct Grain: View {
    var body: some View {
        Canvas { ctx, size in
            var seed: UInt64 = 0x9E3779B97F4A7C15
            func rand() -> CGFloat {
                seed = seed &* 6364136223846793005 &+ 1442695040888963407
                return CGFloat(seed >> 33) / CGFloat(1 << 31)
            }
            let count = Int(size.width * size.height / 40)
            for _ in 0..<count {
                let r = 0.4 + rand() * 0.7
                let rect = CGRect(x: rand() * size.width, y: rand() * size.height, width: r * 2, height: r * 2)
                ctx.fill(Path(ellipseIn: rect), with: .color(.black.opacity(0.04 + rand() * 0.05)))
            }
        }
        .drawingGroup()
    }
}

/// The score as light: a warm wash that settles on the side that is ahead.
private struct ScoreLight: View {
    let share: Double // White's share; White is at the bottom

    var body: some View {
        RadialGradient(colors: [Palette.amber.opacity(0.75), Palette.amber.opacity(0.18), .clear],
                       center: UnitPoint(x: 0.5, y: 0.15 + share * 0.7), startRadius: 0, endRadius: 260)
            .blendMode(.softLight)
            .allowsHitTesting(false)
            .animation(.easeInOut(duration: 1.2), value: share)
    }
}

/// Lines of light from a piece to where it can go. A knight's line bends.
private struct RaysLayer: View {
    let rays: Rays?
    let size: CGFloat

    var body: some View {
        ZStack {
            if let rays {
                let color: Color = switch rays.kind {
                case .options, .chosen: Palette.amber
                case .thinking: Palette.thinking
                case .hint: Palette.great
                }
                ForEach(Array(rays.moves.enumerated()), id: \.element) { k, m in
                    Ray(move: m, knight: rays.knight.contains(m), size: size, color: color,
                        width: size / 8 * (rays.kind == .thinking ? 0.05 : 0.075),
                        opacity: rays.kind == .thinking ? 0.7 : 1, delay: rays.kind == .thinking ? 0 : Double(k) * 0.02)
                }
                .id(kindKey(rays.kind))
            }
        }
        .allowsHitTesting(false)
    }

    private func kindKey(_ k: Rays.Kind) -> String { "\(k)" }
}

private struct Ray: View {
    let move: BoardMove
    let knight: Bool
    let size: CGFloat
    let color: Color
    let width: CGFloat
    let opacity: Double
    let delay: Double
    @State private var drawn = false

    var body: some View {
        let a = center(move.from, size), b = center(move.to, size)
        let q = size / 8
        ZStack {
            Path { p in
                p.move(to: a)
                if knight {
                    let horizontal = abs(b.x - a.x) > abs(b.y - a.y)
                    p.addLine(to: horizontal ? CGPoint(x: b.x, y: a.y) : CGPoint(x: a.x, y: b.y))
                }
                p.addLine(to: b)
            }
            .trim(from: 0, to: drawn ? 1 : 0)
            .stroke(color, style: StrokeStyle(lineWidth: width, lineCap: .round, lineJoin: .round))
            Group {
                if move.capture {
                    Circle().stroke(color, lineWidth: width * 0.8).frame(width: q * 0.76, height: q * 0.76)
                } else {
                    Circle().fill(color).frame(width: q * 0.2, height: q * 0.2)
                }
            }
            .scaleEffect(drawn ? 1 : 0.01)
            .position(b)
        }
        .opacity(opacity)
        .shadow(color: color.opacity(0.35), radius: 1.5, y: 1)
        .onAppear {
            withAnimation(.easeOut(duration: 0.32).delay(delay)) { drawn = true }
        }
    }
}

private struct PiecesLayer: View {
    let model: GameModel
    let size: CGFloat

    var body: some View {
        let q = size / 8
        ZStack {
            ForEach(model.pieces) { piece in
                let lifted = piece.square == model.selected
                let toppled = piece.id == model.toppled
                GlassPiece(code: piece.code, lifted: lifted)
                    .frame(width: q, height: q)
                    .scaleEffect(lifted ? 1.1 : 1, anchor: .bottom)
                    .offset(y: lifted ? -q * 0.09 : 0)
                    .rotationEffect(.degrees(toppled ? 82 : 0), anchor: UnitPoint(x: 0.5, y: 0.88))
                    .offset(x: toppled ? q * 0.15 : 0)
                    .animation(.spring(response: 0.26, dampingFraction: 0.6), value: lifted)
                    .animation(.spring(response: 0.9, dampingFraction: 0.55), value: toppled)
                    .position(center(piece.square, size))
                    .zIndex(piece.id == model.movingPiece ? 20 : Double(Squares.row(piece.square)))
                    .transition(.opacity.animation(.easeOut(duration: 0.12)))
            }
        }
        .frame(width: size, height: size)
        .allowsHitTesting(false)
        .animation(.spring(response: 0.42, dampingFraction: 0.72), value: model.moveSerial)
    }
}

/// A taken piece breaking into glass.
private struct ShardBurst: View {
    let dark: Bool
    let size: CGFloat
    @State private var flown = false
    private let shards: [(dx: CGFloat, dy: CGFloat, rot: Double, delay: Double)] = (0..<14).map { _ in
        let a = Double.random(in: 0..<(2 * .pi)), d = CGFloat.random(in: 0.6...1.6)
        return (CGFloat(cos(a)) * d, CGFloat(sin(a)) * d - 0.3, Double.random(in: -270...270), Double.random(in: 0...0.05))
    }

    var body: some View {
        ZStack {
            ForEach(shards.indices, id: \.self) { i in
                let s = shards[i]
                Shard()
                    .fill(LinearGradient(colors: dark ? [Color(white: 0.5), Color(white: 0.1)] : [.white, Color(white: 0.8)],
                                         startPoint: .topLeading, endPoint: .bottomTrailing))
                    .frame(width: size * 0.26, height: size * 0.26)
                    .shadow(color: .black.opacity(0.3), radius: 1, y: 1)
                    .rotationEffect(.degrees(flown ? s.rot : 0))
                    .offset(x: flown ? s.dx * size : 0, y: flown ? s.dy * size : 0)
                    .scaleEffect(flown ? 0.4 : 1)
                    .opacity(flown ? 0 : 1)
                    .animation(.timingCurve(0.15, 0.7, 0.4, 1, duration: 0.75).delay(s.delay), value: flown)
            }
        }
        .allowsHitTesting(false)
        .onAppear { flown = true }
    }
}

private struct Shard: Shape {
    func path(in r: CGRect) -> Path {
        Path { p in
            p.move(to: CGPoint(x: r.midX, y: r.minY))
            p.addLine(to: CGPoint(x: r.maxX, y: r.minY + r.height * 0.7))
            p.addLine(to: CGPoint(x: r.minX + r.width * 0.2, y: r.maxY))
            p.closeSubpath()
        }
    }
}
