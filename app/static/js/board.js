const chessBoard = document.getElementById('chess-board');
let legalMoves = []; // To store allowed moves for the clicked piece
let selectedPiece = null; // Track the currently selected piece
let selectedSquare = null; // Track the currently selected square
let lastGameData = null; // Most recent game payload (for re-render on flip)

// Standard piece values and full starting material (for captured/advantage calc)
const PIECE_VALUES = { p: 1, n: 3, b: 3, r: 5, q: 9, k: 0 };
const START_COUNTS = { p: 8, n: 2, b: 2, r: 2, q: 1, k: 1 };

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

// Poll handle used while a host waits for an opponent to join.
let waitingPoll = null;

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

// Manual board-flip override (null = follow the player's own colour)
let povOverride = null;

// Render the chessboard
function renderChessBoard(gameData) {
    // Cache a pristine copy so a manual flip can re-render without compounding
    // the in-place board_layout mutations below.
    lastGameData = JSON.parse(JSON.stringify(gameData));

    legalMoves = gameData.legal_moves; // Store the allowed moves
    userData = JSON.parse(localStorage.getItem("userData"));
    if (povOverride) {
        localStorage.setItem("boardPov", povOverride);
    } else if (gameData.white_user && userData.id === gameData.white_user.id) {
        localStorage.setItem("boardPov", "w");
    } else if (gameData.black_user && userData.id === gameData.black_user.id) {
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
        gameData.board_layout.forEach((row, rowIndex) => {
            row.forEach((squareInfo, colIndex) => {
                const [squareKey, color] = squareInfo;

                const square = document.createElement('div');
                square.className = `square ${color === 'w' ? 'light' : 'dark'}`;
                square.dataset.key = squareKey;
                square.dataset.file = squareKey[0];
                square.dataset.rank = squareKey[1];

                // Coordinates only on board edges (lichess-style overlay)
                if (rowIndex === 7) {
                    const fileLabel = document.createElement('span');
                    fileLabel.textContent = squareKey[0];
                    fileLabel.className = 'square-label file';
                    square.appendChild(fileLabel);
                }
                if (colIndex === 0) {
                    const rankLabel = document.createElement('span');
                    rankLabel.textContent = squareKey[1];
                    rankLabel.className = 'square-label rank';
                    square.appendChild(rankLabel);
                }

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

    // Generate avatars locally as SVG data-URIs (no network round-trip).
    const getAvatarUrl = (name) => {
        const initials = (name || "?")
            .split(/\s+/)
            .map(w => w.charAt(0))
            .slice(0, 2)
            .join("")
            .toUpperCase();
        // deterministic hue from the name
        let hash = 0;
        for (let i = 0; i < (name || "").length; i++) {
            hash = (name.charCodeAt(i) + ((hash << 5) - hash)) | 0;
        }
        const hue = Math.abs(hash) % 360;
        const svg =
            `<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80" viewBox="0 0 80 80">` +
            `<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">` +
            `<stop offset="0" stop-color="hsl(${hue},62%,55%)"/>` +
            `<stop offset="1" stop-color="hsl(${(hue + 38) % 360},62%,42%)"/>` +
            `</linearGradient></defs>` +
            `<rect width="80" height="80" rx="20" fill="url(#g)"/>` +
            `<text x="50%" y="50%" dy="0.35em" text-anchor="middle" ` +
            `font-family="Inter,system-ui,sans-serif" font-size="34" font-weight="700" fill="#fff">${initials}</text>` +
            `</svg>`;
        return `data:image/svg+xml,${encodeURIComponent(svg)}`;
    };

    const renderPlayerBars = () => {
        const topBar = document.getElementById("player-bar-top");
        const bottomBar = document.getElementById("player-bar-bottom");
        const pov = localStorage.getItem("boardPov") || "w";
        // top player is the opposite color of the point-of-view
        const topColor = pov === "w" ? "b" : "w";
        const bottomColor = pov === "w" ? "w" : "b";
        const turn = gameData.state.turn;
        const captured = computeCaptured(gameData.current_state);

        const renderCaptured = (container, color) => {
            container.innerHTML = "";
            // pieces THIS player captured = the opponent's lost pieces
            const codes = color === "w" ? captured.whiteCaptured : captured.blackCaptured;
            codes.forEach(code => {
                const img = document.createElement("img");
                img.src = pieceMap[code];
                img.alt = code;
                container.appendChild(img);
            });
            const adv = color === "w" ? captured.adv : -captured.adv;
            if (adv > 0) {
                const span = document.createElement("span");
                span.className = "material";
                span.textContent = `+${adv}`;
                container.appendChild(span);
            }
        };

        const fillBar = (bar, color) => {
            const user = color === "w" ? gameData.white_user : gameData.black_user;
            if (!user) {
                // Seat not yet taken: show a quiet placeholder instead of crashing.
                bar.querySelector(".player-dp").src = getAvatarUrl("?");
                bar.querySelector(".player-name").textContent = "Waiting for opponent…";
                bar.querySelector(".turn-indicator").textContent = "";
                bar.querySelector(".player-timer").textContent = "";
                bar.classList.remove("is-turn");
                bar.querySelector(".player-captured").innerHTML = "";
                return;
            }
            const name = formatUserName(user.name);
            bar.querySelector(".player-dp").src = getAvatarUrl(name);
            bar.querySelector(".player-name").textContent = name;
            bar.querySelector(".turn-indicator").textContent = turn === color ? "To move" : "";
            bar.querySelector(".player-timer").textContent = turn === color ? "🟢" : "⏳";
            bar.classList.toggle("is-turn", turn === color);
            renderCaptured(bar.querySelector(".player-captured"), color);
        };

        fillBar(topBar, topColor);
        fillBar(bottomBar, bottomColor);
    };

    const renderStatus = () => {
        const moveCount = document.getElementById("move-count");
        if (moveCount) moveCount.textContent = gameData.moves.length;
        const status = document.getElementById("game-status");
        if (status) {
            status.textContent = gameData.winner
                ? "Game over"
                : `${gameData.state.turn === "w" ? "White" : "Black"} to move`;
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
        const scroller = document.getElementById("move-history");
        scroller.scrollTop = scroller.scrollHeight;
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
    
        winnerDisplay.style.display = "flex";
    }

    // Host is in the game but the opponent seat is still empty: prompt them to
    // share the invite code, and poll until someone joins (join isn't pushed
    // over the socket).
    const manageWaiting = () => {
        const me = userData?.id;
        const amPlayer = me != null &&
            ((gameData.white_user && gameData.white_user.id === me) ||
             (gameData.black_user && gameData.black_user.id === me));
        const waiting = amPlayer && (!gameData.white_user || !gameData.black_user) && !gameData.winner;
        const overlay = document.getElementById("waiting-display");
        if (!overlay) return;

        if (!waiting) {
            overlay.style.display = "none";
            if (waitingPoll) { clearInterval(waitingPoll); waitingPoll = null; }
            return;
        }

        const codeEl = document.getElementById("waiting-code");
        if (codeEl) codeEl.textContent = gameData.invite_code || "";
        overlay.style.display = "flex";
        if (!waitingPoll) waitingPoll = setInterval(fetchChessState, 3000);
    };

    renderBitBoard();
    renderPlayerBars();
    renderMoves();
    renderStatus();
    highlightLastMove();
    displayWinner();
    manageWaiting();
}

// Compute captured pieces and material advantage from the live board state.
// Returns { whiteCaptured, blackCaptured, adv } where *Captured are arrays of
// piece codes that side has captured, and adv = white material - black material.
function computeCaptured(currentState) {
    const onBoard = { w: { p: 0, n: 0, b: 0, r: 0, q: 0, k: 0 }, b: { p: 0, n: 0, b: 0, r: 0, q: 0, k: 0 } };
    Object.values(currentState || {}).forEach(code => {
        if (!code) return;
        const side = code === code.toUpperCase() ? "w" : "b";
        onBoard[side][code.toLowerCase()]++;
    });

    const order = ["q", "r", "b", "n", "p"];
    const whiteCaptured = []; // black pieces white removed -> black codes (lowercase)
    const blackCaptured = []; // white pieces black removed -> white codes (uppercase)
    let whiteMat = 0, blackMat = 0;
    order.forEach(type => {
        const wLost = Math.max(0, START_COUNTS[type] - onBoard.w[type]);
        const bLost = Math.max(0, START_COUNTS[type] - onBoard.b[type]);
        for (let i = 0; i < bLost; i++) whiteCaptured.push(type);            // black piece
        for (let i = 0; i < wLost; i++) blackCaptured.push(type.toUpperCase()); // white piece
        whiteMat += onBoard.w[type] * PIECE_VALUES[type];
        blackMat += onBoard.b[type] * PIECE_VALUES[type];
    });

    return { whiteCaptured, blackCaptured, adv: whiteMat - blackMat };
}

// Flip the board orientation (works for players and spectators).
(function wireFlipButton() {
    const flipBtn = document.getElementById("flip-board");
    if (!flipBtn) return;
    flipBtn.addEventListener("click", () => {
        const current = localStorage.getItem("boardPov") || "w";
        povOverride = current === "w" ? "b" : "w";
        if (lastGameData) renderChessBoard(JSON.parse(JSON.stringify(lastGameData)));
    });
})();



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
    piece.parentElement.classList.add('selected');
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
    document.querySelectorAll('.square.selected').forEach(square => {
        square.classList.remove('selected');
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

// Wire the waiting-overlay "copy invite code" button once.
(function () {
    const btn = document.getElementById("waiting-copy");
    const codeEl = document.getElementById("waiting-code");
    if (!btn || !codeEl) return;
    btn.addEventListener("click", () => {
        const label = btn.querySelector(".waiting-copy-label");
        navigator.clipboard?.writeText(codeEl.textContent.trim()).then(() => {
            const prev = label ? label.innerHTML : "";
            if (label) label.innerHTML = '<i class="fa fa-check"></i> Copied';
            btn.classList.add("is-copied");
            setTimeout(() => {
                if (label) label.innerHTML = prev;
                btn.classList.remove("is-copied");
            }, 1500);
        }).catch(() => {});
    });
})();
