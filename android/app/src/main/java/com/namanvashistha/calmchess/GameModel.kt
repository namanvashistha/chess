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

/** The three opponents, gentlest first. */
enum class Opponent(val title: String, val engineLevel: String) {
    Easy("Rookie", "beginner"),
    Medium("Master", "easy"),
    Hard("Grandmaster", "medium"),
}

/** What the coach is saying. */
data class CoachCard(val id: Int, val kind: Kind, val badge: String, val title: String, val text: String) {
    enum class Kind {
        Great, Good, Inaccuracy, Mistake, Blunder, Coach, Warning, Opponent;

        /** The app's own words for a verdict: kind, never harsh. */
        val verdict: Pair<String, String>?
            get() = when (this) {
                Great -> "Great move" to "★"
                Good -> "Good move" to "✓"
                Inaccuracy -> "Almost" to "~"
                Mistake -> "Not quite" to "?"
                Blunder -> "Oops" to "↩"
                else -> null
            }

        companion object {
            fun of(raw: String) = entries.firstOrNull { it.name.equals(raw, ignoreCase = true) } ?: Good
        }
    }
}

/**
 * A game against the engine, as the board and the screen see it. The same
 * shape as the iPhone app's GameModel: the engine (shared Go code) owns the
 * rules; this keeps each piece's identity so it can glide, and narrates.
 */
class GameModel(private val scope: CoroutineScope) {
    var pieces by mutableStateOf(listOf<PieceItem>()); private set
    var board by mutableStateOf(List(64) { '.' }); private set
    var turn by mutableStateOf("w"); private set
    var selected by mutableStateOf<Square?>(null); private set
    var rays by mutableStateOf<Rays?>(null); private set
    var lastMove by mutableStateOf<BoardMove?>(null); private set
    var movingPiece by mutableStateOf<Int?>(null); private set
    var moveSerial by mutableStateOf(0); private set
    var thinking by mutableStateOf(false); private set
    var busy by mutableStateOf(false); private set
    var status by mutableStateOf(""); private set
    var inCheck by mutableStateOf(false); private set
    var coach by mutableStateOf<CoachCard?>(null); private set
    var opponent by mutableStateOf(Opponent.Easy); private set

    private var game: Game = Chesscore.newGame()
    private var legal = listOf<String>()
    private val ids = arrayOfNulls<Int>(64)
    private var nextId = 0
    private val moves = mutableListOf<String>()
    private var serial = 0
    /** Bumps whenever pending engine work must be dropped. */
    private var epoch = 0
    private val myColor = "w"

    val isOver get() = status.isNotEmpty()
    val playerToMove get() = !busy && !isOver && turn == myColor

    /** The king in check, if any. */
    val checkSquare: Square?
        get() = if (!inCheck) null else board.indices.firstOrNull { board[it] == (if (turn == "w") 'K' else 'k') }

    /** Squares the selected piece can move to. */
    val targets: List<BoardMove>
        get() {
            val from = selected?.let(Squares::name) ?: return emptyList()
            return legal.filter { it.startsWith(from) }
                .mapNotNull { BoardMove.of(it, board) }
                .distinctBy { it.to }
        }

    init { start(Opponent.Easy) }

    fun start(opponent: Opponent) {
        this.opponent = opponent
        game = Chesscore.newGame()
        epoch++
        ids.fill(null)
        moves.clear()
        lastMove = null
        selected = null
        rays = null
        busy = false
        thinking = false
        refresh()
        moveSerial++
        say(CoachCard.Kind.Coach, "✦", "Your move", "You play White. Tap a piece to see where it can go.")
    }

    // Touch

    fun tap(square: Square) {
        if (!playerToMove) return
        val from = selected
        if (from != null && targets.any { it.to == square }) {
            play(from, square)
            return
        }
        val piece = board[square]
        val mine = piece != '.' && (if (piece.isUpperCase()) "w" else "b") == turn
        if (mine && selected != square) {
            selected = square
            rays = makeRays(Rays.Kind.Options, targets)
            Haptics.select()
        } else {
            if (selected != null) Haptics.putDown()
            selected = null
            rays = null
        }
    }

    /** Picks up one of the mover's pieces to drag. False if it can't move. */
    fun pickUp(square: Square): Boolean {
        if (!playerToMove) return false
        val piece = board[square]
        if (piece == '.' || (if (piece.isUpperCase()) "w" else "b") != turn) return false
        if (selected != square) {
            selected = square
            rays = makeRays(Rays.Kind.Options, targets)
            Haptics.select()
        }
        return true
    }

    /** Drops the dragged piece: a legal square plays it; anywhere else it stays selected. */
    fun drop(on: Square?) {
        val from = selected ?: return
        if (on != null && targets.any { it.to == on }) play(from, on)
    }

    // Moves

    private fun play(from: Square, to: Square) {
        var uci = Squares.name(from) + Squares.name(to)
        if (uci !in legal && "${uci}q" in legal) uci += "q"
        val fenBefore = game.fen()
        val san = apply(uci)
        if (san.isEmpty()) return
        busy = true
        val token = epoch
        scope.launch {
            val verdict = withContext(Dispatchers.Default) { Chesscore.newGameFromFEN(fenBefore)?.assess(uci) }
            if (token != epoch) return@launch
            if (verdict != null) {
                val kind = CoachCard.Kind.of(verdict.kind)
                val words = kind.verdict ?: ("Good move" to "✓")
                say(kind, words.second, words.first, if (kind == CoachCard.Kind.Good) "" else verdict.text)
            }
            busy = false
            if (isOver) return@launch
            engineTurn(token)
        }
    }

    /** The engine shows the moves it is weighing, one by one, then commits. */
    private suspend fun engineTurn(token: Int) {
        busy = true
        thinking = true
        val snap = snapshot()
        val level = opponent.engineLevel
        val (candidates, choice, line) = withContext(Dispatchers.Default) {
            val c = snap.candidates(4).split(" ").filter { it.isNotEmpty() }
            val m = snap.botMove(level)
            Triple(c, m, if (m.isEmpty()) "" else snap.describe(m))
        }
        if (token != epoch) return
        if (choice.isEmpty()) { busy = false; thinking = false; return }
        val shown = candidates.toMutableList()
        if (choice !in shown) { if (shown.isEmpty()) shown += choice else shown[shown.lastIndex] = choice }
        for (k in 1..shown.size) {
            if (token != epoch) return
            rays = makeRays(Rays.Kind.Thinking, shown.take(k).mapNotNull { BoardMove.of(it, board) })
            delay(260)
        }
        if (token != epoch) return
        BoardMove.of(choice, board)?.let { rays = makeRays(Rays.Kind.Chosen, listOf(it)) }
        delay(420)
        // Let the verdict on the player's move be read before it's replaced.
        delay(readingTime(coach))
        if (token != epoch) return
        rays = null
        thinking = false
        val san = apply(choice)
        busy = false
        say(CoachCard.Kind.Opponent, san, "Your move", if (line.isEmpty()) "" else "Your opponent $line.")
    }

    private fun apply(uci: String): String {
        val from = Squares.parse(uci.substring(0, 2)) ?: return ""
        val to = Squares.parse(uci.substring(2, 4)) ?: return ""
        val before = board
        if (!game.play(uci)) return ""
        val san = game.lastSAN()
        moves += uci
        val captured = moveIds(from, to, before)
        lastMove = BoardMove(from, to)
        selected = null
        rays = null
        refresh()
        moveSerial++
        if (captured != null) Haptics.capture() else Haptics.move()
        return san
    }

    /** Moves piece identities to match a move; returns the captured square. */
    private fun moveIds(from: Square, to: Square, before: List<Char>): Square? {
        val piece = before[from].lowercaseChar()
        var captured: Square? = if (before[to] != '.') to else null
        if (piece == 'p' && Squares.file(from) != Squares.file(to) && before[to] == '.') {
            val victim = Squares.row(from) * 8 + Squares.file(to)
            ids[victim] = null
            captured = victim
        }
        if (piece == 'k' && kotlin.math.abs(Squares.file(to) - Squares.file(from)) == 2) {
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
        val items = mutableListOf<PieceItem>()
        for (sq in 0 until 64) {
            if (board[sq] == '.') { ids[sq] = null; continue }
            if (ids[sq] == null) ids[sq] = nextId++
            items += PieceItem(ids[sq]!!, board[sq], sq)
        }
        pieces = items
    }

    /** A private copy for work off the main thread, so the real game can change meanwhile. */
    private fun snapshot(): Game = Chesscore.newGame().also { g -> moves.forEach { g.play(it) } }

    private fun makeRays(kind: Rays.Kind, list: List<BoardMove>) =
        Rays(++serial, kind, list, knight = list.filter { board[it.from].lowercaseChar() == 'n' }.toSet())

    private fun say(kind: CoachCard.Kind, badge: String, title: String, text: String) {
        coach = CoachCard(++serial, kind, badge, title, text)
    }

    /** How long a verdict needs on screen to be read, in milliseconds. */
    private fun readingTime(card: CoachCard?): Long {
        if (card == null || card.kind == CoachCard.Kind.Good) return 0
        return (minOf(4.0, maxOf(2.0, 1.2 + (card.title.length + card.text.length) * 0.035)) * 1000).toLong() - 680
    }
}
