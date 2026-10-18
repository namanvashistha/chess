package com.namanvashistha.calmchess

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/** The app's background: warm daylight, or evening. */
@Composable
fun AppBackground(content: @Composable () -> Unit) {
    val p = LocalPalette.current
    Box(
        Modifier
            .fillMaxSize()
            .background(Brush.verticalGradient(listOf(p.backgroundTop, p.backgroundBottom)))
            .background(Brush.radialGradient(listOf(p.glow, Color.Transparent), radius = 1000f, center = androidx.compose.ui.geometry.Offset(540f, 0f)))
    ) { content() }
}

/** The game: who you're playing, the board, and the coach underneath. */
@Composable
fun GameScreen(model: GameModel) {
    val p = LocalPalette.current
    AppBackground {
        Column(
            Modifier
                .fillMaxSize()
                .safeDrawingPadding()
                .padding(horizontal = 14.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Row(Modifier.fillMaxWidth().padding(vertical = 14.dp), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.size(36.dp).background(p.statFill, RoundedCornerShape(50)))
                Spacer(Modifier.size(10.dp))
                Text(model.opponent.title, style = TextStyle(fontSize = 17.sp, fontWeight = FontWeight.Bold, color = p.ink))
            }
            BoardView(model, Modifier.widthIn(max = 760.dp).fillMaxWidth())
            Spacer(Modifier.size(24.dp))
            CoachView(model.coach)
        }
    }
}

/** The coach's card: a verdict badge, a title in the verdict's colour, and one sentence. */
@Composable
fun CoachView(card: CoachCard?) {
    val p = LocalPalette.current
    AnimatedContent(card, transitionSpec = { fadeIn() togetherWith fadeOut() }, label = "coach") { c ->
        if (c == null) { Spacer(Modifier.heightIn(min = 74.dp)); return@AnimatedContent }
        val (fill, titleColor) = when (c.kind) {
            CoachCard.Kind.Great -> p.greatFill to p.great
            CoachCard.Kind.Inaccuracy -> p.inaccuracyFill to p.inaccuracy
            CoachCard.Kind.Mistake, CoachCard.Kind.Warning -> p.mistakeFill to p.mistake
            CoachCard.Kind.Blunder -> p.blunderFill to p.blunder
            CoachCard.Kind.Coach -> p.accent to p.ink
            else -> p.goodFill to p.ink
        }
        Row(
            Modifier
                .fillMaxWidth()
                .background(p.surface, RoundedCornerShape(20.dp))
                .border(1.dp, p.hairline, RoundedCornerShape(20.dp))
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Box(Modifier.size(46.dp).background(fill, RoundedCornerShape(14.dp)), contentAlignment = Alignment.Center) {
                Text(c.badge, style = TextStyle(fontSize = 17.sp, fontWeight = FontWeight.Bold,
                    color = if (c.kind == CoachCard.Kind.Coach) p.onAccent else p.ink))
            }
            Column(Modifier.weight(1f)) {
                Text(c.title, style = TextStyle(fontSize = 17.sp, fontWeight = FontWeight.ExtraBold, color = titleColor))
                if (c.text.isNotEmpty()) Text(c.text, style = TextStyle(fontSize = 13.5.sp, color = p.mist))
            }
        }
    }
}
