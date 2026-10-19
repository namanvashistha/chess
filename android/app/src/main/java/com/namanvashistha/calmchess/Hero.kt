package com.namanvashistha.calmchess

import androidx.compose.animation.AnimatedVisibilityScope
import androidx.compose.animation.ExperimentalSharedTransitionApi
import androidx.compose.animation.SharedTransitionScope
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.unit.dp

@OptIn(ExperimentalSharedTransitionApi::class)
val LocalSharedScope = compositionLocalOf<SharedTransitionScope?> { null }
val LocalScreenScope = compositionLocalOf<AnimatedVisibilityScope?> { null }

/**
 * Marks the board that carries across between home and the game: going into
 * a game, the board itself glides into place instead of the page changing.
 */
@OptIn(ExperimentalSharedTransitionApi::class)
fun Modifier.heroBoard(): Modifier = composed {
    val shared = LocalSharedScope.current
    val screen = LocalScreenScope.current
    if (shared == null || screen == null) this
    else with(shared) {
        this@composed.sharedElement(
            rememberSharedContentState("board"), screen,
            boundsTransform = { _, _ -> tween(450, easing = FastOutSlowInEasing) },
        )
    }
}

/**
 * Everything but the board on a screen that is arriving: it gathers around the
 * board. Things above the board drift down into place, things below rise, each
 * a beat apart.
 */
fun Modifier.chrome(fromTop: Boolean = false, delay: Int = 140): Modifier = composed {
    var shown by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) { shown = true }
    val v by animateFloatAsState(if (shown) 1f else 0f, tween(500, delay, FastOutSlowInEasing), label = "chrome")
    graphicsLayer {
        alpha = v
        translationY = (1 - v) * (if (fromTop) -10 else 14).dp.toPx()
    }
}
