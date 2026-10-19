package com.namanvashistha.calmchess

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        Prefs.init(this)
        Haptics.init(this)
        setContent { CalmTheme { Root() } }
    }
}

@Composable
private fun Root() {
    val scope = rememberCoroutineScope()
    val model = remember { GameModel(scope) }
    var playing by remember { mutableStateOf(false) }
    AnimatedContent(playing, transitionSpec = { fadeIn() togetherWith fadeOut() }, label = "screen") { inGame ->
        if (inGame) GameScreen(model) { playing = false }
        else HomeStub(
            onContinue = { if (model.resume()) playing = true },
            onBegin = {
                model.start(Mode.Engine, Opponent.entries.getOrElse(Prefs.p.getInt("opponent", 0)) { Opponent.Easy })
                playing = true
            },
        )
    }
}

/** A stand-in home screen until stage 3 ports the real one: Continue a saved game, or Begin. */
@Composable
private fun HomeStub(onContinue: () -> Unit, onBegin: () -> Unit) {
    val p = LocalPalette.current
    val saved = remember { SavedGame.load() }
    AppBackground {
        Column(Modifier.fillMaxSize().safeDrawingPadding().padding(24.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("Calm Chess", style = Typeface.display(34f), color = p.ink)
            Spacer(Modifier.weight(1f))
            if (saved != null) PrimaryButton("Continue", onClick = onContinue)
            PrimaryButton("Begin", onClick = onBegin)
        }
    }
}
