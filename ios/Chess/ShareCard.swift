import CoreTransferable
import SwiftUI
import UniformTypeIdentifiers

/// A moment worth sharing (a great move, or a win), drawn as a picture of the
/// board with a line about it.
struct ShareMoment: Transferable, Equatable, Sendable {
    let board: [Character]
    let move: BoardMove?
    let title: String
    let detail: String
    let footer: String

    static var transferRepresentation: some TransferRepresentation {
        DataRepresentation(exportedContentType: .png) { moment in
            guard let data = await moment.png() else { throw CocoaError(.fileWriteUnknown) }
            return data
        }
        .suggestedFileName("Cozy Chess.png")
    }

    /// Just the board, square, for a Messages bubble.
    @MainActor
    func boardPNG() -> Data? {
        let renderer = ImageRenderer(content: ShareCardView(moment: self, boardOnly: true))
        renderer.scale = 3
        return renderer.uiImage?.pngData()
    }

    @MainActor
    func png() -> Data? {
        let renderer = ImageRenderer(content: ShareCardView(moment: self))
        renderer.scale = 3 // 360 × 450 points → 1080 × 1350 pixels
        return renderer.uiImage?.pngData()
    }
}

/// The picture itself: always in the light palette, so every share looks the same.
private struct ShareCardView: View {
    let moment: ShareMoment
    var boardOnly = false

    var body: some View {
        if boardOnly {
            board
                .frame(width: 320, height: 320)
                .environment(\.colorScheme, .light)
        } else {
            card
        }
    }

    private var card: some View {
        VStack(alignment: .leading, spacing: 14) {
            VStack(alignment: .leading, spacing: 4) {
                Text(moment.title).font(Typeface.display(30)).tracking(-0.5)
                Text(moment.detail).font(.system(size: 15)).foregroundStyle(Palette.mist)
                    .lineLimit(2).fixedSize(horizontal: false, vertical: true)
            }
            board
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .shadow(color: .black.opacity(0.18), radius: 10, y: 6)
            HStack(spacing: 6) {
                Image(systemName: "sparkles")
                Text(moment.footer)
            }
            .font(.system(size: 13, weight: .semibold, design: .rounded))
            .foregroundStyle(Palette.accentText)
        }
        .foregroundStyle(Palette.ink)
        .padding(22)
        .frame(width: 360, height: 450, alignment: .topLeading)
        .background(LinearGradient(colors: [Palette.backgroundTop, Palette.backgroundBottom], startPoint: .top, endPoint: .bottom))
        .environment(\.colorScheme, .light)
    }

    private var board: some View {
        GeometryReader { geo in
            let size = geo.size.width, q = size / 8
            ZStack(alignment: .topLeading) {
                VStack(spacing: 0) {
                    ForEach(0..<8, id: \.self) { r in
                        HStack(spacing: 0) {
                            ForEach(0..<8, id: \.self) { c in
                                let sq = r * 8 + c
                                ZStack {
                                    Rectangle().fill((r + c) % 2 == 1 ? Palette.squareDark : Palette.squareLight)
                                    if let m = moment.move, m.from == sq || m.to == sq {
                                        Rectangle().fill(Palette.great.opacity(sq == m.to ? 0.4 : 0.22))
                                    }
                                }
                                .frame(width: q, height: q)
                            }
                        }
                    }
                }
                ForEach(0..<min(64, moment.board.count), id: \.self) { i in
                    if moment.board[i] != "." {
                        GlassPiece(code: moment.board[i], flat: true)
                            .frame(width: q, height: q)
                            .position(center(i, size))
                            .zIndex(Double(i / 8))
                    }
                }
            }
        }
        .aspectRatio(1, contentMode: .fit)
    }
}

/// A small glass button that shares a moment.
struct ShareMomentButton: View {
    let moment: ShareMoment
    var label: String?
    /// The share sheet's thumbnail, drawn once the button appears.
    @State private var thumbnail: Image?

    var body: some View {
        ShareLink(item: moment, preview: SharePreview(moment.title, image: thumbnail ?? Image(systemName: "sparkles"))) {
            if let label {
                Label(label, systemImage: "square.and.arrow.up")
            } else {
                Image(systemName: "square.and.arrow.up")
                    .font(.system(size: 14, weight: .semibold))
                    .frame(width: 30, height: 30)
            }
        }
        .simultaneousGesture(TapGesture().onEnded { Haptics.tap() })
        .task(id: moment) {
            if let data = moment.png(), let ui = UIImage(data: data) { thumbnail = Image(uiImage: ui) }
        }
        .accessibilityLabel("Share")
    }
}
