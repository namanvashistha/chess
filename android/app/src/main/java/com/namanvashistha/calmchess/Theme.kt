package com.namanvashistha.calmchess

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontVariation
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Type: a rounded face for display, the system face for everything else. The
 * iPhone app uses Apple's SF Rounded, which only Apple's platforms may carry;
 * Nunito is the closest open rounded face.
 */
object Typeface {
    private fun weight(w: Int) = Font(R.font.nunito, FontWeight(w), variationSettings = FontVariation.Settings(FontVariation.weight(w)))
    val rounded = FontFamily(weight(500), weight(600), weight(700), weight(800), weight(900))

    /** Display text, as on iPhone: heavy by default. SF's weights sit a step heavier than Nunito's. */
    fun display(size: Float, weight: FontWeight = FontWeight.Black) = TextStyle(
        fontFamily = rounded,
        fontWeight = when (weight) {
            FontWeight.Black -> FontWeight(900)
            FontWeight.Bold -> FontWeight(850)
            FontWeight.SemiBold -> FontWeight(750)
            else -> weight
        },
        fontSize = size.sp,
    )

    fun body(size: Float, weight: FontWeight = FontWeight.Normal) = TextStyle(fontSize = size.sp, fontWeight = weight)
}

/** The app's background: warm daylight, or evening, with a soft glow at the top. */
@Composable
fun AppBackground(content: @Composable BoxScope.() -> Unit) {
    val p = LocalPalette.current
    Box(
        Modifier
            .fillMaxSize()
            .background(Brush.verticalGradient(listOf(p.backgroundTop, p.backgroundBottom)))
            .drawBehind {
                drawRect(Brush.radialGradient(listOf(p.glow, Color.Transparent), center = Offset(size.width / 2, 0f), radius = 380.dp.toPx()))
            },
        content = content,
    )
}

/**
 * Glass, as close as Android gets to Liquid Glass: a pale translucent fill,
 * a lit top edge fading to a hairline, and a soft shadow underneath.
 */
fun Modifier.glass(shape: Shape, tint: Color? = null): Modifier = composed {
    val p = LocalPalette.current
    val dark = p === Palette.Dark
    this
        .shadow(10.dp, shape, ambientColor = Color.Black.copy(alpha = 0.1f), spotColor = Color.Black.copy(alpha = 0.14f))
        .background(if (dark) Color(0xFF2A2F35).copy(alpha = 0.82f) else Color.White.copy(alpha = 0.66f), shape)
        .then(if (tint != null) Modifier.background(tint, shape) else Modifier)
        .background(Brush.verticalGradient(listOf(Color.White.copy(alpha = if (dark) 0.07f else 0.4f), Color.Transparent)), shape)
        .border(1.dp, Brush.verticalGradient(listOf(Color.White.copy(alpha = if (dark) 0.2f else 0.9f), p.hairline)), shape)
}

/** The primary button: evergreen with a pressed-in edge. */
@Composable
fun PrimaryButton(text: String, modifier: Modifier = Modifier, onClick: () -> Unit) {
    val p = LocalPalette.current
    val source = remember { MutableInteractionSource() }
    val pressed by source.collectIsPressedAsState()
    val sink by animateFloatAsState(if (pressed) 4f else 0f, swiftSpring(0.2f, 0.6f), label = "sink")
    LaunchedEffect(pressed) { if (pressed) Haptics.tap(0.6f) }
    val shape = RoundedCornerShape(20.dp)
    Box(modifier.fillMaxWidth().height(62.dp).clickable(source, null, onClick = onClick)) {
        // The edge underneath, which the face presses down onto.
        Box(Modifier.fillMaxWidth().height(56.dp).offset(y = 6.dp)
            .shadow(14.dp, shape, ambientColor = p.accent.copy(alpha = 0.35f), spotColor = p.accent.copy(alpha = 0.35f))
            .background(p.accentEdge, shape))
        Box(
            Modifier.fillMaxWidth().height(56.dp).offset(y = sink.dp)
                .graphicsLayer { val s = 1f - 0.015f * (sink / 4f); scaleX = s; scaleY = s }
                .background(Brush.verticalGradient(listOf(p.accentTop, p.accent)), shape),
            contentAlignment = Alignment.Center,
        ) {
            Text(text, style = Typeface.display(19f), color = p.onAccent)
        }
    }
}

/** Gives a little under the finger and springs back, like the iPhone's pressable cards. */
fun Modifier.pressable(source: MutableInteractionSource, scale: Float = 0.965f): Modifier = composed {
    val pressed by source.collectIsPressedAsState()
    val s by animateFloatAsState(if (pressed) scale else 1f, swiftSpring(0.25f, 0.55f), label = "press")
    this.graphicsLayer { scaleX = s; scaleY = s }
}
