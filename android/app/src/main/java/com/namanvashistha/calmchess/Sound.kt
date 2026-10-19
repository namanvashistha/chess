package com.namanvashistha.calmchess

import android.content.Context
import android.media.AudioAttributes
import android.media.AudioFormat
import android.media.AudioManager
import android.media.AudioTrack
import kotlin.math.PI
import kotlin.math.asin
import kotlin.math.exp
import kotlin.math.min
import kotlin.math.pow
import kotlin.math.sin
import kotlin.random.Random

/**
 * Glass sounds, synthesized at launch exactly as on iPhone, so the app ships
 * no audio files: a clear tap for a move, a crack for a capture, a rising
 * chime for a great move, a low knock for a bad one. Quiet when the phone is
 * on silent or vibrate, as the iPhone app respects the silent switch.
 */
object Sound {
    private const val RATE = 44_100
    private var audio: AudioManager? = null
    private val tracks = mutableMapOf<Feedback, AudioTrack>()

    /** Settings' Sound switch; on unless turned off. */
    val enabled get() = Prefs.p.getBoolean("sound", true)

    fun init(context: Context) {
        audio = context.getSystemService(Context.AUDIO_SERVICE) as AudioManager
        Thread {
            val made = mapOf(
                Feedback.Select to tones(listOf(p(1320.0, 0.0, 0.12, 0.05))),
                // Lower and softer than picking up, so the two read as a pair.
                Feedback.PutDown to tones(listOf(p(990.0, 0.0, 0.1, 0.035))),
                Feedback.Move to tones(listOf(p(880.0, 0.0, 0.25, 0.09), p(1760.0, 0.0, 0.18, 0.03))),
                Feedback.Capture to tones(listOf(p(1980.0, 0.0, 0.3, 0.05)), noise = 0.35 to 0.12),
                Feedback.Check to tones(listOf(p(660.0, 0.0, 0.2, 0.04), p(660.0, 0.14, 0.2, 0.04)), square = true),
                Feedback.Great to tones(listOf(p(784.0, 0.0, 0.35, 0.07), p(988.0, 0.07, 0.35, 0.07), p(1319.0, 0.14, 0.35, 0.07))),
                Feedback.Good to tones(listOf(p(1046.0, 0.0, 0.25, 0.06))),
                Feedback.Bad to tones(listOf(p(330.0, 0.0, 0.3, 0.07), p(311.0, 0.08, 0.35, 0.05)), triangle = true),
                Feedback.Win to tones(listOf(p(523.0, 0.0, 0.5, 0.07), p(659.0, 0.09, 0.5, 0.07), p(784.0, 0.18, 0.5, 0.07),
                    p(1046.0, 0.27, 0.5, 0.07), p(1319.0, 0.36, 0.5, 0.07))),
                Feedback.Lose to tones(listOf(p(523.0, 0.0, 0.4, 0.06), p(440.0, 0.12, 0.5, 0.06))),
            ).mapValues { (_, pcm) -> track(pcm) }
            synchronized(tracks) { tracks.putAll(made) }
        }.start()
    }

    fun play(kind: Feedback) {
        if (!enabled || audio?.ringerMode != AudioManager.RINGER_MODE_NORMAL) return
        val t = synchronized(tracks) { tracks[kind] } ?: return
        runCatching {
            t.stop()
            t.reloadStaticData()
            t.play()
        }
    }

    private data class Partial(val freq: Double, val start: Double, val dur: Double, val gain: Double)
    private fun p(f: Double, s: Double, d: Double, g: Double) = Partial(f, s, d, g)

    /** Partials with a fast attack and an exponential decay, optionally over a burst of high-passed noise. */
    private fun tones(partials: List<Partial>, noise: Pair<Double, Double>? = null, square: Boolean = false, triangle: Boolean = false): ShortArray {
        val length = maxOf(partials.maxOf { it.start + it.dur }, noise?.first ?: 0.0) + 0.05
        val frames = (length * RATE).toInt()
        val out = FloatArray(frames)
        for ((freq, start, dur, gain) in partials) {
            val s0 = (start * RATE).toInt()
            val n = (dur * RATE).toInt()
            for (k in 0 until n) {
                if (s0 + k >= frames) break
                val t = k.toDouble() / RATE
                val phase = 2 * PI * freq * t
                var wave = sin(phase)
                if (square) wave = if (wave >= 0) 1.0 else -1.0
                if (triangle) wave = 2 / PI * asin(sin(phase))
                val env = min(1.0, t / 0.005) * exp(-5 * t / dur)
                out[s0 + k] += (wave * gain * env).toFloat()
            }
        }
        if (noise != null) {
            var prev = 0f
            val n = (noise.first * RATE).toInt()
            for (k in 0 until minOf(n, frames)) {
                val white = Random.nextFloat() * 2 - 1
                val high = white - prev // a crude high-pass: glassy, not rumbling
                prev = white
                out[k] += high * noise.second.toFloat() * (1 - k.toFloat() / n).pow(3) * 0.5f
            }
        }
        // The iPhone's mixer plays them at 0.8.
        return ShortArray(frames) { (out[it] * 0.8f * Short.MAX_VALUE).toInt().coerceIn(-32768, 32767).toShort() }
    }

    private fun track(pcm: ShortArray): AudioTrack {
        val t = AudioTrack.Builder()
            .setAudioAttributes(AudioAttributes.Builder().setUsage(AudioAttributes.USAGE_GAME)
                .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION).build())
            .setAudioFormat(AudioFormat.Builder().setEncoding(AudioFormat.ENCODING_PCM_16BIT).setSampleRate(RATE)
                .setChannelMask(AudioFormat.CHANNEL_OUT_MONO).build())
            .setTransferMode(AudioTrack.MODE_STATIC)
            .setBufferSizeInBytes(pcm.size * 2)
            .build()
        t.write(pcm, 0, pcm.size)
        return t
    }
}
