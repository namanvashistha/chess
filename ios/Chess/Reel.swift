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
        case "three-seconds": await threeSeconds(model)
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

    /// "Only 3% see it in 3 seconds": the Petrov trap, where Nc6+ is a
    /// discovered check that wins the queen. A held countdown, the app's own
    /// two-step hint, the move, and the knight taking the queen.
    private static func threeSeconds(_ model: GameModel) async {
        model.stage(["e2e4", "e7e5", "g1f3", "g8f6", "f3e5", "f6e4", "d1e2", "e4f6"])
        beat("start")
        await wait(3.6)
        beat("hint"); model.hint()
        await wait(2.2)
        beat("hint2"); model.hint()
        await wait(2.4)
        beat("pick"); model.tap(sq("e5"))
        await wait(0.8)
        model.scriptedReplies = ["f8e7"]
        beat("move"); model.tap(sq("c6"))
        // Black has to answer the check: wait for the reply.
        for _ in 0..<120 where model.turn != "w" || model.busy { await wait(0.1) }
        beat("reply")
        await wait(1.0)
        // The queen is gone either way; the bishop blocks, and the knight takes it.
        guard let queen = model.board.firstIndex(of: "q") else { beat("no-queen"); return }
        beat("pick2"); model.tap(sq("c6"))
        await wait(0.7)
        model.holdReplies = true
        beat("take"); model.tap(queen)
        await wait(4.0)
        beat("end")
    }
}
#endif
