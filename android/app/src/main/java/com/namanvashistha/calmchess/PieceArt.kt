package com.namanvashistha.calmchess

import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Path

/**
 * The piece silhouettes: classic Staunton proportions simplified to a few
 * curves, in a 100 by 100 box standing on a shared plinth. The same path data
 * as the iPhone app, so the pieces match exactly.
 */
object PieceArt {
    private const val PLINTH =
        "M22 86.5C22 84 23.5 82.5 26 82.5H74C76.5 82.5 78 84 78 86.5V87.5C78 89.5 76.5 90.5 74.5 90.5H25.5C23.5 90.5 22 89.5 22 87.5Z"

    class Art(val body: String, val detail: String, val eye: Triple<Float, Float, Float>? = null)

    val art: Map<Char, Art> = mapOf(
        'p' to Art(
            "M33 82.5C35 74 40 67 43 61H57C60 67 65 74 67 82.5Z M37 61C37 58 41 56.5 50 56.5C59 56.5 63 58 63 61Z M50 27A14 14 0 1 1 49.99 27Z",
            "M38 61H62",
        ),
        'r' to Art(
            "M30 82.5C32 76 34 70 35 64V44H65V64C66 70 68 76 70 82.5Z M30 44V24H38V30H45V24H55V30H62V24H70V44Z",
            "M35 44H65M34 64H66",
        ),
        'b' to Art(
            "M32 82.5C35 76 40 71 42 66H58C60 71 65 76 68 82.5Z M38 66C38 63 42 61.5 50 61.5C58 61.5 62 63 62 66Z M50 20C61 28 66 37 66 46C66 54 59 60 50 60C41 60 34 54 34 46C34 37 39 28 50 20Z M50 11A5 5 0 1 1 49.99 11Z",
            "M44 33L57 46M40 66H60",
        ),
        'n' to Art(
            "M31 82.5C32 73 37 66 41 61C34 60 27 59 23 56C18 53 17.5 47.5 20.5 43.5L37 25C40.5 20 45 15 49.5 10.5C52 14 54 18 55.5 22C68 26 76 37 75.5 52C75 64 72.5 73 71.5 82.5Z",
            "", Triple(40f, 33f, 2.6f),
        ),
        'q' to Art(
            "M30 82.5C33 74 38 66 40 58H60C62 66 67 74 70 82.5Z M36 58C36 55 41 53.5 50 53.5C59 53.5 64 55 64 58Z M35 53.5L27 27L38.5 40L44 21.5L50 38L56 21.5L61.5 40L73 27L65 53.5Z M27 20A4 4 0 1 1 26.99 20Z M44 14.5A4 4 0 1 1 43.99 14.5Z M56 14.5A4 4 0 1 1 55.99 14.5Z M73 20A4 4 0 1 1 72.99 20Z",
            "M38 58H62M36 47H64",
        ),
        'k' to Art(
            "M30 82.5C33 74 38 66 40 58H60C62 66 67 74 70 82.5Z M36 58C36 55 41 53.5 50 53.5C59 53.5 64 55 64 58Z M35 53.5C33 46 30 39 31 34C32.5 28 41 27 50 27C59 27 67.5 28 69 34C70 39 67 46 65 53.5Z M46.5 26V19H40.5V13H46.5V6.5H53.5V13H59.5V19H53.5V26Z",
            "M38 58H62M33 40H67",
        ),
    )

    /** The silhouette (plinth and body) laid out in `rect`, scaled by its width. */
    fun silhouette(kind: Char, rect: Rect): Path {
        val a = art[kind] ?: return Path()
        return parse(PLINTH + a.body, rect)
    }

    /** The engraved lines: collars, the bishop's slit. */
    fun detail(kind: Char, rect: Rect): Path = parse(art[kind]?.detail ?: "", rect)

    /** The knight's eye. */
    fun eye(kind: Char, rect: Rect): Path {
        val e = art[kind]?.eye ?: return Path()
        val s = rect.width / 100f
        return Path().apply {
            addOval(Rect(rect.left + (e.first - e.third) * s, rect.top + (e.second - e.third) * s,
                rect.left + (e.first + e.third) * s, rect.top + (e.second + e.third) * s))
        }
    }

    /** The small SVG path subset the art uses: absolute M L H V C Z, and A for whole circles. */
    fun parse(d: String, rect: Rect): Path {
        val s = rect.width / 100f
        fun px(x: Float) = rect.left + x * s
        fun py(y: Float) = rect.top + y * s
        val tokens = mutableListOf<String>()
        val number = StringBuilder()
        fun flush() { if (number.isNotEmpty()) { tokens += number.toString(); number.clear() } }
        for (ch in d) {
            when {
                ch.isLetter() -> { flush(); tokens += ch.toString() }
                ch == ' ' || ch == ',' -> flush()
                ch == '-' && number.isNotEmpty() && !number.endsWith("e") -> { flush(); number.append(ch) }
                else -> number.append(ch)
            }
        }
        flush()

        val path = Path()
        var i = 0
        var cmd = "M"
        var cx = 0f
        var cy = 0f
        fun next(): Float = tokens[i++].toFloatOrNull() ?: 0f
        while (i < tokens.size) {
            val t = tokens[i]
            if (t[0].isLetter()) {
                cmd = t
                i++
                if (cmd == "Z") { path.close(); continue }
            }
            when (cmd) {
                "M" -> { cx = next(); cy = next(); path.moveTo(px(cx), py(cy)); cmd = "L" }
                "L" -> { cx = next(); cy = next(); path.lineTo(px(cx), py(cy)) }
                "H" -> { cx = next(); path.lineTo(px(cx), py(cy)) }
                "V" -> { cy = next(); path.lineTo(px(cx), py(cy)) }
                "C" -> {
                    val x1 = next(); val y1 = next(); val x2 = next(); val y2 = next()
                    cx = next(); cy = next()
                    path.cubicTo(px(x1), py(y1), px(x2), py(y2), px(cx), py(cy))
                }
                "A" -> {
                    // The art only uses arcs for whole circles, starting at the top.
                    val r = next(); next(); next(); next(); next()
                    val ex = next(); val ey = next()
                    val ccx = cx; val ccy = cy + r
                    path.addOval(Rect(px(ccx - r), py(ccy - r), px(ccx + r), py(ccy + r)))
                    cx = ex; cy = ey
                }
                else -> i++
            }
        }
        return path
    }
}
