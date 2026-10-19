package com.namanvashistha.calmchess

import android.content.Context
import org.json.JSONObject

/**
 * Translations, from the iPhone app's own string catalog (`make
 * android-strings`), so both apps say exactly the same thing in Hindi and
 * Spanish. English is the key itself.
 */
object Strings {
    private var table: Map<String, Map<String, String>> = emptyMap()
    /** The app's language: the phone's, or the one picked for this app in system settings. */
    var language = "en"
        private set

    fun init(context: Context) {
        language = context.resources.configuration.locales[0].language
        table = runCatching {
            val o = JSONObject(context.assets.open("strings.json").bufferedReader().use { it.readText() })
            o.keys().asSequence().associateWith { k ->
                val t = o.getJSONObject(k)
                t.keys().asSequence().associateWith { t.getString(it) }
            }
        }.getOrDefault(emptyMap())
    }

    fun lookup(key: String) = table[key]?.get(language) ?: key
}

private val placeholder = Regex("""%(\d+\$)?(@|lld|ld|d)|%%""")

/**
 * A sentence in the player's language. Values fill the translation's
 * placeholders (%@ and %lld, or positional %1$@), so word order can change
 * per language, as with the iPhone's String(localized:).
 */
fun L(key: String, vararg args: Any): String {
    val text = Strings.lookup(key)
    if (args.isEmpty() && !text.contains("%%")) return text
    var next = 0
    return placeholder.replace(text) { m ->
        if (m.value == "%%") "%"
        else {
            val index = m.groupValues[1].dropLast(1).toIntOrNull()?.minus(1) ?: next++
            args.getOrNull(index)?.toString() ?: ""
        }
    }
}

/**
 * The engine writes its coaching in English, from a few fixed templates. This
 * matches each template and rebuilds the sentence in the player's language.
 */
object EngineText {
    private val better = Regex("""(\S+) would have been even better\.""")

    fun localize(english: String): String {
        // In every language: name the piece, not the move. "Your queen had a better move" is enough.
        better.matchEntire(english)?.let { m ->
            val kind = when (m.groupValues[1].first()) {
                'K', 'O' -> "king"; 'Q' -> "queen"; 'R' -> "rook"; 'B' -> "bishop"; 'N' -> "knight"; else -> "pawn"
            }
            return L("Your %@ had a better move here.", L(kind))
        }
        if (english.isEmpty() || Strings.language == "en") return english
        val s = english
        fun match(pattern: String) = Regex(pattern).matchEntire(s)?.groupValues?.drop(1)
        fun piece(name: String) = L(name)
        when (s) {
            "Checkmate.", "King tucked away and the rook brought into play.", "Check, and it keeps the pressure on.",
            "Claims the centre.", "Exactly what the engine would play.", "A solid, sensible move.",
            "There was an even better move here.", "it is checkmate.", "castle to keep your king safe.",
            "it gives check.", "it improves your position the most." -> return L(s)
        }
        match("""You win the (\w+)\. Clean\.""")?.let { return L("You win the %@. Clean.", piece(it[0])) }
        match("""Develops the (\w+) toward the centre\.""")?.let { return L("Develops the %@ toward the centre.", piece(it[0])) }
        match("""Careful: your (\w+) on ([a-h][1-8]) can be taken now\.""")?.let { return L("Careful: your %@ on %@ can be taken now.", piece(it[0]), it[1]) }
        match("""it wins the (\w+)\.""")?.let { return L("it wins the %@.", piece(it[0])) }
        match("""takes your (\w+) on ([a-h][1-8])(\. Checkmate|, with check)?""")?.let { return L("takes your %@ on %@", piece(it[0]), it[1]) + suffix(s) }
        match("""moves the (\w+) to ([a-h][1-8])(\. Checkmate|, with check)?""")?.let { return L("moves the %@ to %@", piece(it[0]), it[1]) + suffix(s) }
        return english
    }

    private fun suffix(s: String) = when {
        s.endsWith(". Checkmate") -> L(". Checkmate")
        s.endsWith(", with check") -> L(", with check")
        else -> ""
    }
}
