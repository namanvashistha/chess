package com.namanvashistha.calmchess

import androidx.compose.animation.core.SpringSpec
import androidx.compose.animation.core.spring
import kotlin.math.PI

/**
 * A spring described the way the iPhone app describes its springs (a response
 * in seconds and a damping fraction), so motion feels the same on both.
 */
fun <T> swiftSpring(response: Float, dampingFraction: Float): SpringSpec<T> {
    val w = 2 * PI / response
    return spring(dampingRatio = dampingFraction, stiffness = (w * w).toFloat())
}
