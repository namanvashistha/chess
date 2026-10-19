package com.namanvashistha.calmchess

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Undo
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.QuestionMark
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material.icons.rounded.ThumbUp
import androidx.compose.ui.graphics.vector.ImageVector

/** The everyday icon for a verdict, no chess notation: the iPhone app's SF Symbols in Material's rounded set. */
fun verdictIcon(k: CoachCard.Kind): ImageVector? = when (k) {
    CoachCard.Kind.Great -> Icons.Rounded.Star
    CoachCard.Kind.Good -> Icons.Rounded.Check
    CoachCard.Kind.Inaccuracy -> Icons.Rounded.ThumbUp
    CoachCard.Kind.Mistake -> Icons.Rounded.QuestionMark
    CoachCard.Kind.Blunder -> Icons.AutoMirrored.Rounded.Undo
    else -> null
}
