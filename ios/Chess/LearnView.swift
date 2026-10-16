import SwiftUI

/// "Learn the moves": one tiny lesson per piece for someone who has never
/// played. A single piece and a few stars; soft dots show where it can go; tap
/// a dot to move. Collect every star to finish. No timer, no way to fail.
struct LearnView: View {
    /// Finished all lessons and wants a first game.
    let play: () -> Void
    @Environment(\.dismiss) private var dismiss

    @State private var index = 0
    @State private var square: Square = 0
    @State private var stars: Set<Square> = []
    @State private var captured: Set<Square> = []
    @State private var pops = 0
    /// The piece is being introduced: shown large in the middle, then it
    /// shrinks onto its square.
    @State private var intro = true

    private var lesson: Lesson? { Lesson.all.indices.contains(index) ? Lesson.all[index] : nil }
    private var done: Bool { stars.isEmpty }

    var body: some View {
        ZStack {
            AppBackground()
            VStack(spacing: 0) {
                header
                if let lesson {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(lesson.title).font(Typeface.display(28)).tracking(-0.4)
                        Text(lesson.text).font(.system(size: 16)).foregroundStyle(Palette.mist)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.top, 22)
                    .id(index)
                    .transition(.opacity)

                    LessonBoard(lesson: lesson, square: square, stars: stars, captured: captured, intro: intro,
                                targets: done || intro ? [] : lesson.moves(from: square, captured: captured)) { to in move(to) }
                        .padding(.top, 22)

                    Spacer(minLength: 16)
                    footer
                } else {
                    finale
                }
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 16)
            .frame(maxWidth: 560)
            .foregroundStyle(Palette.ink)
        }
        .onAppear { load() }
        .sensoryFeedback(.success, trigger: pops)
        .animation(.smooth(duration: 0.35), value: index)
    }

    private var header: some View {
        HStack {
            Button { Haptics.tap(); dismiss() } label: {
                Image(systemName: "xmark").font(.system(size: 14, weight: .semibold)).frame(width: 30, height: 30)
            }
            .buttonStyle(.glass)
            .buttonBorderShape(.circle)
            .accessibilityLabel("Close")
            Spacer()
            HStack(spacing: 5) {
                ForEach(Lesson.all.indices, id: \.self) { i in
                    Capsule().fill(i <= index ? Palette.accent : Palette.mist.opacity(0.3))
                        .frame(width: i == index ? 16 : 6, height: 6)
                }
            }
            Spacer()
            Color.clear.frame(width: 30, height: 30)
        }
        .padding(.top, 8)
        .animation(.smooth, value: index)
    }

    private var footer: some View {
        VStack(spacing: 10) {
            if done {
                Label("Nicely done", systemImage: "star.fill")
                    .font(.system(size: 14, weight: .semibold)).foregroundStyle(Palette.great)
                    .transition(.scale.combined(with: .opacity))
                Button(index == Lesson.all.count - 1 ? "Finish" : "Next piece") {
                    Haptics.tap()
                    withAnimation(.smooth) { index += 1 }
                    load()
                }
                .buttonStyle(PrimaryButtonStyle())
            } else {
                Text(stars.count == 1 ? "1 star to go" : "\(stars.count) stars to go")
                    .font(.system(size: 14, weight: .medium)).foregroundStyle(Palette.mist)
                    .frame(minHeight: 56)
            }
        }
        .animation(.spring(response: 0.4, dampingFraction: 0.7), value: done)
    }

    private var finale: some View {
        VStack(spacing: 16) {
            Spacer()
            GlassPiece(code: "N").frame(width: 96, height: 96)
            Text("You know the moves").font(Typeface.display(30)).tracking(-0.5)
            Text("The rest you'll learn by playing. The coach will help along the way.")
                .font(.system(size: 16)).foregroundStyle(Palette.mist).multilineTextAlignment(.center)
            Spacer()
            Button("Play the Rookie") {
                Haptics.tap()
                UserDefaults.standard.set(true, forKey: "learn.done")
                dismiss()
                play()
            }
            .buttonStyle(PrimaryButtonStyle())
            Button("Back to home") {
                UserDefaults.standard.set(true, forKey: "learn.done")
                dismiss()
            }
            .font(.system(size: 15, weight: .semibold)).foregroundStyle(Palette.mist)
            .buttonStyle(.plain)
            .padding(.bottom, 8)
        }
    }

    private func load() {
        guard let lesson else { return }
        square = lesson.start
        stars = Set(lesson.stars)
        captured = []
        intro = true
        let shown = index
        Task {
            try? await Task.sleep(for: .seconds(1.1))
            guard shown == index else { return }
            withAnimation(.spring(response: 0.75, dampingFraction: 0.82)) { intro = false }
            Haptics.tap(.soft, intensity: 0.6)
        }
    }

    private func move(_ to: Square) {
        guard let lesson else { return }
        withAnimation(.spring(response: 0.38, dampingFraction: 0.75)) {
            square = to
            if lesson.enemies.contains(to) { captured.insert(to) }
            if stars.remove(to) != nil { pops += 1; Sound.shared.play(.great) } else { Sound.shared.play(.move) }
        }
    }
}

/// One lesson: a piece, where it starts, the stars to collect, its own pawns
/// in the way (to jump over or around) and black pieces it can take.
struct Lesson {
    let piece: Character
    let start: Square
    let stars: [Square]
    var friends: [Square] = []
    var enemies: [Square] = []
    let title: String
    let text: String

    // Squares in display order: 0 is a8, 63 is h1.
    static let all: [Lesson] = [
        Lesson(piece: "R", start: 56, stars: [24, 28, 4], title: L("The rook"),
               text: L("Moves in straight lines, up, down or across, as far as it likes.")),
        Lesson(piece: "B", start: 58, stars: [44, 30, 3], title: L("The bishop"),
               text: L("Slides diagonally. It always stays on squares of one colour.")),
        Lesson(piece: "Q", start: 59, stars: [27, 31, 4], title: L("The queen"),
               text: L("The strongest piece: she moves like a rook and a bishop together.")),
        Lesson(piece: "K", start: 60, stars: [52, 45, 37], title: L("The king"),
               text: L("Steps one square in any direction. Keep him safe: that's the whole game.")),
        Lesson(piece: "N", start: 62, stars: [45, 28, 11], friends: [53, 54, 55], title: L("The knight"),
               text: L("Jumps in an L: two squares one way, then one to the side. It leaps over anything.")),
        Lesson(piece: "P", start: 52, stars: [36, 27, 19], enemies: [27], title: L("The pawn"),
               text: L("Steps forward, two squares on its first move. It takes pieces diagonally.")),
    ]

    func moves(from sq: Square, captured: Set<Square>) -> [Square] {
        let r = sq / 8, c = sq % 8
        let blocked = Set(friends)
        let enemy = Set(enemies).subtracting(captured)
        func on(_ r: Int, _ c: Int) -> Bool { (0..<8).contains(r) && (0..<8).contains(c) }
        func slide(_ dirs: [(Int, Int)]) -> [Square] {
            var out: [Square] = []
            for (dr, dc) in dirs {
                var rr = r + dr, cc = c + dc
                while on(rr, cc) {
                    let s = rr * 8 + cc
                    if blocked.contains(s) { break }
                    out.append(s)
                    if enemy.contains(s) { break }
                    rr += dr; cc += dc
                }
            }
            return out
        }
        let straight = [(1, 0), (-1, 0), (0, 1), (0, -1)], diagonal = [(1, 1), (1, -1), (-1, 1), (-1, -1)]
        switch piece {
        case "R": return slide(straight)
        case "B": return slide(diagonal)
        case "Q": return slide(straight + diagonal)
        case "K":
            return (straight + diagonal).map { (r + $0.0, c + $0.1) }.filter { on($0.0, $0.1) }
                .map { $0.0 * 8 + $0.1 }.filter { !blocked.contains($0) }
        case "N":
            return [(2, 1), (2, -1), (-2, 1), (-2, -1), (1, 2), (1, -2), (-1, 2), (-1, -2)]
                .map { (r + $0.0, c + $0.1) }.filter { on($0.0, $0.1) }
                .map { $0.0 * 8 + $0.1 }.filter { !blocked.contains($0) }
        case "P":
            var out: [Square] = []
            let one = (r - 1) * 8 + c
            if r > 0, !enemy.contains(one), !blocked.contains(one) {
                out.append(one)
                let two = (r - 2) * 8 + c
                if r == 6, !enemy.contains(two) { out.append(two) }
            }
            for dc in [-1, 1] where on(r - 1, c + dc) {
                let s = (r - 1) * 8 + c + dc
                if enemy.contains(s) { out.append(s) }
            }
            return out
        default: return []
        }
    }
}

/// The lesson board: stone squares, the piece, stars, and soft dots for where
/// the piece can go. Tapping a dot moves the piece there.
private struct LessonBoard: View {
    let lesson: Lesson
    let square: Square
    let stars: Set<Square>
    let captured: Set<Square>
    let intro: Bool
    let targets: [Square]
    let tap: (Square) -> Void

    var body: some View {
        GeometryReader { geo in
            let size = geo.size.width, q = size / 8
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
                .blur(radius: intro ? 4 : 0)
                .overlay(Color.black.opacity(intro ? 0.12 : 0))
                Group {
                ForEach(lesson.friends, id: \.self) { s in
                    GlassPiece(code: "P").frame(width: q, height: q).position(center(s, size))
                }
                ForEach(lesson.enemies.filter { !captured.contains($0) }, id: \.self) { s in
                    ZStack {
                        GlassPiece(code: "p")
                        Image(systemName: "star.fill").font(.system(size: q * 0.2))
                            .foregroundStyle(Palette.lastMove).offset(x: q * 0.3, y: -q * 0.3)
                    }
                    .frame(width: q, height: q).position(center(s, size))
                    .transition(.opacity.combined(with: .scale(scale: 0.6)))
                }
                // The game's lines of light, from the piece to where it can go.
                ZStack {
                    ForEach(Array(targets.enumerated()), id: \.element) { k, s in
                        Ray(move: BoardMove(from: square, to: s, capture: lesson.enemies.contains(s)),
                            knight: lesson.piece == "N", size: size, color: Palette.ray,
                            width: q * 0.075, opacity: 0.85, delay: Double(k) * 0.03)
                    }
                }
                .id(square) // redraw the lines from each new square
                .allowsHitTesting(false)
                ForEach(targets, id: \.self) { s in
                    Color.clear
                        .frame(width: q, height: q)
                        .contentShape(Rectangle())
                        .position(center(s, size))
                        .onTapGesture { tap(s) }
                }
                // Stars sit above the dots, so a dot never covers one.
                ForEach(Array(stars), id: \.self) { s in
                    Image(systemName: "star.fill")
                        .font(.system(size: q * 0.42))
                        .foregroundStyle(Palette.lastMove)
                        .shadow(color: Palette.lastMove.opacity(0.6), radius: 6)
                        .position(center(s, size))
                        .transition(.scale(scale: 1.8).combined(with: .opacity))
                        .opacity(lesson.enemies.contains(s) && !captured.contains(s) ? 0 : 1)
                }
                }
                .opacity(intro ? 0 : 1)
                // Introduced large in the middle, then it settles onto its square.
                GlassPiece(code: lesson.piece)
                    .frame(width: intro ? size * 0.6 : q, height: intro ? size * 0.6 : q)
                    .shadow(color: .black.opacity(intro ? 0.3 : 0), radius: intro ? 24 : 0, y: intro ? 16 : 0)
                    .position(intro ? CGPoint(x: size / 2, y: size / 2) : center(square, size))
                    .allowsHitTesting(false)
            }
            .frame(width: size, height: size)
            .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
            .shadow(color: .black.opacity(0.22), radius: 16, y: 12)
        }
        .aspectRatio(1, contentMode: .fit)
        .animation(.easeOut(duration: 0.2), value: targets)
    }
}
