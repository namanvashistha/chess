const chessBoard = document.getElementById('chess-board');
const currentTurn = document.getElementById('current-turn');
let allowedMoves = []; // To store allowed moves for the clicked piece
let selectedPiece = null; // Track the currently selected piece
let selectedSquare = null; // Track the currently selected square
let flip = false;

// Mapping pieces to Unicode symbols
const pieceMap = {
    "wP": "static/images/wP.svg",
    "wR": "static/images/wR.svg",
    "wN": "static/images/wN.svg",
    "wB": "static/images/wB.svg",
    "wQ": "static/images/wQ.svg",
    "wK": "static/images/wK.svg",
    "bP": "static/images/bP.svg",
    "bR": "static/images/bR.svg",
    "bN": "static/images/bN.svg",
    "bB": "static/images/bB.svg",
    "bQ": "static/images/bQ.svg",
    "bK": "static/images/bK.svg"
};

function getURLParameter(name) {
    const params = new URLSearchParams(window.location.search);
    return params.get(name);
}
const gameId = getURLParameter('game_id');
// console.log('gameId:', gameId);

console.log('gameId:', gameId);
// Fetch initial chessboard state from the API
fetch(`/api/chess/state/${gameId}`)
    .then(response => {
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    })
    .then(data => {
        if (data && data.data && data.data.board) {
            renderChessBoard(data.data.board, data.data.board_layout, data.data.allowed_moves, data.data.turn);
        } else {
            console.error('Unexpected API response:', data);
        }
    })
    .catch(err => console.error('Error fetching chess state:', err));

// Render the chessboard with drag-and-drop functionality
function renderChessBoard(board, boardLayout, allowedMovesData, turn) {
    chessBoard.innerHTML = ''; // Clear the chessboard
    console.log('turn:', turn);
    currentTurn.innerHTML = turn;
    const flipbutton = document.createElement('button');
    flipbutton.textContent = "FLIP!!";
    currentTurn.appendChild(flipbutton);
    flipbutton.addEventListener('click', () => {
        console.log('flip');
        flip = !flip;
        renderChessBoard(board, boardLayout, allowedMovesData, turn);
    })
    allowedMoves = allowedMovesData; // Store the allowed moves data

    if (flip) {
        boardLayout = boardLayout.map(row => row.reverse());
        boardLayout = boardLayout.reverse();
    }

    // console.log(boardLayout);
    boardLayout.forEach((row, i) => {
        row.forEach((cell, j) => {
            const squareKey = cell; // Get square notation like 'a1'
            const squareData = board[squareKey]; // Get data for this square
            const [color, pieceCode] = squareData;

            const square = document.createElement('div');
            square.className = `square ${color === 'w' ? 'light' : 'dark'}`;
            square.dataset.key = squareKey;
            square.dataset.file = squareKey[0]; // File (a-h)
            square.dataset.rank = squareKey[1]; // Rank (1-8)

            // Create a small label for the squareKey
            const label = document.createElement('span');
            label.className = 'square-label';
            label.textContent = squareKey;
            square.appendChild(label);
            square.addEventListener('dragover', handleDragOver);
            square.addEventListener('drop', handleDrop);
            // Add the piece if it exists
            if (pieceCode !== "---") {
                const piece = document.createElement('span');
                piece.className = 'piece';
                // console.log('Piece map:', pieceCode.slice(0, -1), pieceMap[pieceCode.slice(0, -1)]);
                const pieceImage = pieceMap[pieceCode.slice(0, -1)];
                if (pieceImage) {
                    const img = document.createElement('img');
                    img.src = pieceImage; // Assign image source
                    img.alt = "";  // Use piece code as alt text for accessibility
                    img.className = 'piece-image'; // Optional: Add CSS class for styling
                    img.dataset.code = pieceCode
                    // Set the image as draggable
                    img.draggable = true; // Make the image draggable

                    // // Add drag event listener to the image directly
                    img.addEventListener('dragstart', handleDragStart);

                    piece.appendChild(img);
                } else {
                    // Fallback: Render text if no image found
                    piece.textContent = "";
                }
                piece.dataset.code = pieceCode;
                piece.draggable = true;

                // Add event listener for piece click to highlight allowed moves
                piece.addEventListener('click', () => handlePieceClick(piece, square));

                // Drag event listeners
                piece.addEventListener('dragstart', handleDragStart);
                

                square.appendChild(piece);
            }

            chessBoard.appendChild(square);
        });
    });
}

// Handle piece click - highlight allowed moves
function handlePieceClick(piece, square) {
    // Deselect previous selected piece if any
    if (selectedPiece) {
        clearHighlightedSquares();
    }
    if (selectedSquare) {
        selectedSquare.style.border = '';
    }

    // Highlight the clicked piece's allowed moves
    selectedPiece = piece;
    selectedSquare = square;
    console.log('Selected piece:', selectedPiece);
    console.log('Selected square:', selectedSquare);
    selectedSquare.style.border = '2px solid red'; // Optional: highlight the selected piece
    highlightAllowedMoves(selectedPiece);
}

// Highlight allowed move squares
function highlightAllowedMoves(selectedPiece) {
    // const piecePosition = squareKey; // e.g., "b1", "a2", etc.
    const pieceMoves = allowedMoves[selectedPiece.dataset.code];

    if (!pieceMoves) {
        console.log("No allowed moves found for piece:", selectedPiece); // Debugging line
        return;
    }

    // Clear previously highlighted squares
    clearHighlightedSquares();

    pieceMoves.forEach(move => {
        const targetSquare = document.querySelector(
            `.square[data-file="${move[0]}"][data-rank="${move[1]}"]`
        );
        if (targetSquare) {
            targetSquare.classList.add('highlight');
        }
    });
}


// Clear highlighted squares
function clearHighlightedSquares() {
    const highlightedSquares = document.querySelectorAll('.highlight');
    highlightedSquares.forEach(square => {
        square.classList.remove('highlight');
    });
}

// Get piece code from the square
function getPieceCodeAtSquare(squareKey) {
    const squareData = chessBoard.querySelector(
        `.square[data-file="${squareKey[0]}"][data-rank="${squareKey[1]}"]`
    );
    console.log('Square data:', squareData);
    return pieceMap[squareData];
}

function handleDragStart(event) {
    const piece = event.target;
    var square = event.target.parentElement;
    const pieceCode = piece.dataset.code;
    if (!square.dataset.file) {
        square = square.parentElement;
    }

    // Set the drag data with the piece code, and square positions
    event.dataTransfer.setData('text/plain', JSON.stringify({
        file: square.dataset.file,
        rank: square.dataset.rank,
        pieceCode: pieceCode
    }));

    // piece.style.opacity = 0.5;
}

// Allow dropping
function handleDragOver(event) {
    event.preventDefault(); // Required to allow dropping
}

// Handle drop and make API call
function handleDrop(event) {
    event.preventDefault();

    // Retrieve the dragged data (file, rank, and piece code)
    const draggedData = JSON.parse(event.dataTransfer.getData('text/plain'));
    const sourceFile = draggedData.file;
    const sourceRank = draggedData.rank;
    const pieceCode = draggedData.pieceCode;

    // Determine the target square (either directly or via parent element)
    var targetSquare = event.target.classList.contains('square')
        ? event.target
        : event.target.parentElement;
    if (!targetSquare.dataset.file) {
        targetSquare = targetSquare.parentElement;
    }
    const targetFile = targetSquare.dataset.file;
    const targetRank = targetSquare.dataset.rank;
    console.log('Moving: '+ sourceFile +"|"+ sourceRank + " -> "+ (targetFile + targetRank))
    // Check if the move is allowed (target square is within allowed moves for the piece)
    
    if (allowedMoves[pieceCode] && allowedMoves[pieceCode].some(move => move[0] === targetFile && move[1] === targetRank)) {
        
        // Check if the target square is empty or contains an opponent's piece
        const targetPiece = targetSquare.querySelector('.piece');
        if (targetPiece && targetPiece.dataset.code === pieceCode) {
            console.log('Cannot move to the square as it already contains your own piece');
            return; // Can't move to a square containing your own piece
        }

        // Proceed with the move (move piece visually)
        movePieceOnBoard(sourceFile, sourceRank, targetFile, targetRank, pieceCode);

        // Make API call to update the game state
        makeMoveAPICall(pieceCode, sourceFile + sourceRank, targetFile + targetRank);
    } else {
        console.log('Move not allowed: '+ (sourceFile + sourceRank) + " -> "+ (targetFile + targetRank));
    }
}



function movePieceOnBoard(sourceFile, sourceRank, targetFile, targetRank, pieceCode) {
    const sourceSquare = document.querySelector(
        `.square[data-file="${sourceFile}"][data-rank="${sourceRank}"]`
    );
    const targetSquare = document.querySelector(
        `.square[data-file="${targetFile}"][data-rank="${targetRank}"]`
    );
    console.log('Source square:', sourceSquare);
    console.log('Target square:', targetSquare);

    // Clear the source square
    sourceSquare.innerHTML = '';

    // Create a new piece element and place it on the target square
    const newPiece = document.createElement('span');
    newPiece.className = 'piece';
    newPiece.textContent = pieceMap[pieceCode.slice(0, -1)];
    newPiece.dataset.code = pieceCode;
    targetSquare.appendChild(newPiece);
}

// Make API call to update the game state
function makeMoveAPICall(pieceCode, sourceSquare, destinationSquare) {
    sendMove(pieceCode, sourceSquare, destinationSquare);
    // fetch(`/api/chess/state/${gameId}`)
    // .then(response => {
    //     if (!response.ok) {
    //         throw new Error(`HTTP error! status: ${response.status}`);
    //     }
    //     return response.json();
    // })
    // .then(data => {
    //     if (data && data.data && data.data.board) {
    //         renderChessBoard(data.data.board, data.data.board_layout, data.data.allowed_moves, data.data.turn);
    //     } else {
    //         console.error('Unexpected API response:', data);
    //     }
    // })

    // fetch('/api/chess/state/move', {
    //     method: 'POST',
    //     headers: {
    //         'Content-Type': 'application/json'
    //     },
    //     body: JSON.stringify({
    //         piece: pieceCode,
    //         source: sourceSquare,
    //         destination: destinationSquare,
    //         game_id: gameId
    //     })
    // })
    //     .then(response => response.json())
    //     .then(data => {
    //         sendMove(pieceCode, sourceSquare, destinationSquare);
    //         fetch(`/api/chess/state/${gameId}`)
    //         .then(response => {
    //             if (!response.ok) {
    //                 throw new Error(`HTTP error! status: ${response.status}`);
    //             }
    //             return response.json();
    //         })
    //         .then(data => {
    //             if (data && data.data && data.data.board) {
    //                 renderChessBoard(data.data.board, data.data.board_layout, data.data.allowed_moves, data.data.turn);
    //             } else {
    //                 console.error('Unexpected API response:', data);
    //             }
    //         })
    //         .catch(err => console.error('Error fetching chess state:', err));
    //             console.log('Move successfully made:', data);
    //         })
    //     .catch(err => console.log('Error making move:', err));
    
}
