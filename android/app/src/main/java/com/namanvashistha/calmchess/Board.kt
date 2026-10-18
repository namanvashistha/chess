package com.namanvashistha.calmchess

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
    /** Moves drawn in the danger colour. */
    val danger: Set<BoardMove> = emptySet(),
) {
    enum class Kind { Options, Thinking, Chosen, Hint }
}
