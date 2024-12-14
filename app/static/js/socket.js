const socket = new WebSocket(`${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}/ws`);

socket.onopen = () => {
    console.log("Connected to WebSocket server.");
};

socket.onmessage = (event) => {
    const message = JSON.parse(event.data);
    console.log("Move received:", message);
    updateChessBoard(message);
};

socket.onclose = () => {
    console.log("WebSocket connection closed.");
};

function sendMove(piece, source, destination) {
    data = {
        type: "game_update",
        payload: {
            piece: piece,
            source: source,
            destination: destination,
            game_id: gameId
        }
    }
    console.log("Sending move:", data);
    response = socket.send(JSON.stringify(data));
    console.log("Response:", response);
}

function updateChessBoard(message) {
    // Update the UI based on the move received
    console.log("Updating board for move:", message);
    renderChessBoard(message.payload.board, message.payload.board_layout, message.payload.allowed_moves, message.payload.turn);
}
