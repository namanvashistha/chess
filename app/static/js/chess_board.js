// Fetch the chess state from the API
fetch('/api/chess/state')
    .then(response => response.json())
    .then(data => renderChessBoard(data.data.board))
    .catch(err => console.error('Error fetching chess state:', err));

// Function to render the chessboard
function renderChessBoard(board) {
    const chessBoard = document.getElementById('chess-board');

    // Clear existing board
    chessBoard.innerHTML = '';

    // Render the chessboard
    for (let i = 0; i < board.length; i++) {
        for (let j = 0; j < board[i].length; j++) {
            const cell = document.createElement('div');
            const isWhite = (i + j) % 2 === 0;
            cell.className = `chess-cell ${isWhite ? 'white' : 'black'}`;

            // Add the piece symbol if present
            const piece = board[i][j];
            if (piece) {
                cell.textContent = piece;
            }

            chessBoard.appendChild(cell);
        }
    }
}
