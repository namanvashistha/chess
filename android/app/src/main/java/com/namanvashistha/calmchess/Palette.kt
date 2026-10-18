package com.namanvashistha.calmchess

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

/**
 * The Stone palette, the same values as the iPhone app. Every colour has a
 * light and a dark value: in dark mode the stone dims the way it would in the
 * evening and surfaces turn to dark glass.
 */
data class Palette(
    val squareLight: Color,
    val squareDark: Color,
    val coordinate: Color,
    val lastMove: Color,
    val accent: Color,
    val accentTop: Color,
    val accentEdge: Color,
    val onAccent: Color,
    val ray: Color,
    val thinking: Color,
    val danger: Color,
    val ink: Color,
    val mist: Color,
    val backgroundTop: Color,
    val backgroundBottom: Color,
    val glow: Color,
    val statFill: Color,
    val surface: Color,
    val hairline: Color,
    val great: Color,
    val greatFill: Color,
    val goodFill: Color,
    val inaccuracy: Color,
    val inaccuracyFill: Color,
    val mistake: Color,
    val mistakeFill: Color,
    val blunder: Color,
    val blunderFill: Color,
) {
    companion object {
        private fun hex(v: Long) = Color(0xFF000000 or v)

        val Light = Palette(
            squareLight = hex(0xDDD8CB), squareDark = hex(0x8E9B94), coordinate = hex(0x6F7A70),
            lastMove = hex(0xE8C468),
            accent = hex(0x2E6150), accentTop = hex(0x3D7562), accentEdge = hex(0x1F4538), onAccent = hex(0xF3F1EA),
            ray = hex(0x4FA887), thinking = hex(0x4F7FC9), danger = hex(0xC23B3B),
            ink = hex(0x22272C), mist = hex(0x687076),
            backgroundTop = hex(0xF1EFE9), backgroundBottom = hex(0xE6E3DC), glow = hex(0xFFFAF0),
            statFill = hex(0x22272C).copy(alpha = 0.06f), surface = hex(0xFBFAF7), hairline = hex(0x22272C).copy(alpha = 0.09f),
            great = hex(0x2F9E5C), greatFill = hex(0xBFE9CF), goodFill = hex(0xDFE7F0),
            inaccuracy = hex(0x9A7206), inaccuracyFill = hex(0xF6E3A6),
            mistake = hex(0xB8601B), mistakeFill = hex(0xF7CFAA),
            blunder = hex(0xC2654C), blunderFill = hex(0xF6D3C8),
        )

        val Dark = Palette(
            squareLight = hex(0xC1BBAE), squareDark = hex(0x76827A), coordinate = hex(0x4F5A52),
            lastMove = hex(0xD9B556),
            accent = hex(0x7FB79F), accentTop = hex(0x93C6AF), accentEdge = hex(0x4E7D69), onAccent = hex(0x0F1A15),
            ray = hex(0x8DCBB0), thinking = hex(0x7DA6E6), danger = hex(0xFF8A8A),
            ink = hex(0xECEEF0), mist = hex(0x9AA2A8),
            backgroundTop = hex(0x16191D), backgroundBottom = hex(0x0F1114), glow = hex(0x262A2F),
            statFill = Color.White.copy(alpha = 0.06f), surface = hex(0x1C2025), hairline = Color.White.copy(alpha = 0.09f),
            great = hex(0x6FD69A), greatFill = hex(0x1F4A33), goodFill = hex(0x2A3440),
            inaccuracy = hex(0xF0CC64), inaccuracyFill = hex(0x4A3D14),
            mistake = hex(0xF5A565), mistakeFill = hex(0x4D2F17),
            blunder = hex(0xF29C86), blunderFill = hex(0x4D2C23),
        )
    }
}

val LocalPalette = staticCompositionLocalOf { Palette.Light }

/** Provides the light or dark palette, following the phone. */
@Composable
fun CalmTheme(content: @Composable () -> Unit) {
    CompositionLocalProvider(LocalPalette provides if (isSystemInDarkTheme()) Palette.Dark else Palette.Light) {
        content()
    }
}
