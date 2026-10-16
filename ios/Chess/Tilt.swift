import CoreMotion
import Observation
import UIKit

/// How far the phone leans, as x (roll) and y (pitch) in -1...1. The values are
/// relative to how the phone is being held: a slow-moving baseline follows the
/// raw attitude, so only movement registers and resting at any angle reads as 0.
///
/// The simulator has no motion sensors, so there the board sways on its own to
/// show the effect.
@MainActor
@Observable
final class Tilt {
    private(set) var x: Double = 0
    private(set) var y: Double = 0

    @ObservationIgnored private let motion = CMMotionManager()
    @ObservationIgnored private var baseRoll: Double?
    @ObservationIgnored private var basePitch: Double = 0
    @ObservationIgnored private var timer: Timer?

    /// Radians of lean that map to full deflection.
    private let range = 0.35

    func start() {
        guard !UIAccessibility.isReduceMotionEnabled else { return }
        if motion.isDeviceMotionAvailable {
            motion.deviceMotionUpdateInterval = 1 / 60
            motion.startDeviceMotionUpdates(to: .main) { [weak self] data, _ in
                guard let self, let a = data?.attitude else { return }
                MainActor.assumeIsolated { self.update(roll: a.roll, pitch: a.pitch) }
            }
        } else {
            let start = Date()
            timer = Timer.scheduledTimer(withTimeInterval: 1 / 60, repeats: true) { [weak self] _ in
                let t = Date().timeIntervalSince(start)
                MainActor.assumeIsolated {
                    self?.x = sin(t / 1.7) * 0.7
                    self?.y = cos(t / 2.3) * 0.55
                }
            }
        }
    }

    func stop() {
        motion.stopDeviceMotionUpdates()
        timer?.invalidate()
        timer = nil
        baseRoll = nil
        x = 0
        y = 0
    }

    private func update(roll: Double, pitch: Double) {
        if baseRoll == nil {
            baseRoll = roll
            basePitch = pitch
        }
        baseRoll! += (roll - baseRoll!) * 0.01
        basePitch += (pitch - basePitch) * 0.01
        let nx = max(-1, min(1, (roll - baseRoll!) / range))
        let ny = max(-1, min(1, (pitch - basePitch) / range))
        x += (nx - x) * 0.15
        y += (ny - y) * 0.15
    }
}
