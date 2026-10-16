import SwiftUI

/// The piece silhouettes: classic Staunton proportions simplified to a few
/// curves, in a 100 by 100 box standing on a shared plinth. The same path data
/// drives the web prototype.
enum PieceArt {
    static let plinth = "M22 86.5C22 84 23.5 82.5 26 82.5H74C76.5 82.5 78 84 78 86.5V87.5C78 89.5 76.5 90.5 74.5 90.5H25.5C23.5 90.5 22 89.5 22 87.5Z"

    struct Art {
        let body: String
        let detail: String
        var eye: (x: CGFloat, y: CGFloat, r: CGFloat)? = nil
    }

    static let art: [Character: Art] = [
        "p": Art(body: "M33 82.5C35 74 40 67 43 61H57C60 67 65 74 67 82.5Z M37 61C37 58 41 56.5 50 56.5C59 56.5 63 58 63 61Z M50 27A14 14 0 1 1 49.99 27Z",
                 detail: "M38 61H62"),
        "r": Art(body: "M30 82.5C32 76 34 70 35 64V44H65V64C66 70 68 76 70 82.5Z M30 44V24H38V30H45V24H55V30H62V24H70V44Z",
                 detail: "M35 44H65M34 64H66"),
        "b": Art(body: "M32 82.5C35 76 40 71 42 66H58C60 71 65 76 68 82.5Z M38 66C38 63 42 61.5 50 61.5C58 61.5 62 63 62 66Z M50 20C61 28 66 37 66 46C66 54 59 60 50 60C41 60 34 54 34 46C34 37 39 28 50 20Z M50 11A5 5 0 1 1 49.99 11Z",
                 detail: "M44 33L57 46M40 66H60"),
        "n": Art(body: "M31 82.5C32 73 37 66 41 61C34 60 27 59 23 56C18 53 17.5 47.5 20.5 43.5L37 25C40.5 20 45 15 49.5 10.5C52 14 54 18 55.5 22C68 26 76 37 75.5 52C75 64 72.5 73 71.5 82.5Z",
                 detail: "M23.5 51.5C26 50.5 28 50.5 30 51.5", eye: (40, 33, 2.6)),
        "q": Art(body: "M30 82.5C33 74 38 66 40 58H60C62 66 67 74 70 82.5Z M36 58C36 55 41 53.5 50 53.5C59 53.5 64 55 64 58Z M35 53.5L27 27L38.5 40L44 21.5L50 38L56 21.5L61.5 40L73 27L65 53.5Z M27 20A4 4 0 1 1 26.99 20Z M44 14.5A4 4 0 1 1 43.99 14.5Z M56 14.5A4 4 0 1 1 55.99 14.5Z M73 20A4 4 0 1 1 72.99 20Z",
                 detail: "M38 58H62M36 47H64"),
        "k": Art(body: "M30 82.5C33 74 38 66 40 58H60C62 66 67 74 70 82.5Z M36 58C36 55 41 53.5 50 53.5C59 53.5 64 55 64 58Z M35 53.5C33 46 30 39 31 34C32.5 28 41 27 50 27C59 27 67.5 28 69 34C70 39 67 46 65 53.5Z M46.5 26V19H40.5V13H46.5V6.5H53.5V13H59.5V19H53.5V26Z",
                 detail: "M38 58H62M33 40H67"),
    ]
}

/// A piece's silhouette (plinth and body) as a SwiftUI shape, so it can carry
/// a Liquid Glass effect and clip its own shading.
struct PieceSilhouette: Shape {
    let kind: Character // lowercase

    func path(in rect: CGRect) -> Path {
        guard let art = PieceArt.art[kind] else { return Path() }
        return SVGPath.parse(PieceArt.plinth + art.body, in: rect)
    }
}

/// The engraved lines on a piece: collars, the bishop's slit, the knight's mouth.
struct PieceDetail: Shape {
    let kind: Character

    func path(in rect: CGRect) -> Path {
        guard let art = PieceArt.art[kind] else { return Path() }
        return SVGPath.parse(art.detail, in: rect)
    }
}

/// The knight's eye.
struct PieceEye: Shape {
    let kind: Character

    func path(in rect: CGRect) -> Path {
        guard let eye = PieceArt.art[kind]?.eye else { return Path() }
        let s = rect.width / 100
        return Path(ellipseIn: CGRect(x: rect.minX + (eye.x - eye.r) * s, y: rect.minY + (eye.y - eye.r) * s,
                                      width: eye.r * 2 * s, height: eye.r * 2 * s))
    }
}

/// Parses the small SVG path subset the piece art uses: absolute M, L, H, V,
/// C, Z, and A for circles drawn as a closed arc from their top point.
enum SVGPath {
    static func parse(_ d: String, in rect: CGRect) -> Path {
        let s = rect.width / 100
        func pt(_ x: CGFloat, _ y: CGFloat) -> CGPoint { CGPoint(x: rect.minX + x * s, y: rect.minY + y * s) }

        var tokens: [String] = []
        var number = ""
        for ch in d {
            if ch.isLetter {
                if !number.isEmpty { tokens.append(number); number = "" }
                tokens.append(String(ch))
            } else if ch == " " || ch == "," {
                if !number.isEmpty { tokens.append(number); number = "" }
            } else if ch == "-" && !number.isEmpty && !number.hasSuffix("e") {
                tokens.append(number); number = "-"
            } else {
                number.append(ch)
            }
        }
        if !number.isEmpty { tokens.append(number) }

        var path = Path()
        var i = 0
        var cmd = "M"
        var cur = CGPoint.zero // in 0-100 units
        func next() -> CGFloat { defer { i += 1 }; return CGFloat(Double(tokens[i]) ?? 0) }

        while i < tokens.count {
            if let c = tokens[i].first, c.isLetter {
                cmd = String(c)
                i += 1
                if cmd == "Z" { path.closeSubpath(); continue }
            }
            switch cmd {
            case "M":
                cur = CGPoint(x: next(), y: next()); path.move(to: pt(cur.x, cur.y)); cmd = "L"
            case "L":
                cur = CGPoint(x: next(), y: next()); path.addLine(to: pt(cur.x, cur.y))
            case "H":
                cur.x = next(); path.addLine(to: pt(cur.x, cur.y))
            case "V":
                cur.y = next(); path.addLine(to: pt(cur.x, cur.y))
            case "C":
                let c1 = CGPoint(x: next(), y: next()), c2 = CGPoint(x: next(), y: next())
                cur = CGPoint(x: next(), y: next())
                path.addCurve(to: pt(cur.x, cur.y), control1: pt(c1.x, c1.y), control2: pt(c2.x, c2.y))
            case "A":
                let r = next(); _ = next(); _ = next(); _ = next(); _ = next()
                let end = CGPoint(x: next(), y: next())
                // The art only uses arcs for whole circles, starting at the top.
                let center = CGPoint(x: cur.x, y: cur.y + r)
                path.addEllipse(in: CGRect(x: rect.minX + (center.x - r) * s, y: rect.minY + (center.y - r) * s,
                                           width: 2 * r * s, height: 2 * r * s))
                cur = end
            default:
                i += 1
            }
        }
        return path
    }
}
