package com.namanvashistha.calmchess

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.LinearOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.animateOffsetAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.border
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
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
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.graphics.drawscope.translate
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
import androidx.compose.ui.unit.sp
import androidx.compose.ui.zIndex
import kotlin.math.PI
import kotlin.math.abs
import kotlin.math.cos
import kotlin.math.floor
import kotlin.math.hypot
import kotlin.math.max
import kotlin.math.roundToInt
import kotlin.math.sin
import kotlin.random.Random

/**
 * What the board needs from whatever drives it: a game, the welcome, or a
 * Learn moment. The same board plays all three.
 */
interface BoardSource {
    val shownPieces: List<PieceItem>
    val selected: Square?
    val rays: Rays?
    val bursts: List<Burst>
    val toppled: Int?
    val movingPiece: Int?
    /** Whose move it is, "w" or "b": the board's light rests on that side. */
    val turn: String
    val reviewing: Int?
    val shownLastMove: BoardMove?
    val sparkle: Pair<Int, Square>?
    val nudge: Square?
    val loose: Set<Square>
    val checkSquare: Square?
    val shownMark: Pair<TrailEntry, Square>?
    fun pickUp(square: Square): Boolean
    fun drop(on: Square?)
    fun inspect(square: Square): Boolean
    fun endInspect()
    fun tap(square: Square)
    /** The square a carried piece is over; nil when it's put down. The welcome's coach thinks out loud. */
    fun hover(square: Square?) {}
    /** Pieces arriving are set down by hand rather than appearing. The welcome lays its table this way. */
    val placesPieces: Boolean get() = false
}

/** Square centre in board coordinates. */
fun center(sq: Square, size: Float): Offset {
    val q = size / 8
    return Offset((Squares.file(sq) + 0.5f) * q, (Squares.row(sq) + 0.5f) * q)
}

/** The colour of a verdict: its badge on the board and its title in the coach card. */
fun verdictColor(p: Palette, k: CoachCard.Kind) = when (k) {
    CoachCard.Kind.Great -> p.great
    CoachCard.Kind.Inaccuracy -> p.inaccuracy
    CoachCard.Kind.Mistake, CoachCard.Kind.Warning -> p.mistake
    CoachCard.Kind.Blunder -> p.blunder
    else -> p.accent
}

/**
 * The Stone board: limestone and slate squares with a faint grain, glass
 * pieces that glide and settle, lines of light for moves, and a soft light on
 * the side whose move it is. Tap a piece then a square, carry it, or hold a
 * piece to see what guards and attacks it.
 */
@Composable
fun BoardView(model: BoardSource, modifier: Modifier = Modifier) {
    val palette = LocalPalette.current
    val shape = RoundedCornerShape(6.dp)
    BoxWithConstraints(
        modifier
            .aspectRatio(1f)
            .shadow(18.dp, shape, ambientColor = Color.Black.copy(alpha = 0.3f), spotColor = Color.Black.copy(alpha = 0.3f))
            .clip(shape)
            .border(1.dp, Color.Black.copy(alpha = 0.12f), shape)
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

        SquaresLayer(model, palette)
        RaysLayer(model.rays, sizePx, palette)
        Box(
            Modifier
                .matchParentSize()
                .pointerInput(model, sizePx) {
                    awaitEachGesture {
                        val down = awaitFirstDown()
                        val start = squareAt(down.position) ?: return@awaitEachGesture
                        val pressedAt = System.currentTimeMillis()
                        var dragging = false
                        var moved = false
                        var inspecting = false
                        while (true) {
                            val waiting = !dragging && !moved && !inspecting
                            val left = 450 - (System.currentTimeMillis() - pressedAt)
                            val event = if (waiting && left > 0) withTimeoutOrNull(left) { awaitPointerEvent() } else awaitPointerEvent()
                            if (event == null) {
                                // Held still: show what guards and attacks this piece.
                                inspecting = model.inspect(start)
                                if (!inspecting) moved = true
                                continue
                            }
                            val change = event.changes.firstOrNull { it.id == down.id } ?: break
                            if (!change.pressed) {
                                when {
                                    inspecting -> model.endInspect()
                                    dragging -> model.drop(squareAt(change.position))
                                    !moved -> model.tap(start)
                                }
                                break
                            }
                            val d = change.position - down.position
                            if (!dragging && !moved && !inspecting && hypot(d.x, d.y) > q * 0.2f) {
                                if (model.pickUp(start)) { dragging = true; dragFrom = start } else moved = true
                            }
                            if (dragging) {
                                dragPoint = change.position
                                model.hover(squareAt(change.position))
                                if (change.positionChange() != Offset.Zero) change.consume()
                            }
                        }
                        model.hover(null)
                        dragPoint = null
                        dragFrom = null
                    }
                }
        ) {
            for (piece in model.shownPieces) {
                key(piece.id) {
                    val lifted = piece.square == model.selected
                    val dragged = dragFrom == piece.square && lifted && dragPoint != null
                    val toppled = piece.id == model.toppled
                    val glide by animateOffsetAsState(center(piece.square, sizePx), swiftSpring(0.42f, 0.72f), label = "glide")
                    val scale by animateFloatAsState(if (dragged) 1.28f else if (lifted) 1.06f else 1f, swiftSpring(0.22f, 0.7f), label = "scale")
                    val tip by animateFloatAsState(if (toppled) 1f else 0f, swiftSpring(0.9f, 0.55f), label = "topple")
                    // Set down by hand: from just above, a little large and clear, settling into place.
                    val placed = remember { Animatable(if (model.placesPieces) 0f else 1f) }
                    LaunchedEffect(Unit) { placed.animateTo(1f, swiftSpring(0.42f, 0.62f)) }
                    // Held a little above the finger so the finger doesn't hide it.
                    val at = if (dragged) dragPoint!! + Offset(0f, -q * 0.25f) else glide
                    GlassPiece(
                        piece.code,
                        lifted = lifted,
                        modifier = Modifier
                            .size(qDp)
                            .offset { IntOffset((at.x - q / 2 + q * 0.15f * tip).roundToInt(), (at.y - q / 2 - q * 0.5f * (1 - placed.value)).roundToInt()) }
                            .graphicsLayer {
                                val drop = 1f + 0.18f * (1 - placed.value)
                                scaleX = scale * drop; scaleY = scale * drop
                                alpha = placed.value.coerceIn(0f, 1f)
                                rotationZ = 82f * tip
                                transformOrigin = TransformOrigin(0.5f, if (tip > 0f) 0.88f else 1f)
                            }
                            .zIndex(if (dragged) 30f else if (piece.id == model.movingPiece) 20f else Squares.row(piece.square).toFloat()),
                    )
                }
            }
            MarkLayer(model, sizePx, palette)
            model.sparkle?.let { (id, sq) -> key(id) { SparkleBurst(center(sq, sizePx), q, palette) } }
            for (b in model.bursts) key(b.id) { ShardBurst(center(b.square, sizePx), q, b.dark) }
        }
    }
}

/** The squares, the last move's honey wash, check, a hint's nudge, loose pieces, coordinates, grain and the turn light. */
@Composable
private fun SquaresLayer(model: BoardSource, palette: Palette) {
    val measurer = rememberTextMeasurer()
    val last = model.shownLastMove
    val check = model.checkSquare
    val nudge = model.nudge
    val loose = if (model.rays == null && model.reviewing == null) model.loose else emptySet()
    val lightY by animateFloatAsState(if (model.turn == "w") 0.85f else 0.15f, tween(900, easing = FastOutSlowInEasing), label = "light")
    Canvas(
        Modifier
            .fillMaxSize()
            .drawWithCache {
                // A fine stone grain, the same every time.
                val w = size.width
                val h = size.height
                val rng = Random(7)
                val dots = List((w * h / 120).toInt()) {
                    Triple(Offset(rng.nextFloat() * w, rng.nextFloat() * h), 0.6f + rng.nextFloat() * 1.1f, 0.04f + rng.nextFloat() * 0.05f)
                }
                val q = w / 8
                val style = TextStyle(color = palette.coordinate.copy(alpha = 0.6f), fontSize = (q * 0.15f / density / fontScale).sp,
                    fontWeight = FontWeight.Bold, fontFamily = Typeface.rounded)
                val ranks = (0..7).map { measurer.measure("${8 - it}", style) }
                val files = (0..7).map { measurer.measure("${'a' + it}", style) }
                onDrawBehind {
                    for (sq in 0 until 64) {
                        val dark = (Squares.row(sq) + Squares.file(sq)) % 2 == 1
                        val tl = Offset(Squares.file(sq) * q, Squares.row(sq) * q)
                        val c = tl + Offset(q / 2, q / 2)
                        drawRect(if (dark) palette.squareDark else palette.squareLight, tl, Size(q, q))
                        if (last != null && (last.from == sq || last.to == sq)) {
                            // The last move: a honey wash, warmer toward the base where the piece landed.
                            drawRect(palette.lastMove.copy(alpha = if (sq == last.to) 0.4f else 0.26f), tl, Size(q, q))
                            if (sq == last.to) drawRect(Brush.radialGradient(listOf(palette.lastMove.copy(alpha = 0.45f), Color.Transparent),
                                center = tl + Offset(q / 2, q * 0.65f), radius = q * 0.55f), tl, Size(q, q))
                        }
                        if (sq == check) drawRect(Brush.radialGradient(listOf(palette.danger.copy(alpha = 0.75f), Color.Transparent),
                            center = c, radius = q * 0.6f), tl, Size(q, q))
                        if (sq == nudge) {
                            // A hint's first step: this piece has a good move.
                            drawRect(palette.great.copy(alpha = 0.22f), tl, Size(q, q))
                            drawCircle(palette.great, q * 0.45f - 1.25.dp.toPx(), c, style = Stroke(2.5.dp.toPx()))
                        }
                        if (sq in loose) drawCircle(palette.danger.copy(alpha = 0.7f), q * 0.44f - 1.dp.toPx(), c, style = Stroke(2.dp.toPx()))
                    }
                    for ((pt, r, a) in dots) drawCircle(Color.Black.copy(alpha = a * 0.35f), r, pt)
                    // The light rests on the side whose move it is.
                    drawRect(Brush.radialGradient(listOf(Color.White.copy(alpha = 0.38f), Color.White.copy(alpha = 0.09f), Color.Transparent),
                        center = Offset(w / 2, h * lightY), radius = 260.dp.toPx()), blendMode = BlendMode.Softlight)
                    // Coordinates, quiet, in the corners of the edge squares.
                    for (r in 0..7) drawText(ranks[r], topLeft = Offset(q * 0.035f, r * q + q * 0.02f))
                    for (c in 0..7) {
                        val t = files[c]
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
        val total = 320 + rays.moves.size * 20
        val progress = remember { Animatable(0f) }
        LaunchedEffect(Unit) { progress.animateTo(1f, tween(total, easing = FastOutSlowInEasing)) }
        val q = size / 8
        val width = q * (if (rays.kind == Rays.Kind.Thinking) 0.05f else 0.075f)
        val alpha = if (rays.kind == Rays.Kind.Thinking) 0.7f else 1f
        Canvas(Modifier.fillMaxSize()) {
            rays.moves.forEachIndexed { k, m ->
                val c = if (m in rays.danger) palette.danger else color
                val local = ((progress.value * total - k * 20) / 320f).coerceIn(0f, 1f)
                if (local <= 0f) return@forEachIndexed
                val a = center(m.from, size)
                val b = center(m.to, size)
                val path = Path().apply {
                    moveTo(a.x, a.y)
                    if (m in rays.knight) {
                        if (abs(b.x - a.x) > abs(b.y - a.y)) lineTo(b.x, a.y) else lineTo(a.x, b.y)
                    }
                    lineTo(b.x, b.y)
                }
                val measure = PathMeasure().apply { setPath(path, false) }
                val drawn = Path()
                measure.getSegment(0f, measure.length * local, drawn, true)
                drawPath(drawn, c.copy(alpha = alpha), style = Stroke(width, cap = StrokeCap.Round, join = StrokeJoin.Round))
                if (m.capture) drawCircle(c.copy(alpha = alpha), q * 0.38f * local, b, style = Stroke(width * 0.8f))
                else drawCircle(c.copy(alpha = alpha), q * 0.1f * local, b)
            }
        }
    }
}

/** The verdict on the last move, as a small badge on the piece that made it. */
@Composable
private fun MarkLayer(model: BoardSource, size: Float, palette: Palette) {
    val m = model.shownMark ?: return
    val icon = verdictIcon(m.first.card.kind) ?: return
    key(m.first.id) {
        val q = size / 8
        val pop = remember { Animatable(0.2f) }
        LaunchedEffect(Unit) { pop.animateTo(1f, swiftSpring(0.35f, 0.55f)) }
        val c = center(m.second, size)
        val d = q * 0.4f
        val dDp = with(LocalDensity.current) { d.toDp() }
        Box(
            Modifier
                .zIndex(25f)
                .offset { IntOffset((c.x + q * 0.33f - d / 2).roundToInt(), (c.y - q * 0.33f - d / 2).roundToInt()) }
                .size(dDp)
                .graphicsLayer { scaleX = pop.value; scaleY = pop.value; alpha = pop.value }
                .shadow(2.dp, RoundedCornerShape(50))
        ) {
            Canvas(Modifier.fillMaxSize()) {
                drawCircle(verdictColor(palette, m.first.card.kind))
                drawCircle(Color.White.copy(alpha = 0.9f), style = Stroke(max(1f, q * 0.03f)))
            }
            androidx.compose.material3.Icon(icon, null, tint = Color.White,
                modifier = Modifier.size(dDp * 0.55f).align(androidx.compose.ui.Alignment.Center))
        }
    }
}

/** A great move: a ring and a few sparkles burst from the piece and fade, once. */
@Composable
private fun SparkleBurst(at: Offset, q: Float, palette: Palette) {
    val bits = remember { List(7) { i -> Triple(i / 7.0 * 2 * PI + Random.nextDouble(-0.25, 0.25), Random.nextDouble(0.55, 0.85).toFloat(), Random.nextDouble(0.7, 1.1).toFloat()) } }
    val t = remember { Animatable(0f) }
    LaunchedEffect(Unit) { t.animateTo(1f, tween(800, easing = LinearOutSlowInEasing)) }
    Canvas(Modifier.fillMaxSize().zIndex(26f)) {
        val v = t.value
        drawCircle(palette.great.copy(alpha = 0.6f * (1 - v)), q * (0.25f + 0.5f * v), at, style = Stroke(2.dp.toPx()))
        bits.forEachIndexed { i, (angle, dist, s) ->
            val p = at + Offset(cos(angle).toFloat() * q * dist * v, sin(angle).toFloat() * q * dist * v)
            sparkleStar(p, q * 0.11f * s * (0.2f + 0.8f * v), (if (i % 2 == 0) palette.great else palette.lastMove).copy(alpha = 1 - v))
        }
    }
}

/** A four-pointed sparkle. */
fun DrawScope.sparkleStar(c: Offset, r: Float, color: Color) {
    val path = Path().apply {
        moveTo(c.x, c.y - r)
        quadraticTo(c.x, c.y, c.x + r, c.y)
        quadraticTo(c.x, c.y, c.x, c.y + r)
        quadraticTo(c.x, c.y, c.x - r, c.y)
        quadraticTo(c.x, c.y, c.x, c.y - r)
        close()
    }
    drawPath(path, color)
}

/** A taken piece breaking into glass. */
@Composable
private fun ShardBurst(at: Offset, q: Float, dark: Boolean) {
    val shards = remember {
        List(14) {
            val a = Random.nextDouble(0.0, 2 * PI)
            val d = Random.nextDouble(0.6, 1.6).toFloat()
            listOf(cos(a).toFloat() * d, sin(a).toFloat() * d - 0.3f, Random.nextDouble(-270.0, 270.0).toFloat(), Random.nextDouble(0.0, 0.05).toFloat())
        }
    }
    val t = remember { Animatable(0f) }
    LaunchedEffect(Unit) { t.animateTo(1f, tween(800, easing = CubicBezierEasing(0.15f, 0.7f, 0.4f, 1f))) }
    Canvas(Modifier.fillMaxSize().zIndex(27f)) {
        val colors = if (dark) listOf(Color(0xFF808080), Color(0xFF1A1A1A)) else listOf(Color.White, Color(0xFFCCCCCC))
        val s = q * 0.26f
        for ((dx, dy, rot, delay) in shards) {
            val v = ((t.value - delay) / (1 - delay)).coerceIn(0f, 1f)
            val p = at + Offset(dx * q * v, dy * q * v)
            val scale = 1 - 0.6f * v
            translate(p.x, p.y) {
                rotate(rot * v, pivot = Offset.Zero) {
                    val path = Path().apply {
                        moveTo(0f, -s / 2 * scale)
                        lineTo(s / 2 * scale, (-s / 2 + s * 0.7f) * scale)
                        lineTo((-s / 2 + s * 0.2f) * scale, s / 2 * scale)
                        close()
                    }
                    drawPath(path, Brush.linearGradient(colors, Offset(-s / 2, -s / 2), Offset(s / 2, s / 2)), alpha = 1 - v)
                }
            }
        }
    }
}
