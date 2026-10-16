import SwiftUI

/// One chess piece in glass: frosted milk glass for White, polished smoked
/// glass for Black, lit from the top left, with a contact shadow where its base
/// meets the board.
///
/// The body is the system Liquid Glass material cut to the piece's own shape,
/// so it picks up the square beneath it. A gradient over it gives the milk or
/// smoke, and two clipped strokes give the lit and the shaded edge.
struct GlassPiece: View {
    let code: Character // FEN letter
    var lifted = false

    private var isWhite: Bool { code.isUppercase }
    private var kind: Character { Character(code.lowercased()) }

    var body: some View {
        GeometryReader { geo in
            let s = geo.size.width
            let shape = PieceSilhouette(kind: kind)
            ZStack {
                contactShadow(s)
                Color.clear
                    .glassEffect(.regular.tint(isWhite ? .white.opacity(0.35) : .black.opacity(0.55)), in: shape)
                shape.fill(fill)
                    .overlay(shape.fill(highlight).blendMode(.screen))
                shape.stroke(litEdge, lineWidth: s * 0.06)
                    .clipShape(shape)
                shape.stroke(shadeEdge, lineWidth: s * 0.06)
                    .clipShape(shape)
                PieceDetail(kind: kind)
                    .stroke(isWhite ? Color(white: 0.5, opacity: 0.55) : Color(white: 0.9, opacity: 0.35),
                            style: StrokeStyle(lineWidth: s * 0.016, lineCap: .round))
                PieceEye(kind: kind)
                    .fill(isWhite ? Color(white: 0.28, opacity: 0.6) : Color(white: 0.92, opacity: 0.6))
            }
            .padding(.horizontal, s * 0.04)
            .padding(.top, s * 0.02)
            .padding(.bottom, s * 0.05)
        }
        .aspectRatio(1, contentMode: .fit)
        .accessibilityHidden(true)
    }

    private func contactShadow(_ s: CGFloat) -> some View {
        ZStack {
            Ellipse()
                .fill(RadialGradient(colors: [lifted ? Palette.amber.opacity(0.18) : .black.opacity(0.14), .clear],
                                     center: .center, startRadius: 0, endRadius: s * 0.5))
                .frame(width: s * 0.96, height: s * 0.22)
                .offset(x: s * 0.06, y: s * 0.4)
            Ellipse()
                .fill(RadialGradient(colors: [lifted ? Palette.amber.opacity(0.5) : .black.opacity(0.42), .clear],
                                     center: .center, startRadius: 0, endRadius: s * 0.36))
                .frame(width: s * 0.72, height: s * 0.12)
                .offset(x: s * 0.03, y: s * 0.37 + (lifted ? s * 0.05 : 0))
                .scaleEffect(lifted ? 1.25 : 1)
        }
    }

    private var fill: LinearGradient {
        isWhite
            ? LinearGradient(stops: [.init(color: .white.opacity(0.9), location: 0),
                                     .init(color: Color(white: 0.965, opacity: 0.84), location: 0.45),
                                     .init(color: Color(red: 0.74, green: 0.765, blue: 0.80, opacity: 0.94), location: 1)],
                             startPoint: .topLeading, endPoint: .bottomTrailing)
            : LinearGradient(stops: [.init(color: Color(red: 0.31, green: 0.34, blue: 0.38, opacity: 0.86), location: 0),
                                     .init(color: Color(red: 0.13, green: 0.15, blue: 0.18, opacity: 0.92), location: 0.5),
                                     .init(color: Color(red: 0.055, green: 0.063, blue: 0.08, opacity: 0.96), location: 1)],
                             startPoint: .topLeading, endPoint: .bottomTrailing)
    }

    private var highlight: RadialGradient {
        RadialGradient(colors: [.white.opacity(isWhite ? 0.9 : 0.5), .clear],
                       center: UnitPoint(x: 0.34, y: 0.22), startRadius: 0, endRadius: 22)
    }

    private var litEdge: LinearGradient {
        let c = isWhite ? Color.white : Color(red: 0.91, green: 0.93, blue: 0.96)
        return LinearGradient(stops: [.init(color: c.opacity(isWhite ? 1 : 0.95), location: 0),
                                      .init(color: c.opacity(0.25), location: 0.55),
                                      .init(color: c.opacity(0), location: 1)],
                              startPoint: .topLeading, endPoint: .bottomTrailing)
    }

    private var shadeEdge: LinearGradient {
        let c = Color(red: 0.16, green: 0.19, blue: 0.21)
        return LinearGradient(stops: [.init(color: c.opacity(0), location: 0),
                                      .init(color: c.opacity(0.24), location: 0.5),
                                      .init(color: c.opacity(0.72), location: 1)],
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
