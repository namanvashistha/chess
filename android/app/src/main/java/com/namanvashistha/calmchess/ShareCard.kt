package com.namanvashistha.calmchess

import android.content.ClipData
import android.content.Context
import android.content.Intent
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Canvas
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.graphics.drawscope.CanvasDrawScope
import androidx.compose.ui.graphics.drawscope.clipPath
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.text.TextMeasurer
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.font.createFontFamilyResolver
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.sp
import androidx.core.content.FileProvider
import java.io.File

/** A moment worth sharing (a great move, or a win), drawn as a picture of the board with a line about it. */
data class ShareMoment(val board: List<Char>, val move: BoardMove?, val title: String, val detail: String, val footer: String)

object ShareCard {
    /**
     * The picture itself, as on iPhone: 360 by 450 points at 3x, so 1080 by
     * 1350 pixels, always in the light palette so every share looks the same.
     */
    fun render(context: Context, m: ShareMoment): ImageBitmap {
        val scale = 3f
        val density = Density(scale, 1f)
        val p = Palette.Light
        val image = ImageBitmap((360 * scale).toInt(), (450 * scale).toInt())
        val text = TextMeasurer(createFontFamilyResolver(context), density, LayoutDirection.Ltr)
        CanvasDrawScope().draw(density, LayoutDirection.Ltr, Canvas(image), Size(image.width.toFloat(), image.height.toFloat())) {
            val pad = 22 * scale
            val width = size.width - pad * 2
            drawRect(Brush.verticalGradient(listOf(p.backgroundTop, p.backgroundBottom)))
            // The title and the line about it.
            val title = text.measure(m.title, Typeface.display(30f).copy(color = p.ink, letterSpacing = (-0.5).sp), maxLines = 1,
                overflow = TextOverflow.Ellipsis, constraints = Constraints(maxWidth = width.toInt()))
            drawText(title, topLeft = Offset(pad, pad))
            val detail = text.measure(m.detail, TextStyle(fontSize = 15.sp, color = p.mist), maxLines = 2,
                overflow = TextOverflow.Ellipsis, constraints = Constraints(maxWidth = width.toInt()))
            drawText(detail, topLeft = Offset(pad, pad + title.size.height + 4 * scale))
            // The board, with the move in green.
            val top = pad + title.size.height + 4 * scale + detail.size.height + 14 * scale
            val boardSize = minOf(width, size.height - top - 40 * scale)
            val left = pad
            val radius = CornerRadius(6 * scale)
            drawRoundRect(androidx.compose.ui.graphics.Color.Black.copy(alpha = 0.08f), Offset(left, top + 6 * scale), Size(boardSize, boardSize), radius)
            val shape = Path().apply { addRoundRect(androidx.compose.ui.geometry.RoundRect(left, top, left + boardSize, top + boardSize, radius)) }
            clipPath(shape) {
                val q = boardSize / 8
                for (sq in 0 until 64) {
                    val tl = Offset(left + (sq % 8) * q, top + (sq / 8) * q)
                    drawRect(if ((sq / 8 + sq % 8) % 2 == 1) p.squareDark else p.squareLight, tl, Size(q, q))
                    m.move?.let { mv -> if (mv.from == sq || mv.to == sq) drawRect(p.great.copy(alpha = if (sq == mv.to) 0.4f else 0.22f), tl, Size(q, q)) }
                }
                for (row in 0..7) for (col in 0..7) {
                    val c = m.board.getOrNull(row * 8 + col) ?: continue
                    if (c == '.') continue
                    translate(left + col * q, top + row * q) {
                        contactShadow(q, 0f, p.accent)
                        glass(c, q)
                    }
                }
            }
            // The footer: a sparkle and where it was played.
            val footer = text.measure(m.footer, TextStyle(fontFamily = Typeface.rounded, fontWeight = FontWeight(750), fontSize = 13.sp, color = p.accent))
            val fy = top + boardSize + 14 * scale
            sparkleStar(Offset(pad + 6 * scale, fy + footer.size.height / 2f - 2 * scale), 5 * scale, p.accent)
            sparkleStar(Offset(pad + 11 * scale, fy + footer.size.height / 2f + 3 * scale), 3 * scale, p.accent)
            drawText(footer, topLeft = Offset(pad + 20 * scale, fy))
        }
        return image
    }

    /** Opens the share sheet with the picture, as a PNG named after the app. */
    fun share(context: Context, m: ShareMoment) {
        val dir = File(context.cacheDir, "share").apply { mkdirs() }
        val file = File(dir, "Calm Chess.png")
        file.outputStream().use { render(context, m).asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, it) }
        val uri = FileProvider.getUriForFile(context, "${context.packageName}.share", file)
        val send = Intent(Intent.ACTION_SEND).apply {
            type = "image/png"
            putExtra(Intent.EXTRA_STREAM, uri)
            putExtra(Intent.EXTRA_TITLE, m.title)
            clipData = ClipData.newRawUri(m.title, uri)
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }
        context.startActivity(Intent.createChooser(send, null))
    }
}
