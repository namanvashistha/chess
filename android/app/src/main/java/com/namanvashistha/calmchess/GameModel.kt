package com.namanvashistha.calmchess

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.namanvashistha.calmchess.engine.chesscore.Chesscore
import com.namanvashistha.calmchess.engine.chesscore.Game
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlin.math.abs
import kotlin.math.exp

/** A one-off event for haptics and sound. */
enum class Feedback { Select, PutDown, Move, Capture, Check, Great, Good, Bad, Win, Lose }

/**
 * The game screen's state, ported from the iPhone app's GameModel. The Go
 * engine owns the rules and the coaching; this keeps what the screen needs on
 * top and runs the engine off the main thread.
 */
class GameModel(private val scope: CoroutineScope) {
    var mode by mutableStateOf(Mode.Engine); private set
    /** Always White against the engine. */
    val myColor = "w"
    var opponent by mutableStateOf(Opponent.Easy); private set

    var board by mutableStateOf(List(64) { '.' }); private set
    var pieces by mutableStateOf(listOf<PieceItem>()); private set
    var turn by mutableStateOf("w"); private set
    var status by mutableStateOf(""); private set
    var inCheck by mutableStateOf(false); private set
    var lastMove by mutableStateOf<BoardMove?>(null); private set
    var loose by mutableStateOf(setOf<Square>()); private set
    var selected by mutableStateOf<Square?>(null); private set
    var rays by mutableStateOf<Rays?>(null); private set
    var coach by mutableStateOf<CoachCard?>(null); private set
    var bursts by mutableStateOf(listOf<Burst>()); private set
    var toppled by mutableStateOf<Int?>(null); private set
    var finish by mutableStateOf<Finish?>(null); private set
    var movingPiece by mutableStateOf<Int?>(null); private set
    var moveSerial by mutableStateOf(0); private set
    /** White's share of the position, 0 to 1. */
    var whiteShare by mutableStateOf(0.5); private set
    var busy by mutableStateOf(false); private set
    var thinking by mutableStateOf(false); private set
    var trail by mutableStateOf(listOf<TrailEntry>()); private set
    var hintsLeft by mutableStateOf(3); private set
    /** The game so far in UCI. */
    var moves by mutableStateOf(listOf<String>()); private set
    /** The trail entry the coach card is showing, if the player stepped back to one. */
    var reviewing by mutableStateOf<Int?>(null); private set
    var reviewBoard by mutableStateOf<List<Char>?>(null); private set
    var reviewMove by mutableStateOf<BoardMove?>(null); private set
    /** The verdict on the last move, pinned to the piece that made it. */
    var mark by mutableStateOf<Pair<TrailEntry, Square>?>(null); private set
    /** A great move's sparkle: an id and where it landed. */
    var sparkle by mutableStateOf<Pair<Int, Square>?>(null); private set
    /** The piece a hint is pointing at, before it shows the whole move. */
    var nudge by mutableStateOf<Square?>(null); private set
    var hintRevealed by mutableStateOf(false); private set
    /** The engine is looking for a hint; the Hint button stays as it is meanwhile. */
    var findingHint by mutableStateOf(false); private set
    var feedback by mutableStateOf<Pair<Int, Feedback>?>(null); private set

    private var evals = mutableListOf<Int?>()
    private var losses = mutableListOf<Int>()
    private var comebacks = 0
    private var game: Game = Chesscore.newGame()
    private var beforeInspect: CoachCard? = null
    private var quietTurns = 0
    private var ideaTurn = 0
    /** When the verdict on the player's last move appeared. */
    private var verdictShownAt: Long? = null
    private var legal = listOf<String>()
    private var ids = arrayOfNulls<Int>(64)
    private var nextId = 0
    private val history = mutableListOf<Pair<Array<Int?>, BoardMove?>>()
    private var serial = 0
    /** Bumps whenever pending work must be dropped: a new game, or an undo while the engine thinks. */
    private var epoch = 0
    private var great = 0
    private var blunders = 0
    /** The hint already paid for in this position, shown again for free. */
    private var hintShown: Triple<Int, String, String>? = null

    init { refresh() }

    val isOver get() = status.isNotEmpty()
    val playerToMove get() = !busy && !isOver && (mode == Mode.PassAndPlay || turn == myColor)
    val hintIsFree get() = hintShown?.first == moves.size

    /** Accuracy on Lichess's curve: each move scored by how much it lowers the chance of winning. */
    val accuracy: Int?
        get() {
            if (mode == Mode.PassAndPlay || losses.size < ACCURACY_AFTER) return null
            fun win(cp: Double) = 50 + 50 * (2 / (1 + exp(-0.00368208 * cp)) - 1)
            val scores = losses.map { loss ->
                val drop = 50 - win(-loss.toDouble())
                (103.1668 * exp(-0.04354 * drop) - 3.1669).coerceIn(0.0, 100.0)
            }
            return Math.round(scores.sum() / scores.size).toInt()
        }

    val targets: List<BoardMove>
        get() {
            val from = selected?.let(Squares::name) ?: return emptyList()
            return legal.filter { it.startsWith(from) }.mapNotNull { BoardMove.of(it, board) }.distinctBy { it.to }
        }

    val checkSquare: Square?
        get() = if (!inCheck) null else board.indexOf(if (turn == "w") 'K' else 'k').takeIf { it >= 0 }

    /** The pieces to draw: the live game's, or the reviewed position's. */
    val shownPieces: List<PieceItem>
        get() = reviewBoard?.let { b -> b.indices.filter { b[it] != '.' }.map { PieceItem(-1 - it, b[it], it) } } ?: pieces

    val shownLastMove get() = if (reviewBoard == null) lastMove else reviewMove

    /** The mark to draw: hidden while reviewing, or once its piece is gone. */
    val shownMark: Pair<TrailEntry, Square>?
        get() {
            val m = mark ?: return null
            if (reviewing != null) return null
            val piece = board[m.second]
            val moverIsWhite = m.first.ply % 2 == 0
            return if (piece != '.' && piece.isUpperCase() == moverIsWhite) m else null
        }

    // Starting

    fun start(mode: Mode, opponent: Opponent) = start(mode, opponent, keepingSave = false)

    private fun start(mode: Mode, opponent: Opponent, keepingSave: Boolean) {
        this.mode = mode
        this.opponent = opponent
        game = Chesscore.newGame()
        epoch++
        ids = arrayOfNulls(64)
        history.clear()
        lastMove = null
        selected = null
        rays = null
        bursts = emptyList()
        toppled = null
        finish = null
        losses = mutableListOf()
        great = 0
        blunders = 0
        busy = false
        thinking = false
        whiteShare = 0.5
        evals = mutableListOf()
        trail = emptyList()
        moves = emptyList()
        hintsLeft = if (mode == Mode.Engine) opponent.startingHints else 3
        comebacks = 0
        reviewing = null; reviewBoard = null; reviewMove = null
        hintShown = null
        nudge = null
        mark = null
        sparkle = null
        hintRevealed = false
        if (!keepingSave) SavedGame.clear()
        refresh()
        moveSerial++
        say(CoachCard.Kind.Coach, "✦", "Your move",
            if (mode == Mode.Engine) "You play White. Tap a piece to see where it can go."
            else "White starts. Pass the phone after each move.")
    }

    /** Picks up the saved game, if there is one. */
    fun resume(): Boolean {
        val saved = SavedGame.load() ?: return false
        start(if (saved.passAndPlay) Mode.PassAndPlay else Mode.Engine, Opponent.entries.getOrElse(saved.opponent) { Opponent.Easy }, keepingSave = true)
        val played = mutableListOf<String>()
        for (uci in saved.moves) { if (!game.play(uci)) break; played += uci }
        moves = played
        trail = saved.trail
        losses = saved.losses.toMutableList()
        great = saved.great
        blunders = saved.blunders
        hintsLeft = saved.hintsLeft
        comebacks = saved.comebacks
        evals = saved.evals.take(moves.size).toMutableList()
        evals.lastOrNull()?.let { whiteShare = share(it) }
        ids = arrayOfNulls(64)
        refresh()
        moves.lastOrNull()?.let { lastMove = BoardMove.of(it, board) }
        moveSerial++
        say(CoachCard.Kind.Coach, "✦", "Welcome back",
            if (mode == Mode.Engine && turn == "b" && !isOver) "Your game is where you left it. They're thinking."
            else "Your game is where you left it. Your move.")
        save()
        fillEvals()
        if (mode == Mode.Engine && turn == "b" && !isOver) engineTurn()
        return true
    }

    /** Measures any evaluations a save is missing, so the finish card's arc tells the whole story. */
    private fun fillEvals() {
        val list = moves
        val have = evals.toList()
        if (have.size >= list.size && have.none { it == null }) return
        val token = epoch
        scope.launch {
            val out = withContext(Dispatchers.Default) {
                val g = Chesscore.newGame()
                buildList {
                    for ((i, m) in list.withIndex()) {
                        if (!g.play(m)) break
                        add(have.getOrNull(i) ?: g.eval(2).toInt())
                    }
                }
            }
            if (token != epoch) return@launch
            val merged = out.toMutableList<Int?>()
            for (i in evals.indices) { if (i < merged.size) { evals[i]?.let { merged[i] = it } } else merged += evals[i] }
            evals = merged.take(moves.size).toMutableList()
            if (moves.size == out.size) out.lastOrNull()?.let { whiteShare = share(it) }
            save()
        }
    }

    private fun save() {
        if (isOver) { SavedGame.clear(); return }
        SavedGame(mode == Mode.PassAndPlay, opponent.ordinal, moves, trail, losses.toList(), great, blunders,
            hintsLeft, evals.toList(), comebacks, game.fen()).store()
    }

    // Touch

    fun pickUp(square: Square): Boolean {
        if (!playerToMove || reviewing != null) return false
        val piece = board[square]
        if (piece == '.' || (if (piece.isUpperCase()) "w" else "b") != turn) return false
        if (selected != square) {
            selected = square
            rays = makeRays(Rays.Kind.Options, targets)
            emit(Feedback.Select)
        }
        return true
    }

    /** Drops the dragged piece: a legal square plays it; anywhere else it stays selected. */
    fun drop(on: Square?) {
        val from = selected ?: return
        if (on != null && on != from && targets.any { it.to == on }) play(from, on)
    }

    /** Long-press: lines from every piece guarding this one and every piece attacking it. */
    fun inspect(square: Square): Boolean {
        if (reviewing != null || busy || board[square] == '.') return false
        val name = Squares.name(square)
        fun lines(list: String, danger: Boolean) =
            list.split(" ").mapNotNull { Squares.parse(it) }.map { BoardMove(it, square, danger) }
        val guards = lines(game.guards(name), false)
        val attackers = lines(game.attackers(name), true)
        selected = null
        rays = makeRays(Rays.Kind.Options, guards + attackers).copy(danger = attackers.toSet())
        beforeInspect = coach
        val piece = board[square]
        val what = pieceName(piece)
        val mine = (if (piece.isUpperCase()) "w" else "b") == myColor
        val title = when (mode) {
            Mode.Engine -> if (mine) "Your $what on $name" else "Your opponent's $what on $name"
            Mode.PassAndPlay -> if (piece.isUpperCase()) "White's $what on $name" else "Black's $what on $name"
        }
        val g = guards.size
        val a = attackers.size
        val text = when {
            g == 0 && a == 0 -> "Nothing attacks it, and nothing guards it."
            a == 0 -> "Nothing attacks it, and it's guarded by $g of your pieces."
            g == 0 -> "Attacked by $a of their pieces, and nothing guards it."
            else -> "Attacked by $a, guarded by $g."
        }
        say(CoachCard.Kind.Coach, "◎", title, text)
        emit(Feedback.Select)
        return true
    }

    fun endInspect() {
        rays = null
        beforeInspect?.let { coach = it }
        beforeInspect = null
    }

    fun tap(square: Square) {
        if (reviewing != null) { endReview(); return }
        if (!playerToMove) return
        val from = selected
        if (from != null && targets.any { it.to == square }) { play(from, square); return }
        val piece = board[square]
        val mine = piece != '.' && (if (piece.isUpperCase()) "w" else "b") == turn
        if (mine && selected != square) {
            selected = square
            rays = makeRays(Rays.Kind.Options, targets)
            emit(Feedback.Select)
        } else {
            // Put back down: the quieter half of picking it up.
            if (selected != null) emit(Feedback.PutDown)
            selected = null
            rays = null
        }
    }

    /** Tapping the mark: the coach shows that move's verdict again. */
    fun revealMark() {
        val m = shownMark ?: return
        coach = m.first.card
        emit(Feedback.Select)
    }

    // Hints, undo, resign

    /** Hints come in two steps: first the piece to move; asking again shows the move and why, free. */
    fun hint() {
        if (!playerToMove) return
        selected = null
        val shown = hintShown
        if (shown != null && shown.first == moves.size) {
            BoardMove.of(shown.second, board)?.let { m ->
                nudge = null
                hintRevealed = true
                rays = makeRays(Rays.Kind.Hint, listOf(m))
                say(CoachCard.Kind.Great, "✦", "Try this", shown.third)
                emit(Feedback.Good)
            }
            return
        }
        if (hintsLeft <= 0) return
        hintsLeft--
        save()
        rays = null
        busy = true
        findingHint = true
        val snap = snapshot()
        val ply = moves.size
        val token = epoch
        scope.launch {
            val (best, san, why) = withContext(Dispatchers.Default) {
                val b = snap.search(3, 0)?.bestMove ?: ""
                Triple(b, if (b.isEmpty()) "" else snap.san(b), if (b.isEmpty()) "" else EngineText.localize(snap.explain(b)))
            }
            findingHint = false
            if (token != epoch) return@launch
            busy = false
            val m = BoardMove.of(best, board) ?: return@launch
            hintShown = Triple(ply, best, "$san: $why")
            hintRevealed = false
            nudge = m.from
            val piece = pieceName(board[m.from])
            say(CoachCard.Kind.Coach, "✦", "Look at your $piece",
                "Your $piece on ${Squares.name(m.from)} has a good move. Find it, or tap Show the move.")
            emit(Feedback.Select)
        }
    }

    fun undo() {
        mark = null
        sparkle = null
        nudge = null
        hintRevealed = false
        var steps = if (mode == Mode.Engine) 2 else 1
        if (mode == Mode.Engine && turn == "b") steps = 1 // the engine has not replied yet
        if (status == "resigned" || moves.size < steps) return
        // Undo works at any moment: whatever the engine was weighing is dropped.
        epoch++
        busy = false
        thinking = false
        val list = moves.toMutableList()
        repeat(steps) {
            game.undo()
            list.removeAt(list.lastIndex)
            val snap = history.removeLastOrNull()
            if (snap != null) { ids = snap.first; lastMove = snap.second } else { ids = arrayOfNulls(64); lastMove = null }
        }
        moves = list
        // Drop the verdicts on the moves taken back.
        val taken = trail.filter { it.ply >= moves.size }
        trail = trail.filter { it.ply < moves.size }
        repeat(minOf(taken.size, losses.size)) { losses.removeAt(losses.lastIndex) }
        great = maxOf(0, great - taken.count { it.card.kind == CoachCard.Kind.Great })
        blunders = maxOf(0, blunders - taken.count { it.card.kind == CoachCard.Kind.Blunder })
        evals = evals.take(moves.size).toMutableList()
        hintShown = null
        reviewing = null; reviewBoard = null; reviewMove = null
        selected = null
        rays = null
        toppled = null
        finish = null
        movingPiece = null
        refresh()
        moveSerial++
        say(CoachCard.Kind.Coach, "↶", "Rewound", "Your move again. Take another look.")
        save()
    }

    /** Ends the game as a loss. */
    fun resign() {
        if (mode == Mode.PassAndPlay || isOver || busy) return
        status = "resigned"
        selected = null
        rays = null
        loose = emptySet()
        board.indexOf(if (myColor == "w") 'K' else 'k').takeIf { it >= 0 }?.let { toppled = ids[it] }
        val count = (moves.size + 1) / 2
        val f = Finish(Finish.Outcome.Lost, "Good game", "Resigned on move $count", accuracy, great, blunders)
        scope.launch {
            delay(900)
            f.goodMoves = trail.count { it.card.kind == CoachCard.Kind.Good || it.card.kind == CoachCard.Kind.Great }
            f.best = bestMove()
            finish = withArc(f)
            emit(Feedback.Lose)
            Stats.record(f.accuracy, f.great)
            SavedGame.clear()
            Stats.addRecord(GameRecord(System.currentTimeMillis(), opponent.title, "lost", f.accuracy, count, opponent.ordinal, f.great))
        }
    }

    // Reviewing past moves

    /** Steps the review one rated move earlier (-1) or later (+1); past the latest returns to the game. */
    fun step(delta: Int) {
        if (trail.isEmpty()) return
        val index = reviewing?.let { id -> trail.indexOfFirst { it.id == id } } ?: trail.size
        val next = index + delta
        if (next >= trail.size) {
            if (reviewing != null) { endReview(); emit(Feedback.Select) }
            return
        }
        if (next < 0 || next == index) return
        selected = null
        rays = null
        review(trail[next])
        emit(Feedback.Select)
    }

    private fun review(entry: TrailEntry) {
        reviewing = entry.id
        position(entry.ply)?.let { reviewBoard = it.first; reviewMove = it.second }
        coach = CoachCard(++serial, entry.card.kind, entry.card.badge,
            "${entry.ply / 2 + 1}. ${entry.san} · ${entry.card.kind.verdict?.first ?: entry.card.title}",
            entry.card.text.replace(" Undo and try again.", ""))
    }

    fun endReview() {
        if (reviewing == null) return
        reviewing = null; reviewBoard = null; reviewMove = null
        coach = trail.lastOrNull()?.card ?: coach
    }

    // Moves

    private fun play(from: Square, to: Square) {
        var uci = Squares.name(from) + Squares.name(to)
        if (uci !in legal && "${uci}q" in legal) uci += "q"
        val fenBefore = game.fen()
        val ply = moves.size
        val san = apply(uci)
        if (san.isEmpty()) return
        busy = true
        val snap = snapshot()
        val token = epoch
        scope.launch {
            val (verdict, eval) = withContext(Dispatchers.Default) {
                Chesscore.newGameFromFEN(fenBefore)?.assess(uci) to snap.eval(2).toInt()
            }
            if (token != epoch) return@launch
            if (verdict != null) judged(verdict.kind, verdict.loss.toInt(), verdict.text, ply, san)
            setEval(eval)
            busy = false
            if (endIfOver()) return@launch
            if (mode == Mode.Engine) engineTurn()
        }
    }

    private fun judged(rawKind: String, loss: Int, rawText: String, ply: Int, san: String) {
        losses += loss
        val kind = CoachCard.Kind.of(rawKind)
        if (kind == CoachCard.Kind.Great) great++
        if (kind == CoachCard.Kind.Blunder) blunders++
        val words = kind.verdict ?: ("Good move" to "✓")
        val nudgeText = if (kind == CoachCard.Kind.Blunder) " Undo and try again." else ""
        // A calm coach: a good move is the norm, so it gets the badge and a word, no explanation.
        val reason = EngineText.localize(rawText)
        say(kind, words.second, words.first, if (kind == CoachCard.Kind.Good) "" else reason + nudgeText)
        verdictShownAt = if (kind == CoachCard.Kind.Good) null else System.currentTimeMillis()
        coach?.let { card ->
            val entry = TrailEntry(++serial, ply, san, card.copy(text = reason))
            trail = trail + entry
            lastMove?.to?.let { to ->
                mark = entry to to
                if (kind == CoachCard.Kind.Great) {
                    sparkle = entry.id to to
                    // A one-time burst: gone once it has played.
                    scope.launch { delay(1200); if (sparkle?.first == entry.id) sparkle = null }
                }
            }
        }
        reviewing = null; reviewBoard = null; reviewMove = null
        emit(when (kind) { CoachCard.Kind.Great -> Feedback.Great; CoachCard.Kind.Good -> Feedback.Good; else -> Feedback.Bad })
        save()
    }

    /** The engine shows the moves it is weighing, one by one, then commits. */
    private fun engineTurn() {
        busy = true
        thinking = true
        val snap = snapshot()
        val level = opponent.engineLevel
        val token = epoch
        scope.launch {
            val (candidates, choice, line) = withContext(Dispatchers.Default) {
                val c = snap.candidates(4).split(" ").filter { it.isNotEmpty() }
                val m = snap.botMove(level)
                Triple(c, m, if (m.isEmpty()) "" else EngineText.localize(snap.describe(m)))
            }
            if (token != epoch) return@launch
            if (choice.isEmpty()) { busy = false; thinking = false; return@launch }
            val shown = candidates.toMutableList()
            if (choice !in shown) { if (shown.isEmpty()) shown += choice else shown[shown.lastIndex] = choice }
            for (k in 1..shown.size) {
                if (token != epoch) return@launch
                rays = makeRays(Rays.Kind.Thinking, shown.take(k).mapNotNull { BoardMove.of(it, board) })
                delay(260)
            }
            if (token != epoch) return@launch
            BoardMove.of(choice, board)?.let { rays = makeRays(Rays.Kind.Chosen, listOf(it)) }
            delay(420)
            // Let the verdict on the player's move be read before it's replaced.
            verdictShownAt?.let { at ->
                val left = readingTime(trail.lastOrNull()?.card) - (System.currentTimeMillis() - at)
                if (left > 0) delay(left)
            }
            if (token != epoch) return@launch
            rays = null
            thinking = false
            val taken = BoardMove.of(choice, board)?.let { if (it.capture) board[it.to] else null }
            val san = apply(choice)
            busy = false
            if (endIfOver()) {
                say(CoachCard.Kind.Opponent, san, "Your opponent $line.", "That's the game.")
                return@launch
            }
            val after = snapshot()
            val eval = withContext(Dispatchers.Default) { after.eval(2).toInt() }
            if (token != epoch) return@launch
            setEval(eval)
            val tip = watchFor(lastMove, taken)
            val extra = if (offerComebackHint(eval)) " You're behind, so here's an extra hint." else ""
            say(if (tip.third) CoachCard.Kind.Warning else CoachCard.Kind.Opponent, san, tip.first, tip.second + extra)
            verdictShownAt = null
        }
    }

    /** When the player is clearly behind and nearly out of hints, gives one more. */
    private fun offerComebackHint(eval: Int): Boolean {
        if (mode != Mode.Engine || comebacks >= opponent.comebackHints || hintsLeft > 1 || eval > -opponent.behindBy) return false
        comebacks++
        hintsLeft++
        save()
        return true
    }

    private fun apply(uci: String): String {
        val from = Squares.parse(uci.substring(0, 2)) ?: return ""
        val to = Squares.parse(uci.substring(2, 4)) ?: return ""
        val before = board
        if (!game.play(uci)) return ""
        val san = game.lastSAN()
        moves = moves + uci
        history += ids.copyOf() to lastMove
        // A new move by the player replaces the old mark; the opponent's reply keeps it.
        if (mode == Mode.PassAndPlay || turn == myColor) mark = null
        nudge = null
        hintRevealed = false
        val captured = moveIds(from, to, before)
        lastMove = BoardMove(from, to)
        selected = null
        rays = null
        refresh()
        moveSerial++
        if (captured != null) {
            val burst = Burst(++serial, captured, !before[captured].isUpperCase())
            bursts = bursts + burst
            scope.launch { delay(900); bursts = bursts - burst }
            emit(Feedback.Capture)
        } else {
            emit(Feedback.Move)
        }
        if (inCheck) emit(Feedback.Check)
        save()
        return san
    }

    private fun endIfOver(): Boolean {
        if (!isOver) return false
        busy = false
        val mate = status == Chesscore.StatusCheckmate
        if (mate) checkSquare?.let { toppled = ids[it] }
        val count = (moves.size + 1) / 2
        var f: Finish = if (mate) {
            val whiteWon = turn == "b"
            if (mode == Mode.PassAndPlay) Finish(Finish.Outcome.Won, if (whiteWon) "White wins" else "Black wins",
                "Checkmate in $count moves.", accuracy, great, blunders)
            else Finish(if (whiteWon) Finish.Outcome.Won else Finish.Outcome.Lost, if (whiteWon) "You won!" else "Good fight",
                if (whiteWon) "Checkmate in $count moves." else "Checkmate after $count moves. Next time is yours.",
                accuracy, great, blunders)
        } else {
            val why = mapOf(
                Chesscore.StatusStalemate to "Stalemate: no legal moves, and no check.",
                Chesscore.StatusRepetition to "The same position came up three times.",
                Chesscore.StatusFiftyMove to "Fifty moves without a capture or a pawn move.",
                Chesscore.StatusInsufficient to "Neither side has enough left to mate.",
            )[status] ?: "The game is drawn."
            Finish(Finish.Outcome.Draw, "A draw", why, accuracy, great, blunders)
        }
        f.goodMoves = trail.count { it.card.kind == CoachCard.Kind.Good || it.card.kind == CoachCard.Kind.Great }
        if (mode == Mode.Engine && f.outcome == Finish.Outcome.Won) {
            val next = opponent.next
            val wasLocked = next?.let { !it.isUnlocked } ?: false
            val firstWin = Stats.wins(opponent) == 0
            Stats.addWin(opponent)
            f = f.copy(title = "You beat the ${opponent.title}!", firstWin = firstWin)
            if (next != null && wasLocked) {
                if (next.isUnlocked) f.unlocked = next else f.towardNext = next to Stats.wins(opponent)
            }
        }
        f.best = bestMove()
        f = withArc(f)
        if (mode == Mode.PassAndPlay) {
            f.sides = trail.count { it.ply % 2 == 0 && it.card.kind == CoachCard.Kind.Great } to
                trail.count { it.ply % 2 == 1 && it.card.kind == CoachCard.Kind.Great }
        }
        val final = f
        scope.launch {
            delay(1100)
            finish = final
            emit(if (final.outcome == Finish.Outcome.Won) Feedback.Win else Feedback.Lose)
            if (mode == Mode.Engine) Stats.record(final.accuracy, final.great)
            Stats.addRecord(GameRecord(System.currentTimeMillis(), if (mode == Mode.Engine) opponent.title else "Pass & play",
                final.outcome.name.lowercase().let { if (it == "won" || it == "lost") it else "draw" },
                final.accuracy, count, if (mode == Mode.Engine) opponent.ordinal else null, final.great))
            SavedGame.clear()
        }
        return true
    }

    // What to watch

    /** After the opponent moves: what it means for the player, as a title, a line and urgency. */
    private fun watchFor(m: BoardMove?, taken: Char?): Triple<String, String, Boolean> {
        val me = if (mode == Mode.PassAndPlay) turn else myColor
        val them = if (me == "w") "b" else "w"
        fun name(sq: Square) = "${pieceName(board[sq])} on ${Squares.name(sq)}"
        fun squares(s: String) = s.split(" ").mapNotNull { Squares.parse(it) }
        fun isMine(sq: Square) = board[sq] != '.' && (if (board[sq].isUpperCase()) "w" else "b") == me
        fun value(sq: Square) = VALUE[board[sq].lowercaseChar()] ?: 0

        if (inCheck) return Triple("Your king is in check", "Move it, block the line, or take the checking piece.", true)
        if (m != null) {
            val mover = pieceName(board[m.to])
            val hit = (0 until 64).filter { isMine(it) && board[it] != 'K' && m.to in squares(game.attackers(Squares.name(it))) }
                .sortedByDescending { value(it) }
            hit.firstOrNull { game.guards(Squares.name(it)).isEmpty() }?.let {
                return Triple("Watch your ${name(it)}", "Their $mover now attacks it, and nothing guards it.", true)
            }
            if (taken != null) {
                val back = game.attackers(Squares.name(m.to)).isNotEmpty()
                return Triple("They took your ${pieceName(taken)}",
                    if (back) "You can take back on ${Squares.name(m.to)}." else "Look for a way to win it back.", false)
            }
            hit.firstOrNull()?.let { sq ->
                if (value(sq) > value(m.to)) return Triple("Their $mover eyes your ${pieceName(board[sq])}",
                    "It's guarded, but their $mover is worth less, so a trade would cost you.", true)
            }
        }
        loose.minOrNull()?.let { return Triple("Your ${name(it)} is loose", "It's attacked and nothing guards it.", true) }
        squares(game.loose(them)).firstOrNull()?.let {
            return Triple("A chance", "Their ${name(it)} is under attack and unguarded.", false)
        }
        // Nothing pressing: a quiet "Your move", with a tip only every other time.
        quietTurns++
        return Triple("Your move", if (quietTurns % 2 == 1) tip() else "", false)
    }

    /** A tip that fits the position, or one of the app's gestures, each shown once. */
    private fun tip(): String {
        val castling = game.fen().split(" ").getOrElse(2) { "" }
        val white = (if (mode == Mode.PassAndPlay) turn else myColor) == "w"
        val home = if (white) listOf(57 to 'N', 62 to 'N', 58 to 'B', 61 to 'B') else listOf(1 to 'n', 6 to 'n', 2 to 'b', 5 to 'b')
        if (moves.size < 16 && home.count { board[it.first] == it.second } >= 2) return "Bring a knight or bishop toward the centre."
        if (moves.size < 30 && board[if (white) 60 else 4] == (if (white) 'K' else 'k') &&
            castling.any { if (white) it == 'K' || it == 'Q' else it == 'k' || it == 'q' }
        ) return "Castling tucks your king away and brings a rook into play."
        val gestures = listOf(
            "tip.hold" to "Hold any piece to see what guards it and what attacks it.",
            "tip.drag" to "You can drag pieces as well as tap them.",
            "tip.swipe" to "Swipe this card to look back at your earlier moves.",
        )
        gestures.firstOrNull { !Prefs.p.getBoolean(it.first, false) }?.let {
            Prefs.p.edit().putBoolean(it.first, true).apply()
            return it.second
        }
        // Ideas, never moves: something to look for, not something to play.
        val ideas = listOf(
            "Before you move, ask what their last move wants.",
            "Pieces in the centre reach the most squares.",
            "Look for checks, captures and threats, for both sides.",
            "A knight on the edge reaches only half its squares.",
            "Rooks love open lines with no pawns in the way.",
            "Ahead in pieces? Trading down makes it simpler.",
            "Pawns can't move back, so push them with a purpose.",
            "Every piece helping beats one piece attacking alone.",
        )
        ideaTurn++
        return ideas[(ideaTurn - 1) % ideas.size]
    }

    // Bookkeeping

    /** Moves piece identities to match a move; returns the captured square. */
    private fun moveIds(from: Square, to: Square, before: List<Char>): Square? {
        val piece = before[from].lowercaseChar()
        var captured: Square? = if (before[to] != '.') to else null
        if (piece == 'p' && Squares.file(from) != Squares.file(to) && before[to] == '.') {
            val victim = Squares.row(from) * 8 + Squares.file(to)
            ids[victim] = null
            captured = victim
        }
        if (piece == 'k' && abs(Squares.file(to) - Squares.file(from)) == 2) {
            val row = Squares.row(from) * 8
            val (rf, rt) = if (Squares.file(to) == 6) row + 7 to row + 5 else row to row + 3
            ids[rt] = ids[rf]
            ids[rf] = null
        }
        movingPiece = ids[from]
        ids[to] = ids[from]
        ids[from] = null
        return captured
    }

    private fun refresh() {
        board = game.board().toList()
        turn = game.turn()
        status = game.status()
        inCheck = game.inCheck()
        legal = game.legalMoves().split(" ").filter { it.isNotEmpty() }
        val mine = if (mode == Mode.PassAndPlay) turn else myColor
        loose = if (isOver) emptySet() else game.loose(mine).split(" ").mapNotNull { Squares.parse(it) }.toSet()
        val items = mutableListOf<PieceItem>()
        for (sq in 0 until 64) {
            if (board[sq] == '.') { ids[sq] = null; continue }
            if (ids[sq] == null) ids[sq] = nextId++
            items += PieceItem(ids[sq]!!, board[sq], sq)
        }
        pieces = items
    }

    /** A private copy for work off the main thread, replayed from the moves so it keeps repetition history. */
    private fun snapshot(): Game = Chesscore.newGame().also { g -> moves.forEach { g.play(it) } }

    /** The board after a half-move, and that move. */
    private fun position(ply: Int): Pair<List<Char>, BoardMove?>? {
        val g = Chesscore.newGame()
        var before = g.board().toList()
        for ((i, uci) in moves.take(ply + 1).withIndex()) {
            if (i == ply) before = g.board().toList()
            g.play(uci)
        }
        return g.board().toList() to moves.getOrNull(ply)?.let { BoardMove.of(it, before) }
    }

    /** The game's arc from the player's side: the chance to win after every half-move. */
    private fun withArc(f: Finish): Finish {
        val out = f.copy(plies = moves.size)
        val mine = if (mode == Mode.PassAndPlay) "w" else myColor
        var last = 0.5
        val arc = mutableListOf(0.5)
        for (i in moves.indices) {
            evals.getOrNull(i)?.let { cp -> val w = share(cp); last = if (mine == "w") w else 1 - w }
            arc += last
        }
        if (status == Chesscore.StatusCheckmate) arc[arc.lastIndex] = if (out.outcome == Finish.Outcome.Won) 1.0 else 0.0
        return out.reading(arc, if (mine == "w") 0 else 1, mode == Mode.PassAndPlay)
    }

    /** The player's best move: the strongest verdict, the latest among equals. */
    private fun bestMove(): BestMove? {
        val rank = mapOf(CoachCard.Kind.Great to 4, CoachCard.Kind.Good to 3, CoachCard.Kind.Inaccuracy to 2,
            CoachCard.Kind.Mistake to 1, CoachCard.Kind.Blunder to 0)
        val e = trail.maxWithOrNull(compareBy<TrailEntry>({ rank[it.card.kind] ?: 0 }, { it.ply })) ?: return null
        if ((rank[e.card.kind] ?: 0) < 3) return null
        val after = position(e.ply) ?: return null
        val move = after.second ?: return null
        val before = position(e.ply - 1) ?: return null
        return BestMove(if (mode == Mode.PassAndPlay) (if (e.ply % 2 == 0) "White" else "Black") else null,
            before.first, after.first, move, e.ply / 2 + 1, e.card.kind, e.card.text, e.ply, e.san)
    }

    /** Pieces each side has taken, biggest first. */
    fun captured(by: String): List<Char> {
        val start = mapOf('q' to 1, 'r' to 2, 'b' to 2, 'n' to 2, 'p' to 8)
        val opponentIsWhite = by == "b"
        val have = mutableMapOf<Char, Int>()
        for (c in board) if (c != '.' && c.isUpperCase() == opponentIsWhite) have[c.lowercaseChar()] = (have[c.lowercaseChar()] ?: 0) + 1
        return "qrbnp".flatMap { k -> List(maxOf(0, start[k]!! - (have[k] ?: 0))) { if (opponentIsWhite) k.uppercaseChar() else k } }
    }

    fun materialLead(side: String): Int {
        var score = 0
        for (c in board) if (c != '.') score += (if (c.isUpperCase()) 1 else -1) * (VALUE[c.lowercaseChar()] ?: 0)
        return if (side == "w") score else -score
    }

    private fun setEval(cp: Int) {
        whiteShare = share(cp)
        val i = moves.size - 1
        if (i < 0) return
        while (evals.size <= i) evals += null
        evals[i] = cp
    }

    private fun share(cp: Int) = if (abs(cp) >= 9000) (if (cp > 0) 1.0 else 0.0) else 1 / (1 + exp(-cp / 220.0))

    private fun makeRays(kind: Rays.Kind, list: List<BoardMove>) =
        Rays(++serial, kind, list, knight = list.filter { board[it.from].lowercaseChar() == 'n' }.toSet())

    private fun say(kind: CoachCard.Kind, badge: String, title: String, text: String) {
        coach = CoachCard(++serial, kind, badge, title, text)
    }

    private fun emit(kind: Feedback) {
        feedback = ++serial to kind
        Haptics.play(kind)
    }

    companion object {
        /** Accuracy is shown once there is enough to say something. */
        const val ACCURACY_AFTER = 3
        private val VALUE = mapOf('p' to 1, 'n' to 3, 'b' to 3, 'r' to 5, 'q' to 9, 'k' to 0)

        fun pieceName(c: Char) = mapOf('p' to "pawn", 'n' to "knight", 'b' to "bishop", 'r' to "rook", 'q' to "queen", 'k' to "king")[c.lowercaseChar()] ?: "piece"

        /** How long the coach's verdict needs on screen to be read, in ms: never under 2 s or over 4 s. */
        private fun readingTime(card: CoachCard?): Long {
            card ?: return 0
            return (minOf(4.0, maxOf(2.0, 1.2 + (card.title.length + card.text.length) * 0.035)) * 1000).toLong()
        }
    }
}
