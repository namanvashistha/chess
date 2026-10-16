import Chesscore
import Messages
import SwiftUI

/// Cozy Chess inside Messages. Each move is a message bubble with a picture of
/// the board; the game itself travels in the bubble's link, so there's no
/// server and nothing leaves the conversation.
final class MessagesViewController: MSMessagesAppViewController {
    private let model = GameModel()
    private lazy var link = MessageLink(controller: self, model: model)
    private var host: UIHostingController<MessagesRoot>?

    override func viewDidLoad() {
        super.viewDidLoad()
        let root = UIHostingController(rootView: MessagesRoot(model: model, link: link, expanded: false))
        root.view.backgroundColor = .clear
        addChild(root)
        root.view.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(root.view)
        NSLayoutConstraint.activate([
            root.view.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            root.view.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            root.view.topAnchor.constraint(equalTo: view.topAnchor),
            root.view.bottomAnchor.constraint(equalTo: view.bottomAnchor),
        ])
        root.didMove(toParent: self)
        host = root
    }

    override func willBecomeActive(with conversation: MSConversation) {
        super.willBecomeActive(with: conversation)
        if let message = conversation.selectedMessage { link.open(message, in: conversation) }
        render()
    }

    override func didSelect(_ message: MSMessage, conversation: MSConversation) {
        super.didSelect(message, conversation: conversation)
        link.open(message, in: conversation)
    }

    override func didReceive(_ message: MSMessage, conversation: MSConversation) {
        super.didReceive(message, conversation: conversation)
        link.received(message, in: conversation)
    }

    override func didTransition(to style: MSMessagesAppPresentationStyle) {
        super.didTransition(to: style)
        render()
    }

    private func render() {
        host?.rootView = MessagesRoot(model: model, link: link, expanded: presentationStyle == .expanded)
    }
}

/// Reads and writes games as message links: `moves` (UCI, comma-separated),
/// `white` (the participant playing White) and `resigned`.
@MainActor
final class MessageLink: MatchLink {
    private weak var controller: MSMessagesAppViewController?
    private let model: GameModel
    private var session: MSSession?
    private var white: String = ""

    init(controller: MSMessagesAppViewController, model: GameModel) {
        self.controller = controller
        self.model = model
    }

    private var conversation: MSConversation? { controller?.activeConversation }

    /// Starts a new game: this player is White and moves first.
    func newGame() {
        guard let conversation else { return }
        session = MSSession()
        white = conversation.localParticipantIdentifier.uuidString
        model.startFriend(name: GameModel.unnamedFriend, color: "w", moves: [], link: self)
        controller?.requestPresentationStyle(.expanded)
    }

    /// Opens the game in a bubble the player tapped.
    func open(_ message: MSMessage, in conversation: MSConversation) {
        guard let game = Self.parse(message) else { return }
        session = message.session
        white = game.white
        let color = game.white == conversation.localParticipantIdentifier.uuidString ? "w" : "b"
        model.startFriend(name: GameModel.unnamedFriend, color: color, moves: game.moves, link: self)
        if game.resigned, message.senderParticipantIdentifier != conversation.localParticipantIdentifier {
            model.receive(moves: game.moves, resigned: true)
        }
    }

    /// A move arrived while the game is open.
    func received(_ message: MSMessage, in conversation: MSConversation) {
        guard let game = Self.parse(message), message.session == session else { return }
        model.receive(moves: game.moves, resigned: game.resigned)
    }

    // MARK: MatchLink

    func didMove(_ model: GameModel) { send(resigned: false) }
    func didResign(_ model: GameModel) { send(resigned: true) }
    func rematch(_ model: GameModel) { newGame() }

    /// Puts the move in the message field as a bubble; the player taps send.
    private func send(resigned: Bool) {
        guard let conversation else { return }
        let message = MSMessage(session: session ?? MSSession())
        var parts = URLComponents()
        parts.queryItems = [URLQueryItem(name: "moves", value: model.moves.joined(separator: ",")),
                            URLQueryItem(name: "white", value: white),
                            URLQueryItem(name: "resigned", value: resigned ? "1" : "0")]
        message.url = parts.url

        let layout = MSMessageTemplateLayout()
        let san = model.moves.isEmpty ? "" : ChesscoreSANs(model.moves)
        let title: String
        if resigned { title = L("I resigned. Good game!") }
        else if model.isOver { title = model.status == ChesscoreStatusCheckmate ? L("Checkmate!") : L("It's a draw") }
        else { title = L("Your move") }
        if let data = ShareMoment(board: model.board, move: model.lastMove, title: title,
                                  detail: san, footer: "Cozy Chess").boardPNG() {
            layout.image = UIImage(data: data)
        }
        layout.caption = title
        layout.subcaption = san.isEmpty ? "Cozy Chess" : L("\(san) · Cozy Chess")
        message.layout = layout
        message.summaryText = L("Played \(san)")

        conversation.insert(message) { _ in }
        // Back to the chat, where the move waits in the field to be sent.
        Task {
            try? await Task.sleep(for: .seconds(1.2))
            self.controller?.requestPresentationStyle(.compact)
        }
    }

    private static func parse(_ message: MSMessage) -> (moves: [String], white: String, resigned: Bool)? {
        guard let url = message.url, let items = URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems else { return nil }
        func value(_ k: String) -> String? { items.first { $0.name == k }?.value }
        let moves = (value("moves") ?? "").split(separator: ",").map(String.init)
        return (moves, value("white") ?? "", value("resigned") == "1")
    }
}

/// The last move, in notation, for the bubble.
private func ChesscoreSANs(_ moves: [String]) -> String {
    guard let g = ChesscoreNewGame() else { return "" }
    for m in moves.dropLast() { _ = g.play(m) }
    return moves.last.map { g.san($0) } ?? ""
}

/// What the extension shows: a small invitation in the keyboard space, or the
/// whole game screen when expanded.
struct MessagesRoot: View {
    let model: GameModel
    let link: MessageLink
    let expanded: Bool

    var body: some View {
        ZStack {
            AppBackground()
            if expanded && model.mode == .friend {
                GameScreen(model: model) { link.collapse() }
            } else {
                VStack(spacing: 14) {
                    Text("Cozy Chess").font(Typeface.display(22))
                    Text("A calm game of chess, one move at a time.")
                        .font(.system(size: 14)).foregroundStyle(Palette.mist)
                    Button("Start a game") { Haptics.tap(); link.newGame() }
                        .buttonStyle(PrimaryButtonStyle())
                        .padding(.horizontal, 40)
                }
                .foregroundStyle(Palette.ink)
                .padding()
            }
        }
    }
}

extension MessageLink {
    func collapse() { controller?.requestPresentationStyle(.compact) }
}
