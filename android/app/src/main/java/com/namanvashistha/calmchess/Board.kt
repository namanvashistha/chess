package com.namanvashistha.calmchess

import org.json.JSONObject

/** A square index in display order: 0 is a8, 7 is h8, 63 is h1. Matches the engine's board(). */
typealias Square = Int

object Squares {
    fun name(sq: Square): String = "${'a' + sq % 8}${8 - sq / 8}"

    fun parse(name: CharSequence): Square? {
        if (name.length != 2) return null
        val f = name[0] - 'a'
        val r = name[1] - '0'
        if (f !in 0..7 || r !in 1..8) return null
        return (8 - r) * 8 + f
    }

    fun file(sq: Square) = sq % 8
    fun row(sq: Square) = sq / 8
}

/** A move between two squares, as the board draws it. */
data class BoardMove(val from: Square, val to: Square, val capture: Boolean = false) {
    companion object {
        fun of(uci: String, board: List<Char>): BoardMove? {
            if (uci.length < 4) return null
            val f = Squares.parse(uci.substring(0, 2)) ?: return null
            val t = Squares.parse(uci.substring(2, 4)) ?: return null
            return BoardMove(f, t, board.getOrNull(t)?.let { it != '.' } ?: false)
        }
    }
}

/** A piece on the board with a stable identity, so it can glide between squares. */
data class PieceItem(val id: Int, val code: Char, val square: Square)

/** Lines of light drawn on the board. */
data class Rays(
    val id: Int,
    val kind: Kind,
    val moves: List<BoardMove>,
    /** Moves drawn as a knight's L rather than a straight line. */
    val knight: Set<BoardMove> = emptySet(),
    /** Moves drawn in the danger colour: attackers, when inspecting a piece. */
    val danger: Set<BoardMove> = emptySet(),
) {
    enum class Kind { Options, Thinking, Chosen, Hint }
}

/** What the coach is saying. */
data class CoachCard(val id: Int, val kind: Kind, val badge: String, val title: String, val text: String) {
    enum class Kind {
        Great, Good, Inaccuracy, Mistake, Blunder, Coach, Warning, Opponent;

        /** The app's own words for a verdict: kind, never harsh. */
        val verdict: Pair<String, String>?
            get() = when (this) {
                Great -> L("Great move") to "★"
                Good -> L("Good move") to "✓"
                Inaccuracy -> L("Almost") to "~"
                Mistake -> L("Not quite") to "?"
                Blunder -> L("Oops") to "↩"
                else -> null
            }

        companion object {
            fun of(raw: String) = entries.firstOrNull { it.name.equals(raw, ignoreCase = true) } ?: Good
        }
    }

    fun toJson() = JSONObject().apply { put("id", id); put("kind", kind.name); put("badge", badge); put("title", title); put("text", text) }

    companion object {
        fun from(o: JSONObject) = CoachCard(o.getInt("id"), Kind.of(o.getString("kind")), o.getString("badge"), o.getString("title"), o.getString("text"))
    }
}

/** One of the player's moves with the coach's verdict, for the move trail. */
data class TrailEntry(val id: Int, val ply: Int, val san: String, val card: CoachCard) {
    fun toJson() = JSONObject().apply { put("id", id); put("ply", ply); put("san", san); put("card", card.toJson()) }

    companion object {
        fun from(o: JSONObject) = TrailEntry(o.getInt("id"), o.getInt("ply"), o.getString("san"), CoachCard.from(o.getJSONObject("card")))
    }
}

/** A move worth replaying: the boards either side of it and what the coach said. */
data class BestMove(
    val side: String?,
    val before: List<Char>,
    val after: List<Char>,
    val move: BoardMove,
    val moveNumber: Int,
    val kind: CoachCard.Kind,
    val text: String,
    val ply: Int,
    val san: String,
)

/** A capture's glass shards. */
data class Burst(val id: Int, val square: Square, val dark: Boolean)

/** How the game ended, for the finish card. */
data class Finish(
    val outcome: Outcome,
    val title: String,
    val text: String,
    val accuracy: Int?,
    val great: Int,
    val blunders: Int,
    var goodMoves: Int = 0,
    var best: BestMove? = null,
    /** Pass and play: great moves for each side, in place of accuracy. */
    var sides: Pair<Int, Int>? = null,
    var unlocked: Opponent? = null,
    var firstWin: Boolean = false,
    var towardNext: Pair<Opponent, Int>? = null,
    var plies: Int = 0,
    /** The player's chance to win, 0 to 1, from the start and after every half-move. */
    var arc: List<Double> = emptyList(),
    var turnPly: Int? = null,
    var insight: String? = null,
) {
    enum class Outcome { Won, Lost, Draw }

    /** Reads the arc: where it turned on the player's own move, and the one sentence worth saying. */
    fun reading(arc: List<Double>, playerParity: Int, twoPlayers: Boolean): Finish {
        val f = copy()
        if (arc.size < 5) return f
        f.arc = arc
        if (twoPlayers) return f
        var worst: Pair<Int, Double>? = null
        for (i in 1 until arc.size) {
            if ((i - 1) % 2 != playerParity) continue
            val drop = arc[i - 1] - arc[i]
            if (drop > (worst?.second ?: 0.15)) worst = (i - 1) to drop
        }
        f.turnPly = if (outcome == Outcome.Won) null else worst?.first
        fun move(i: Int) = (i - 1) / 2 + 1
        fun pct(v: Double) = Math.round(v * 100).toInt()
        val played = (1 until arc.size).toList()
        val peak = played.maxBy { arc[it] }
        val low = played.minBy { arc[it] }
        when (outcome) {
            Outcome.Won ->
                if (arc[low] <= 0.35) f.insight = L("Came back from %lld%% on move %lld.", pct(arc[low]), move(low))
                else if (played.all { arc[it] >= 0.45 }) f.insight = L("You never fell behind.")
            else ->
                if (arc[peak] >= 0.6) f.insight = L("You were winning: %lld%% on move %lld.", pct(arc[peak]), move(peak))
                else worst?.let { f.insight = L("It turned on move %lld.", it.first / 2 + 1) }
        }
        return f
    }
}

/** The three opponents, gentlest first. */
enum class Opponent {
    Easy, Medium, Hard;

    /** Chess titles, so each win reads as a step up. */
    val title get() = listOf(L("Rookie"), L("Master"), L("Grandmaster"))[ordinal]
    val subtitle get() = listOf(L("Beginner"), L("Intermediate"), L("Expert"))[ordinal]
    val engineLevel get() = listOf("beginner", "easy", "medium")[ordinal]
    val hints get() = listOf(5, 4, 3)[ordinal]

    /** Hints to start a game with, matched to how the player has been doing lately. */
    val startingHints: Int
        get() {
            val games = Stats.recent.filter { it.resolvedLevel == ordinal }.take(5)
            val accuracy = games.mapNotNull { it.accuracy }
            if (accuracy.size < 2) return hints
            val avg = accuracy.sum() / accuracy.size
            var extra = if (avg < 60) 2 else if (avg < 72) 1 else if (avg >= 88) -1 else 0
            // Three losses in a row: a little more help, whatever the accuracy.
            if (games.size >= 3 && games.take(3).all { it.outcome == "lost" }) extra += 1
            return (hints + extra).coerceIn(2, 7)
        }

    /** Extra hints handed out when the player falls behind, at most this many a game. */
    val comebackHints get() = listOf(4, 2, 0)[ordinal]
    /** How far behind, in centipawns, before an extra hint is offered. */
    val behindBy get() = listOf(100, 150, 0)[ordinal]

    val previous get() = entries.getOrNull(ordinal - 1)
    val next get() = entries.getOrNull(ordinal + 1)
    val isUnlocked get() = previous?.let { Stats.wins(it) >= WINS_TO_UNLOCK } ?: true

    companion object {
        const val WINS_TO_UNLOCK = 3
    }
}

enum class Mode { Engine, PassAndPlay }
