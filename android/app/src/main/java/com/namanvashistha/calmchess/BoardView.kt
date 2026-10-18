package com.namanvashistha.calmchess

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.animateOffsetAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathMeasure
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.TransformOrigin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.input.pointer.positionChange
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.zIndex
import kotlin.math.abs
import kotlin.math.floor
import kotlin.math.hypot
import kotlin.math.roundToInt

/** Square centre in board coordinates. */
fun center(sq: Square, size: Float): Offset {
    val q = size / 8
    return Offset((Squares.file(sq) + 0.5f) * q, (Squares.row(sq) + 0.5f) * q)
}

/**
 * The Stone board: limestone and slate squares with a faint grain, glass
 * pieces that glide and settle, lines of light for moves, and a soft light on
 * the side whose move it is. One finger: tap a piece then a square, or carry it.
 */
@Composable
fun BoardView(model: GameModel, modifier: Modifier = Modifier) {
    val palette = LocalPalette.current
    BoxWithConstraints(
        modifier
            .aspectRatio(1f)
            .shadow(18.dp, RoundedCornerShape(6.dp), ambientColor = Color.Black.copy(alpha = 0.3f), spotColor = Color.Black.copy(alpha = 0.3f))
            .clip(RoundedCornerShape(6.dp))
    ) {
        val sizePx = with(LocalDensity.current) { maxWidth.toPx() }
        val qDp = maxWidth / 8
        val q = sizePx / 8
        var dragPoint by remember { mutableStateOf<Offset?>(null) }
        var dragFrom by remember { mutableStateOf<Square?>(null) }

        fun squareAt(p: Offset): Square? {
            val c = floor(p.x / q).toInt()
            val r = floor(p.y / q).toInt()
            return if (c in 0..7 && r in 0..7) r * 8 + c else null
        }

        Squares(model, palette)
        RaysLayer(model.rays, sizePx, palette)
        Box(
            Modifier
                .matchParentSize()
                .pointerInput(model, sizePx) {
                    awaitEachGesture {
                        val down = awaitFirstDown()
                        val start = squareAt(down.position) ?: return@awaitEachGesture
                        var dragging = false
                        var moved = false
                        while (true) {
                            val event = awaitPointerEvent()
                            val change = event.changes.firstOrNull { it.id == down.id } ?: break
                            if (!change.pressed) {
                                if (dragging) model.drop(squareAt(change.position))
                                else if (!moved) model.tap(start)
                                break
                            }
                            val d = change.position - down.position
                            if (!dragging && !moved && hypot(d.x, d.y) > q * 0.2f) {
                                if (model.pickUp(start)) { dragging = true; dragFrom = start } else moved = true
                            }
                            if (dragging) {
                                dragPoint = change.position
                                if (change.positionChange() != Offset.Zero) change.consume()
                            }
                        }
                        dragPoint = null
                        dragFrom = null
                    }
                }
        ) {
            for (piece in model.pieces) {
                key(piece.id) {
                    val lifted = piece.square == model.selected
                    val dragged = dragFrom == piece.square && lifted && dragPoint != null
                    val target = center(piece.square, sizePx)
                    val glide by animateOffsetAsState(target, swiftSpring(0.42f, 0.72f), label = "glide")
                    val scale by animateFloatAsState(if (dragged) 1.28f else if (lifted) 1.06f else 1f, swiftSpring(0.22f, 0.7f), label = "scale")
                    // Held a little above the finger so the finger doesn't hide it.
                    val at = if (dragged) dragPoint!! + Offset(0f, -q * 0.25f) else glide
                    GlassPiece(
                        piece.code,
                        lifted = lifted,
                        modifier = Modifier
                            .size(qDp)
                            .offset { IntOffset((at.x - q / 2).roundToInt(), (at.y - q / 2).roundToInt()) }
                            .graphicsLayer {
                                scaleX = scale; scaleY = scale
                                transformOrigin = TransformOrigin(0.5f, 1f)
                            }
                            .zIndex(if (dragged) 30f else if (piece.id == model.movingPiece) 20f else Squares.row(piece.square).toFloat()),
                    )
                }
            }
        }
    }
}

/** The squares, the last move's honey wash, check, coordinates, grain and the turn light. */
@Composable
private fun Squares(model: GameModel, palette: Palette) {
    val measurer = rememberTextMeasurer()
    val last = model.lastMove
    val check = model.checkSquare
    val lightY by animateFloatAsState(if (model.turn == "w") 0.85f else 0.15f, tween(900, easing = FastOutSlowInEasing), label = "light")
    Canvas(
        Modifier
            .aspectRatio(1f)
            .drawWithCache {
                // A fine stone grain, the same every time.
                val w = size.width
                val h = size.height
                val dots = buildList {
                    var seed = 0x9E3779B97F4A7C15uL
                    fun rand(): Float {
                        seed = seed * 6364136223846793005uL + 1442695040888963407uL
                        return (seed shr 40).toFloat() / (1 shl 24).toFloat()
                    }
                    repeat((w * h / 120).toInt()) {
                        add(Triple(Offset(rand() * w, rand() * h), 0.6f + rand() * 1.1f, 0.04f + rand() * 0.05f))
                    }
                }
                onDrawBehind {
                    val q = size.width / 8
                    for (sq in 0 until 64) {
                        val dark = (Squares.row(sq) + Squares.file(sq)) % 2 == 1
                        val tl = Offset(Squares.file(sq) * q, Squares.row(sq) * q)
                        drawRect(if (dark) palette.squareDark else palette.squareLight, tl, Size(q, q))
                        if (last != null && (last.from == sq || last.to == sq)) {
                            drawRect(palette.lastMove.copy(alpha = if (sq == last.to) 0.4f else 0.26f), tl, Size(q, q))
                        }
                        if (sq == check) {
                            drawRect(Brush.radialGradient(listOf(palette.danger.copy(alpha = 0.75f), Color.Transparent),
                                center = tl + Offset(q / 2, q / 2), radius = q * 0.6f), tl, Size(q, q))
                        }
                    }
                    for ((p, r, a) in dots) drawCircle(Color.Black.copy(alpha = a * 0.35f), r, p)
                    // The light rests on the side whose move it is.
                    drawRect(
                        Brush.radialGradient(listOf(Color.White.copy(alpha = 0.38f), Color.White.copy(alpha = 0.09f), Color.Transparent),
                            center = Offset(size.width / 2, size.height * lightY), radius = 260.dp.toPx()),
                        blendMode = BlendMode.Softlight,
                    )
                    // Coordinates, quiet, in the corners of the edge squares.
                    val style = TextStyle(color = palette.coordinate.copy(alpha = 0.6f), fontSize = (q * 0.15f / density).let { androidx.compose.ui.unit.TextUnit(it, androidx.compose.ui.unit.TextUnitType.Sp) } , fontWeight = FontWeight.Bold)
                    for (r in 0..7) {
                        drawText(measurer, "${8 - r}", Offset(q * 0.035f, r * q + q * 0.02f), style)
                    }
                    for (c in 0..7) {
                        val t = measurer.measure("${'a' + c}", style)
                        drawText(t, topLeft = Offset((c + 1) * q - t.size.width - q * 0.035f, 8 * q - t.size.height - q * 0.02f))
                    }
                }
            }
    ) {}
}

/** Lines of light from a piece to where it can go. A knight's line bends. */
@Composable
private fun RaysLayer(rays: Rays?, size: Float, palette: Palette) {
    if (rays == null) return
    key(rays.id) {
        val color = when (rays.kind) {
            Rays.Kind.Options, Rays.Kind.Chosen -> palette.ray
            Rays.Kind.Thinking -> palette.thinking
            Rays.Kind.Hint -> palette.great
        }
        val progress = remember { Animatable(0f) }
        LaunchedEffect(Unit) { progress.animateTo(1f, tween(320 + rays.moves.size * 20, easing = FastOutSlowInEasing)) }
        val q = size / 8
        val width = q * (if (rays.kind == Rays.Kind.Thinking) 0.05f else 0.075f)
        val alpha = if (rays.kind == Rays.Kind.Thinking) 0.7f else 1f
        Canvas(Modifier.aspectRatio(1f)) {
            rays.moves.forEachIndexed { k, m ->
                val c = if (m in rays.danger) palette.danger else color
                val local = ((progress.value * (rays.moves.size * 20 + 320) - k * 20) / 320f).coerceIn(0f, 1f)
                if (local <= 0f) return@forEachIndexed
                val a = center(m.from, size)
                val b = center(m.to, size)
                val path = Path().apply {
                    moveTo(a.x, a.y)
                    if (m in rays.knight) {
                        val horizontal = abs(b.x - a.x) > abs(b.y - a.y)
                        if (horizontal) lineTo(b.x, a.y) else lineTo(a.x, b.y)
                    }
                    lineTo(b.x, b.y)
                }
                val measure = PathMeasure().apply { setPath(path, false) }
                val drawn = Path()
                measure.getSegment(0f, measure.length * local, drawn, true)
                drawPath(drawn, c.copy(alpha = alpha), style = Stroke(width, cap = StrokeCap.Round, join = StrokeJoin.Round))
                if (m.capture) {
                    drawCircle(c.copy(alpha = alpha), q * 0.38f * local, b, style = Stroke(width * 0.8f))
                } else {
                    drawCircle(c.copy(alpha = alpha), q * 0.1f * local, b)
                }
            }
        }
    }
}
