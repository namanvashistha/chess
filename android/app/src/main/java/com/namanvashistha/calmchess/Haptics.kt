package com.namanvashistha.calmchess

import android.content.Context
import android.os.VibrationEffect
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

    private fun primitive(id: Int, scale: Float) {
        val v = vibrator ?: return
        if (!v.areAllPrimitivesSupported(id)) {
            v.vibrate(VibrationEffect.createPredefined(VibrationEffect.EFFECT_TICK))
            return
        }
        v.vibrate(VibrationEffect.startComposition().addPrimitive(id, scale).compose())
    }

    fun select() = primitive(VibrationEffect.Composition.PRIMITIVE_TICK, 0.6f)
    fun putDown() = primitive(VibrationEffect.Composition.PRIMITIVE_TICK, 0.35f)
    fun move() = primitive(VibrationEffect.Composition.PRIMITIVE_CLICK, 0.4f)
    fun capture() = primitive(VibrationEffect.Composition.PRIMITIVE_CLICK, 0.8f)
}
