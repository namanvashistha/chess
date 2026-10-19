package com.namanvashistha.calmchess

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Redo
import androidx.compose.material.icons.automirrored.rounded.Undo
import androidx.compose.material.icons.outlined.Flag
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.Share
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material.icons.rounded.ChevronLeft
import androidx.compose.material.icons.rounded.LockOpen
import androidx.compose.material.icons.rounded.MoreHoriz
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.PathMeasure
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.zIndex
import kotlin.math.abs
import kotlin.random.Random

/** The game: who you're playing, the board, the coach and the dock. */
@Composable
fun GameScreen(model: GameModel, back: () -> Unit) {
    val p = LocalPalette.current
    var confirm by remember { mutableStateOf<Confirm?>(null) }
    BackHandler { if (confirm != null) confirm = null else back() }
    AppBackground {
        BoxWithConstraints(Modifier.fillMaxSize().safeDrawingPadding().padding(horizontal = 14.dp).padding(bottom = 8.dp)) {
            val width = minOf(760.dp, maxHeight * 0.6f).coerceAtLeast(maxWidth.coerceAtMost(maxHeight * 0.6f))
            Column(
                Modifier.widthIn(max = width).fillMaxSize().align(Alignment.TopCenter),
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Box(Modifier.chrome(fromTop = true, delay = 120)) { TopBar(model, back) }
                Box(Modifier.chrome(fromTop = true, delay = 160)) { CapturedRow(model.captured("b"), model.materialLead("b")) }
                BoardView(model, Modifier.heroBoard().fillMaxWidth())
                Box(Modifier.chrome(delay = 180)) { CapturedRow(model.captured("w"), model.materialLead("w")) }
                Box(Modifier.chrome(delay = 220)) { CoachArea(model) }
                Spacer(Modifier.weight(1f))
                Box(Modifier.chrome(delay = 280)) { Dock(model) { confirm = it } }
            }
        }
        model.finish?.let { f ->
            FinishCard(
                f, king = 'K', share = model.finishMoment,
                again = { model.start(model.mode, model.opponent) },
                tryNext = f.unlocked?.let { next -> { Prefs.p.edit().putInt("opponent", next.ordinal).apply(); model.start(Mode.Engine, next) } },
                home = back,
            )
        }
        confirm?.let { c ->
            ConfirmSheet(c, onCancel = { confirm = null }) {
                confirm = null
                Haptics.tap(0.7f)
                if (c == Confirm.NewGame) model.start(model.mode, model.opponent) else model.resign()
            }
        }
    }
}

enum class Confirm(private val t: String, private val a: String, private val m: String) {
    NewGame("Start a new game?", L("New game"), "This game will be lost."),
    Resign("Resign this game?", L("Resign"), "It counts as a loss in your recent games.");

    val title get() = L(t)
    val action get() = L(a)
    val message get() = L(m)
}

@Composable
private fun TopBar(model: GameModel, back: () -> Unit) {
    val p = LocalPalette.current
    Row(Modifier.padding(top = 4.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        GlassIconButton(Icons.Rounded.ChevronLeft, L("Home"), circle = true) { Haptics.tap(); back() }
        if (model.mode == Mode.Engine) OpponentAvatar(model.opponent, 36f)
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(if (model.mode == Mode.Engine) model.opponent.title else L("Pass & play"), style = Typeface.display(17f, FontWeight.Bold), color = p.ink)
            AnimatedVisibility(model.thinking, enter = fadeIn(tween(200)), exit = fadeOut(tween(200))) { ThinkingDots() }
        }
    }
}

@Composable
private fun ThinkingDots() {
    val p = LocalPalette.current
    val t = rememberInfiniteTransition(label = "dots")
    Row(horizontalArrangement = Arrangement.spacedBy(3.dp)) {
        for (i in 0..2) {
            val a by t.animateFloat(0.2f, 1f, infiniteRepeatable(tween(500, delayMillis = i * 150, easing = FastOutSlowInEasing), RepeatMode.Reverse), label = "dot$i")
            Box(Modifier.size(4.dp).graphicsLayer { alpha = a }.background(p.accent, CircleShape))
        }
    }
}

/** A small glass button with one icon: a circle for Home, a capsule in the dock. */
@Composable
fun GlassIconButton(icon: androidx.compose.ui.graphics.vector.ImageVector, label: String, circle: Boolean = false,
                    enabled: Boolean = true, modifier: Modifier = Modifier, onClick: () -> Unit) {
    val p = LocalPalette.current
    val source = remember { MutableInteractionSource() }
    Box(
        modifier
            .then(if (circle) Modifier.size(44.dp) else Modifier.height(56.dp).width(70.dp))
            .pressable(source)
            .glass(if (circle) CircleShape else RoundedCornerShape(50))
            .clickable(source, null, enabled = enabled, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(icon, label, tint = if (enabled) p.ink else p.mist.copy(alpha = 0.5f), modifier = Modifier.size(if (circle) 24.dp else 22.dp))
    }
}

/** Each opponent is a stone: a pale river pebble, a cut flint, an obsidian star. */
@Composable
fun OpponentAvatar(opponent: Opponent, size: Float) {
    val dark = LocalPalette.current === Palette.Dark
    val tint = listOf(
        if (dark) Color(0xFFA9BDB3) else Color(0xFF8FA39A),
        if (dark) Color(0xFFC9AE85) else Color(0xFFA2855C),
        if (dark) Color(0xFFA9B2C3) else Color(0xFF3A3F4A),
    )[opponent.ordinal]
    Box(Modifier.size(size.dp).glass(CircleShape, tint.copy(alpha = 0.18f)), contentAlignment = Alignment.Center) {
        Canvas(Modifier.size((size * 0.5f).dp)) {
            val brush = Brush.linearGradient(listOf(tint.copy(alpha = 0.75f), tint), Offset.Zero, Offset(this.size.width, this.size.height))
            val w = this.size.width
            when (opponent) {
                Opponent.Easy -> drawCircle(brush, w * 0.42f)
                Opponent.Medium -> drawPath(Path().apply {
                    moveTo(w / 2, w * 0.04f); lineTo(w * 0.96f, w / 2); lineTo(w / 2, w * 0.96f); lineTo(w * 0.04f, w / 2); close()
                }, brush)
                Opponent.Hard -> drawPath(Path().apply {
                    for (i in 0 until 10) {
                        val r = if (i % 2 == 0) w * 0.5f else w * 0.21f
                        val a = Math.PI * (i / 5.0) - Math.PI / 2
                        val x = w / 2 + (r * kotlin.math.cos(a)).toFloat()
                        val y = w * 0.53f + (r * kotlin.math.sin(a)).toFloat()
                        if (i == 0) moveTo(x, y) else lineTo(x, y)
                    }
                    close()
                }, brush)
            }
        }
    }
}

@Composable
private fun CapturedRow(pieces: List<Char>, lead: Int) {
    val p = LocalPalette.current
    Row(Modifier.fillMaxWidth().height(22.dp).padding(horizontal = 6.dp), verticalAlignment = Alignment.CenterVertically) {
        Row(horizontalArrangement = Arrangement.spacedBy((-5).dp)) {
            for (c in pieces) GlassPiece(c, Modifier.size(20.dp))
        }
        if (lead > 0) Text(L("+%lld", lead), style = Typeface.body(12f, FontWeight.Bold), color = p.mist, modifier = Modifier.padding(start = 12.dp))
    }
}

/** The coach card, with its swipe to step through past moves, and a way back to the game while looking back. */
@Composable
private fun CoachArea(model: GameModel) {
    val p = LocalPalette.current
    Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(6.dp)) {
        var dx by remember { mutableFloatStateOf(0f) }
        Box(
            Modifier.padding(top = 4.dp).pointerInput(model) {
                detectHorizontalDragGestures(
                    onDragStart = { dx = 0f },
                    onDragEnd = { if (abs(dx) > 40.dp.toPx()) model.step(if (dx > 0) -1 else 1) },
                ) { _, amount -> dx += amount }
            }
        ) { CoachView(model.coach, model.opponent) }
        AnimatedVisibility(model.reviewing != null, enter = fadeIn(), exit = fadeOut()) {
            val source = remember { MutableInteractionSource() }
            Row(
                Modifier.pressable(source).glass(RoundedCornerShape(50))
                    .clickable(source, null) { Haptics.tap(); model.endReview() }
                    .padding(horizontal = 14.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Icon(Icons.AutoMirrored.Rounded.Redo, null, tint = p.ink, modifier = Modifier.size(16.dp))
                Text(L("Back to game"), style = Typeface.body(13.5f, FontWeight.SemiBold), color = p.ink)
            }
        }
    }
}

/** The coach's card: a verdict badge, a title in the verdict's colour, and one sentence. */
@Composable
fun CoachView(card: CoachCard?, opponent: Opponent, compact: Boolean = false) {
    val p = LocalPalette.current
    Box(Modifier.fillMaxWidth().heightIn(min = 74.dp)) {
        AnimatedContent(
            card,
            transitionSpec = {
                (fadeIn(tween(220)) + scaleIn(swiftSpring(0.45f, 0.7f), initialScale = 0.94f,
                    transformOrigin = androidx.compose.ui.graphics.TransformOrigin(0.5f, 1f))) togetherWith fadeOut(tween(150))
            },
            contentKey = { it?.id },
            label = "coach",
        ) { c ->
            if (c == null) return@AnimatedContent
            Row(
                Modifier.fillMaxWidth().glass(RoundedCornerShape(20.dp)).padding(12.dp),
                verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Badge(c, opponent)
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(c.title, style = Typeface.display(17f), color = when (c.kind) {
                        CoachCard.Kind.Good, CoachCard.Kind.Coach, CoachCard.Kind.Opponent -> p.ink
                        else -> verdictColor(p, c.kind)
                    })
                    if (c.text.isNotEmpty()) Text(c.text, style = Typeface.body(if (compact) 12f else 13.5f), color = p.mist)
                }
            }
        }
    }
}

@Composable
private fun Badge(card: CoachCard, opponent: Opponent) {
    val p = LocalPalette.current
    if (card.kind == CoachCard.Kind.Opponent || card.kind == CoachCard.Kind.Warning) {
        Box(Modifier.size(46.dp)) {
            OpponentAvatar(opponent, 46f)
            Text(card.badge, style = Typeface.body(11f, FontWeight.Bold), color = p.ink, maxLines = 1,
                modifier = Modifier.align(Alignment.BottomEnd).offset(6.dp, 4.dp)
                    .background(if (card.kind == CoachCard.Kind.Warning) p.mistakeFill else p.goodFill, RoundedCornerShape(50))
                    .padding(horizontal = 5.dp, vertical = 2.dp))
        }
        return
    }
    val shape = RoundedCornerShape(14.dp)
    val fill = when (card.kind) {
        CoachCard.Kind.Great -> Modifier.background(p.greatFill, shape)
        CoachCard.Kind.Inaccuracy -> Modifier.background(p.inaccuracyFill, shape)
        CoachCard.Kind.Mistake -> Modifier.background(p.mistakeFill, shape)
        CoachCard.Kind.Blunder -> Modifier.background(p.blunderFill, shape)
        CoachCard.Kind.Coach -> Modifier.background(Brush.verticalGradient(listOf(p.accentTop, p.accent)), shape)
        else -> Modifier.background(p.goodFill, shape)
    }
    val color = if (card.kind == CoachCard.Kind.Coach) p.onAccent else p.ink
    Box(Modifier.size(46.dp).then(fill), contentAlignment = Alignment.Center) {
        // Glyphs the rounded face lacks are drawn as icons, at the same size as the rest.
        val icon = verdictIcon(card.kind) ?: if (card.badge == "↶") Icons.AutoMirrored.Rounded.Undo else null
        if (icon != null) Icon(icon, null, tint = color, modifier = Modifier.size(22.dp))
        else Text(card.badge, style = Typeface.display(20f), color = color)
    }
}

/** Undo, the game menu, and the hint: the controls under the coach. */
@Composable
private fun Dock(model: GameModel, ask: (Confirm) -> Unit) {
    val p = LocalPalette.current
    var menu by remember { mutableStateOf(false) }
    val canHint = model.findingHint || (model.playerToMove && (model.hintsLeft > 0 || model.hintIsFree))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
        GlassIconButton(Icons.AutoMirrored.Rounded.Undo, L("Undo"), enabled = model.moves.isNotEmpty()) { Haptics.tap(); model.undo() }
        Box {
            GlassIconButton(Icons.Rounded.MoreHoriz, L("Game options"), enabled = !model.busy) { Haptics.tap(); menu = true }
            DropdownMenu(menu, { menu = false }, shape = RoundedCornerShape(16.dp), containerColor = p.surface) {
                DropdownMenuItem({ Text(L("New game"), color = p.ink) }, { menu = false; ask(Confirm.NewGame) },
                    leadingIcon = { Icon(Icons.Rounded.Add, null, tint = p.ink) })
                if (model.mode != Mode.PassAndPlay && !model.isOver) {
                    DropdownMenuItem({ Text(L("Resign"), color = p.danger) }, { menu = false; ask(Confirm.Resign) },
                        leadingIcon = { Icon(Icons.Outlined.Flag, null, tint = p.danger) })
                }
            }
        }
        // The hint: evergreen while it can help, quiet glass when it can't.
        val source = remember { MutableInteractionSource() }
        val shape = RoundedCornerShape(50)
        val free = model.hintIsFree && !model.hintRevealed
        Row(
            Modifier.weight(1f).height(56.dp).pressable(source)
                .then(if (canHint) Modifier.shadow(10.dp, shape, ambientColor = p.accent.copy(alpha = 0.3f), spotColor = p.accent.copy(alpha = 0.3f))
                    .background(Brush.verticalGradient(listOf(p.accentTop, p.accent)), shape) else Modifier.glass(shape))
                .clickable(source, null, enabled = canHint) { if (!model.findingHint) { Haptics.tap(); model.hint() } },
            horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
        ) {
            val color = if (canHint) p.onAccent else p.mist
            Icon(Icons.Rounded.AutoAwesome, null, tint = color, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(6.dp))
            Text(if (free) L("Show the move") else if (model.hintsLeft > 0 || model.findingHint) L("Hint") else L("No hints"),
                style = Typeface.display(14f, FontWeight.Bold), color = color, maxLines = 1)
            if (model.hintsLeft > 0 && !free && !model.findingHint) {
                Spacer(Modifier.width(6.dp))
                Text("${model.hintsLeft}", style = Typeface.body(12f, FontWeight.Bold), color = color,
                    modifier = Modifier.background(if (canHint) Color.White.copy(alpha = 0.25f) else p.mist.copy(alpha = 0.15f), RoundedCornerShape(50))
                        .padding(horizontal = 6.dp, vertical = 1.dp))
            }
        }
    }
}

/** A question before something that can't be undone, from the bottom, like iPhone's action sheet. */
@Composable
private fun ConfirmSheet(c: Confirm, onCancel: () -> Unit, onConfirm: () -> Unit) {
    val p = LocalPalette.current
    var shown by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) { shown = true }
    Box(Modifier.fillMaxSize().zIndex(50f)) {
        AnimatedVisibility(shown, enter = fadeIn(), exit = fadeOut()) {
            Box(Modifier.fillMaxSize().background(Color.Black.copy(alpha = 0.3f)).clickable(remember { MutableInteractionSource() }, null, onClick = onCancel))
        }
        AnimatedVisibility(shown, Modifier.align(Alignment.BottomCenter), enter = slideInVertically { it } + fadeIn(), exit = slideOutVertically { it } + fadeOut()) {
            Column(Modifier.navigationBarsPadding().padding(10.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(p.surface)) {
                    Column(Modifier.fillMaxWidth().padding(16.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(c.title, style = Typeface.body(13f, FontWeight.SemiBold), color = p.mist)
                        Text(c.message, style = Typeface.body(13f), color = p.mist)
                    }
                    Box(Modifier.fillMaxWidth().height(1.dp).background(p.hairline))
                    Box(Modifier.fillMaxWidth().clickable(onClick = onConfirm).padding(vertical = 16.dp), contentAlignment = Alignment.Center) {
                        Text(c.action, style = Typeface.body(19f), color = p.danger)
                    }
                }
                Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(p.surface).clickable(onClick = onCancel).padding(vertical = 16.dp),
                    contentAlignment = Alignment.Center) {
                    Text(L("Cancel"), style = Typeface.body(19f, FontWeight.SemiBold), color = p.accent)
                }
            }
        }
    }
}

/** The end of a game, in one calm column: what happened, the arc, the numbers, what's next, and the way on. */
@Composable
private fun FinishCard(finish: Finish, king: Char, share: ShareMoment?, again: () -> Unit, tryNext: (() -> Unit)?, home: () -> Unit) {
    val p = LocalPalette.current
    val enter = remember { Animatable(0f) }
    LaunchedEffect(finish) { enter.snapTo(0f); enter.animateTo(1f, swiftSpring(0.5f, 0.75f)) }
    Box(
        Modifier.fillMaxSize().zIndex(40f).graphicsLayer { alpha = enter.value.coerceIn(0f, 1f) }
            .background(Brush.verticalGradient(listOf(p.backgroundTop.copy(alpha = 0.88f), p.backgroundBottom.copy(alpha = 0.96f))))
            .clickable(remember { MutableInteractionSource() }, null) {},
        contentAlignment = Alignment.Center,
    ) {
        val tint = emblemTint(p, finish.outcome)
        val shape = RoundedCornerShape(30.dp)
        Box(Modifier.padding(16.dp).widthIn(max = 440.dp).graphicsLayer { val s = 0.94f + 0.06f * enter.value; scaleX = s; scaleY = s }) {
            Column(
                Modifier.fillMaxWidth()
                    .shadow(30.dp, shape, ambientColor = Color.Black.copy(alpha = 0.28f), spotColor = Color.Black.copy(alpha = 0.28f))
                    .background(p.surface, shape)
                    .drawBehind { drawRect(Brush.radialGradient(listOf(tint.copy(alpha = 0.16f), Color.Transparent), Offset.Zero, 300.dp.toPx())) }
                    .padding(24.dp),
            ) {
                // What happened, in a line.
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                    FinishEmblem(finish.outcome, king)
                    Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                        Text(finish.title, style = Typeface.display(24f), color = p.ink, maxLines = 2)
                        Text(if (finish.firstWin) L("Your first win · %@", finish.text) else finish.text,
                            style = Typeface.body(14f), color = p.mist, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    }
                }
                // How it went: the arc of the game, and what it says.
                if (finish.arc.isNotEmpty()) {
                    Column(Modifier.padding(top = 16.dp).fillMaxWidth().background(p.statFill, RoundedCornerShape(16.dp)).padding(16.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Text(L("Your chances"), style = Typeface.body(12f, FontWeight.SemiBold), color = p.mist)
                        GameArc(finish.arc, finish.best?.let { it.ply + 1 }, finish.turnPly?.let { it + 1 }, Modifier.fillMaxWidth().height(78.dp))
                        Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                            finish.best?.let { b -> Legend(p.great, L("Best · move %lld", b.moveNumber) + if (b.san.isEmpty()) "" else " · ${b.san}") }
                            finish.turnPly?.let { Legend(p.blunder, L("Turned · move %lld", it / 2 + 1)) }
                        }
                    }
                    finish.insight?.let { Text(it, style = Typeface.body(15f, FontWeight.SemiBold), color = p.ink, modifier = Modifier.padding(top = 16.dp)) }
                }
                // The numbers, as tiles you can read at a glance.
                val stats = buildList {
                    val s = finish.sides
                    if (s != null) { if (s.first + s.second > 0) { add("${s.first}" to L("White")); add("${s.second}" to L("Black")) } }
                    else {
                        finish.accuracy?.let { add(L("%lld%%", it) to L("Accuracy")) }
                        add("${finish.great}" to L("Great"))
                        if (finish.plies > 0) add("${(finish.plies + 1) / 2}" to L("Moves"))
                    }
                }
                if (stats.isNotEmpty()) {
                    Row(Modifier.padding(top = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        for ((value, label) in stats) {
                            Column(Modifier.weight(1f).background(p.statFill, RoundedCornerShape(14.dp)).padding(vertical = 10.dp),
                                horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(1.dp)) {
                                Text(value, style = Typeface.display(20f, FontWeight.Bold), color = p.ink)
                                Text(label, style = Typeface.body(11.5f, FontWeight.Medium), color = p.mist)
                            }
                        }
                    }
                }
                // What comes next.
                val next = finish.unlocked
                val toward = finish.towardNext
                if (next != null) {
                    Row(Modifier.fillMaxWidth().padding(top = 16.dp), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Rounded.LockOpen, null, tint = p.great, modifier = Modifier.size(16.dp))
                        Spacer(Modifier.width(6.dp))
                        Text(L("The %@ is open", next.title), style = Typeface.body(13.5f, FontWeight.SemiBold), color = p.great)
                    }
                } else if (toward != null) {
                    Row(Modifier.fillMaxWidth().padding(top = 16.dp), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
                        Row(horizontalArrangement = Arrangement.spacedBy(3.dp)) {
                            for (k in 0 until Opponent.WINS_TO_UNLOCK) Box(Modifier.size(6.dp).background(if (k < toward.second) p.accent else p.mist.copy(alpha = 0.3f), CircleShape))
                        }
                        Spacer(Modifier.width(8.dp))
                        val left = Opponent.WINS_TO_UNLOCK - toward.second
                        Text(if (left == 1) L("1 more win to face the %@", toward.first.title) else L("%lld more wins to face the %@", left, toward.first.title),
                            style = Typeface.body(13.5f, FontWeight.Medium), color = p.mist)
                    }
                }
                // The way on: one clear next step, then Home, quiet.
                Column(Modifier.padding(top = 24.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (next != null && tryNext != null) PrimaryButton(L("Play the %@", next.title), onClick = tryNext)
                    else PrimaryButton(L("Play again"), onClick = again)
                    // Home and Share beside each other, quiet.
                    val context = androidx.compose.ui.platform.LocalContext.current
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        QuietButton(Icons.Outlined.Home, L("Home"), Modifier.weight(1f)) { Haptics.tap(); home() }
                        if (share != null) QuietButton(Icons.Outlined.Share, L("Share"), Modifier.weight(1f)) { Haptics.tap(); ShareCard.share(context, share) }
                    }
                }
            }
            if (finish.outcome == Finish.Outcome.Won) Celebration(Modifier.align(Alignment.TopStart).offset(54.dp, 54.dp))
        }
    }
}

/** A secondary action: plain words that dim under the finger, no frame. */
@Composable
private fun QuietButton(icon: androidx.compose.ui.graphics.vector.ImageVector, text: String, modifier: Modifier, onClick: () -> Unit) {
    val p = LocalPalette.current
    val source = remember { MutableInteractionSource() }
    val pressed by source.collectIsPressedAsState()
    Row(modifier.height(48.dp).graphicsLayer { alpha = if (pressed) 0.5f else 1f }.clickable(source, null, onClick = onClick),
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
        Icon(icon, null, tint = p.ink, modifier = Modifier.size(20.dp))
        Spacer(Modifier.width(8.dp))
        Text(text, style = Typeface.body(15f, FontWeight.SemiBold), color = p.ink)
    }
}

@Composable
private fun Legend(color: Color, text: String) {
    val p = LocalPalette.current
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        Box(Modifier.size(7.dp).background(color, CircleShape))
        Text(text, style = Typeface.body(12f, FontWeight.Medium), color = p.mist, maxLines = 1)
    }
}

fun emblemTint(p: Palette, o: Finish.Outcome) = when (o) {
    Finish.Outcome.Won -> p.lastMove
    Finish.Outcome.Lost -> p.squareLight
    Finish.Outcome.Draw -> p.squareDark
}

/** The outcome at a glance: your king standing in honey, tipped over for a loss, beside its rival for a draw. */
@Composable
private fun FinishEmblem(outcome: Finish.Outcome, king: Char) {
    val p = LocalPalette.current
    val tint = emblemTint(p, outcome)
    Box(
        Modifier.size(60.dp)
            .shadow(12.dp, CircleShape, ambientColor = tint.copy(alpha = 0.4f), spotColor = tint.copy(alpha = 0.4f))
            .background(Brush.linearGradient(listOf(lerp(tint, Color.White, 0.25f), tint)), CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        Canvas(Modifier.fillMaxSize()) { drawCircle(Color.White.copy(alpha = 0.3f), style = Stroke(1.dp.toPx())) }
        when (outcome) {
            Finish.Outcome.Won -> GlassPiece(king, Modifier.size(40.dp).offset(y = (-1).dp))
            Finish.Outcome.Lost -> GlassPiece(king, Modifier.size(40.dp).offset(10.dp, (-3).dp).graphicsLayer {
                rotationZ = -55f; transformOrigin = androidx.compose.ui.graphics.TransformOrigin(0.5f, 0.86f)
            }, grounded = false)
            Finish.Outcome.Draw -> Row(Modifier.offset(y = (-1).dp), horizontalArrangement = Arrangement.spacedBy((-13).dp)) {
                GlassPiece(king, Modifier.size(34.dp))
                GlassPiece(if (king == 'K') 'k' else 'K', Modifier.size(34.dp))
            }
        }
    }
}

/** The game as one line: green above even, coral below, the best move and the turning point marked. It draws itself in once. */
@Composable
private fun GameArc(arc: List<Double>, best: Int?, turn: Int?, modifier: Modifier) {
    val p = LocalPalette.current
    val drawn = remember { Animatable(0f) }
    LaunchedEffect(Unit) { kotlinx.coroutines.delay(250); drawn.animateTo(1f, tween(1100, easing = FastOutSlowInEasing)) }
    Canvas(modifier) {
        val w = size.width
        val h = size.height
        val pts = arc.indices.map { Offset(w * it / maxOf(1, arc.size - 1), h * (1 - arc[it].toFloat())) }
        val line = Path().apply {
            moveTo(pts[0].x, pts[0].y)
            for (i in 1 until pts.size) {
                val a = pts[i - 1]; val b = pts[i]
                quadraticTo(a.x, a.y, (a.x + b.x) / 2, (a.y + b.y) / 2)
                if (i == pts.lastIndex) quadraticTo(b.x, b.y, b.x, b.y)
            }
        }
        val area = Path().apply { addPath(line); lineTo(w, h / 2); lineTo(0f, h / 2); close() }
        drawLine(p.mist.copy(alpha = 0.35f), Offset(0f, h / 2), Offset(w, h / 2), 1.dp.toPx(),
            pathEffect = PathEffect.dashPathEffect(floatArrayOf(3.dp.toPx(), 4.dp.toPx())))
        clipRect(bottom = h / 2) { drawPath(area, p.great.copy(alpha = 0.24f)) }
        clipRect(top = h / 2) { drawPath(area, p.blunder.copy(alpha = 0.24f * drawn.value)) }
        val m = PathMeasure().apply { setPath(line, false) }
        val part = Path()
        m.getSegment(0f, m.length * drawn.value, part, true)
        drawPath(part, p.ink.copy(alpha = 0.8f), style = Stroke(1.75.dp.toPx(), cap = StrokeCap.Round, join = StrokeJoin.Round))
        for ((i, color) in listOf(best to p.great, turn to p.blunder)) {
            if (i == null || i !in pts.indices) continue
            val shown = ((drawn.value - i.toFloat() / arc.size) * 6).coerceIn(0f, 1f)
            if (shown <= 0f) continue
            drawCircle(p.surface, 6.5.dp.toPx() * shown, pts[i])
            drawCircle(color, 4.5.dp.toPx() * shown, pts[i])
        }
    }
}

/** A win: glass flecks in the app's colours rise and fall once. */
@Composable
fun Celebration(modifier: Modifier) {
    val p = LocalPalette.current
    val flecks = remember {
        List(28) {
            listOf(Random.nextDouble(-170.0, 170.0).toFloat(), Random.nextDouble(-190.0, -60.0).toFloat(), Random.nextDouble(-360.0, 360.0).toFloat(),
                Random.nextDouble(0.0, 0.15).toFloat(), Random.nextInt(3).toFloat(), Random.nextDouble(6.0, 11.0).toFloat())
        }
    }
    val t = remember { Animatable(0f) }
    LaunchedEffect(Unit) { t.animateTo(1f, tween(1550, easing = androidx.compose.animation.core.CubicBezierEasing(0.1f, 0.8f, 0.3f, 1f))) }
    Canvas(modifier.size(1.dp)) {
        val colors = listOf(p.great, p.accent, p.inaccuracy)
        for ((dx, dy, rot, delay, c, s) in flecks.map { it }) {
            val v = ((t.value - delay) / (1 - delay)).coerceIn(0f, 1f)
            val at = Offset(dx.dp.toPx() * v, dy.dp.toPx() * v)
            rotate(rot * v, at) {
                drawRoundRect(colors[c.toInt()].copy(alpha = 1 - v), at - Offset(s.dp.toPx() / 2, s.dp.toPx() * 0.3f),
                    androidx.compose.ui.geometry.Size(s.dp.toPx(), s.dp.toPx() * 0.6f), androidx.compose.ui.geometry.CornerRadius(2.dp.toPx()))
            }
        }
    }
}

private operator fun <T> List<T>.component6() = this[5]
