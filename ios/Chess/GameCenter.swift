import Chesscore
import GameKit
import SwiftUI

/// Turn-based games with friends through Game Center. Apple hosts the games,
/// the invitations and the "your move" notifications; the app keeps no server
/// and no account of its own.
///
/// A game travels as its move list. The player who started it is White.
@MainActor
@Observable
final class GameCenter: NSObject {
    static let shared = GameCenter()

    /// Signed in to Game Center on this device.
    private(set) var signedIn = false
    /// Games with friends that are still going, newest first.
    private(set) var matches: [GKTurnBasedMatch] = []

    /// The board, and a way to show the game screen.
    @ObservationIgnored weak var model: GameModel?
    @ObservationIgnored var showGame: () -> Void = {}
    /// The game on the board, if it's a Game Center game.
    @ObservationIgnored private var current: GKTurnBasedMatch?
    /// Game Center's own sign-in screen, kept until the player asks to play a friend.
    @ObservationIgnored private var signIn: UIViewController?

    private struct Payload: Codable { var moves: [String] }

    // MARK: Signing in

    /// Signs in quietly at launch. If Game Center needs the player to sign in,
    /// its screen waits until they ask to play a friend.
    func authenticate() {
        GKLocalPlayer.local.authenticateHandler = { [weak self] vc, _ in
            Task { @MainActor in
                guard let self else { return }
                self.signIn = vc
                self.signedIn = GKLocalPlayer.local.isAuthenticated
                if self.signedIn {
                    GKLocalPlayer.local.register(self)
                    await self.reload()
                }
            }
        }
    }

    func reload() async {
        guard signedIn else { return }
        let all = (try? await GKTurnBasedMatch.loadMatches()) ?? []
        matches = all.filter { $0.status == .open || $0.status == .matching }
            .sorted { ($0.creationDate) > ($1.creationDate) }
    }

    // MARK: Starting and opening

    /// Opens Game Center's invite screen to start a game with a friend.
    func invite() {
        if !signedIn {
            if let signIn { present(signIn) }
            return
        }
        let request = GKMatchRequest()
        request.minPlayers = 2
        request.maxPlayers = 2
        request.inviteMessage = "Let's play a calm game of chess."
        let vc = GKTurnBasedMatchmakerViewController(matchRequest: request)
        vc.turnBasedMatchmakerDelegate = self
        vc.showExistingMatches = false
        present(vc)
    }

    /// Shows a game on the board.
    func open(_ match: GKTurnBasedMatch) {
        guard let model else { return }
        current = match
        let me = match.participants.firstIndex { $0.player?.gamePlayerID == GKLocalPlayer.local.gamePlayerID } ?? 0
        let color = me == 0 ? "w" : "b"
        model.startFriend(name: Self.friendName(in: match), color: color, moves: Self.moves(in: match), link: self)
        if Self.friendQuit(in: match) { model.receive(moves: Self.moves(in: match), resigned: true) }
        showGame()
        settleIfFriendQuit(match)
    }

    /// The friend's name, or a placeholder while the invitation is out.
    static func friendName(in match: GKTurnBasedMatch) -> String {
        let other = match.participants.first { $0.player?.gamePlayerID != GKLocalPlayer.local.gamePlayerID }
        return other?.player?.displayName ?? "Your friend"
    }

    static func moves(in match: GKTurnBasedMatch) -> [String] {
        guard let data = match.matchData, !data.isEmpty,
              let p = try? JSONDecoder().decode(Payload.self, from: data) else { return [] }
        return p.moves
    }

    static func isMyTurn(_ match: GKTurnBasedMatch) -> Bool {
        match.currentParticipant?.player?.gamePlayerID == GKLocalPlayer.local.gamePlayerID
    }

    private static func friendQuit(in match: GKTurnBasedMatch) -> Bool {
        match.participants.contains { $0.player?.gamePlayerID != GKLocalPlayer.local.gamePlayerID && $0.matchOutcome == .quit }
    }

    private func data(_ model: GameModel) -> Data {
        (try? JSONEncoder().encode(Payload(moves: model.moves))) ?? Data()
    }

    private func others(_ match: GKTurnBasedMatch) -> [GKTurnBasedParticipant] {
        match.participants.filter { $0.player?.gamePlayerID != GKLocalPlayer.local.gamePlayerID }
    }

    private func mine(_ match: GKTurnBasedMatch) -> GKTurnBasedParticipant? {
        match.participants.first { $0.player?.gamePlayerID == GKLocalPlayer.local.gamePlayerID }
    }

    /// If the friend resigned and it's now this player's turn, close the game
    /// on Game Center so it leaves both lists.
    private func settleIfFriendQuit(_ match: GKTurnBasedMatch) {
        guard Self.friendQuit(in: match), Self.isMyTurn(match), match.status == .open else { return }
        mine(match)?.matchOutcome = .won
        Task {
            try? await match.endMatchInTurn(withMatch: match.matchData ?? Data())
            await reload()
        }
    }

    // MARK: Presenting Game Center's screens

    private func present(_ vc: UIViewController) {
        #if MESSAGES
        // The Messages extension has no app window; it never shows Game Center.
        #else
        guard let root = UIApplication.shared.connectedScenes
            .compactMap({ ($0 as? UIWindowScene)?.keyWindow?.rootViewController }).first else { return }
        var top = root
        while let next = top.presentedViewController { top = next }
        top.present(vc, animated: true)
        #endif
    }
}

// MARK: - Sending moves

extension GameCenter: MatchLink {
    func didMove(_ model: GameModel) {
        guard let match = current else { return }
        let data = data(model)
        Task {
            if model.isOver {
                let iWon = model.status == ChesscoreStatusCheckmate && model.turn != model.myColor
                let draw = model.status != ChesscoreStatusCheckmate
                mine(match)?.matchOutcome = draw ? .tied : iWon ? .won : .lost
                for p in others(match) { p.matchOutcome = draw ? .tied : iWon ? .lost : .won }
                try? await match.endMatchInTurn(withMatch: data)
            } else {
                try? await match.endTurn(withNextParticipants: others(match), turnTimeout: GKTurnTimeoutDefault, match: data)
            }
            await reload()
        }
    }

    func didResign(_ model: GameModel) {
        guard let match = current else { return }
        let data = data(model)
        Task {
            if Self.isMyTurn(match) {
                try? await match.participantQuitInTurn(with: .quit, nextParticipants: others(match),
                                                      turnTimeout: GKTurnTimeoutDefault, match: data)
            } else {
                try? await match.participantQuitOutOfTurn(with: .quit)
            }
            await reload()
        }
    }

    func rematch(_ model: GameModel) {
        guard let match = current else { return }
        Task {
            if let next = try? await match.rematch() { open(next) }
            await reload()
        }
    }
}

// MARK: - Game Center events

extension GameCenter: GKLocalPlayerListener {
    nonisolated func player(_ player: GKPlayer, receivedTurnEventFor match: GKTurnBasedMatch, didBecomeActive: Bool) {
        Task { @MainActor in
            await self.reload()
            // Tapped a notification, accepted an invite, or just created a game: show it.
            if didBecomeActive || (self.current == nil && Self.moves(in: match).isEmpty && Self.isMyTurn(match)) {
                self.open(match)
                return
            }
            // The friend moved in the game on the board: let the moves glide in.
            if let current = self.current, current.matchID == match.matchID, let model = self.model, model.mode == .friend {
                self.current = match
                model.renameFriend(Self.friendName(in: match))
                model.receive(moves: Self.moves(in: match), resigned: Self.friendQuit(in: match))
                self.settleIfFriendQuit(match)
            }
        }
    }

    nonisolated func player(_ player: GKPlayer, matchEnded match: GKTurnBasedMatch) {
        Task { @MainActor in
            await self.reload()
            if let current = self.current, current.matchID == match.matchID, let model = self.model {
                model.receive(moves: Self.moves(in: match), resigned: Self.friendQuit(in: match))
            }
        }
    }
}

extension GameCenter: GKTurnBasedMatchmakerViewControllerDelegate {
    nonisolated func turnBasedMatchmakerViewControllerWasCancelled(_ vc: GKTurnBasedMatchmakerViewController) {
        Task { @MainActor in vc.dismiss(animated: true) }
    }

    nonisolated func turnBasedMatchmakerViewController(_ vc: GKTurnBasedMatchmakerViewController, didFailWithError error: Error) {
        Task { @MainActor in vc.dismiss(animated: true) }
    }
}
