import SwiftUI

/// One chess piece in glass: milk glass for White, polished smoked glass for
/// Black, lit from the top left, with a contact shadow where its base meets the
/// board.
///
/// The glass is baked: a gradient gives the milk or smoke, two clipped strokes
/// give the lit and the shaded edge, and a halo and fine contour (dark on
/// White, light on Black) keep each side apart from both squares. It used to be the live
/// Liquid Glass material, but
/// 32 live refractions redrawn every frame ran the GPU hot; this looks nearly
/// the same and renders once, as one cached layer.
struct GlassPiece: View {
    let code: Character // FEN letter
    var lifted = false
    /// Kept for callers that draw pieces into images; pieces are always baked now.
    var flat = false
    /// Standing on a board, with a shadow under it. Off for a piece shown on
    /// its own, like a tipped king.
    var grounded = true

    private var isWhite: Bool { code.isUppercase }
    private var kind: Character { Character(code.lowercased()) }

    var body: some View {
        GeometryReader { geo in
            let s = geo.size.width
            let shape = PieceSilhouette(kind: kind)
            ZStack {
                if grounded { contactShadow(s) }
                glass(s, shape)
                    // A lifted piece rises off the board; its shadow stays behind.
                    .offset(y: lifted ? -s * 0.09 : 0)
            }
            .animation(.spring(response: 0.3, dampingFraction: 0.82), value: lifted)
        }
        .aspectRatio(1, contentMode: .fit)
        .accessibilityHidden(true)
    }

    private func glass(_ s: CGFloat, _ shape: PieceSilhouette) -> some View {
        ZStack {
            // The tint under the glass, dense enough that the square only
            // hints through.
            shape.fill(isWhite ? Color.white.opacity(0.8) : Color.black.opacity(0.8))
            // A halo just outside the silhouette, dark around White and light
            // around Black: it reads as the edge of an object on a surface and
            // keeps each side clear of both squares.
            shape.fill(fill)
                .shadow(color: isWhite ? .black.opacity(0.58) : .white.opacity(0.46), radius: s * 0.016)
                .overlay(shape.fill(highlight(s)).blendMode(.screen))
            shape.stroke(litEdge, lineWidth: s * 0.06)
                .clipShape(shape)
            shape.stroke(shadeEdge, lineWidth: s * 0.06)
                .clipShape(shape)
            // A fine contour of the same weight on both sides, so the shape
            // holds even on a square of its own tone.
            shape.stroke(isWhite ? Color.black.opacity(0.6) : Color.white.opacity(0.38),
                         style: StrokeStyle(lineWidth: max(1, s * 0.017), lineJoin: .round))
            PieceDetail(kind: kind)
                .stroke(isWhite ? Color(white: 0.4, opacity: 0.6) : Color(white: 0.9, opacity: 0.45),
                        style: StrokeStyle(lineWidth: s * 0.016, lineCap: .round))
            PieceEye(kind: kind)
                .fill(isWhite ? Color(white: 0.25, opacity: 0.7) : Color(white: 0.92, opacity: 0.65))
        }
        // A touch larger than the art box, standing on its plinth, so the
        // piece fills more of its square; no more, or the king's cross and
        // the bishop's tip would reach past the square's edge.
        .scaleEffect(1.05, anchor: UnitPoint(x: 0.5, y: 0.9))
        .padding(.horizontal, s * 0.04)
        .padding(.top, s * 0.02)
        .padding(.bottom, s * 0.05)
        // One offscreen layer per piece, drawn once and reused while it glides.
        .drawingGroup()
    }

    /// Where the piece meets the board, lit from the top left: a soft ambient
    /// shadow falling down and to the right, and a tight dark line of contact
    /// right under the plinth. Lifted, the piece leaves the ground: the contact
    /// fades, the shadow spreads and pales, and a soft pool of the accent marks
    /// the square it came from.
    private func contactShadow(_ s: CGFloat) -> some View {
        ZStack {
            softEllipse(lifted ? Palette.accent.opacity(0.32) : .clear,
                        width: s * 0.86, height: s * 0.26)
                .offset(y: s * 0.35)
            softEllipse(.black.opacity(lifted ? 0.12 : 0.2), width: s * 0.94, height: s * 0.22)
                .scaleEffect(lifted ? 1.12 : 1)
                .offset(x: s * (lifted ? 0.07 : 0.05), y: s * 0.38)
            softEllipse(.black.opacity(lifted ? 0.14 : 0.5), width: s * 0.66, height: s * 0.085)
                .scaleEffect(lifted ? 1.18 : 1)
                .offset(x: s * 0.015, y: s * 0.357)
        }
    }

    /// An ellipse that fades out evenly to every edge: a round gradient,
    /// squashed, so there is no hard rim where it meets the board.
    private func softEllipse(_ color: Color, width: CGFloat, height: CGFloat) -> some View {
        Circle()
            .fill(RadialGradient(stops: [.init(color: color, location: 0),
                                         .init(color: color.opacity(0.45), location: 0.45),
                                         .init(color: color.opacity(0), location: 1)],
                                 center: .center, startRadius: 0, endRadius: width / 2))
            .frame(width: width, height: width)
            .scaleEffect(x: 1, y: height / width)
    }

    private var fill: LinearGradient {
        isWhite
            ? LinearGradient(stops: [.init(color: .white, location: 0),
                                     .init(color: Color(red: 0.975, green: 0.97, blue: 0.96), location: 0.45),
                                     .init(color: Color(red: 0.80, green: 0.815, blue: 0.84), location: 1)],
                             startPoint: .topLeading, endPoint: .bottomTrailing)
            // A deep, cool charcoal that belongs with the slate, short of pure
            // black so it keeps its form.
            : LinearGradient(stops: [.init(color: Color(red: 0.22, green: 0.235, blue: 0.26), location: 0),
                                     .init(color: Color(red: 0.13, green: 0.14, blue: 0.16), location: 0.5),
                                     .init(color: Color(red: 0.075, green: 0.08, blue: 0.095), location: 1)],
                             startPoint: .topLeading, endPoint: .bottomTrailing)
    }

    private func highlight(_ s: CGFloat) -> RadialGradient {
        RadialGradient(colors: [.white.opacity(isWhite ? 0.9 : 0.16), .clear],
                       center: UnitPoint(x: 0.34, y: 0.22), startRadius: 0, endRadius: s * 0.42)
    }

    private var litEdge: LinearGradient {
        let c = isWhite ? Color.white : Color(red: 0.91, green: 0.93, blue: 0.96)
        return LinearGradient(stops: [.init(color: c.opacity(isWhite ? 1 : 0.4), location: 0),
                                      .init(color: c.opacity(isWhite ? 0.25 : 0.08), location: 0.55),
                                      .init(color: c.opacity(0), location: 1)],
                              startPoint: .topLeading, endPoint: .bottomTrailing)
    }

    private var shadeEdge: LinearGradient {
        let c = Color(red: 0.16, green: 0.19, blue: 0.21)
        return LinearGradient(stops: [.init(color: c.opacity(0), location: 0),
                                      .init(color: c.opacity(0.32), location: 0.5),
                                      .init(color: c.opacity(isWhite ? 0.5 : 0.6), location: 1)],
                              startPoint: .topLeading, endPoint: .bottomTrailing)
    }
}

#Preview {
    HStack(spacing: 0) {
        ForEach(Array("KQRBNPkqrbnp"), id: \.self) { p in
            GlassPiece(code: p)
                .frame(width: 60, height: 60)
                .background(Palette.squareLight)
        }
    }
}
