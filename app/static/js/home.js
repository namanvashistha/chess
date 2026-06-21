document.addEventListener('DOMContentLoaded', () => {
    const createGameButton = document.getElementById('create-game');
    const joinGameButton = document.getElementById('join-game');
    const inviteCodeInput = document.getElementById('invite-code');
    const refreshButton = document.getElementById('refresh-games');
    const gamesList = document.getElementById('games-list');
    const statOpen = document.getElementById('stat-open');
    const statMine = document.getElementById('stat-mine');

    // ---- identity ----
    function renderUserName() {
        const data = localStorage.getItem('userData');
        const el = document.getElementById('user-name');
        if (el && data) {
            const name = JSON.parse(data)?.name;
            if (name) el.textContent = formatUserName(name);
        }
    }
    renderUserName();

    function currentUserId() {
        try {
            return JSON.parse(localStorage.getItem('userData'))?.id ?? null;
        } catch {
            return null;
        }
    }

    // ---- decorative board: starting position after 1. e4 ----
    function renderMotifBoard() {
        const el = document.getElementById('motif-board');
        if (!el) return;
        const back = ['R', 'N', 'B', 'Q', 'K', 'B', 'N', 'R'];
        // board[row][file], row 0 = rank 8 (top), row 7 = rank 1 (bottom)
        const board = Array.from({ length: 8 }, () => Array(8).fill(null));
        for (let f = 0; f < 8; f++) {
            board[0][f] = { c: 'b', p: back[f] };
            board[1][f] = { c: 'b', p: 'P' };
            board[6][f] = { c: 'w', p: 'P' };
            board[7][f] = { c: 'w', p: back[f] };
        }
        // 1. e4 : white pawn e2 (row 6, file 4) -> e4 (row 4, file 4)
        board[4][4] = board[6][4];
        board[6][4] = null;
        const lastMove = new Set(['6,4', '4,4']);

        let html = '';
        for (let r = 0; r < 8; r++) {
            for (let f = 0; f < 8; f++) {
                const dark = (r + f) % 2 === 1;
                const last = lastMove.has(`${r},${f}`) ? ' is-last' : '';
                const sq = board[r][f];
                html += `<div class="msq ${dark ? 'msq--d' : 'msq--l'}${last}">`;
                if (sq) html += `<img src="/static/images/${sq.c}${sq.p}.svg" alt="">`;
                html += '</div>';
            }
        }
        el.innerHTML = html;
    }
    renderMotifBoard();

    // ---- games panel ----
    function isFinished(game) {
        return game.winner === 'w' || game.winner === 'b';
    }

    function statusFor(game, role) {
        if (isFinished(game)) return game.winner === 'w' ? 'White won' : 'Black won';
        if (role === 'mine') return 'You’re in';
        if (role === 'open') return 'Open seat';
        return 'In progress';
    }

    function seatMarkup(user, myId) {
        if (!user) return '<span class="loader">waiting<span class="dots">...</span></span>';
        const you = user.id === myId ? ' <span class="gi-you">you</span>' : '';
        return `${formatUserName(user.name)}${you}`;
    }

    function gameCard(game, role, myId) {
        const action = role === 'open' ? 'join' : 'view';
        const label = role === 'mine' ? 'Resume' : role === 'open' ? 'Join game' : 'View game';
        const statusKind = isFinished(game) ? 'done' : role === 'open' ? 'join' : role === 'mine' ? 'mine' : 'live';

        const item = document.createElement('div');
        item.className = 'game-item';
        item.innerHTML = `
            <div class="gi-head">
                <span class="gi-id">Game #${game.id}</span>
                <span class="gi-status gi-status--${statusKind}">${statusFor(game, role)}</span>
            </div>
            <button type="button" class="gi-code" title="Copy invite code">
                <i class="fa fa-hashtag"></i>
                <span class="gi-code-value">${game.invite_code}</span>
                <i class="fa fa-copy gi-code-icon"></i>
            </button>
            <div class="gi-players">
                <span class="gi-player"><span class="gi-side gi-side--w"></span>${seatMarkup(game.white_user, myId)}</span>
                <span class="gi-player"><span class="gi-side gi-side--b"></span>${seatMarkup(game.black_user, myId)}</span>
            </div>
            <button type="button" class="join-game-button" data-action="${action}">${label}</button>
        `;

        // click-to-copy invite code
        const codeBtn = item.querySelector('.gi-code');
        codeBtn.addEventListener('click', () => {
            const icon = codeBtn.querySelector('.gi-code-icon');
            navigator.clipboard?.writeText(game.invite_code).then(() => {
                codeBtn.classList.add('is-copied');
                icon.className = 'fa fa-check gi-code-icon';
                setTimeout(() => {
                    codeBtn.classList.remove('is-copied');
                    icon.className = 'fa fa-copy gi-code-icon';
                }, 1400);
            }).catch(() => {});
        });

        // join (claim seat) or view/spectate
        item.querySelector('.join-game-button').addEventListener('click', () => {
            localStorage.setItem('inviteCode', game.invite_code);
            if (action === 'view') {
                window.location.href = `/game/${game.id}`;
                return;
            }
            fetch('/api/chess/game/join', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    invite_code: game.invite_code,
                    token: localStorage.getItem('userToken'),
                }),
            })
            .then(res => res.json())
            .then(data => {
                if (data.response_key === 'SUCCESS') {
                    window.location.href = `/game/${data.data}`;
                } else {
                    alert('Could not join this game.');
                }
            })
            .catch(err => console.error('Error joining game:', err));
        });

        return item;
    }

    function renderGroup(title, games, role, myId) {
        const section = document.createElement('div');
        section.className = 'game-group';
        const head = document.createElement('div');
        head.className = 'game-group-head';
        head.innerHTML = `<span>${title}</span><span class="game-group-count">${games.length}</span>`;
        section.appendChild(head);
        games.forEach(g => section.appendChild(gameCard(g, role, myId)));
        return section;
    }

    function showSkeleton() {
        let html = '';
        for (let i = 0; i < 3; i++) {
            html += '<div class="game-skeleton"><span></span><span></span><span></span></div>';
        }
        gamesList.innerHTML = html;
    }

    function showEmpty() {
        gamesList.innerHTML = `
            <div class="tables-empty">
                <span class="tables-empty-mark">♟</span>
                <p class="tables-empty-title">No games yet</p>
                <p class="tables-empty-sub">Create a table and share the code to get the first game going.</p>
            </div>`;
    }

    function showError() {
        gamesList.innerHTML = `
            <div class="tables-empty">
                <span class="tables-empty-mark">⚠</span>
                <p class="tables-empty-title">Couldn't load games</p>
                <button type="button" class="btn-join" id="retry-games">Try again</button>
            </div>`;
        const retry = document.getElementById('retry-games');
        if (retry) retry.addEventListener('click', fetchChessGames);
    }

    function fetchChessGames() {
        showSkeleton();
        fetch('/api/chess/game')
            .then(res => res.json())
            .then(data => {
                const games = data.response_key === 'SUCCESS' ? data.data : null;
                if (!Array.isArray(games) || games.length === 0) {
                    statOpen.textContent = '0';
                    statMine.textContent = '0';
                    showEmpty();
                    return;
                }

                const myId = currentUserId();
                const amPlayer = (g) =>
                    myId != null &&
                    ((g.white_user && g.white_user.id === myId) ||
                     (g.black_user && g.black_user.id === myId));
                const hasOpenSeat = (g) => !g.white_user || !g.black_user;

                const mine = games.filter(g => amPlayer(g));
                const open = games.filter(g => !amPlayer(g) && hasOpenSeat(g) && !isFinished(g));
                const watch = games.filter(g => !amPlayer(g) && !(hasOpenSeat(g) && !isFinished(g)));

                statOpen.textContent = String(open.length);
                statMine.textContent = String(mine.length);

                gamesList.innerHTML = '';
                if (mine.length) gamesList.appendChild(renderGroup('Your games', mine, 'mine', myId));
                if (open.length) gamesList.appendChild(renderGroup('Open tables', open, 'open', myId));
                if (watch.length) gamesList.appendChild(renderGroup('Watch', watch, 'watch', myId));
            })
            .catch(err => {
                console.error('Error fetching games:', err);
                showError();
            });
    }
    fetchChessGames();

    if (refreshButton) {
        refreshButton.addEventListener('click', () => {
            refreshButton.classList.add('is-spinning');
            fetchChessGames();
            setTimeout(() => refreshButton.classList.remove('is-spinning'), 600);
        });
    }

    // ---- create a new game ----
    createGameButton.addEventListener('click', () => {
        const token = localStorage.getItem('userToken');
        createGameButton.disabled = true;
        fetch('/api/chess/game', {
            method: 'POST',
            body: JSON.stringify({ token }),
            headers: { 'Content-Type': 'application/json' },
        })
        .then(res => res.json())
        .then(data => {
            if (data.response_key === 'SUCCESS') {
                // Take the host straight into their game (waiting-for-opponent state)
                // instead of dropping them back on the lobby.
                window.location.href = `/game/${data.data}`;
            } else {
                createGameButton.disabled = false;
            }
        })
        .catch(err => {
            console.error('Error creating game:', err);
            createGameButton.disabled = false;
        });
    });

    // ---- play against the bot ----
    const createBotButton = document.getElementById('create-bot-game');
    const botLevelButtons = document.querySelectorAll('.bot-level');
    let botLevel = localStorage.getItem('botLevel') || 'medium';

    function syncBotLevels() {
        botLevelButtons.forEach(b => b.classList.toggle('is-active', b.dataset.level === botLevel));
    }
    syncBotLevels();
    botLevelButtons.forEach(b => {
        b.addEventListener('click', () => {
            botLevel = b.dataset.level;
            localStorage.setItem('botLevel', botLevel);
            syncBotLevels();
        });
    });

    if (createBotButton) {
        createBotButton.addEventListener('click', () => {
            const token = localStorage.getItem('userToken');
            createBotButton.disabled = true;
            fetch('/api/chess/game/bot', {
                method: 'POST',
                body: JSON.stringify({ token, difficulty: botLevel }),
                headers: { 'Content-Type': 'application/json' },
            })
            .then(res => res.json())
            .then(data => {
                if (data.response_key === 'SUCCESS') {
                    window.location.href = `/game/${data.data}`;
                } else {
                    createBotButton.disabled = false;
                }
            })
            .catch(err => {
                console.error('Error creating bot game:', err);
                createBotButton.disabled = false;
            });
        });
    }

    // ---- join with a pasted invite code ----
    joinGameButton.addEventListener('click', () => {
        const inviteCode = inviteCodeInput.value.trim();
        if (!inviteCode) {
            inviteCodeInput.focus();
            return;
        }
        fetch('/api/chess/game/join', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                invite_code: inviteCode,
                token: localStorage.getItem('userToken'),
            }),
        })
        .then(res => res.json())
        .then(data => {
            if (data.response_key === 'SUCCESS') {
                window.location.href = `/game/${data.data}`;
            } else {
                alert('Invalid invite code!');
            }
        })
        .catch(err => console.error('Error joining game:', err));
    });

    inviteCodeInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') joinGameButton.click();
    });
});
