package com.namanvashistha.calmchess

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Paint
import androidx.compose.ui.graphics.PaintingStyle
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipPath
import androidx.compose.ui.graphics.drawscope.drawIntoCanvas
import androidx.compose.ui.graphics.drawscope.scale
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.toArgb
import kotlin.math.max

/**
 * One chess piece in glass: milk glass for White, polished smoked glass for
 * Black, lit from the top left, with a contact shadow where its base meets the
 * board. Drawn exactly as on iPhone: a tint, a gradient fill with a halo, a
 * highlight, a lit and a shaded edge, a fine contour, then the details.
 */
@Composable
fun GlassPiece(
    code: Char,
    modifier: Modifier = Modifier,
    lifted: Boolean = false,
    /** Standing on a board, with a shadow under it. */
    grounded: Boolean = true,
) {
    val palette = LocalPalette.current
    // A lifted piece rises off the board; its shadow stays behind.
    val lift by animateFloatAsState(if (lifted) 1f else 0f, swiftSpring(0.3f, 0.82f), label = "lift")
    Box(modifier.aspectRatio(1f)) {
        Canvas(Modifier.matchParentSize()) {
            val s = size.width
            if (grounded) contactShadow(s, lift, palette.accent)
            translate(top = -s * 0.09f * lift) { glass(code, s) }
        }
    }
}

fun DrawScope.contactShadow(s: Float, lift: Float, accent: Color) {
    fun lerp(a: Float, b: Float) = a + (b - a) * lift
    val c = Offset(s / 2, s / 2)
    // A soft pool of the accent marks the square a lifted piece came from.
    if (lift > 0f) softEllipse(accent.copy(alpha = 0.32f * lift), c + Offset(0f, s * 0.35f), s * 0.86f, s * 0.26f)
    softEllipse(Color.Black.copy(alpha = lerp(0.2f, 0.12f)), c + Offset(s * lerp(0.05f, 0.07f), s * 0.38f),
        s * 0.94f * lerp(1f, 1.12f), s * 0.22f * lerp(1f, 1.12f))
    softEllipse(Color.Black.copy(alpha = lerp(0.5f, 0.14f)), c + Offset(s * 0.015f, s * 0.357f),
        s * 0.66f * lerp(1f, 1.18f), s * 0.085f * lerp(1f, 1.18f))
}

/** An ellipse that fades out evenly to every edge: a round gradient, squashed. */
private fun DrawScope.softEllipse(color: Color, center: Offset, width: Float, height: Float) {
    if (width <= 0f || height <= 0f) return
    val r = width / 2
    scale(1f, height / width, pivot = center) {
        drawCircle(
            Brush.radialGradient(
                0f to color, 0.45f to color.copy(alpha = color.alpha * 0.45f), 1f to color.copy(alpha = 0f),
                center = center, radius = r,
            ),
            radius = r, center = center,
        )
    }
}

fun DrawScope.glass(code: Char, s: Float) {
    val white = code.isUpperCase()
    val kind = code.lowercaseChar()
    // The art box: padded, then a touch larger, standing on its plinth.
    val frame = Rect(s * 0.04f, s * 0.02f, s - s * 0.04f, s - s * 0.05f)
    val pivot = Offset(frame.left + frame.width * 0.5f, frame.top + frame.height * 0.9f)
    val shape = PieceArt.silhouette(kind, frame)
    val start = frame.topLeft
    val end = frame.bottomRight

    scale(1.05f, pivot = pivot) {
        // The tint under the glass, dense enough that the square only hints through.
        drawPath(shape, if (white) Color.White.copy(alpha = 0.8f) else Color.Black.copy(alpha = 0.8f))

        // The fill, with a halo just outside the silhouette: dark around White,
        // light around Black, so each side stays clear of both squares.
        val fill = if (white) Brush.linearGradient(
            0f to Color.White, 0.45f to Color(0.975f, 0.97f, 0.96f), 1f to Color(0.80f, 0.815f, 0.84f),
            start = start, end = end,
        ) else Brush.linearGradient(
            0f to Color(0.22f, 0.235f, 0.26f), 0.5f to Color(0.13f, 0.14f, 0.16f), 1f to Color(0.075f, 0.08f, 0.095f),
            start = start, end = end,
        )
        val halo = if (white) Color.Black.copy(alpha = 0.58f) else Color.White.copy(alpha = 0.46f)
        drawIntoCanvas { canvas ->
            val paint = Paint().apply { style = PaintingStyle.Fill }
            fill.applyTo(Size(s, s), paint, 1f)
            paint.asFrameworkPaint().apply {
                isAntiAlias = true
                setShadowLayer(s * 0.016f, 0f, 0f, halo.toArgb())
            }
            canvas.drawPath(shape, paint)
        }

        clipPath(shape) {
            // The highlight, from the top left.
            drawRect(
                Brush.radialGradient(
                    listOf(Color.White.copy(alpha = if (white) 0.9f else 0.16f), Color.Transparent),
                    center = Offset(frame.left + frame.width * 0.34f, frame.top + frame.height * 0.22f),
                    radius = s * 0.42f,
                ),
                blendMode = BlendMode.Screen,
            )
            // The lit edge and the shaded edge.
            val lit = if (white) Color.White else Color(0.91f, 0.93f, 0.96f)
            drawPath(
                shape,
                Brush.linearGradient(
                    0f to lit.copy(alpha = if (white) 1f else 0.4f),
                    0.55f to lit.copy(alpha = if (white) 0.25f else 0.08f),
                    1f to lit.copy(alpha = 0f),
                    start = start, end = end,
                ),
                style = Stroke(width = s * 0.06f),
            )
            val shade = Color(0.16f, 0.19f, 0.21f)
            drawPath(
                shape,
                Brush.linearGradient(
                    0f to shade.copy(alpha = 0f), 0.5f to shade.copy(alpha = 0.32f),
                    1f to shade.copy(alpha = if (white) 0.5f else 0.6f),
                    start = start, end = end,
                ),
                style = Stroke(width = s * 0.06f),
            )
        }

        // A fine contour of the same weight on both sides.
        drawPath(
            shape,
            if (white) Color.Black.copy(alpha = 0.6f) else Color.White.copy(alpha = 0.38f),
            style = Stroke(width = max(1f, s * 0.017f), join = StrokeJoin.Round),
        )
        drawPath(
            PieceArt.detail(kind, frame),
            if (white) Color(0.4f, 0.4f, 0.4f, 0.6f) else Color(0.9f, 0.9f, 0.9f, 0.45f),
            style = Stroke(width = s * 0.016f, cap = StrokeCap.Round),
        )
        drawPath(PieceArt.eye(kind, frame), if (white) Color(0.25f, 0.25f, 0.25f, 0.7f) else Color(0.92f, 0.92f, 0.92f, 0.65f))
    }
}
