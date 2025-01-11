let socket;
let reconnectInterval = 5000; // 5 seconds
const moveSound = new Audio('https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/move-opponent.mp3');

// Initialize WebSocket connection
function createWebSocket() {
    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    socket = new WebSocket(`${protocol}://${window.location.host}/ws/${gameId}`);

    socket.onopen = () => {
        console.log("Connected to WebSocket server.");
    };

    socket.onmessage = (event) => {
        const message = JSON.parse(event.data);
        if (message.status === "error") {
            console.log("Error:", message.message);
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
            game_id: gameId,
            token: localStorage.getItem('userToken'),
        }
    };

    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(data));
    } else {
        console.log("WebSocket is not open, cannot send move.");
    }
}

// Function to update the chessboard based on the received message
function updateChessBoard(message) {
    // Update the UI based on the move received
    const chessState = message.payload.chess_state;

    // Assuming `renderChessBoard` is defined elsewhere in your code
    renderChessBoard(message.payload);
    if (message.status === "success") {
        moveSound.play().catch((err) => {
            console.error("Error playing move sound:", err);
        });
    }
}

// Initialize WebSocket connection
createWebSocket();
