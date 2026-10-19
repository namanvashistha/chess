package com.namanvashistha.calmchess

import android.os.SystemClock
import android.util.Log
import kotlinx.coroutines.delay

/**
 * The reel studio: scripted scenes played on the real app, for recording
 * marketing reels. Debug builds only, started with `--es reel <scene>`. Every
 * beat is logged with its uptime ("CalmReel"), so captions and sound can be
 * laid over the recording at exactly the right moments.
 */
object Reel {
    private fun beat(name: String) = Log.d("CalmReel", "$name ${SystemClock.uptimeMillis()}")

    private fun sq(name: String) = Squares.parse(name)!!

    /** "Mate in 1. Easy.": the Scholar's Mate is on, the queen goes to h7 instead, the coach is kind, undo, mate. */
    suspend fun mateInOne(model: GameModel) {
        model.stage(listOf("e2e4", "e7e5", "f1c4", "b8c6", "d1h5", "g8f6"))
        beat("start")
        delay(1700)
        beat("pick"); model.tap(sq("h5"))
        delay(1000)
        beat("blunder"); model.tap(sq("h7"))
        delay(2900)
        beat("undo"); model.undo()
        delay(1300)
        beat("pick2"); model.tap(sq("h5"))
        delay(900)
        beat("mate"); model.tap(sq("f7"))
        delay(5200)
        beat("end")
    }

    suspend fun play(scene: String, model: GameModel) = when (scene) {
        "mate-in-one" -> mateInOne(model)
        else -> Log.w("CalmReel", "no scene $scene")
    }
}
