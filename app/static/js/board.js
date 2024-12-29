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
            renderChessBoard(
                data.data,
                chessState.board,
                chessState.board_layout,
                chessState.allowed_moves,
                data.data.white_user,
                data.data.black_user,
                chessState.turn
            );
        })
        .catch(err => console.error('Error fetching chess state:', err));
}

// Render the chessboard
function renderChessBoard(gameData, board, boardLayout, legalMovesData, whiteUser, blackUser, turn) {
    legalMoves = gameData.legal_moves; // Store the allowed moves
    userData = JSON.parse(localStorage.getItem("userData"));
    if (userData.id === whiteUser.id) {
        localStorage.setItem("boardPov", "white");
    } else if (userData.id === blackUser.id) {
        localStorage.setItem("boardPov", "black");
    }

    const renderBitBoard = () => {

        chessBoard.innerHTML = ''; // Clear the chessboard

        if (localStorage.getItem("boardPov") === "black") {
            boardLayout = boardLayout.map(row => row.reverse()).reverse();
        } else if (localStorage.getItem("boardPov") === "white" && boardLayout[0][0] === "h1") {
            boardLayout = boardLayout.map(row => row.reverse()).reverse();
        }
        currentState = gameData.current_state;
        boardLayout.forEach((row, _) => {
            row.forEach((squareInfo, _) => {
                const [squareKey, color] = squareInfo;

                const square = document.createElement('div');
                square.className = `square ${color === 'w' ? 'light' : 'dark'}`;
                square.textContent = squareKey;
                square.dataset.key = squareKey;
                square.dataset.file = squareKey[0];
                square.dataset.rank = squareKey[1];

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

    const renderBoard = () => {
        
        chessBoard.innerHTML = ''; // Clear the chessboard
        if (localStorage.getItem("boardPov") === "black") {
            boardLayout = boardLayout.map(row => row.reverse()).reverse();
        }
        else if (localStorage.getItem("boardPov") === "white" && boardLayout[0][0] == "h1") {
            boardLayout = boardLayout.map(row => row.reverse()).reverse();
        }

        boardLayout.forEach((row, i) => {
            row.forEach((squareInfo, j) => {
                const [squareKey, color] = squareInfo;
                const pieceCode = board[squareKey];

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
    };

    // const renderBitBoard = () => {

    //     chessBoard.innerHTML = '';
    //     if (localStorage.getItem("boardPov") === "black") {
    //         boardLayout = boardLayout.map(row => row.reverse()).reverse();
    //     }
    //     else if (localStorage.getItem("boardPov") === "white" && boardLayout[0][0] == "h1") {
    //         boardLayout = boardLayout.map(row => row.reverse()).reverse();
    //     }

    //     boardLayout.forEach((row, i) => {
    //         row.forEach((squareKey, j) => {
    //             const squareData = board[squareKey];
    //             const [color, pieceCode] = squareData;

    //             const square = document.createElement('div');
    //             square.className = `square ${color === 'w' ? 'light' : 'dark'}`;
    //             square.dataset.key = squareKey;
    //             square.dataset.file = squareKey[0];
    //             square.dataset.rank = squareKey[1];

    //             if (pieceCode !== "---") {
    //                 const piece = document.createElement('span');
    //                 piece.className = 'piece';
    //                 piece.dataset.code = pieceCode;
    //                 const img = document.createElement('img');
    //                 img.src = pieceMap[pieceCode.slice(0, -1)];
    //                 img.alt = pieceCode;
    //                 img.className = 'piece-image';
    //                 img.draggable = true;
    //                 img.dataset.code = pieceCode;

    //                 img.addEventListener('dragstart', handleDragStart);
    //                 piece.addEventListener('click', () => handlePieceClick(piece, square));

    //                 piece.appendChild(img);
    //                 square.appendChild(piece);
    //             }

    //             square.addEventListener('dragover', handleDragOver);
    //             square.addEventListener('drop', handleDrop);
    //             chessBoard.appendChild(square);
    //         });
    //     });
    // };

    const getAvatarUrl = (name) => `https://avatar.iran.liara.run/username?username=${encodeURIComponent(name)}`;

    const renderPlayerBars = () => {
        const topBar = document.getElementById("player-bar-top");
        const bottomBar = document.getElementById("player-bar-bottom");
        if ((localStorage.getItem("boardPov") || "white") === "white") {
            // White is at the bottom
            topBar.querySelector(".player-dp").src = getAvatarUrl(formatUserName(blackUser.name));
            topBar.querySelector(".player-name").textContent = formatUserName(blackUser.name);
            topBar.querySelector(".turn-indicator").textContent = turn === "black" ? "Move" : "";
            topBar.querySelector(".turn-indicator").style.color = turn === "black" ? "green" : "gray";
            topBar.querySelector(".player-timer").textContent = turn === "black" ? "🟢" : "⏳";


            bottomBar.querySelector(".player-dp").src = getAvatarUrl(formatUserName(whiteUser.name));
            bottomBar.querySelector(".player-name").textContent = formatUserName(whiteUser.name);
            bottomBar.querySelector(".turn-indicator").textContent = turn === "white" ? "Move" : "";
            bottomBar.querySelector(".turn-indicator").style.color = turn === "white" ? "green" : "gray";
            bottomBar.querySelector(".player-timer").textContent = turn === "white" ? "🟢" : "⏳";

        } else {
            // Black is at the bottom
            topBar.querySelector(".player-dp").src = getAvatarUrl(formatUserName(whiteUser.name));
            topBar.querySelector(".player-name").textContent = formatUserName(whiteUser.name);
            topBar.querySelector(".turn-indicator").textContent = turn === "white" ? "Move" : "";
            topBar.querySelector(".turn-indicator").style.color = turn === "white" ? "green" : "gray";
            topBar.querySelector(".player-timer").textContent = turn === "white" ? "🟢" : "⏳";


            bottomBar.querySelector(".player-dp").src = getAvatarUrl(formatUserName(blackUser.name));
            bottomBar.querySelector(".player-name").textContent = formatUserName(blackUser.name);
            bottomBar.querySelector(".turn-indicator").textContent = turn === "black" ? "Move" : "";
            bottomBar.querySelector(".turn-indicator").style.color = turn === "black" ? "green" : "gray";
            bottomBar.querySelector(".player-timer").textContent = turn === "black" ? "🟢" : "⏳";
        }
    };

    renderBitBoard();
    // renderBoard();
    renderPlayerBars();

    document.getElementById("flip-board").addEventListener("click", () => {
        boardPov = localStorage.getItem("boardPov") === "white";
        localStorage.setItem("boardPov", boardPov ? "black" : "white");
        renderBitBoard();
        renderPlayerBars();
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
