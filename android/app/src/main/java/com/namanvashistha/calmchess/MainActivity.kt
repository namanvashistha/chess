package com.namanvashistha.calmchess

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExperimentalSharedTransitionApi
import androidx.compose.animation.SharedTransitionLayout
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        installSplashScreen()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        Prefs.init(this)
        Strings.init(this)
        Haptics.init(this)
        Sound.init(this)
        setContent { CalmTheme { Root() } }
    }
}

/** The app: home, and the game it starts or resumes. The board carries across between them. */
@OptIn(ExperimentalSharedTransitionApi::class)
@Composable
private fun Root() {
    val scope = rememberCoroutineScope()
    val model = remember { GameModel(scope) }
    var playing by remember { mutableStateOf(false) }
    // Refreshes home's saved game and history on return.
    var homeId by remember { mutableIntStateOf(0) }
    // Decided before the first frame, so a first run never glimpses home.
    var welcoming by remember { mutableStateOf(firstRun()) }
    var learning by remember { mutableStateOf(false) }
    fun startGame(mode: Mode, opponent: Opponent) { model.start(mode, opponent); playing = true }
    SharedTransitionLayout {
        CompositionLocalProvider(LocalSharedScope provides this) {
            // A screen arrives without fading (its board carries across fully drawn;
            // the rest gathers on its own) and leaves by fading.
            AnimatedContent(playing, transitionSpec = { EnterTransition.None togetherWith fadeOut(tween(300)) }, label = "screen") { inGame ->
                CompositionLocalProvider(LocalScreenScope provides this) {
                    if (inGame) GameScreen(model) { homeId++; playing = false }
                    else key(homeId) {
                        HomeScreen(
                            start = { mode, opponent -> startGame(mode, opponent) },
                            resume = { if (model.resume()) playing = true },
                            learn = { learning = true },
                            welcome = { Prefs.p.edit().putBoolean("welcome.seen", false).apply(); welcoming = true },
                        )
                    }
                }
            }
        }
    }
    // The welcome sits on top from the first frame; Learn covers everything while it's open.
    if (learning) LearnScreen(close = { learning = false; homeId++ }, play = { startGame(Mode.Engine, Opponent.Easy) })
    if (welcoming) WelcomeScreen(enter = { welcoming = false }, learn = { learning = true })
}

/** A true first run. Someone who has already played doesn't need the welcome and is marked as having seen it. */
private fun firstRun(): Boolean {
    val p = Prefs.p
    if (!p.getBoolean("welcome.seen", false) && (Stats.recent.isNotEmpty() || SavedGame.load() != null)) {
        p.edit().putBoolean("welcome.seen", true).apply()
    }
    return !p.getBoolean("welcome.seen", false)
}
