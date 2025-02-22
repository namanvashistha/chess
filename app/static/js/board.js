const chessBoard = document.getElementById('chess-board');
let legalMoves = []; // To store allowed moves for the clicked piece
let selectedPiece = null; // Track the currently selected piece
let selectedSquare = null; // Track the currently selected square

function bitboardToBits(title, bitboard) {
    const bits = [];
    // Convert BigInt to string, pad it to 64 bits, and split into chunks of 8 bits
    const bitString = bitboard.toString(2).padStart(64, '0');  // Ensuring 64 bits
    console.log(title, bitString);
    // Convert the bit string into an 8x8 grid
    for (let i = 0; i < 8; i++) {
        bits.push(bitString.slice(i * 8, (i + 1) * 8));
    }

    console.table(title, bitboard);
    return bits;
}



// Mapping pieces to image paths
const pieceMap = {
    "P": "/static/images/wP.svg",
    "R": "/static/images/wR.svg",
    "N": "/static/images/wN.svg",
    "B": "/static/images/wB.svg",
    "Q": "/static/images/wQ.svg",
    "K": "/static/images/wK.svg",
    "p": "/static/images/bP.svg",
    "r": "/static/images/bR.svg",
    "n": "/static/images/bN.svg",
    "b": "/static/images/bB.svg",
    "q": "/static/images/bQ.svg",
    "k": "/static/images/bK.svg"
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
            renderChessBoard(data.data);
        })
        .catch(err => console.error('Error fetching chess state:', err));
}

// Render the chessboard
function renderChessBoard(gameData) {
    legalMoves = gameData.legal_moves; // Store the allowed moves
    userData = JSON.parse(localStorage.getItem("userData"));
    if (userData.id === gameData.white_user.id) {
        localStorage.setItem("boardPov", "w");
    } else if (userData.id === gameData.black_user.id) {
        localStorage.setItem("boardPov", "b");
    }

    const renderBitBoard = () => {

        chessBoard.innerHTML = ''; // Clear the chessboard

        if (localStorage.getItem("boardPov") === "b") {
            gameData.board_layout = gameData.board_layout.map(row => row.reverse()).reverse();
        } else if (localStorage.getItem("boardPov") === "w" && gameData.board_layout[0][0] === "h1") {
            gameData.board_layout = gameData.board_layout.map(row => row.reverse()).reverse();
        }
        currentState = gameData.current_state;
        gameData.board_layout.forEach((row, _) => {
            row.forEach((squareInfo, _) => {
                const [squareKey, color] = squareInfo;

                const square = document.createElement('div');
                square.className = `square ${color === 'w' ? 'light' : 'dark'}`;
                // square.textContent = squareKey;
                square.dataset.key = squareKey;
                square.dataset.file = squareKey[0];
                square.dataset.rank = squareKey[1];

                const label = document.createElement('span');
                label.textContent = squareKey;
                label.className = 'square-label';
                square.appendChild(label);

                const pieceCode = currentState[squareKey];
                if (pieceCode) {
                    const piece = document.createElement('span');
                    piece.className = 'piece';
                    piece.dataset.code = pieceCode;
                    const img = document.createElement('img');
                    img.src = pieceMap[pieceCode];
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

    const getAvatarUrl = (name) => `https://avatar.iran.liara.run/username?username=${encodeURIComponent(name)}`;

    const renderPlayerBars = () => {
        const topBar = document.getElementById("player-bar-top");
        const bottomBar = document.getElementById("player-bar-bottom");
        if ((localStorage.getItem("boardPov") || "w") === "w") {
            // White is at the bottom
            topBar.querySelector(".player-dp").src = getAvatarUrl(formatUserName(gameData.black_user.name));
            topBar.querySelector(".player-name").textContent = formatUserName(gameData.black_user.name);
            topBar.querySelector(".turn-indicator").textContent = gameData.state.turn === "b" ? "Move" : "";
            topBar.querySelector(".turn-indicator").style.color = gameData.state.turn === "b" ? "green" : "gray";
            topBar.querySelector(".player-timer").textContent = gameData.state.turn === "b" ? "🟢" : "⏳";
            topBar.style.backgroundColor = "#606c76";
            topBar.style.color = "#f9f9f9";


            bottomBar.querySelector(".player-dp").src = getAvatarUrl(formatUserName(gameData.white_user.name));
            bottomBar.querySelector(".player-name").textContent = formatUserName(gameData.white_user.name);
            bottomBar.querySelector(".turn-indicator").textContent = gameData.state.turn === "w" ? "Move" : "";
            bottomBar.querySelector(".turn-indicator").style.color = gameData.state.turn === "w" ? "green" : "gray";
            bottomBar.querySelector(".player-timer").textContent = gameData.state.turn === "w" ? "🟢" : "⏳";
            bottomBar.style.backgroundColor = "f9f9f9";
            bottomBar.style.color = "#606c76";

        } else {
            // Black is at the bottom
            topBar.querySelector(".player-dp").src = getAvatarUrl(formatUserName(gameData.white_user.name));
            topBar.querySelector(".player-name").textContent = formatUserName(gameData.white_user.name);
            topBar.querySelector(".turn-indicator").textContent = gameData.state.turn === "w" ? "Move" : "";
            topBar.querySelector(".turn-indicator").style.color = gameData.state.turn === "w" ? "green" : "gray";
            topBar.querySelector(".player-timer").textContent = gameData.state.turn === "w" ? "🟢" : "⏳";
            topBar.style.backgroundColor = "#f9f9f9";
            topBar.style.color = "#606c76";



            bottomBar.querySelector(".player-dp").src = getAvatarUrl(formatUserName(gameData.black_user.name));
            bottomBar.querySelector(".player-name").textContent = formatUserName(gameData.black_user.name);
            bottomBar.querySelector(".turn-indicator").textContent = gameData.state.turn === "b" ? "Move" : "";
            bottomBar.querySelector(".turn-indicator").style.color = gameData.state.turn === "b" ? "green" : "gray";
            bottomBar.querySelector(".player-timer").textContent = gameData.state.turn === "b" ? "🟢" : "⏳";
            bottomBar.style.backgroundColor = "#606c76";
            bottomBar.style.color = "#f9f9f9";

        }
    };

    const renderMoves = () => {
        const historyTableBody = document.querySelector("#history-table tbody");
        historyTableBody.innerHTML = ""; // Clear any existing rows
        const moves = gameData.moves;
    
        for (let i = 0; i < moves.length; i += 2) {
            const moveRow = document.createElement("tr");
    
            // Add the move number
            const moveNumberCell = document.createElement("td");
            moveNumberCell.textContent = `${Math.floor(i / 2) + 1}.`;
            moveRow.appendChild(moveNumberCell);
    
            // Add White's move
            const whiteMoveCell = document.createElement("td");
            whiteMoveCell.textContent = moves[i]?.move || ""; // If no move, leave blank
            moveRow.appendChild(whiteMoveCell);
    
            // Add Black's move
            const blackMoveCell = document.createElement("td");
            blackMoveCell.textContent = moves[i + 1]?.move || ""; // If no move, leave blank
            moveRow.appendChild(blackMoveCell);
    
            historyTableBody.appendChild(moveRow);
        }
        const moveHistory = document.getElementById("move-history");
        moveHistory.scrollTop = moveHistory.scrollHeight;
    };
    const highlightLastMove = () => {
        const lastMove = gameData.state.last_move;
        if (!lastMove) return;  // Ensure there's a move to highlight
    
        const source = lastMove.substring(1, 3);  // Extract "from" square (e.g., "a4")
        const target = lastMove.substring(3);      // Extract "to" square (e.g., "c5")

        const sourceSquare = document.querySelector(`.square[data-key="${source}"]`);
        const targetSquare = document.querySelector(`.square[data-key="${target}"]`);

        if (sourceSquare) {
            sourceSquare.classList.add('last-move-highlight-source');
        }
        if (targetSquare) {
            targetSquare.classList.add('last-move-highlight-target');
        }
    }

    const displayWinner = () => {
        const winner = gameData.winner;
        if (!winner) return;
        console.log(winner);
        const winnerMessage = document.getElementById("winner-message");
        const winnerDisplay = document.getElementById("winner-display");
        
        if (winner == "w") {
            winnerMessage.textContent = 'White wins the game!';
        } else if (winner == "b"){
            winnerMessage.textContent = 'Black wins the game!';
        }
        else {
            winnerMessage.textContent = "It's a draw!";
        }
    
        winnerDisplay.style.display = "block";
    }

    renderBitBoard();
    renderPlayerBars();
    renderMoves();
    highlightLastMove();
    displayWinner();
    // document.getElementById("flip-board").addEventListener("click", () => {
    //     boardPov = localStorage.getItem("boardPov") === "w";
    //     localStorage.setItem("boardPov", boardPov ? "b" : "w");
    //     renderBitBoard();
    //     renderPlayerBars();
    // });
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
    const pieceMoves = legalMoves[piece.parentElement.dataset.key];

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
