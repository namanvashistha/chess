package com.namanvashistha.calmchess

import android.content.Context
import android.os.VibrationEffect
import android.os.VibrationEffect.Composition
import android.os.Vibrator
import android.os.VibratorManager

/**
 * Small taps under the finger, built from Android's haptic primitives so they
 * feel crisp rather than buzzy. The same moments as on iPhone.
 */
object Haptics {
    private var vibrator: Vibrator? = null

    fun init(context: Context) {
        vibrator = (context.getSystemService(Context.VIBRATOR_MANAGER_SERVICE) as VibratorManager).defaultVibrator
    }

    /** Primitives as (id, scale, delay ms); falls back to a plain tick on phones without them. */
    private fun compose(vararg parts: Triple<Int, Float, Int>) {
        val v = vibrator ?: return
        if (!v.areAllPrimitivesSupported(*parts.map { it.first }.toIntArray())) {
            v.vibrate(VibrationEffect.createPredefined(VibrationEffect.EFFECT_TICK))
            return
        }
        val c = VibrationEffect.startComposition()
        for ((id, scale, delay) in parts) c.addPrimitive(id, scale, delay)
        v.vibrate(c.compose())
    }

    fun tap(scale: Float = 0.5f) = compose(Triple(Composition.PRIMITIVE_TICK, scale, 0))
    fun select() = tap(0.6f)
    fun putDown() = tap(0.35f)

    fun play(kind: Feedback) = when (kind) {
        Feedback.Select -> select()
        Feedback.PutDown -> putDown()
        Feedback.Move -> compose(Triple(Composition.PRIMITIVE_CLICK, 0.4f, 0))
        Feedback.Capture -> compose(Triple(Composition.PRIMITIVE_CLICK, 0.8f, 0), Triple(Composition.PRIMITIVE_TICK, 0.4f, 40))
        Feedback.Check -> compose(Triple(Composition.PRIMITIVE_THUD, 0.6f, 0))
        Feedback.Great -> compose(Triple(Composition.PRIMITIVE_CLICK, 0.5f, 0), Triple(Composition.PRIMITIVE_CLICK, 0.8f, 90))
        Feedback.Good -> tap(0.3f)
        Feedback.Bad -> compose(Triple(Composition.PRIMITIVE_LOW_TICK, 0.6f, 0))
        Feedback.Win -> compose(Triple(Composition.PRIMITIVE_QUICK_RISE, 0.6f, 0), Triple(Composition.PRIMITIVE_CLICK, 1f, 60))
        Feedback.Lose -> compose(Triple(Composition.PRIMITIVE_QUICK_FALL, 0.4f, 0))
    }
}
