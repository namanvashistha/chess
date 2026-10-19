package com.namanvashistha.calmchess

import android.content.Context
import android.content.pm.ApplicationInfo
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Canvas
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.graphics.drawscope.CanvasDrawScope
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.clipPath
import androidx.compose.ui.graphics.drawscope.scale
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.text.drawText
import java.io.File
import kotlin.random.Random

/**
 * Renders the launcher icon's layers from the app's own knight and stone, so
 * the icon always matches the pieces, as the iPhone app renders its icon. A
 * debug build launched with `--ez renderIcon true` writes them to the app's
 * files folder: background.png, foreground.png and monochrome.png, each 432
 * pixels square (108 dp at 4x).
 */
object IconRenderer {
    private const val SIZE = 432

    fun renderIfAsked(context: Context, asked: Boolean) {
        if (!asked || context.applicationInfo.flags and ApplicationInfo.FLAG_DEBUGGABLE == 0) return
        val dir = File(context.filesDir, "icon").apply { mkdirs() }
        save(File(dir, "background.png")) { background() }
        save(File(dir, "foreground.png")) { knight() }
        save(File(dir, "monochrome.png")) { silhouette() }
        // The store's icon: the full square, layers together, 512 pixels.
        save(File(dir, "store.png"), 512) { scale(512f / SIZE, Offset.Zero) { background(); knight() } }
        feature(context, File(dir, "feature.png"))
    }

    /**
     * Google Play's feature graphic, 1024 by 500: the app's name and its line
     * on the stone background, beside a board from a calm opening with a line
     * of light, as the app draws them.
     */
    private fun feature(context: Context, file: File) {
        val w = 1024
        val h = 500
        val image = ImageBitmap(w, h)
        val text = androidx.compose.ui.text.TextMeasurer(androidx.compose.ui.text.font.createFontFamilyResolver(context), Density(1f), LayoutDirection.Ltr)
        val p = Palette.Light
        CanvasDrawScope().draw(Density(1f), LayoutDirection.Ltr, Canvas(image), Size(w.toFloat(), h.toFloat())) {
            drawRect(Brush.verticalGradient(listOf(p.backgroundTop, p.backgroundBottom)))
            drawRect(Brush.radialGradient(listOf(p.glow, Color.Transparent), Offset(260f, 0f), 520f))
            // The board on the right, tilted toward you by nothing but its shadow.
            val board = 420f
            val left = w - board - 56f
            val top = (h - board) / 2
            val radius = androidx.compose.ui.geometry.CornerRadius(6f)
            drawRoundRect(Color.Black.copy(alpha = 0.10f), Offset(left, top + 12f), Size(board, board), radius)
            val q = board / 8
            val position = "r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R"
            val squares = buildList { for (ch in position) if (ch != '/') { if (ch.isDigit()) repeat(ch.digitToInt()) { add('.') } else add(ch) } }
            val clip = androidx.compose.ui.graphics.Path().apply { addRoundRect(androidx.compose.ui.geometry.RoundRect(left, top, left + board, top + board, radius)) }
            clipPath(clip) {
                for (sq in 0 until 64) drawRect(if ((sq / 8 + sq % 8) % 2 == 1) p.squareDark else p.squareLight,
                    Offset(left + (sq % 8) * q, top + (sq / 8) * q), Size(q, q))
                // The last move in honey, and a line of light from the bishop.
                for (sq in listOf(1, 18)) drawRect(p.lastMove.copy(alpha = if (sq == 18) 0.4f else 0.26f), Offset(left + (sq % 8) * q, top + (sq / 8) * q), Size(q, q))
                val from = Offset(left + 5.5f * q, top + 7.5f * q)
                val to = Offset(left + 1.5f * q, top + 3.5f * q)
                drawLine(p.ray, from, to, q * 0.075f, androidx.compose.ui.graphics.StrokeCap.Round)
                drawCircle(p.ray, q * 0.1f, to)
                for (sq in 0 until 64) if (squares[sq] != '.') translate(left + (sq % 8) * q, top + (sq / 8) * q) {
                    contactShadow(q, 0f, p.accent); glass(squares[sq], q)
                }
            }
            // The name and the line.
            val name = text.measure("Calm Chess", Typeface.display(76f).copy(color = p.ink, letterSpacing = androidx.compose.ui.unit.TextUnit(-1.2f, androidx.compose.ui.unit.TextUnitType.Sp)))
            drawText(name, topLeft = Offset(64f, h / 2f - name.size.height + 4f))
            val line = text.measure("A kind coach beside you", androidx.compose.ui.text.TextStyle(fontSize = androidx.compose.ui.unit.TextUnit(28f, androidx.compose.ui.unit.TextUnitType.Sp), color = p.mist))
            drawText(line, topLeft = Offset(68f, h / 2f + 14f))
        }
        file.outputStream().use { image.asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, it) }
    }

    private fun save(file: File, px: Int = SIZE, draw: DrawScope.() -> Unit) {
        val image = ImageBitmap(px, px)
        CanvasDrawScope().draw(Density(1f), LayoutDirection.Ltr, Canvas(image), Size(px.toFloat(), px.toFloat()), draw)
        file.outputStream().use { image.asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, it) }
    }

    /** Four squares of the board, limestone and slate, with grain and the board's light from the top left. */
    private fun DrawScope.background() {
        val p = Palette.Light
        val h = SIZE / 2f
        drawRect(p.squareLight, Offset.Zero, Size(h, h))
        drawRect(p.squareDark, Offset(h, 0f), Size(h, h))
        drawRect(p.squareDark, Offset(0f, h), Size(h, h))
        drawRect(p.squareLight, Offset(h, h), Size(h, h))
        val rng = Random(7)
        repeat(SIZE * SIZE / 120) {
            drawCircle(Color.Black.copy(alpha = (0.04f + rng.nextFloat() * 0.05f) * 0.35f), 0.3f + rng.nextFloat() * 0.5f,
                Offset(rng.nextFloat() * SIZE, rng.nextFloat() * SIZE))
        }
        drawRect(Brush.radialGradient(listOf(Color.White.copy(alpha = 0.45f), Color.Transparent), Offset(SIZE * 0.3f, SIZE * 0.25f), SIZE * 0.68f),
            blendMode = BlendMode.Softlight)
    }

    /** The white knight, at the same share of the visible icon as on iPhone, standing a little low like a piece on its square. */
    private fun DrawScope.knight() {
        // The visible icon is the middle 72 of 108 dp; the iPhone's knight box fills 78% of its icon.
        val box = SIZE * (72f / 108f) * 0.78f
        translate((SIZE - box) / 2, (SIZE - box) / 2 + box * 0.02f) {
            contactShadow(box, 0f, Palette.Light.accent)
            glass('N', box)
        }
    }

    /** The knight's shape alone, for themed icons: the system tints it. */
    private fun DrawScope.silhouette() {
        val box = SIZE * (72f / 108f) * 0.78f
        translate((SIZE - box) / 2, (SIZE - box) / 2 + box * 0.02f) {
            val frame = Rect(box * 0.04f, box * 0.02f, box - box * 0.04f, box - box * 0.05f)
            scale(1.05f, Offset(frame.left + frame.width * 0.5f, frame.top + frame.height * 0.9f)) {
                drawPath(PieceArt.silhouette('n', frame), Color.White)
            }
        }
    }
}
