#if DEBUG
import Chesscore
import Foundation

/// A pretend friend for testing friend games in the simulator: launched with
/// `-friendTest`, you play Black and "Sam" answers with the beginner bot.
@MainActor
final class DebugLink: MatchLink {
    static let shared = DebugLink()

    func didMove(_ model: GameModel) {
        let moves = model.moves
        guard !model.isOver else { return }
        Task.detached {
            guard let g = ChesscoreNewGame() else { return }
            for m in moves { _ = g.play(m) }
            let reply = g.botMove("beginner")
            try? await Task.sleep(for: .seconds(1.5))
            await MainActor.run { model.receive(moves: moves + [reply]) }
        }
    }

    func didResign(_ model: GameModel) {}

    func rematch(_ model: GameModel) { start(model) }

    func start(_ model: GameModel) {
        model.startFriend(name: "Sam", color: "b", moves: [], link: self)
        didMove(model) // Sam plays White's first move
    }
}
#endif
