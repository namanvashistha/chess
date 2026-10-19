package com.namanvashistha.calmchess

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Close
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.zIndex
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * L("Learn the moves"): ten tiny moments for someone who has never played. Each
 * is one idea, one short line and one thing to do, on the game's own board.
 * Solve it and the next one comes on its own: no counters, no way to fail.
 */
@Composable
fun LearnScreen(close: () -> Unit, play: () -> Unit) {
    val p = LocalPalette.current
    val scope = rememberCoroutineScope()
    val lesson = remember { LessonModel(scope) }
    // Where you are, kept so you can leave and come back.
    var step by remember { mutableIntStateOf(Prefs.p.getInt("learn.step", 0)) }
    var solved by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    val moment = Moment.all.getOrNull(step)
    val progress by animateFloatAsState((step + if (solved) 1 else 0).toFloat() / Moment.all.size, swiftSpring(0.5f, 0.8f), label = "progress")
    var celebrate by remember { mutableIntStateOf(0) }

    fun setStep(s: Int) { step = s; Prefs.p.edit().putInt("learn.step", s).apply() }

    fun load() {
        val m = Moment.all.getOrNull(step) ?: return
        solved = false
        busy = false
        lesson.load(m)
        val g = m.goal
        if (g is Moment.Goal.Escape) lesson.showOptions(BoardMove(g.from, m.start, true)) else lesson.showOptions()
    }

    fun finish(at: Square?, last: Boolean = false) {
        solved = true
        Haptics.play(Feedback.Great)
        if (at != null) lesson.celebrate(at)
        if (last) celebrate++
        Sound.play(if (last) Feedback.Win else Feedback.Great)
        val shown = step
        scope.launch {
            delay(if (last) 2200 else 1100)
            if (shown != step) return@launch
            setStep(step + 1)
            load()
        }
    }

    fun carryOn() { busy = false; lesson.showOptions() }

    /** Not this one: a soft bump, and the piece goes back where it was. */
    suspend fun miss(back: Square) {
        Haptics.play(Feedback.Bad)
        delay(250)
        lesson.slide(back)
        delay(300)
        busy = false
        val m = moment ?: return
        val g = m.goal
        if (g is Moment.Goal.Escape) lesson.showOptions(BoardMove(g.from, back, true)) else lesson.showOptions()
    }

    fun move(from: Square, to: Square) {
        val m = Moment.all.getOrNull(step) ?: return
        if (busy) return
        busy = true
        val shown = step
        // Exactly as in a game: the piece glides, and what it takes goes at once.
        if (lesson.slide(to)) { Haptics.play(Feedback.Capture); Sound.play(Feedback.Capture) } else Sound.play(Feedback.Move)
        scope.launch {
            delay(300)
            if (shown != step) return@launch
            when (val g = m.goal) {
                Moment.Goal.TakeAll -> if (lesson.others.values.none { it.isLowerCase() }) finish(to) else carryOn()
                Moment.Goal.Promote -> if (to < 8) { lesson.promote(); finish(to) } else carryOn()
                is Moment.Goal.Check -> if (lesson.attacks(g.king)) { lesson.showCheck(to, g.king); finish(null) } else miss(from)
                is Moment.Goal.Escape -> if (to in g.safe) finish(to) else miss(from)
                is Moment.Goal.Mate -> if (to in g.squares) {
                    lesson.showCheck(to, g.king)
                    delay(300)
                    lesson.topple(g.king)
                    finish(null, last = true)
                } else miss(from)
            }
        }
    }

    LaunchedEffect(Unit) { lesson.onMove = { f, t -> move(f, t) }; load() }
    BackHandler { if (moment == null) setStep(0); close() }

    AppBackground {
        Column(
            Modifier.fillMaxSize().safeDrawingPadding().padding(horizontal = 20.dp).padding(bottom = 16.dp)
                .widthIn(max = 560.dp).align(Alignment.TopCenter)
        ) {
            // Close, and a thin line that fills as you go.
            Row(Modifier.padding(top = 8.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                GlassIconButton(Icons.Rounded.Close, L("Close"), circle = true) { Haptics.tap(); if (moment == null) setStep(0); close() }
                BoxWithConstraints(Modifier.weight(1f).height(6.dp)) {
                    Box(Modifier.fillMaxSize().background(p.mist.copy(alpha = 0.2f), RoundedCornerShape(50)))
                    Box(Modifier.fillMaxHeight().width(maxOf(6.dp, maxWidth * progress)).background(p.accent, RoundedCornerShape(50)))
                }
                Spacer(Modifier.size(44.dp))
            }
            if (moment != null) {
                AnimatedContent(step, transitionSpec = { fadeIn(tween(400)) togetherWith fadeOut(tween(200)) }, label = "moment") { s ->
                    val m = Moment.all.getOrNull(s) ?: return@AnimatedContent
                    Column(Modifier.fillMaxWidth().padding(top = 28.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(m.title, style = Typeface.display(30f).copy(letterSpacing = (-0.5).sp), color = p.ink)
                        Text(m.text, style = Typeface.body(17f), color = p.mist)
                    }
                }
                Box(Modifier.padding(top = 24.dp)) {
                    BoardView(lesson, Modifier.fillMaxWidth())
                    if (celebrate > 0) androidx.compose.runtime.key(celebrate) { Celebration(Modifier.align(Alignment.TopCenter).offset(y = 80.dp)) }
                }
            } else {
                Finale(
                    play = { Haptics.tap(); Prefs.p.edit().putBoolean("learn.done", true).apply(); setStep(0); close(); play() },
                    home = { Prefs.p.edit().putBoolean("learn.done", true).apply(); setStep(0); close() },
                )
            }
        }
    }
}

/** The payoff: an empty board fills with every piece you just learned, until a whole game is set up. */
@Composable
private fun androidx.compose.foundation.layout.ColumnScope.Finale(play: () -> Unit, home: () -> Unit) {
    val p = LocalPalette.current
    Column(Modifier.fillMaxWidth().padding(top = 28.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(L("You know every piece"), style = Typeface.display(30f).copy(letterSpacing = (-0.5).sp), color = p.ink)
        Text(L("That's the whole game. Time to play."), style = Typeface.body(17f), color = p.mist)
    }
    FinaleBoard(Modifier.padding(top = 24.dp)) { Haptics.play(Feedback.Win); Sound.play(Feedback.Win) }
    Spacer(Modifier.weight(1f).heightIn(min = 16.dp))
    Column(verticalArrangement = Arrangement.spacedBy(20.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        PrimaryButton(L("Play the Rookie"), onClick = play)
        Box(Modifier.heightIn(min = 44.dp).clickable(onClick = home), contentAlignment = Alignment.Center) {
            Text(L("Back to home"), style = Typeface.body(15f, FontWeight.SemiBold), color = p.mist)
        }
    }
}

/** Empty stone, then the pieces arrive one after another in the order they were learned, and settle. */
@Composable
private fun FinaleBoard(modifier: Modifier, landed: () -> Unit) {
    val arrivals = remember {
        val order = "RBQNKP"
        val back = "RNBQKBNR"
        val white = ((0..7).map { back[it] to 56 + it } + (0..7).map { 'P' to 48 + it }).sortedBy { order.indexOf(it.first) }
        white + white.map { (c, sq) -> c.lowercaseChar() to (sq % 8 + (7 - sq / 8) * 8) }
    }
    val t = remember { Animatable(0f) }
    val total = 300 + arrivals.size * 50 + 450
    LaunchedEffect(Unit) {
        launch { t.animateTo(total.toFloat(), tween(total, easing = androidx.compose.animation.core.LinearEasing)) }
        delay((300 + arrivals.size * 50 + 300).toLong())
        landed()
    }
    BoxWithConstraints(modifier.fillMaxWidth().aspectRatio(1f)
        .shadow(18.dp, RoundedCornerShape(6.dp), ambientColor = Color.Black.copy(alpha = 0.28f), spotColor = Color.Black.copy(alpha = 0.28f))
        .clip(RoundedCornerShape(6.dp))) {
        val q = maxWidth / 8
        StaticBoard("8/8/8/8/8/8/8/8", Modifier.fillMaxSize())
        arrivals.forEachIndexed { i, (code, sq) ->
            // Each one drops in from a little above its square, with a spring's settle.
            val local = ((t.value - 300 - i * 50) / 450f).coerceIn(0f, 1f)
            val s = springish(local)
            GlassPiece(code, Modifier.size(q).offset(q * (sq % 8), q * (sq / 8) - q * 1.2f * (1 - s))
                .graphicsLayer { val sc = 0.4f + 0.6f * s; scaleX = sc; scaleY = sc; alpha = local.coerceIn(0f, 1f) }
                .zIndex((sq / 8).toFloat()))
        }
    }
}

/** A spring-like settle: overshoots a little, then comes to rest. */
private fun springish(x: Float): Float {
    if (x <= 0f) return 0f
    if (x >= 1f) return 1f
    return (1 - Math.exp(-6.0 * x) * Math.cos(9.0 * x)).toFloat()
}

/**
 * A Learn moment as the game board sees it: the one piece you move, what else
 * stands on the board, and the lines from your piece to where it can go.
 */
class LessonModel(private val scope: CoroutineScope) : BoardSource {
    override var shownPieces by mutableStateOf(listOf<PieceItem>()); private set
    override var selected by mutableStateOf<Square?>(null); private set
    override var rays by mutableStateOf<Rays?>(null); private set
    override var bursts by mutableStateOf(listOf<Burst>()); private set
    override var toppled by mutableStateOf<Int?>(null); private set
    override var movingPiece by mutableStateOf<Int?>(null); private set
    /** You always move in a lesson, so the light rests on your side. */
    override val turn = "w"
    override val reviewing: Int? = null
    override val shownLastMove: BoardMove? = null
    override var sparkle by mutableStateOf<Pair<Int, Square>?>(null); private set
    override val nudge: Square? = null
    override val loose = emptySet<Square>()
    override var checkSquare by mutableStateOf<Square?>(null); private set
    override val shownMark: Pair<TrailEntry, Square>? = null
    var onMove: ((Square, Square) -> Unit)? = null
    private var serial = 0

    private val mover get() = shownPieces.firstOrNull { it.id == MOVER }

    /** Everything but your piece: black pieces to take, white ones in the way. */
    val others: Map<Square, Char> get() = shownPieces.filter { it.id != MOVER }.associate { it.square to it.code }

    private val targets: List<Square> get() = mover?.let { Moment.moves(it.code, it.square, others) } ?: emptyList()

    fun load(m: Moment) {
        shownPieces = listOf(PieceItem(MOVER, m.piece, m.start)) +
            m.others.entries.sortedBy { it.key }.mapIndexed { k, e -> PieceItem(k + 1, e.value, e.key) }
        selected = null; rays = null; bursts = emptyList(); toppled = null; checkSquare = null; movingPiece = null; sparkle = null
    }

    /** The lines from your piece to where it can go, and a line of danger to your king if it is in check. */
    fun showOptions(danger: BoardMove? = null) {
        val m = mover ?: return
        val moves = targets.map { BoardMove(m.square, it, others[it]?.isLowerCase() == true) } + listOfNotNull(danger)
        rays = Rays(++serial, Rays.Kind.Options, moves, knight = if (m.code == 'N') moves.filter { it != danger }.toSet() else emptySet(),
            danger = setOfNotNull(danger))
        checkSquare = danger?.to
    }

    fun showCheck(from: Square, king: Square) {
        val line = BoardMove(from, king, true)
        rays = Rays(++serial, Rays.Kind.Chosen, listOf(line), danger = setOf(line))
        checkSquare = king
    }

    /** Glides your piece to a square, taking a black piece there at once, in glass. True if it took one. */
    fun slide(square: Square): Boolean {
        rays = null; checkSquare = null; selected = null
        var took = false
        val victim = shownPieces.firstOrNull { it.id != MOVER && it.square == square && it.code.isLowerCase() }
        if (victim != null) {
            shownPieces = shownPieces - victim
            val burst = Burst(++serial, square, true)
            bursts = bursts + burst
            scope.launch { delay(900); bursts = bursts - burst }
            took = true
        }
        movingPiece = MOVER
        shownPieces = shownPieces.map { if (it.id == MOVER) it.copy(square = square) else it }
        return took
    }

    fun celebrate(square: Square) { sparkle = ++serial to square }
    fun promote() { shownPieces = shownPieces.map { if (it.id == MOVER) it.copy(code = 'Q') else it } }
    fun attacks(square: Square) = square in targets
    fun topple(square: Square) { toppled = shownPieces.firstOrNull { it.square == square && it.id != MOVER }?.id }

    override fun tap(square: Square) {
        val m = mover ?: return
        if (rays?.kind == Rays.Kind.Options && square in targets) onMove?.invoke(m.square, square)
    }

    override fun pickUp(square: Square): Boolean {
        if (rays?.kind != Rays.Kind.Options || square != mover?.square) return false
        selected = square
        return true
    }

    override fun drop(on: Square?) {
        selected = null
        val m = mover ?: return
        if (on != null && rays?.kind == Rays.Kind.Options && on in targets) onMove?.invoke(m.square, on)
    }

    override fun inspect(square: Square) = false
    override fun endInspect() {}

    companion object { private const val MOVER = 0 }
}

/** One moment: a piece to move, what else is on the board, and what solves it. */
data class Moment(val piece: Char, val start: Square, val others: Map<Square, Char>, val goal: Goal, val title: String, val text: String) {
    sealed interface Goal {
        data object TakeAll : Goal
        data object Promote : Goal
        data class Check(val king: Square) : Goal
        data class Escape(val safe: Set<Square>, val from: Square) : Goal
        data class Mate(val squares: Set<Square>, val king: Square) : Goal
    }

    companion object {
        fun sq(name: String) = Squares.parse(name)!!
        private fun board(vararg list: Pair<String, Char>) = list.associate { sq(it.first) to it.second }

        val all get() = listOf(
            Moment('R', sq("a1"), board("a5" to 'p', "e5" to 'p', "e8" to 'p'), Goal.TakeAll, L("The rook"), L("Take the pawns.")),
            Moment('B', sq("c1"), board("e3" to 'p', "g5" to 'p', "d8" to 'p'), Goal.TakeAll, L("The bishop"), L("Take the pawns.")),
            Moment('Q', sq("d1"), board("d5" to 'p', "h5" to 'p', "e8" to 'p'), Goal.TakeAll, L("The queen"), L("Take the pawns.")),
            Moment('N', sq("g1"), board("f2" to 'P', "g2" to 'P', "h2" to 'P', "f3" to 'p', "e5" to 'p', "d7" to 'p'), Goal.TakeAll, L("The knight"), L("Take the pawns.")),
            Moment('K', sq("e1"), board("e2" to 'p', "f3" to 'p', "f4" to 'p'), Goal.TakeAll, L("The king"), L("Take the pawns.")),
            Moment('P', sq("e2"), board("d5" to 'p'), Goal.TakeAll, L("The pawn"), L("Take the pawn.")),
            Moment('P', sq("e7"), emptyMap(), Goal.Promote, L("Promotion"), L("Reach the far side.")),
            Moment('R', sq("a1"), board("e8" to 'k'), Goal.Check(sq("e8")), L("Check"), L("Attack the king.")),
            Moment('K', sq("e1"), board("e8" to 'r'), Goal.Escape(setOf(sq("d1"), sq("d2"), sq("f1"), sq("f2")), sq("e8")), L("Escape"), L("Step out of check.")),
            Moment('R', sq("a1"), board("g8" to 'k', "f7" to 'p', "g7" to 'p', "h7" to 'p'), Goal.Mate(setOf(sq("a8")), sq("g8")), L("Checkmate"), L("Trap the king.")),
        )

        /** Where a white piece can go: slides stop at the first piece, knights jump, pawns step and take diagonally. */
        fun moves(code: Char, s: Square, board: Map<Square, Char>): List<Square> {
            val r = s / 8
            val c = s % 8
            val out = mutableListOf<Square>()
            fun at(rr: Int, cc: Int) = if (rr in 0..7 && cc in 0..7) rr * 8 + cc else null
            fun slide(dirs: List<Pair<Int, Int>>) {
                for ((dr, dc) in dirs) {
                    var rr = r + dr; var cc = c + dc
                    while (true) {
                        val t = at(rr, cc) ?: break
                        val p = board[t]
                        if (p != null) { if (p.isLowerCase()) out += t; break }
                        out += t
                        rr += dr; cc += dc
                    }
                }
            }
            fun step(offs: List<Pair<Int, Int>>) {
                for ((dr, dc) in offs) at(r + dr, c + dc)?.let { t -> if (board[t]?.isUpperCase() != true) out += t }
            }
            val orth = listOf(1 to 0, -1 to 0, 0 to 1, 0 to -1)
            val diag = listOf(1 to 1, 1 to -1, -1 to 1, -1 to -1)
            when (code) {
                'R' -> slide(orth)
                'B' -> slide(diag)
                'Q' -> slide(orth + diag)
                'K' -> step(orth + diag)
                'N' -> step(listOf(2 to 1, 2 to -1, -2 to 1, -2 to -1, 1 to 2, 1 to -2, -1 to 2, -1 to -2))
                'P' -> {
                    at(r - 1, c)?.let { t -> if (board[t] == null) { out += t; if (r == 6) at(r - 2, c)?.let { t2 -> if (board[t2] == null) out += t2 } } }
                    for (dc in listOf(-1, 1)) at(r - 1, c + dc)?.let { t -> if (board[t]?.isLowerCase() == true) out += t }
                }
            }
            return out
        }
    }
}
