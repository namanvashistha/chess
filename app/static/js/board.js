const chessBoard = document.getElementById('chess-board');
const currentTurn = document.getElementById('current-turn');
let allowedMoves = []; // To store allowed moves for the clicked piece
let selectedPiece = null; // Track the currently selected piece
let selectedSquare = null; // Track the currently selected square
let flip = false;

// Mapping pieces to image paths
const pieceMap = {
    "wP": "/static/images/wP.svg",
    "wR": "/static/images/wR.svg",
    "wN": "/static/images/wN.svg",
    "wB": "/static/images/wB.svg",
    "wQ": "/static/images/wQ.svg",
    "wK": "/static/images/wK.svg",
    "bP": "/static/images/bP.svg",
    "bR": "/static/images/bR.svg",
    "bN": "/static/images/bN.svg",
    "bB": "/static/images/bB.svg",
    "bQ": "/static/images/bQ.svg",
    "bK": "/static/images/bK.svg"
};

// Function to extract gameId from the URL path
function getGameIdFromURL() {
    const pathSegments = window.location.pathname.split('/');
    return pathSegments[pathSegments.length - 1]; // Assumes game ID is the last segment
}

const gameId = getGameIdFromURL();

// Fetch initial chessboard state from the API
function fetchChessState() {
    fetch(`/api/chess/game/${gameId}`)
        .then(response => response.json())
        .then(data => {
            chessState = data.data.chess_state;
            renderChessBoard(chessState.board, chessState.board_layout, chessState.allowed_moves, chessState.turn);
        })
        .catch(err => console.error('Error fetching chess state:', err));
}

// Render the chessboard
function renderChessBoard(board, boardLayout, allowedMovesData, turn) {
    chessBoard.innerHTML = ''; // Clear the chessboard
    currentTurn.innerHTML = `Current Turn: ${turn}`;

    allowedMoves = allowedMovesData; // Store the allowed moves

    if (flip) {
        boardLayout = boardLayout.map(row => row.reverse()).reverse();
    }

    boardLayout.forEach((row, i) => {
        row.forEach((squareKey, j) => {
            const squareData = board[squareKey];
            const [color, pieceCode] = squareData;

            const square = document.createElement('div');
            square.className = `square ${color === 'w' ? 'light' : 'dark'}`;
            square.dataset.key = squareKey;
            square.dataset.file = squareKey[0];
            square.dataset.rank = squareKey[1];

            if (pieceCode !== "---") {
                const piece = document.createElement('span');
                piece.className = 'piece';
                piece.dataset.code = pieceCode;
                const img = document.createElement('img');
                img.src = pieceMap[pieceCode.slice(0, -1)];
                img.alt = pieceCode;
                img.className = 'piece-image';
                img.draggable = true;
                img.dataset.code = pieceCode;

                img.addEventListener('dragstart', handleDragStart);
                piece.addEventListener('click', () => handlePieceClick(piece, square));

                piece.appendChild(img);
                square.appendChild(piece);
            }

            square.addEventListener('dragover', handleDragOver);
            square.addEventListener('drop', handleDrop);
            chessBoard.appendChild(square);
        });
    });
}

// Drag event handlers
function handleDragStart(event) {
    const piece = event.target;
    const square = piece.closest('.square');
    event.dataTransfer.setData('text/plain', JSON.stringify({
        pieceCode: piece.dataset.code,
        file: square.dataset.file,
        rank: square.dataset.rank
    }));
}

function handleDragOver(event) {
    event.preventDefault();
}

function handleDrop(event) {
    event.preventDefault();
    const data = JSON.parse(event.dataTransfer.getData('text/plain'));
    const targetSquare = event.target.closest('.square');
    if (targetSquare) {
        movePieceOnBoard(data.file, data.rank, targetSquare.dataset.file, targetSquare.dataset.rank, data.pieceCode);
        sendMove(data.pieceCode, data.file + data.rank, targetSquare.dataset.file + targetSquare.dataset.rank);
    }
}

// Piece click handlers
function handlePieceClick(piece, square) {
    if (selectedPiece === piece) {
        // If the same piece is clicked again, deselect it
        clearHighlightedSquares();
        selectedPiece = null;
        selectedSquare = null;
        return;
    }
    if (selectedPiece) {
        const isHighlighted = square.classList.contains('highlight');
        if (isHighlighted) {
            // Move the piece to the clicked square
            const sourceFile = selectedSquare.dataset.file;
            const sourceRank = selectedSquare.dataset.rank;
            const targetFile = square.dataset.file;
            const targetRank = square.dataset.rank;
            const pieceCode = selectedPiece.querySelector('img').dataset.code;
            movePieceOnBoard(sourceFile, sourceRank, targetFile, targetRank, pieceCode);
            sendMove(pieceCode, sourceFile + sourceRank, targetFile + targetRank);
            clearHighlightedSquares();
        } else {
            // Clear the previously selected piece and highlight the new piece
            clearHighlightedSquares();
            selectedPiece = piece;
            selectedSquare = square;
            highlightAllowedMoves(selectedPiece);
        }
        
    } else {
        selectedPiece = piece;
        selectedSquare = square;
        highlightAllowedMoves(selectedPiece);
    }
}

function highlightAllowedMoves(piece) {
    const pieceMoves = allowedMoves[piece.dataset.code];

    if (!pieceMoves) {
        console.log("No allowed moves for this piece.");
        return;
    }

    pieceMoves.forEach(move => {
        const targetSquare = document.querySelector(
            `.square[data-file="${move[0]}"][data-rank="${move[1]}"]`
        );
        if (targetSquare) {
            targetSquare.classList.add('highlight');
            targetSquare.addEventListener('click', handleSquareClick);
        }
    });
}

// Clear highlighted squares
function clearHighlightedSquares() {
    document.querySelectorAll('.highlight').forEach(square => {
        square.classList.remove('highlight');
        square.removeEventListener('click', handleSquareClick);
    });
}

// Handle clicking on highlighted squares
function handleSquareClick(event) {
    const targetSquare = event.target.closest('.square');

    if (selectedPiece && selectedSquare && targetSquare) {
        const sourceFile = selectedSquare.dataset.file;
        const sourceRank = selectedSquare.dataset.rank;
        const targetFile = targetSquare.dataset.file;
        const targetRank = targetSquare.dataset.rank;
        const pieceCode = selectedPiece.querySelector('img').dataset.code;

        movePieceOnBoard(sourceFile, sourceRank, targetFile, targetRank, pieceCode);
        sendMove(pieceCode, sourceFile + sourceRank, targetFile + targetRank);

        // Clear selection after moving
        selectedPiece = null;
        selectedSquare = null;
        clearHighlightedSquares();
    }
}

// Move piece with animation
function movePieceOnBoard(sourceFile, sourceRank, targetFile, targetRank, pieceCode) {
    const sourceSquare = document.querySelector(`.square[data-file="${sourceFile}"][data-rank="${sourceRank}"]`);
    const targetSquare = document.querySelector(`.square[data-file="${targetFile}"][data-rank="${targetRank}"]`);
    const pieceElement = sourceSquare.querySelector('.piece');

    // If the target square already has a piece, clear it before placing the new piece
    // const existingPiece = targetSquare.querySelector('.piece');
    // if (existingPiece) {
    //     targetSquare.removeChild(existingPiece); // Remove the existing piece
    // }

    if (pieceElement) {
        // targetSquare.appendChild(pieceElement); // Move the selected piece to the target square
        sourceSquare.innerHTML = ''; // Clear the source square
    }
}

// Fetch and render the chessboard initially
fetchChessState();
