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

    /// One finger on the board: a tap, a drag that carries a piece, or a
    /// long press that shows what guards and attacks a piece.
    struct Touch: Equatable {
        var start: Square
        var point: CGPoint
        var dragging = false
        var inspecting = false
        var moved = false
    }
    @State private var touch: Touch?
    @State private var hold: Task<Void, Never>?

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
                PiecesLayer(model: model, size: size, drag: touch?.dragging == true ? touch : nil)
                if !still { MarkLayer(model: model, size: size) }
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
            .gesture(touchGesture(size), including: still ? .none : .all)
        }
        .aspectRatio(1, contentMode: .fit)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("Chess board")
    }
}

extension BoardView {
    private func square(at p: CGPoint, _ size: CGFloat) -> Square? {
        let q = size / 8
        let col = Int(floor(p.x / q)), row = Int(floor(p.y / q))
        guard (0..<8).contains(col), (0..<8).contains(row) else { return nil }
        return row * 8 + col
    }

    private func touchGesture(_ size: CGFloat) -> some Gesture {
        DragGesture(minimumDistance: 0)
            .onChanged { v in
                if touch == nil {
                    guard let sq = square(at: v.startLocation, size) else { return }
                    touch = Touch(start: sq, point: v.location)
                    hold?.cancel()
                    hold = Task { @MainActor in
                        try? await Task.sleep(for: .milliseconds(420))
                        guard !Task.isCancelled, var t = touch, !t.dragging, !t.moved else { return }
                        if model.inspect(t.start) { t.inspecting = true; touch = t }
                    }
                }
                guard var t = touch else { return }
                t.point = v.location
                if !t.dragging && !t.inspecting && !t.moved && hypot(v.translation.width, v.translation.height) > size / 8 * 0.2 {
                    hold?.cancel()
                    if model.pickUp(t.start) { t.dragging = true } else { t.moved = true }
                }
                touch = t
            }
            .onEnded { v in
                hold?.cancel()
                guard let t = touch else { return }
                if t.inspecting {
                    model.endInspect()
                    touch = nil
                } else if t.dragging {
                    let target = square(at: v.location, size)
                    withAnimation(.spring(response: 0.32, dampingFraction: 0.7)) {
                        touch = nil
                        model.drop(on: target)
                    }
                } else {
                    touch = nil
                    guard !t.moved else { return }
                    // The verdict badge sits on the top-right corner of its square.
                    let q = size / 8
                    if let m = model.shownMark, m.square == t.start,
                       v.location.x - CGFloat(Squares.file(m.square)) * q > q * 0.55,
                       v.location.y - CGFloat(Squares.row(m.square)) * q < q * 0.45 {
                        model.revealMark()
                    } else {
                        model.tap(t.start)
                    }
                }
            }
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
        let last = model.shownLastMove
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
                                // The last move: a honey wash on both squares, lighter where
                                // the piece left. Where it landed, the wash warms toward the
                                // piece's base, so the eye finds it without a hard edge.
                                Rectangle().fill(Palette.lastMove.opacity(sq == last.to ? 0.4 : 0.26))
                                if sq == last.to {
                                    RadialGradient(colors: [Palette.lastMove.opacity(0.45), .clear],
                                                   center: UnitPoint(x: 0.5, y: 0.65), startRadius: 0, endRadius: q * 0.55)
                                }
                            }
                            if sq == check {
                                RadialGradient(colors: [Palette.danger.opacity(0.75), .clear], center: .center,
                                               startRadius: 0, endRadius: q * 0.6)
                            }
                            if sq == model.nudge {
                                // A hint's first step: this piece has a good move.
                                Rectangle().fill(Palette.great.opacity(0.22))
                                Circle().strokeBorder(Palette.great, lineWidth: 2.5).padding(q * 0.05)
                            }
                            if model.loose.contains(sq) && model.rays == nil && model.reviewing == nil {
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
        // daylight, not a colour: the board is simply brighter on the side that is ahead
        RadialGradient(colors: [Color.white.opacity(0.6), Color.white.opacity(0.14), .clear],
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
                case .options, .chosen: Palette.accent
                case .thinking: Palette.thinking
                case .hint: Palette.great
                }
                ForEach(Array(rays.moves.enumerated()), id: \.element) { k, m in
                    Ray(move: m, knight: rays.knight.contains(m), size: size,
                        color: rays.danger.contains(m) ? Palette.danger : color,
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
    /// The piece under the finger, if one is being dragged.
    let drag: BoardView.Touch?

    var body: some View {
        let q = size / 8
        ZStack {
            ForEach(model.shownPieces) { piece in
                let lifted = piece.square == model.selected
                let toppled = piece.id == model.toppled
                let dragged = drag?.start == piece.square && lifted
                GlassPiece(code: piece.code, lifted: lifted)
                    .frame(width: q, height: q)
                    .scaleEffect(dragged ? 1.28 : lifted ? 1.1 : 1, anchor: .bottom)
                    .animation(.spring(response: 0.22, dampingFraction: 0.65), value: dragged)
                    .offset(y: lifted ? -q * 0.09 : 0)
                    .rotationEffect(.degrees(toppled ? 82 : 0), anchor: UnitPoint(x: 0.5, y: 0.88))
                    .offset(x: toppled ? q * 0.15 : 0)
                    .animation(.spring(response: 0.26, dampingFraction: 0.6), value: lifted)
                    .animation(.spring(response: 0.9, dampingFraction: 0.55), value: toppled)
                    // Held a little above the finger so the finger doesn't hide it.
                    .position(dragged ? CGPoint(x: drag!.point.x, y: drag!.point.y - q * 0.25) : center(piece.square, size))
                    .zIndex(dragged ? 30 : piece.id == model.movingPiece ? 20 : Double(Squares.row(piece.square)))
                    .transition(.opacity.animation(.easeOut(duration: 0.12)))
            }
        }
        .frame(width: size, height: size)
        .allowsHitTesting(false)
        .animation(.spring(response: 0.42, dampingFraction: 0.72), value: model.moveSerial)
    }
}

/// The verdict on the last move, as a small badge on the piece that made it.
/// It pops in once and stays put until the next move.
private struct MarkLayer: View {
    let model: GameModel
    let size: CGFloat

    var body: some View {
        let q = size / 8
        ZStack {
            if let m = model.shownMark, let symbol = m.entry.card.kind.symbol {
                let c = center(m.square, size)
                Image(systemName: symbol)
                    .font(.system(size: q * 0.19, weight: .bold))
                    .foregroundStyle(.white)
                    .frame(width: q * 0.4, height: q * 0.4)
                    .background(verdictColor(m.entry.card.kind), in: .circle)
                    .overlay(Circle().strokeBorder(.white.opacity(0.9), lineWidth: max(1, q * 0.03)))
                    .shadow(color: .black.opacity(0.25), radius: 2, y: 1)
                    .position(x: c.x + q * 0.33, y: c.y - q * 0.33)
                    .transition(.scale(scale: 0.2).combined(with: .opacity))
                    .id(m.entry.id)
            }
        }
        .frame(width: size, height: size)
        .allowsHitTesting(false)
        .animation(.spring(response: 0.35, dampingFraction: 0.55), value: model.shownMark?.entry.id)
        .accessibilityHidden(true)
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
