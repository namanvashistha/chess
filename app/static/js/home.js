document.addEventListener('DOMContentLoaded', () => {
    const createGameButton = document.getElementById('create-game');
    const joinGameButton = document.getElementById('join-game');
    const inviteCodeInput = document.getElementById('invite-code');
    const refreshButton = document.getElementById('refresh-games');
    const gamesList = document.getElementById('games-list');

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

    function hasOpenSeat(game) {
        return !game.white_user || !game.black_user;
    }

    // One status per game, with a colour "kind" the CSS maps to a pill style.
    function statusMeta(game, role) {
        if (isFinished(game)) {
            const label = game.winner === 'w' ? 'White won'
                : game.winner === 'b' ? 'Black won' : 'Draw';
            return { kind: 'done', label };
        }
        if (role === 'open') return { kind: 'join', label: 'Open seat' };
        if (role === 'mine') {
            return hasOpenSeat(game)
                ? { kind: 'wait', label: 'Waiting' }
                : { kind: 'mine', label: 'Your game' };
        }
        return { kind: 'live', label: 'Live' };
    }

    function seatName(user, myId) {
        if (!user) return '<span class="g-name g-name--empty">Open seat</span>';
        const you = user.id === myId ? ' <i class="g-you">you</i>' : '';
        return `<span class="g-name">${formatUserName(user.name)}${you}</span>`;
    }

    function gameCard(game, role, myId) {
        const action = role === 'open' ? 'join' : 'view';
        const meta = statusMeta(game, role);
        // Only surface the invite code when it's actually useful to share:
        // an open table, or your own game that's still waiting for an opponent.
        const showCode = role === 'open' || (role === 'mine' && hasOpenSeat(game));

        const item = document.createElement('div');
        item.className = 'game-item';
        item.innerHTML = `
            <button type="button" class="g-row" data-action="${action}">
                <span class="g-players">
                    <span class="g-seat">
                        <span class="g-chip g-chip--w"></span>
                        ${seatName(game.white_user, myId)}
                    </span>
                    <span class="g-vs">vs</span>
                    <span class="g-seat">
                        <span class="g-chip g-chip--b"></span>
                        ${seatName(game.black_user, myId)}
                    </span>
                </span>
                <span class="g-end">
                    <span class="g-status g-status--${meta.kind}">${meta.label}</span>
                    <i class="fa fa-chevron-right g-go"></i>
                </span>
            </button>
            ${showCode ? `
            <button type="button" class="g-code" title="Copy invite code">
                <i class="fa fa-hashtag g-code-hash"></i>
                <span class="g-code-value">${game.invite_code}</span>
                <span class="g-code-copy"><i class="fa fa-copy"></i> Copy</span>
            </button>` : ''}
        `;

        // click-to-copy invite code (don't trigger the row's primary action)
        const codeBtn = item.querySelector('.g-code');
        if (codeBtn) {
            codeBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                const copy = codeBtn.querySelector('.g-code-copy');
                navigator.clipboard?.writeText(game.invite_code).then(() => {
                    codeBtn.classList.add('is-copied');
                    copy.innerHTML = '<i class="fa fa-check"></i> Copied';
                    setTimeout(() => {
                        codeBtn.classList.remove('is-copied');
                        copy.innerHTML = '<i class="fa fa-copy"></i> Copy';
                    }, 1400);
                }).catch(() => {});
            });
        }

        // primary action: the whole row joins (claim seat) or views/spectates
        item.querySelector('.g-row').addEventListener('click', () => {
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

    // ---- tabbed games browser ----
    const tabButtons = document.querySelectorAll('.tab');
    const counts = {
        mine: document.getElementById('count-mine'),
        open: document.getElementById('count-open'),
        watch: document.getElementById('count-watch'),
    };
    const roleForTab = { mine: 'mine', open: 'open', watch: 'watch' };
    const emptyCopy = {
        mine: { mark: '♟', title: 'No games yet', sub: 'Create a table or challenge the bot to get going.' },
        open: { mark: '⏳', title: 'No open tables', sub: 'Create a game and share its code to invite a friend.' },
        watch: { mark: '👁', title: 'Nothing to watch', sub: 'Games in progress will appear here to spectate.' },
    };

    let buckets = { mine: [], open: [], watch: [] };
    let activeTab = 'mine';
    let userPickedTab = false;

    function showSkeleton() {
        let html = '';
        for (let i = 0; i < 4; i++) html += '<div class="game-skeleton"><span></span><span></span></div>';
        gamesList.innerHTML = html;
    }

    function showEmpty(copy) {
        gamesList.innerHTML = `
            <div class="tables-empty">
                <span class="tables-empty-mark">${copy.mark}</span>
                <p class="tables-empty-title">${copy.title}</p>
                <p class="tables-empty-sub">${copy.sub}</p>
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

    function renderActive() {
        const games = buckets[activeTab];
        if (!games.length) { showEmpty(emptyCopy[activeTab]); return; }
        const myId = currentUserId();
        gamesList.innerHTML = '';
        games.forEach(g => gamesList.appendChild(gameCard(g, roleForTab[activeTab], myId)));
    }

    function setActiveTab(tab) {
        activeTab = tab;
        tabButtons.forEach(b => b.classList.toggle('is-active', b.dataset.tab === tab));
        renderActive();
    }

    function fetchChessGames() {
        showSkeleton();
        fetch('/api/chess/game')
            .then(res => res.json())
            .then(data => {
                const games = data.response_key === 'SUCCESS' ? data.data : null;
                const all = Array.isArray(games) ? games : [];
                const myId = currentUserId();
                const amPlayer = (g) =>
                    myId != null &&
                    ((g.white_user && g.white_user.id === myId) ||
                     (g.black_user && g.black_user.id === myId));

                buckets.mine = all.filter(g => amPlayer(g));
                buckets.open = all.filter(g => !amPlayer(g) && hasOpenSeat(g) && !isFinished(g));
                buckets.watch = all.filter(g => !amPlayer(g) && !(hasOpenSeat(g) && !isFinished(g)));

                counts.mine.textContent = buckets.mine.length;
                counts.open.textContent = buckets.open.length;
                counts.watch.textContent = buckets.watch.length;

                // First load: land on the most relevant tab automatically.
                if (!userPickedTab) {
                    activeTab = buckets.mine.length ? 'mine' : buckets.open.length ? 'open' : 'mine';
                    tabButtons.forEach(b => b.classList.toggle('is-active', b.dataset.tab === activeTab));
                }
                renderActive();
            })
            .catch(err => {
                console.error('Error fetching games:', err);
                showError();
            });
    }

    tabButtons.forEach(b => b.addEventListener('click', () => {
        userPickedTab = true;
        setActiveTab(b.dataset.tab);
    }));

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
