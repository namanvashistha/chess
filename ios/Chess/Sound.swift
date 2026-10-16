import AVFoundation
import UIKit

/// A small tap felt under the finger when a control is pressed. Light by
/// default; respects the haptics setting.
@MainActor
enum Haptics {
    static func tap(_ style: UIImpactFeedbackGenerator.FeedbackStyle = .light, intensity: CGFloat = 0.7) {
        guard Sound.hapticsOn else { return }
        UIImpactFeedbackGenerator(style: style).impactOccurred(intensity: intensity)
    }
}

/// Glass sounds, synthesized at launch so the app ships no audio files: a
/// clear tap for a move, a crack for a capture, a rising chime for a great
/// move, a low knock for a bad one. Plays alongside other audio and respects
/// the silent switch.
@MainActor
final class Sound {
    static let shared = Sound()

    private let engine = AVAudioEngine()
    private var players: [AVAudioPlayerNode] = []
    private var next = 0
    private var buffers: [Feedback.Kind: AVAudioPCMBuffer] = [:]
    private let format = AVAudioFormat(standardFormatWithSampleRate: 44_100, channels: 1)!

    private init() {
        try? AVAudioSession.sharedInstance().setCategory(.ambient, options: [.mixWithOthers])
        for _ in 0..<4 {
            let p = AVAudioPlayerNode()
            engine.attach(p)
            engine.connect(p, to: engine.mainMixerNode, format: format)
            players.append(p)
        }
        engine.mainMixerNode.outputVolume = 0.8
        buffers = [
            .select: tones([(1320, 0, 0.12, 0.05)]),
            .move: tones([(880, 0, 0.25, 0.09), (1760, 0, 0.18, 0.03)]),
            .capture: tones([(1980, 0, 0.3, 0.05)], noise: (0.35, 0.12)),
            .check: tones([(660, 0, 0.2, 0.04), (660, 0.14, 0.2, 0.04)], square: true),
            .great: tones([(784, 0, 0.35, 0.07), (988, 0.07, 0.35, 0.07), (1319, 0.14, 0.35, 0.07)]),
            .good: tones([(1046, 0, 0.25, 0.06)]),
            .bad: tones([(330, 0, 0.3, 0.07), (311, 0.08, 0.35, 0.05)], triangle: true),
            .win: tones([(523, 0, 0.5, 0.07), (659, 0.09, 0.5, 0.07), (784, 0.18, 0.5, 0.07), (1046, 0.27, 0.5, 0.07), (1319, 0.36, 0.5, 0.07)]),
            .lose: tones([(523, 0, 0.4, 0.06), (440, 0.12, 0.5, 0.06)]),
        ]
    }

    /// Settings: sound and haptics, both on unless turned off.
    static var soundOn: Bool { UserDefaults.standard.object(forKey: "sound") as? Bool ?? true }
    static var hapticsOn: Bool { UserDefaults.standard.object(forKey: "haptics") as? Bool ?? true }

    func play(_ kind: Feedback.Kind) {
        guard Self.soundOn, let buffer = buffers[kind] else { return }
        if !engine.isRunning { try? engine.start() }
        guard engine.isRunning else { return }
        let p = players[next]
        next = (next + 1) % players.count
        p.scheduleBuffer(buffer, at: nil, options: .interrupts)
        if !p.isPlaying { p.play() }
    }

    /// Renders (frequency, start, length, gain) partials with a fast attack and
    /// an exponential decay, optionally over a burst of high-passed noise.
    private func tones(_ partials: [(Double, Double, Double, Double)], noise: (Double, Double)? = nil,
                       square: Bool = false, triangle: Bool = false) -> AVAudioPCMBuffer {
        let rate = format.sampleRate
        let length = max(partials.map { $0.1 + $0.2 }.max() ?? 0, noise?.0 ?? 0) + 0.05
        let frames = AVAudioFrameCount(length * rate)
        let buffer = AVAudioPCMBuffer(pcmFormat: format, frameCapacity: frames)!
        buffer.frameLength = frames
        let out = buffer.floatChannelData![0]
        for i in 0..<Int(frames) { out[i] = 0 }
        for (freq, start, dur, gain) in partials {
            let s0 = Int(start * rate), n = Int(dur * rate)
            for k in 0..<n where s0 + k < Int(frames) {
                let t = Double(k) / rate
                let phase = 2 * Double.pi * freq * t
                var wave = sin(phase)
                if square { wave = wave >= 0 ? 1 : -1 }
                if triangle { wave = 2 / Double.pi * asin(sin(phase)) }
                let attack = min(1, t / 0.005)
                let env = attack * exp(-5 * t / dur)
                out[s0 + k] += Float(wave * gain * env)
            }
        }
        if let (dur, gain) = noise {
            var prev: Float = 0
            let n = Int(dur * rate)
            for k in 0..<n where k < Int(frames) {
                let white = Float.random(in: -1...1)
                let high = white - prev // a crude high-pass: glassy, not rumbling
                prev = white
                let env = pow(1 - Float(k) / Float(n), 3)
                out[k] += high * Float(gain) * env * 0.5
            }
        }
        return buffer
    }
}
