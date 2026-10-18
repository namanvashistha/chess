import GroupActivities
import SwiftUI

/// A game of Calm Chess on a FaceTime call.
struct ChessActivity: GroupActivity {
    var metadata: GroupActivityMetadata {
        var m = GroupActivityMetadata()
        m.title = L("Calm Chess")
        m.subtitle = L("A calm game of chess")
        m.type = .generic
        return m
    }
}

/// Live games over SharePlay: both phones on the call show the same board.
/// Whoever starts the game plays White. Moves travel through FaceTime, so
/// there's no server here either.
@MainActor
@Observable
final class SharePlay: NSObject {
    static let shared = SharePlay()

    /// On a FaceTime call where SharePlay can start.
    private(set) var available = false

    @ObservationIgnored weak var model: GameModel?
    @ObservationIgnored var showGame: () -> Void = {}
    @ObservationIgnored private let observer = GroupStateObserver()
    @ObservationIgnored private var session: GroupSession<ChessActivity>?
    @ObservationIgnored private var messenger: GroupSessionMessenger?
    @ObservationIgnored private var tasks: [Task<Void, Never>] = []
    /// This phone started the activity, so it plays White and keeps others in sync.
    @ObservationIgnored private var host = false
    @ObservationIgnored private var white: UUID?

    private struct State: Codable {
        var moves: [String]
        var white: UUID
        var resigned = false
    }

    /// Starts listening for SharePlay sessions and for FaceTime calls.
    func listen() {
        available = observer.isEligibleForGroupSession
        tasks.append(Task {
            for await eligible in observer.$isEligibleForGroupSession.values { self.available = eligible }
        })
        tasks.append(Task {
            for await session in ChessActivity.sessions() { self.configure(session) }
        })
    }

    /// Offers the game to everyone on the call.
    func start() {
        host = true
        Task { _ = try? await ChessActivity().activate() }
    }

    private func configure(_ session: GroupSession<ChessActivity>) {
        self.session?.leave()
        self.session = session
        let messenger = GroupSessionMessenger(session: session)
        self.messenger = messenger
        session.join()

        tasks.append(Task {
            for await (state, _) in messenger.messages(of: State.self) { self.apply(state) }
        })
        tasks.append(Task {
            // Someone joined late: the host shows them the board as it is.
            for await _ in session.$activeParticipants.values where self.host { self.broadcast() }
        })
        if host {
            white = session.localParticipant.id
            open(moves: [])
            broadcast()
        }
    }

    private func apply(_ state: State) {
        guard let model, let session else { return }
        let color = state.white == session.localParticipant.id ? "w" : "b"
        if white != state.white || model.link !== self {
            white = state.white
            model.startFriend(name: GameModel.unnamedFriend, color: color, moves: state.moves, link: self)
            showGame()
        }
        model.receive(moves: state.moves, resigned: state.resigned)
    }

    private func open(moves: [String]) {
        guard let model else { return }
        model.startFriend(name: GameModel.unnamedFriend, color: "w", moves: moves, link: self)
        showGame()
    }

    private func broadcast(resigned: Bool = false) {
        guard let messenger, let white, let model else { return }
        let state = State(moves: model.mode == .friend && model.link === self ? model.moves : [], white: white,
                          resigned: resigned)
        Task { try? await messenger.send(state) }
    }
}

extension SharePlay: MatchLink {
    func didMove(_ model: GameModel) { broadcast() }
    func didResign(_ model: GameModel) { broadcast(resigned: true) }
    func rematch(_ model: GameModel) {
        guard let session else { return }
        host = true
        white = session.localParticipant.id
        open(moves: [])
        broadcast()
    }
}
