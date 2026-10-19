package com.namanvashistha.calmchess

import android.content.Context
import android.content.SharedPreferences
import org.json.JSONArray
import org.json.JSONObject
import java.util.Calendar

/** The app's small key-value store, like UserDefaults on iPhone. */
object Prefs {
    lateinit var p: SharedPreferences
        private set

    fun init(context: Context) {
        p = context.getSharedPreferences("calmchess", Context.MODE_PRIVATE)
    }
}

/** A finished game, for the home screen's list. */
data class GameRecord(
    val date: Long,
    val opponent: String,
    val outcome: String, // "won", "lost", "draw"
    val accuracy: Int?,
    val moves: Int,
    val level: Int?,
    val great: Int?,
) {
    val resolvedLevel: Int? get() = level

    fun toJson() = JSONObject().apply {
        put("date", date); put("opponent", opponent); put("outcome", outcome)
        accuracy?.let { put("accuracy", it) }; put("moves", moves)
        level?.let { put("level", it) }; great?.let { put("great", it) }
    }

    companion object {
        fun from(o: JSONObject) = GameRecord(
            o.getLong("date"), o.getString("opponent"), o.getString("outcome"),
            if (o.has("accuracy")) o.getInt("accuracy") else null, o.getInt("moves"),
            if (o.has("level")) o.getInt("level") else null, if (o.has("great")) o.getInt("great") else null,
        )
    }
}

/** The player's running numbers: a day streak, average accuracy and recent games. */
object Stats {
    private val p get() = Prefs.p

    val streak get() = p.getInt("streak", 0)
    val averageAccuracy: Int? get() = if (p.getInt("games", 0) == 0) null else Math.round(p.getFloat("accuracy", 0f))
    val greatMoves get() = p.getInt("greatMoves", 0)

    val recent: List<GameRecord>
        get() = runCatching {
            val a = JSONArray(p.getString("recent", "[]"))
            (0 until a.length()).map { GameRecord.from(a.getJSONObject(it)) }
        }.getOrDefault(emptyList())

    /** Keeps the last 20 games: the home screen lists 5 and charts 10. */
    fun addRecord(r: GameRecord) {
        val list = (listOf(r) + recent).take(20)
        p.edit().putString("recent", JSONArray(list.map { it.toJson() }).toString()).apply()
    }

    fun wins(o: Opponent) = p.getInt("wins.${o.ordinal}", 0)
    fun addWin(o: Opponent) = p.edit().putInt("wins.${o.ordinal}", wins(o) + 1).apply()

    fun record(accuracy: Int?, great: Int = 0) {
        val e = p.edit()
        e.putInt("greatMoves", greatMoves + great)
        val today = dayNumber(System.currentTimeMillis())
        val last = p.getInt("lastDay", -1)
        if (last != today) {
            e.putInt("streak", if (last == today - 1) streak + 1 else 1)
            e.putInt("lastDay", today)
        }
        if (accuracy != null) {
            val n = p.getInt("games", 0)
            val avg = p.getFloat("accuracy", 0f)
            // A moving average that leans toward recent games.
            e.putFloat("accuracy", if (n == 0) accuracy.toFloat() else avg * 0.8f + accuracy * 0.2f)
            e.putInt("games", n + 1)
        }
        e.apply()
    }

    /** Days since the epoch in the phone's time zone. */
    private fun dayNumber(ms: Long): Int {
        val c = Calendar.getInstance().apply { timeInMillis = ms }
        return ((ms + c.get(Calendar.ZONE_OFFSET) + c.get(Calendar.DST_OFFSET)) / 86_400_000L).toInt()
    }
}

/** A game in progress, saved after every move so it survives quitting. */
data class SavedGame(
    val passAndPlay: Boolean,
    val opponent: Int,
    val moves: List<String>,
    val trail: List<TrailEntry>,
    val losses: List<Int>,
    val great: Int,
    val blunders: Int,
    val hintsLeft: Int,
    val evals: List<Int?>,
    val comebacks: Int,
) {
    fun store() {
        val o = JSONObject().apply {
            put("passAndPlay", passAndPlay); put("opponent", opponent)
            put("moves", JSONArray(moves)); put("losses", JSONArray(losses))
            put("great", great); put("blunders", blunders); put("hintsLeft", hintsLeft); put("comebacks", comebacks)
            put("evals", JSONArray(evals.map { it ?: JSONObject.NULL }))
            put("trail", JSONArray(trail.map { it.toJson() }))
        }
        Prefs.p.edit().putString(KEY, o.toString()).apply()
    }

    companion object {
        private const val KEY = "savedGame"

        fun load(): SavedGame? = runCatching {
            val o = JSONObject(Prefs.p.getString(KEY, null) ?: return null)
            fun ints(name: String) = o.getJSONArray(name).let { a -> (0 until a.length()).map { a.getInt(it) } }
            val ev = o.getJSONArray("evals")
            val tr = o.getJSONArray("trail")
            SavedGame(
                o.getBoolean("passAndPlay"), o.getInt("opponent"),
                o.getJSONArray("moves").let { a -> (0 until a.length()).map { a.getString(it) } },
                (0 until tr.length()).map { TrailEntry.from(tr.getJSONObject(it)) },
                ints("losses"), o.getInt("great"), o.getInt("blunders"), o.getInt("hintsLeft"),
                (0 until ev.length()).map { if (ev.isNull(it)) null else ev.getInt(it) },
                o.optInt("comebacks", 0),
            )
        }.getOrNull()

        fun clear() = Prefs.p.edit().remove(KEY).apply()
    }
}
