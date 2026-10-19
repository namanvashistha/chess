package com.namanvashistha.calmchess

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ChevronRight
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.blur
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.zIndex
import com.namanvashistha.calmchess.engine.chesscore.Chesscore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * The first time the app opens: a short story, then you play it out.
 *
 * The lore in one sentence: chess is something you look at, most apps made it
 * something you're scored on, and this table is calm, with a friend at your
 * side. It opens calm, the table laying itself. Then, briefly, the noise chess
 * usually comes with drifts over the board and fades. Then you meet the coach.
 * Then it's yours: pick up a piece and hold it while the coach says what it
 * sees on each square. It ends on the promise: see the game.
 *
 * Nothing moves on by itself: every beat waits for Next or a tap.
 */
@Composable
fun WelcomeScreen(enter: () -> Unit, learn: () -> Unit) {
    val p = LocalPalette.current
    val scope = rememberCoroutineScope()
    val board = remember { WelcomeModel(scope) }
    var stage by remember { mutableStateOf(Stage.Calm) }
    var fading by remember { mutableStateOf(false) }
    var leaving by remember { mutableStateOf(false) }
    var height by remember { mutableIntStateOf(2400) }
    var asked by remember { mutableStateOf(0L) }
    val fade by animateFloatAsState(if (fading) 0f else 1f, tween(300), label = "fade")
    // The curtain: it falls away, gathering speed, and home is behind it.
    val fall by animateFloatAsState(if (leaving) 1f else 0f, tween(500, easing = androidx.compose.animation.core.CubicBezierEasing(0.42f, 0f, 1f, 1f)), label = "fall")

    fun seen() = Prefs.p.edit().putBoolean("welcome.seen", true).apply()

    fun finish(then: (() -> Unit)? = null) {
        seen()
        fading = true
        scope.launch { delay(300); enter(); then?.invoke() }
    }

    /** The clock you felt in the loud beat doesn't stop dead: a few last taps, softer each time, then one slow breath. */
    fun windDown() = scope.launch {
        for ((gap, level) in listOf(250L to 0.22f, 300L to 0.12f)) {
            delay(gap)
            if (stage != Stage.Hush) return@launch
            Haptics.rigid(level)
        }
        delay(250)
        if (stage == Stage.Hush) Haptics.breath()
    }

    fun goOn() {
        val now = System.currentTimeMillis()
        if (now - asked < 200) return
        asked = now
        Haptics.tap(0.6f)
        when (stage) {
            Stage.Calm -> { board.showAll(); stage = Stage.Noise }
            Stage.Noise -> { stage = Stage.Hush; windDown() }
            Stage.Hush -> { stage = Stage.Coach; board.greet() }
            Stage.Coach -> { stage = Stage.Pick; board.open = true; board.hint(); scope.launch { board.invite() } }
            Stage.Verdict -> { stage = Stage.Settle; board.takeBack() }
            else -> Unit
        }
    }

    fun begin() {
        Haptics.tap(0.8f)
        Sound.play(Feedback.Great)
        seen()
        leaving = true
        scope.launch { delay(500); enter() }
    }

    LaunchedEffect(Unit) {
        board.onMove = { stage = Stage.Verdict }
        board.setUp()
    }
    BackHandler { finish() }

    Box(
        Modifier.fillMaxSize().zIndex(10f).onSizeChanged { height = it.height + 300 }
            .graphicsLayer { alpha = fade; translationY = fall * height }
            // It covers home completely: touches never reach what's underneath.
            .clickable(remember { MutableInteractionSource() }, null) {}
    ) {
        AppBackground {
            Column(
                Modifier.fillMaxSize().safeDrawingPadding().padding(horizontal = 20.dp).padding(bottom = 16.dp)
                    .widthIn(max = 560.dp).align(Alignment.TopCenter)
            ) {
                Spacer(Modifier.height(44.dp))
                // One line at a time, in a space of its own height so the board never shifts.
                Box(Modifier.fillMaxWidth().height(70.dp).padding(top = 12.dp)) {
                    AnimatedContent(stage.title, transitionSpec = {
                        (fadeIn(tween(500, 200)) + slideInVertically(tween(500, 200)) { 8 }) togetherWith fadeOut(tween(200))
                    }, label = "title") { t ->
                        if (t != null) Text(t, style = Typeface.display(30f).copy(letterSpacing = (-0.5).sp), color = p.ink, maxLines = 1)
                    }
                }
                // The noise drifts over the board, which dims under it, then clears.
                Box(Modifier.padding(top = 24.dp)) {
                    val dim by animateFloatAsState(if (stage == Stage.Noise) 0.25f else 1f, tween(1000), label = "dim")
                    BoardView(board, Modifier.fillMaxWidth().graphicsLayer { alpha = dim })
                    androidx.compose.animation.AnimatedVisibility(stage == Stage.Noise, Modifier.matchParentSize(), enter = fadeIn(tween(900)), exit = fadeOut(tween(900))) { Noise() }
                    if (stage == Stage.Pick && board.inviting) {
                        Shimmer(board.invitePass, Modifier.matchParentSize())
                        TouchRipple(board.invitePawn, board.touchPass, Modifier.matchParentSize())
                    }
                }
                // Under the board: whoever is speaking, then the way on.
                Box(Modifier.padding(top = 20.dp)) {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        AnimatedVisibility(stage == Stage.Coach || stage == Stage.Pick || stage == Stage.Verdict, enter = fadeIn(), exit = fadeOut(tween(150))) {
                            CoachView(board.coach, Opponent.Easy, compact = true)
                        }
                        AnimatedVisibility(stage != Stage.Pick && stage != Stage.Settle, enter = fadeIn(), exit = fadeOut()) {
                            // Bottom right, under Skip: the two corners are the two ways on.
                            Row(Modifier.fillMaxWidth().heightIn(min = 44.dp), horizontalArrangement = Arrangement.End) { NextButton(stage) { goOn() } }
                        }
                    }
                    androidx.compose.animation.AnimatedVisibility(stage == Stage.Settle, enter = fadeIn(tween(500, 250)) + slideInVertically(tween(500, 250)) { 36 }) {
                        Column(verticalArrangement = Arrangement.spacedBy(18.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                            // L("See the game.") above, L("Show me") here: the line and the answer to it.
                            PrimaryButton(L("Show me")) { begin() }
                            Box(Modifier.heightIn(min = 44.dp).clickable { Haptics.tap(); finish(learn) }, contentAlignment = Alignment.Center) {
                                Text(L("New to chess? Learn the moves"), style = Typeface.body(15f, FontWeight.SemiBold), color = p.mist)
                            }
                        }
                    }
                }
            }
            // Skip stays above it, so the way out is never swallowed.
            AnimatedVisibility(stage != Stage.Settle, Modifier.align(Alignment.TopEnd).safeDrawingPadding().padding(end = 12.dp), exit = fadeOut()) {
                Box(Modifier.heightIn(min = 44.dp).clickable { Haptics.tap(); finish() }.padding(horizontal = 8.dp), contentAlignment = Alignment.Center) {
                    Text(L("Skip"), style = Typeface.body(15f, FontWeight.SemiBold), color = p.mist)
                }
            }
        }
    }
}

private enum class Stage {
    Calm, Noise, Hush, Coach, Pick, Verdict, Settle;

    /** The narrator's line, until the coach takes over. */
    val title: String?
        get() = when (this) {
            Calm -> L("Take a seat.")
            Noise -> L("Most chess is loud.")
            Hush -> L("This table is calm.")
            Coach -> L("A friend at your side.")
            Pick, Verdict -> null
            Settle -> L("See the game.")
        }
}

/** Next: a soft capsule that breathes slowly, so it reads as the thing to press without calling out. */
@Composable
private fun NextButton(stage: Stage, onClick: () -> Unit) {
    val p = LocalPalette.current
    val source = remember { MutableInteractionSource() }
    val pressed by source.collectIsPressedAsState()
    val t by rememberInfiniteTransition(label = "breathe").animateFloat(0f, 1f, infiniteRepeatable(tween(1600, easing = FastOutSlowInEasing), RepeatMode.Reverse), label = "t")
    val scale by animateFloatAsState(if (pressed) 0.95f else 1f, swiftSpring(0.25f, 0.6f), label = "press")
    // The chevron nudges right once each beat.
    val wiggle = remember { Animatable(0f) }
    LaunchedEffect(stage) { repeat(2) { wiggle.animateTo(3f, tween(120)); wiggle.animateTo(0f, tween(120)) } }
    val shape = RoundedCornerShape(50)
    Row(
        Modifier.graphicsLayer { scaleX = scale; scaleY = scale }
            .shadow((if (pressed) 3f else 5f + t * 9f).dp, shape, ambientColor = Color.Black.copy(alpha = 0.1f + t * 0.14f), spotColor = Color.Black.copy(alpha = 0.1f + t * 0.14f))
            .background(p.surface, shape)
            .background(p.ink.copy(alpha = if (pressed) 0.14f else 0.03f + t * 0.07f), shape)
            .border(1.dp, p.hairline, shape)
            .clickable(source, null, onClick = onClick)
            .padding(horizontal = 18.dp, vertical = 11.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Text(L("Next"), style = Typeface.body(15f, FontWeight.SemiBold), color = p.ink)
        Icon(Icons.Rounded.ChevronRight, null, tint = p.ink, modifier = Modifier.size(16.dp).offset { IntOffset(wiggle.value.dp.roundToPx(), 0) })
    }
}

/** The noise the app leaves out: a clock running down, a rating slipping, a harsh verdict, half out of focus. */
@Composable
private fun Noise() {
    val p = LocalPalette.current
    var seconds by remember { mutableIntStateOf(12) }
    LaunchedEffect(Unit) {
        // Round and round: it runs out, and another one starts. The clock you can feel.
        while (isActive) {
            delay(450)
            seconds = if (seconds > 0) seconds - 1 else 12
            Haptics.rigid(0.35f)
        }
    }
    // It never settles while you sit on it: a frozen clock is just a picture.
    val drift by rememberInfiniteTransition(label = "drift").animateFloat(6f, -10f, infiniteRepeatable(tween(3000, easing = FastOutSlowInEasing), RepeatMode.Reverse), label = "y")
    val pulse by rememberInfiniteTransition(label = "pulse").animateFloat(0.45f, 1f, infiniteRepeatable(tween(1100), RepeatMode.Reverse), label = "o")
    BoxWithConstraints(Modifier.aspectRatio(1f).graphicsLayer { translationY = drift.dp.toPx() }) {
        val w = maxWidth
        @Composable
        fun fragment(text: String, size: Float, color: Color, blur: Float, x: Float, y: Float, mono: Boolean = false, alpha: Float = 1f) {
            Text(text, style = TextStyle(fontSize = size.sp, fontWeight = FontWeight.Bold,
                fontFamily = if (mono) FontFamily.Monospace else Typeface.rounded, color = color.copy(alpha = 0.75f * alpha)),
                modifier = Modifier.offset(w * x, w * y).blur(blur.dp, androidx.compose.ui.draw.BlurredEdgeTreatment.Unbounded))
        }
        fragment("0:%02d".format(seconds), 46f, p.danger, 0.6f, 0.15f, 0.22f, mono = true)
        fragment("1452  ▾ −12", 24f, p.mist, 1.2f, 0.48f, 0.47f, mono = true)
        fragment("?? " + L("Blunder"), 22f, p.danger, 1.8f, 0.2f, 0.68f, alpha = pulse)
        fragment(L("Rated · 3 min"), 16f, p.mist, 2.4f, 0.6f, 0.17f)
        fragment(L("Time's up"), 18f, p.mist, 2.8f, 0.58f, 0.83f)
    }
}

/** The invitation to touch: a soft band of light sweeping across your two ranks. One pass per `pass`. */
@Composable
private fun Shimmer(pass: Int, modifier: Modifier) {
    val x = remember { Animatable(-0.3f) }
    LaunchedEffect(pass) { x.snapTo(-0.3f); x.animateTo(1.3f, tween(1300, easing = FastOutSlowInEasing)) }
    Canvas(modifier) {
        val w = size.width
        val q = w / 8
        val c = Offset(w * x.value, q * 7)
        clipRect(top = q * 6) {
            rotate(12f, c) {
                drawRect(Brush.horizontalGradient(listOf(Color.Transparent, Color.White.copy(alpha = 0.8f), Color.Transparent), c.x - q * 1.2f, c.x + q * 1.2f),
                    Offset(c.x - q * 1.2f, c.y - q), androidx.compose.ui.geometry.Size(q * 2.4f, q * 2), blendMode = BlendMode.Softlight)
            }
        }
    }
}

/** A touch, shown rather than drawn as a hand: a ring of light that presses in on a square and spreads out. */
@Composable
private fun TouchRipple(square: Square, pass: Int, modifier: Modifier) {
    val p = LocalPalette.current
    val spread = remember { Animatable(0f) }
    LaunchedEffect(pass) { if (pass > 0) { spread.snapTo(0f); spread.animateTo(1f, tween(900, easing = androidx.compose.animation.core.LinearOutSlowInEasing)) } }
    if (pass == 0) return
    Canvas(modifier) {
        val q = size.width / 8
        val c = center(square, size.width)
        val v = spread.value
        val a = 1 - v
        drawCircle(Color.White.copy(alpha = 0.35f * a), q * 0.25f * (0.6f + 0.4f * v), c)
        drawCircle(p.accent.copy(alpha = a), q * 0.45f * (0.7f + 0.9f * v), c, style = Stroke(2.5.dp.toPx()))
    }
}

/**
 * The welcome board: a real game at the start, where you play one move for
 * White, which then glides quietly back. Its coach reads every first move as
 * soon as the board is set, so while you hold a piece it can say, live, what
 * it thinks of the square under your finger.
 */
class WelcomeModel(private val scope: CoroutineScope) : BoardSource {
    override var shownPieces by mutableStateOf(listOf<PieceItem>()); private set
    override var selected by mutableStateOf<Square?>(null); private set
    override var rays by mutableStateOf<Rays?>(null); private set
    override val bursts = emptyList<Burst>()
    override val toppled: Int? = null
    override var movingPiece by mutableStateOf<Int?>(null); private set
    override var turn by mutableStateOf("w"); private set
    override val reviewing: Int? = null
    override var shownLastMove by mutableStateOf<BoardMove?>(null); private set
    override val sparkle: Pair<Int, Square>? = null
    override val nudge: Square? = null
    override val loose = emptySet<Square>()
    override val checkSquare: Square? = null
    override val shownMark: Pair<TrailEntry, Square>? = null
    override val placesPieces = true
    var coach by mutableStateOf<CoachCard?>(null); private set
    var inviting by mutableStateOf(false); private set
    var invitePass by mutableIntStateOf(0); private set
    var touchPass by mutableIntStateOf(0); private set
    /** The board is handed over: pieces can be picked up. */
    var open = false
    var onMove: (() -> Unit)? = null
    /** The pawn that shows how a touch works: e2. */
    val invitePawn: Square = 52

    private val game = Chesscore.newGame()
    private val legal = game.legalMoves().split(" ").filter { it.isNotEmpty() }
    private var played: Triple<Int, Square, Square>? = null
    private var reads = mapOf<String, Pair<CoachCard.Kind, String>>()
    private var hovered: Square? = null
    private var serial = 0
    private var hurried = false
    private var touched = false

    init {
        // The coach's read of every first move, worked out while you look.
        val fen = game.fen()
        scope.launch {
            reads = withContext(Dispatchers.Default) {
                legal.mapNotNull { m -> Chesscore.newGameFromFEN(fen)?.assess(m)?.let { m to (CoachCard.Kind.of(it.kind) to EngineText.localize(it.text)) } }.toMap()
            }
        }
    }

    companion object {
        /** The order a board is set by hand: every rank from the middle out, White then Black. */
        private val ORDER: List<Square> = listOf(4, 3, 5, 2, 6, 1, 7, 0).let { out ->
            out.map { 56 + it } + out.map { 48 + it } + out + out.map { 8 + it }
        }
    }

    /** The table laying itself, piece by piece: the screen arriving, not the story running. */
    suspend fun setUp() {
        val board = game.board().toList()
        if (shownPieces.isEmpty()) delay(700)
        for ((k, sq) in ORDER.withIndex()) {
            if (hurried) return
            if (board[sq] == '.' || shownPieces.any { it.id == sq }) continue
            shownPieces = shownPieces + PieceItem(sq, board[sq], sq)
            // A light tap as each piece is set down: firmer for the back rank.
            Haptics.soft(if (k % 16 < 8) 0.5f else 0.3f)
            delay(if (k % 16 < 8) 70 else 38)
        }
    }

    /** You moved on while it was still laying: the rest arrives at once. */
    fun showAll() {
        hurried = true
        val board = game.board().toList()
        shownPieces = shownPieces + ORDER.filter { sq -> board[sq] != '.' && shownPieces.none { it.id == sq } }.map { PieceItem(it, board[it], it) }
    }

    /** Its first word, before anything is asked of you: it is here to think along, and keeps no score. */
    fun greet() { if (coach == null) say(CoachCard.Kind.Coach, L("Your second"), L("I'll tell you what I see.")) }

    fun hint() { if (!touched) say(CoachCard.Kind.Coach, L("Pick up any piece"), L("I'll follow along.")) }

    /** Until the first touch: light sweeps your pieces, a touch ripples on the king's pawn, it lifts and its lines draw out. */
    suspend fun invite() {
        inviting = true
        delay(600)
        while (!touched) {
            invitePass++
            delay(900); if (touched) break
            touchPass++
            delay(150); if (touched) break
            selected = invitePawn
            rays = Rays(++serial, Rays.Kind.Options, targets(invitePawn))
            delay(1300)
            if (!touched) { selected = null; rays = null }
            delay(2200)
        }
        inviting = false
    }

    private fun targets(from: Square): List<BoardMove> {
        val name = Squares.name(from)
        return legal.filter { it.startsWith(name) }.mapNotNull { m -> Squares.parse(m.substring(2, 4))?.let { BoardMove(from, it) } }.distinctBy { it.to }
    }

    private val canPlay get() = open && played == null

    private fun say(kind: CoachCard.Kind, title: String, text: String, keep: Boolean = true) {
        // One id while you're thinking, so the words change in place rather than popping a new card.
        if (!keep) serial++
        coach = CoachCard(serial, kind, kind.verdict?.second ?: "✦", title, text)
    }

    private fun select(square: Square) {
        touched = true
        if (selected != square) Sound.play(Feedback.Select)
        selected = square
        hovered = null
        val moves = targets(square)
        val code = shownPieces.firstOrNull { it.square == square }?.code ?: 'P'
        rays = Rays(++serial, Rays.Kind.Options, moves, knight = if (code == 'N') moves.toSet() else emptySet())
        say(CoachCard.Kind.Coach, L("Tap a square, or drag"), "", keep = false)
    }

    /** The coach, live, on the square under the carried piece. */
    override fun hover(square: Square?) {
        val from = selected ?: return
        if (!canPlay || square == hovered) return
        hovered = square
        val uci = square?.let { sq -> legal.firstOrNull { it.startsWith(Squares.name(from) + Squares.name(sq)) } }
        if (uci == null) { if (square == null || square == from) select(from); return }
        val read = reads[uci] ?: run { say(CoachCard.Kind.Coach, L("Let me look…"), ""); return }
        val title = when (read.first) {
            CoachCard.Kind.Great -> L("Ooh, a strong square")
            CoachCard.Kind.Good -> L("A good square")
            CoachCard.Kind.Inaccuracy -> L("Hmm, maybe")
            else -> L("Careful there")
        }
        say(if (read.first == CoachCard.Kind.Great || read.first == CoachCard.Kind.Good) read.first else CoachCard.Kind.Coach,
            title, read.second.ifEmpty { L("Nothing wrong with it. Put it down if you like it.") })
    }

    private fun play(from: Square, to: Square) {
        val i = shownPieces.indexOfFirst { it.square == from }
        if (i < 0) return
        val uci = Squares.name(from) + Squares.name(to)
        if (uci !in legal) return
        val read = reads[uci]
        game.play(uci)
        val piece = shownPieces[i]
        played = Triple(piece.id, from, to)
        selected = null
        hovered = null
        rays = null
        movingPiece = piece.id
        shownPieces = shownPieces.toMutableList().also { it[i] = piece.copy(square = to) }
        shownLastMove = BoardMove(from, to)
        turn = "b"
        Sound.play(Feedback.Move)
        val kind = read?.first ?: CoachCard.Kind.Good
        val words = kind.verdict ?: (L("Good move") to "✓")
        say(kind, words.first, read?.second?.ifEmpty { null } ?: L("A calm, steady start."), keep = false)
        Sound.play(if (kind == CoachCard.Kind.Great) Feedback.Great else Feedback.Good)
        onMove?.invoke()
    }

    /** The move glides back, leaving the board as home keeps it. */
    fun takeBack() {
        val (id, from, _) = played ?: return
        Sound.play(Feedback.Move)
        game.undo()
        movingPiece = id
        shownPieces = shownPieces.map { if (it.id == id) it.copy(square = from) else it }
        shownLastMove = null
        turn = "w"
    }

    override fun tap(square: Square) {
        if (!canPlay) return
        // Before the first touch, a lifted pawn is only the board's demo: the first tap picks up what you tapped.
        if (!touched) { selected = null; rays = null }
        touched = true
        val sel = selected
        if (sel != null && rays?.moves?.any { it.to == square } == true) play(sel, square)
        else if (selected != square && shownPieces.any { it.square == square && it.code.isUpperCase() } && targets(square).isNotEmpty()) select(square)
        else {
            if (selected != null) { Sound.play(Feedback.PutDown); Haptics.putDown() }
            selected = null
            rays = null
        }
    }

    override fun pickUp(square: Square): Boolean {
        if (!touched) { selected = null; rays = null }
        touched = true
        if (!canPlay || shownPieces.none { it.square == square && it.code.isUpperCase() } || targets(square).isEmpty()) return false
        if (selected != square) select(square)
        return true
    }

    override fun drop(on: Square?) {
        val sel = selected ?: return
        if (!canPlay) return
        if (on != null && rays?.moves?.any { it.to == on } == true) play(sel, on) else select(sel)
    }

    override fun inspect(square: Square) = false
    override fun endInspect() {}
}
