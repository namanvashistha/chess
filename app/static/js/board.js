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
console.log('gameId:', gameId);

// Fetch initial chessboard state from the API
function fetchChessState() {
    fetch(`/api/chess/state/${gameId}`)
        .then(response => response.json())
        .then(data => {
            renderChessBoard(data.data.board, data.data.board_layout, data.data.allowed_moves, data.data.turn);
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
    console.log('Drop:', targetSquare);
    if (targetSquare) {
        movePieceOnBoard(data.file, data.rank, targetSquare.dataset.file, targetSquare.dataset.rank, data.pieceCode);
        sendMove(data.pieceCode, data.file + data.rank, targetSquare.dataset.file + targetSquare.dataset.rank);
    }
}

// Piece click handlers
function handlePieceClick(piece, square) {
    if (selectedPiece) {
        clearHighlightedSquares();
    }

    selectedPiece = piece;
    selectedSquare = square;
    highlightAllowedMoves(selectedPiece);
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
        }
    });
}

function clearHighlightedSquares() {
    document.querySelectorAll('.highlight').forEach(square => square.classList.remove('highlight'));
}

// Move piece with animation
function movePieceOnBoard(sourceFile, sourceRank, targetFile, targetRank, pieceCode) {
    const sourceSquare = document.querySelector(`.square[data-file="${sourceFile}"][data-rank="${sourceRank}"]`);
    const targetSquare = document.querySelector(`.square[data-file="${targetFile}"][data-rank="${targetRank}"]`);
    const pieceElement = sourceSquare.querySelector('.piece');

    if (pieceElement) {
        targetSquare.appendChild(pieceElement);
        sourceSquare.innerHTML = ''; // Clear the source square
    }
}

// Fetch and render the chessboard initially
fetchChessState();
