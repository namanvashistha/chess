const chessBoard = document.getElementById('chess-board');

// Mapping pieces to Unicode symbols
const pieceMap = {
    wR: "♖", wN: "♘", wB: "♗", wQ: "♕", wK: "♔", wP: "♙",
    bR: "♜", bN: "♞", bB: "♝", bQ: "♛", bK: "♚", bP: "♟",
    "---": "",
};

// Fetch initial chessboard state from the API
fetch('/api/chess/state')
    .then(response => {
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    })
    .then(data => {
        if (data && data.data && data.data.board) {
            renderChessBoard(data.data.board, data.data.board_layout);
        } else {
            console.error('Unexpected API response:', data);
        }
    })
    .catch(err => console.error('Error fetching chess state:', err));

// Render the chessboard with drag-and-drop functionality
function renderChessBoard(board, boardLayout) {
    chessBoard.innerHTML = ''; // Clear the chessboard

    boardLayout.forEach((row, i) => {
        row.forEach((cell, j) => {
            const squareKey = cell; // Get square notation like 'a1'
            const squareData = board[squareKey]; // Get data for this square
            const [color, pieceCode] = squareData;

            const square = document.createElement('div');
            square.className = `square ${color === 'w' ? 'light' : 'dark'}`;
            square.dataset.file = squareKey[0]; // File (a-h)
            square.dataset.rank = squareKey[1]; // Rank (1-8)

            // Create a small label for the squareKey
            const label = document.createElement('span');
            label.className = 'square-label';
            label.textContent = squareKey;
            square.appendChild(label);

            // Add the piece if it exists
            if (pieceCode !== "---") {
                const piece = document.createElement('span');
                piece.className = 'piece';
                piece.textContent = pieceMap[pieceCode.slice(0, -1)];
                piece.draggable = true;

                // Drag event listeners
                piece.addEventListener('dragstart', handleDragStart);
                square.addEventListener('dragover', handleDragOver);
                square.addEventListener('drop', handleDrop);

                square.appendChild(piece);
            }

            chessBoard.appendChild(square);
        });
    });
}

// Handle drag start
function handleDragStart(event) {
    const square = event.target.parentElement;
    event.dataTransfer.setData('text/plain', JSON.stringify({
        file: square.dataset.file,
        rank: square.dataset.rank,
        piece: event.target.textContent
    }));
}

// Allow dropping
function handleDragOver(event) {
    event.preventDefault(); // Required to allow dropping
}

// Handle drop
function handleDrop(event) {
    event.preventDefault();

    const draggedData = JSON.parse(event.dataTransfer.getData('text/plain'));
    const sourceFile = draggedData.file;
    const sourceRank = draggedData.rank;
    const piece = draggedData.piece;

    const targetSquare = event.target.classList.contains('square')
        ? event.target
        : event.target.parentElement;
    const targetFile = targetSquare.dataset.file;
    const targetRank = targetSquare.dataset.rank;

    // Clear the source square
    const sourceSquare = document.querySelector(
        `.square[data-file="${sourceFile}"][data-rank="${sourceRank}"]`
    );
    sourceSquare.innerHTML = '';

    // Place the piece in the target square
    const newPiece = document.createElement('span');
    newPiece.className = 'piece';
    newPiece.textContent = piece;
    newPiece.draggable = true;

    // Re-attach drag listeners
    newPiece.addEventListener('dragstart', handleDragStart);
    targetSquare.appendChild(newPiece);

    console.log(`Moved ${piece} from ${sourceFile}${sourceRank} to ${targetFile}${targetRank}`);
}
