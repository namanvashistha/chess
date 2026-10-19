package com.namanvashistha.calmchess

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.interaction.MutableInteractionSource
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
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.outlined.People
import androidx.compose.material.icons.outlined.School
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material.icons.automirrored.rounded.OpenInNew
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material.icons.outlined.Smartphone
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Lock
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.ColorMatrix
import androidx.compose.ui.graphics.Paint
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.drawIntoCanvas
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Locale
import kotlin.math.roundToInt

private const val SETUP = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"

/**
 * Home is a seat at the table. The board is the hero: your game as you left
 * it, or the pieces set up and waiting. Your opponent sits across it; swipe
 * the board to change who. One button sits below: Continue, or Begin.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(start: (Mode, Opponent) -> Unit, resume: () -> Unit, learn: () -> Unit, welcome: () -> Unit) {
    val p = LocalPalette.current
    val scope = rememberCoroutineScope()
    val saved = remember { SavedGame.load() }
    val recent = remember { Stats.recent }
    var seat by remember {
        mutableStateOf(
            saved?.takeIf { !it.passAndPlay }?.let { Opponent.entries.getOrNull(it.opponent) }
                ?: Opponent.entries.getOrNull(Prefs.p.getInt("opponent", 0))?.takeIf { it.isUnlocked } ?: Opponent.Easy
        )
    }
    var swiped by remember { mutableStateOf(Prefs.p.getBoolean("home.swiped", false)) }
    /** The board's sideways offset: the spring, in dp, plus the finger while it drags, in px. */
    val swipe = remember { Animatable(0f) }
    var drag by remember { mutableFloatStateOf(0f) }
    val density = androidx.compose.ui.platform.LocalDensity.current.density
    var settings by remember { mutableStateOf(false) }
    var history by remember { mutableStateOf(false) }
    val savedHere = saved?.takeIf { !it.passAndPlay && it.opponent == seat.ordinal }
    // The position to show: saves from before positions were kept are replayed from their moves.
    val savedPlacement = remember(savedHere) {
        savedHere?.let { g ->
            g.fen.ifEmpty {
                com.namanvashistha.calmchess.engine.chesscore.Chesscore.newGame().also { e -> g.moves.forEach { e.play(it) } }.fen()
            }.substringBefore(' ')
        } ?: SETUP
    }

    fun choose(o: Opponent) {
        if (o == seat) return
        seat = o
        swiped = true
        Prefs.p.edit().putBoolean("home.swiped", true).apply()
        Haptics.select()
        if (o.isUnlocked) Prefs.p.edit().putInt("opponent", o.ordinal).apply()
    }

    fun go() {
        Haptics.tap()
        if (savedHere != null) resume() else start(Mode.Engine, seat)
    }

    /** The dots' tap, played as a swipe: out to the left, the next one in from the right. */
    fun slideToNext() = scope.launch {
        val next = Opponent.entries[(seat.ordinal + 1) % Opponent.entries.size]
        swipe.animateTo(-48f, tween(140))
        choose(next)
        swipe.snapTo(48f)
        swipe.animateTo(0f, swiftSpring(0.38f, 0.8f))
    }

    // On the first few visits, until the first swipe, the board peeks toward the next opponent.
    LaunchedEffect(Unit) {
        val peeks = Prefs.p.getInt("home.peeks", 0)
        if (swiped || peeks >= 3) return@LaunchedEffect
        delay(900)
        if (swiped || swipe.value != 0f) return@LaunchedEffect
        swipe.animateTo(if (seat == Opponent.entries.last()) 26f else -26f, tween(320, easing = FastOutSlowInEasing))
        swipe.animateTo(0f, swiftSpring(0.5f, 0.55f))
        Prefs.p.edit().putInt("home.peeks", peeks + 1).apply()
    }

    AppBackground {
        Column(
            Modifier.fillMaxSize().safeDrawingPadding().verticalScroll(rememberScrollState())
                .padding(horizontal = 20.dp).padding(bottom = 24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Column(Modifier.widthIn(max = 560.dp).fillMaxWidth()) {
                // The greeting, with settings tucked beside it.
                Row(Modifier.padding(top = 8.dp).chrome(fromTop = true, delay = 100), verticalAlignment = Alignment.CenterVertically) {
                    Text(greeting(), style = Typeface.display(26f), color = p.ink, modifier = Modifier.weight(1f))
                    Box(Modifier.size(44.dp).clip(CircleShape).clickable { Haptics.tap(); settings = true }, contentAlignment = Alignment.Center) {
                        Icon(Icons.Outlined.Settings, "Settings", tint = p.mist, modifier = Modifier.size(24.dp))
                    }
                }

                // Across the table: who you'd be playing.
                Column(Modifier.padding(top = 24.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Row(Modifier.chrome(fromTop = true), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Box(Modifier.graphicsLayer { alpha = if (seat.isUnlocked) 1f else 0.5f }) { OpponentAvatar(seat, 42f) }
                        Row(verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(6.dp), modifier = Modifier.weight(1f)) {
                            Text("vs", style = Typeface.body(15f, FontWeight.Medium), color = p.mist, modifier = Modifier.padding(bottom = 2.dp))
                            Text(seat.title, style = Typeface.display(19f), color = p.ink)
                        }
                        // One dot per opponent; a tap moves on to the next, like a swipe.
                        Row(
                            Modifier.clip(RoundedCornerShape(50)).clickable(remember { MutableInteractionSource() }, null) { slideToNext() }
                                .padding(horizontal = 8.dp, vertical = 13.dp),
                            horizontalArrangement = Arrangement.spacedBy(5.dp),
                        ) {
                            for (o in Opponent.entries) {
                                val w by androidx.compose.animation.core.animateDpAsState(if (o == seat) 14.dp else 6.dp, label = "dot")
                                Box(Modifier.width(w).height(6.dp).background(if (o == seat) p.accent else p.mist.copy(alpha = 0.3f), RoundedCornerShape(50)))
                            }
                        }
                    }

                    Box(
                        Modifier.fillMaxWidth()
                            .offset { IntOffset((swipe.value.dp.toPx() + drag).roundToInt(), 0) }
                            .draggable(
                                rememberDraggableState { delta -> drag += delta * 0.35f },
                                Orientation.Horizontal,
                                onDragStopped = {
                                    val dx = drag / 0.35f / density
                                    val i = seat.ordinal
                                    val next = if (dx < -50) minOf(i + 1, Opponent.entries.lastIndex) else if (dx > 50) maxOf(i - 1, 0) else i
                                    choose(Opponent.entries[next])
                                    // Hand the finger's offset to the spring, which settles it back.
                                    swipe.snapTo(drag / density)
                                    drag = 0f
                                    swipe.animateTo(0f, swiftSpring(0.4f, 0.8f))
                                },
                            )
                            .clickable(remember { MutableInteractionSource() }, null) { if (seat.isUnlocked) go() },
                        contentAlignment = Alignment.Center,
                    ) {
                        StaticBoard(
                            savedPlacement,
                            Modifier.heroBoard()
                                .shadow(16.dp, RoundedCornerShape(6.dp), ambientColor = Color.Black.copy(alpha = 0.22f), spotColor = Color.Black.copy(alpha = 0.22f))
                                .clip(RoundedCornerShape(6.dp))
                                .saturation(if (seat.isUnlocked) 1f else 0.2f)
                                .graphicsLayer { alpha = if (seat.isUnlocked) 1f else 0.55f },
                        )
                        seat.previous?.takeIf { !seat.isUnlocked }?.let { prev -> LockedCard(prev) }
                    }

                    // Until the first swipe, a quiet word on how to change opponents.
                    AnimatedVisibility(!swiped && seat.isUnlocked, Modifier.align(Alignment.CenterHorizontally), enter = fadeIn(), exit = fadeOut()) {
                        Text("Swipe for another opponent", style = Typeface.body(12f), color = p.mist.copy(alpha = 0.8f), modifier = Modifier.chrome(delay = 200))
                    }
                }

                // The one button, and the quieter ways in.
                Column(Modifier.padding(top = 24.dp).chrome(delay = 180), verticalArrangement = Arrangement.spacedBy(18.dp)) {
                    if (seat.isUnlocked) PrimaryButton(if (savedHere == null) "Begin" else "Continue") { go() }
                    Row(Modifier.fillMaxWidth()) {
                        if (savedHere != null) {
                            QuietLink(Icons.Rounded.Add, "New game", Modifier.weight(1f)) { Haptics.tap(); start(Mode.Engine, seat) }
                        }
                        FriendLink(Modifier.weight(1f)) { start(Mode.PassAndPlay, seat) }
                    }
                }
                if (saved?.passAndPlay == true) {
                    Row(Modifier.align(Alignment.CenterHorizontally).padding(top = 10.dp).clickable { Haptics.tap(); resume() },
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        Icon(Icons.Outlined.Smartphone, null, tint = p.mist, modifier = Modifier.size(16.dp))
                        Text("Continue your game on this phone", style = Typeface.body(14f, FontWeight.Medium), color = p.mist)
                    }
                }
                // Your games: a quiet line, not a card. One abandoned on the first move doesn't count.
                if (recent.any { it.moves >= 2 }) {
                    val source = remember { MutableInteractionSource() }
                    Row(
                        Modifier.align(Alignment.CenterHorizontally).padding(top = 40.dp).heightIn(min = 44.dp).chrome(delay = 260)
                            .pressable(source, 0.98f).clickable(source, null) { Haptics.tap(); history = true },
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        val trend = recent.accuracyTrend
                        if (trend.size >= 3) AccuracyTrend(trend, compact = true, modifier = Modifier.width(36.dp).height(14.dp))
                        Text("Your games", style = Typeface.body(14f, FontWeight.SemiBold), color = p.ink)
                        Icon(Icons.AutoMirrored.Rounded.KeyboardArrowRight, null, tint = p.mist, modifier = Modifier.size(18.dp))
                    }
                }
            }
        }
    }

    if (settings) {
        ModalBottomSheet({ settings = false }, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
            containerColor = p.surface, dragHandle = null) {
            SettingsSheet(
                learn = { settings = false; scope.launch { delay(300); learn() } },
                welcome = { settings = false; scope.launch { delay(400); welcome() } },
            )
        }
    }
    if (history) {
        ModalBottomSheet({ history = false }, containerColor = p.surface) {
            Column(Modifier.verticalScroll(rememberScrollState()).padding(horizontal = 24.dp).padding(bottom = 24.dp).navigationBarsPadding()) {
                RecentGames(recent, limit = 20)
            }
        }
    }
}

/** For a locked opponent: one line that counts down, and the wins so far as a bar. */
@Composable
private fun LockedCard(prev: Opponent) {
    val p = LocalPalette.current
    val left = maxOf(0, Opponent.WINS_TO_UNLOCK - Stats.wins(prev))
    Column(Modifier.glass(RoundedCornerShape(20.dp)).padding(horizontal = 20.dp, vertical = 16.dp),
        horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Icon(Icons.Rounded.Lock, null, tint = p.ink, modifier = Modifier.size(24.dp))
        Text(
            when (left) {
                Opponent.WINS_TO_UNLOCK -> "Beat the ${prev.title} $left times"
                1 -> "Beat the ${prev.title} once more"
                else -> "Beat the ${prev.title} $left more times"
            },
            style = Typeface.display(16f), color = p.ink,
        )
        Row(horizontalArrangement = Arrangement.spacedBy(3.dp)) {
            for (k in 0 until Opponent.WINS_TO_UNLOCK) {
                Box(Modifier.width(22.dp).height(4.dp).background(if (k < Opponent.WINS_TO_UNLOCK - left) p.accent else p.mist.copy(alpha = 0.3f), RoundedCornerShape(50)))
            }
        }
    }
}

@Composable
private fun QuietLink(icon: androidx.compose.ui.graphics.vector.ImageVector, text: String, modifier: Modifier, onClick: () -> Unit) {
    val p = LocalPalette.current
    val source = remember { MutableInteractionSource() }
    Row(modifier.heightIn(min = 48.dp).clickable(source, null, onClick = onClick).graphicsLayer { },
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
        Icon(icon, null, tint = p.ink, modifier = Modifier.size(20.dp))
        Spacer(Modifier.width(8.dp))
        Text(text, style = Typeface.body(15f, FontWeight.SemiBold), color = p.ink)
    }
}

/** Play a friend: on this phone, passing it between moves. */
@Composable
private fun FriendLink(modifier: Modifier, onPhone: () -> Unit) {
    val p = LocalPalette.current
    var open by remember { mutableStateOf(false) }
    Box(modifier) {
        QuietLink(Icons.Outlined.People, "Play a friend", Modifier.fillMaxWidth()) { Haptics.tap(); open = true }
        DropdownMenu(open, { open = false }, shape = RoundedCornerShape(16.dp), containerColor = p.surface) {
            DropdownMenuItem({ Text("Play on this phone", color = p.ink) }, { open = false; onPhone() },
                leadingIcon = { Icon(Icons.Outlined.Smartphone, null, tint = p.ink) })
        }
    }
}

private fun greeting() = when (Calendar.getInstance().get(Calendar.HOUR_OF_DAY)) {
    in 5..11 -> "Good morning"
    in 12..16 -> "Good afternoon"
    else -> "Good evening"
}

/** Desaturates whatever it's applied to, for a locked opponent's board. */
fun Modifier.saturation(amount: Float): Modifier = if (amount >= 1f) this else drawWithCache {
    val paint = Paint().apply { colorFilter = ColorFilter.colorMatrix(ColorMatrix().apply { setToSaturation(amount) }) }
    onDrawWithContent {
        drawIntoCanvas { c ->
            c.saveLayer(androidx.compose.ui.geometry.Rect(Offset.Zero, size), paint)
            drawContent()
            c.restore()
        }
    }
}

/** A board drawn from a FEN placement, without a game behind it. */
@Composable
fun StaticBoard(placement: String, modifier: Modifier = Modifier) {
    val p = LocalPalette.current
    val board = remember(placement) {
        buildList { for (ch in placement) if (ch != '/') { if (ch.isDigit()) repeat(ch.digitToInt()) { add('.') } else add(ch) } }
    }
    BoxWithConstraints(modifier.aspectRatio(1f)) {
        val q = maxWidth / 8
        Canvas(Modifier.fillMaxSize()) {
            val s = size.width / 8
            for (sq in 0 until 64) drawRect(if ((sq / 8 + sq % 8) % 2 == 1) p.squareDark else p.squareLight,
                Offset((sq % 8) * s, (sq / 8) * s), Size(s, s))
        }
        for (i in 0 until minOf(64, board.size)) {
            if (board[i] != '.') GlassPiece(board[i], Modifier.size(q).offset(q * (i % 8), q * (i / 8)))
        }
    }
}

/** Accuracy over the last games as one soft line, the latest marked: the same hand as the game's arc. */
@Composable
fun AccuracyTrend(values: List<Int>, compact: Boolean = false, modifier: Modifier = Modifier) {
    val p = LocalPalette.current
    Canvas(modifier) {
        val w = size.width
        val h = size.height
        val lo = maxOf(0, values.min() - 8).toFloat()
        val hi = minOf(100, values.max() + 8).toFloat()
        val pts = values.indices.map { Offset(w * it / maxOf(1, values.size - 1), h * (1 - (values[it] - lo) / maxOf(1f, hi - lo))) }
        val line = Path().apply {
            moveTo(pts[0].x, pts[0].y)
            for (i in 1 until pts.size) {
                val a = pts[i - 1]; val b = pts[i]
                quadraticTo(a.x, a.y, (a.x + b.x) / 2, (a.y + b.y) / 2)
                if (i == pts.lastIndex) quadraticTo(b.x, b.y, b.x, b.y)
            }
        }
        drawPath(line, p.accent, style = Stroke((if (compact) 1.5f else 1.75f).dp.toPx(), cap = StrokeCap.Round, join = StrokeJoin.Round))
        if (!compact) {
            drawCircle(p.surface, 5.dp.toPx(), pts.last())
            drawCircle(p.accent, 3.dp.toPx(), pts.last())
        }
    }
}

/** Your games: two figures, the accuracy line, then each game by day. */
@Composable
private fun RecentGames(records: List<GameRecord>, limit: Int) {
    val p = LocalPalette.current
    val shown = records.filter { it.moves >= 2 }.take(limit)
    val trend = records.accuracyTrend
    Column(Modifier.fillMaxWidth()) {
        Text("Your games", style = Typeface.display(24f), color = p.ink)
        if (trend.size >= 2) {
            Row(Modifier.fillMaxWidth().padding(top = 16.dp), verticalAlignment = Alignment.Bottom) {
                Figure("${trend.sum() / trend.size}%", "Accuracy · last ${trend.size} games", Modifier.weight(1f))
                if (Stats.greatMoves > 0) Figure("${Stats.greatMoves}", "Great moves", trailing = true)
            }
        }
        if (trend.size >= 3) AccuracyTrend(trend, modifier = Modifier.fillMaxWidth().height(44.dp).padding(top = 16.dp))
        if (shown.isEmpty()) Text("Your finished games will show up here, with how accurately you played.",
            style = Typeface.body(14f), color = p.mist, modifier = Modifier.padding(top = 16.dp))
        var lastDay: String? = null
        for (r in shown) {
            val day = dayLabel(r.date)
            if (day != lastDay) {
                Text(day, style = Typeface.body(12f, FontWeight.SemiBold), color = p.mist, modifier = Modifier.padding(top = 24.dp, bottom = 4.dp))
                lastDay = day
            }
            GameRow(r)
        }
    }
}

@Composable
private fun Figure(value: String, label: String, modifier: Modifier = Modifier, trailing: Boolean = false) {
    val p = LocalPalette.current
    Column(modifier, horizontalAlignment = if (trailing) Alignment.End else Alignment.Start, verticalArrangement = Arrangement.spacedBy(2.dp)) {
        Text(value, style = Typeface.display(28f, FontWeight.Bold), color = p.ink)
        Text(label, style = Typeface.body(12.5f), color = p.mist)
    }
}

@Composable
private fun GameRow(r: GameRecord) {
    val p = LocalPalette.current
    val twoPlayers = r.opponent == "Pass & play"
    Row(Modifier.fillMaxWidth().padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(14.dp)) {
        ResultMark(if (twoPlayers) null else r.outcome)
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(if (twoPlayers) "Pass & play" else r.opponentName, style = Typeface.body(16f, FontWeight.SemiBold), color = p.ink)
            val moves = if (r.moves == 1) "1 move" else "${r.moves} moves"
            val result = mapOf("won" to "Won", "lost" to "Lost", "draw" to "Draw")[r.outcome]
            Text(if (twoPlayers || result == null) moves else "$result · $moves", style = Typeface.body(13f), color = p.mist)
        }
        if (!twoPlayers) r.accuracy?.let { Text("$it%", style = Typeface.display(17f), color = p.ink) }
    }
}

/** How a game ended: filled green for a win, a ring for a loss, a quiet dot for a draw, a dashed ring for two players. */
@Composable
private fun ResultMark(outcome: String?) {
    val p = LocalPalette.current
    Canvas(Modifier.size(10.dp)) {
        val r = size.width / 2
        val w = 1.5.dp.toPx()
        when (outcome) {
            "won" -> drawCircle(p.great)
            "lost" -> drawCircle(p.mist.copy(alpha = 0.7f), r - w / 2, style = Stroke(w))
            "draw" -> drawCircle(p.mist.copy(alpha = 0.5f))
            else -> drawCircle(p.mist.copy(alpha = 0.35f), r - w / 2, style = Stroke(w,
                pathEffect = androidx.compose.ui.graphics.PathEffect.dashPathEffect(floatArrayOf(2.dp.toPx(), 2.dp.toPx()))))
        }
    }
}

private fun dayLabel(ms: Long): String {
    val day = Calendar.getInstance().apply { timeInMillis = ms }
    val today = Calendar.getInstance()
    fun same(a: Calendar, b: Calendar) = a.get(Calendar.YEAR) == b.get(Calendar.YEAR) && a.get(Calendar.DAY_OF_YEAR) == b.get(Calendar.DAY_OF_YEAR)
    if (same(day, today)) return "Today"
    if (same(day, (today.clone() as Calendar).apply { add(Calendar.DAY_OF_YEAR, -1) })) return "Yesterday"
    return SimpleDateFormat("EEEE d MMM", Locale.getDefault()).format(day.time)
}

/** Settings: sound and haptics, and a word on appearance. */
@Composable
private fun SettingsSheet(learn: () -> Unit, welcome: () -> Unit) {
    val p = LocalPalette.current
    var sound by remember { mutableStateOf(Prefs.p.getBoolean("sound", true)) }
    var haptics by remember { mutableStateOf(Prefs.p.getBoolean("haptics", true)) }
    Column(Modifier.fillMaxWidth().padding(24.dp).navigationBarsPadding(), verticalArrangement = Arrangement.spacedBy(18.dp)) {
        Text("Settings", style = Typeface.display(22f), color = p.ink)
        Toggle("Sound", sound) { sound = it; Prefs.p.edit().putBoolean("sound", it).apply() }
        Toggle("Haptics", haptics) { haptics = it; Prefs.p.edit().putBoolean("haptics", it).apply(); if (it) Haptics.tap() }
        SettingsRow(Icons.Outlined.School, "Learn the moves", onClick = learn)
        SettingsRow(Icons.Rounded.AutoAwesome, "How it works", onClick = welcome)
        Text("Light and dark follow your phone's appearance.", style = Typeface.body(13f), color = p.mist)
    }
}

@Composable
private fun SettingsRow(icon: androidx.compose.ui.graphics.vector.ImageVector, label: String, trailing: String? = null,
                        external: Boolean = false, onClick: () -> Unit) {
    val p = LocalPalette.current
    Row(Modifier.fillMaxWidth().heightIn(min = 32.dp).clickable(onClick = onClick), verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Icon(icon, null, tint = p.accent, modifier = Modifier.size(22.dp))
        Text(label, style = Typeface.body(16f, FontWeight.Medium), color = p.ink, modifier = Modifier.weight(1f))
        if (trailing != null) Text(trailing, style = Typeface.body(16f), color = p.mist)
        Icon(if (external) Icons.AutoMirrored.Rounded.OpenInNew else Icons.AutoMirrored.Rounded.KeyboardArrowRight, null,
            tint = p.mist, modifier = Modifier.size(if (external) 16.dp else 20.dp))
    }
}

@Composable
private fun Toggle(label: String, on: Boolean, set: (Boolean) -> Unit) {
    val p = LocalPalette.current
    Row(Modifier.fillMaxWidth().clickable { set(!on) }, verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = Typeface.body(16f, FontWeight.Medium), color = p.ink, modifier = Modifier.weight(1f))
        Switch(on, set, colors = SwitchDefaults.colors(checkedTrackColor = p.accent, checkedThumbColor = Color.White,
            uncheckedTrackColor = p.mist.copy(alpha = 0.2f), uncheckedBorderColor = Color.Transparent, uncheckedThumbColor = Color.White))
    }
}
