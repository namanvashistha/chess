let socket;
let reconnectInterval = 5000; // 5 seconds

// Initialize WebSocket connection
function createWebSocket() {
    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    socket = new WebSocket(`${protocol}://${window.location.host}/ws`);

    socket.onopen = () => {
        console.log("Connected to WebSocket server.");
    };

    socket.onmessage = (event) => {
        const message = JSON.parse(event.data);
        console.log("Move received:", message);
        if (message.status === "error") {
            console.log("Error:", message.status);
        }
        updateChessBoard(message);
    };

    socket.onerror = (error) => {
        console.error("WebSocket Error: ", error);
    };

    socket.onclose = (event) => {
        console.log("WebSocket connection closed:", event);
        // Automatically attempt to reconnect if the connection is closed
        if (!event.wasClean) {
            console.log("Reconnecting WebSocket...");
            setTimeout(createWebSocket, reconnectInterval);
        }
    };
}

// Function to send a move to the server
function sendMove(piece, source, destination) {
    const data = {
        type: "game_update",
        payload: {
            piece: piece,
            source: source,
            destination: destination,
            game_id: gameId
        }
    };
    console.log("Sending move:", data);

    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(data));
    } else {
        console.log("WebSocket is not open, cannot send move.");
    }
}

// Function to update the chessboard based on the received message
function updateChessBoard(message) {
    // Update the UI based on the move received
    console.log("Updating board for move:", message);
    const chessState = message.payload.chess_state;

    // Assuming `renderChessBoard` is defined elsewhere in your code
    renderChessBoard(chessState.board, chessState.board_layout, chessState.allowed_moves, chessState.turn);
}

// Initialize WebSocket connection
createWebSocket();
