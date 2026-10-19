#if DEBUG
import Foundation
import os

/// The reel studio: scripted scenes played on the real app, for recording
/// marketing reels. Launched with `-reel <scene>`. Every beat is logged
/// ("CalmReel"), so captions and sound can be laid over the recording.
@MainActor
enum Reel {
    private static let log = Logger(subsystem: "com.namanvashistha.cozychess", category: "reel")

    private static func beat(_ name: String) {
        log.notice("CalmReel \(name, privacy: .public) \(Date.now.timeIntervalSince1970, privacy: .public)")
    }

    private static func sq(_ name: String) -> Square { Squares.parse(name)! }

    private static func wait(_ seconds: Double) async { try? await Task.sleep(for: .seconds(seconds)) }

    static func play(_ scene: String, _ model: GameModel) async {
        switch scene {
        case "mate-in-one": await mateInOne(model)
        default: log.error("no scene \(scene, privacy: .public)")
        }
    }

    /// "Mate in 1. Easy.": the Scholar's Mate is on, the queen goes to h7
    /// instead, the coach is kind, undo, and then the mate.
    private static func mateInOne(_ model: GameModel) async {
        model.stage(["e2e4", "e7e5", "f1c4", "b8c6", "d1h5", "g8f6"])
        beat("start")
        await wait(1.7)
        beat("pick"); model.tap(sq("h5"))
        await wait(1.0)
        beat("blunder"); model.tap(sq("h7"))
        await wait(2.9)
        beat("undo"); model.undo()
        await wait(1.3)
        beat("pick2"); model.tap(sq("h5"))
        await wait(0.9)
        beat("mate"); model.tap(sq("f7"))
        await wait(5.2)
        beat("end")
    }
}
#endif
