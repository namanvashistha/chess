document.addEventListener('DOMContentLoaded', () => {
    const createGameButton = document.getElementById('create-game');
    const joinGameButton = document.getElementById('join-game');
    const inviteCodeInput = document.getElementById('invite-code');
    const inviteCodeDisplay = document.getElementById('invite-code-display');
    const gameStatus = document.getElementById('game-status');
    const gamesList = document.getElementById('games-list'); // To display active games

    // Function to format the user name
    function renderUserName() {
        if (document.getElementById('user-name') && localStorage.getItem('userData')) {
            userName = JSON.parse(localStorage.getItem('userData'))?.name;
            document.getElementById('user-name').textContent = formatUserName(userName);
        }
    }
    renderUserName();

    function fetchChessGames() {
        fetch('/api/chess/game')
            .then(res => res.json())
            .then(data => {
                if (data.response_key === "SUCCESS") {
                    const games = data.data;
                    gamesList.innerHTML = ''; // Clear the previous list
                    games.forEach(game => {
                        const gameItem = document.createElement('div');
                        gameItem.classList.add('game-item');
                        gameItem.innerHTML = `
                            <p>Game ID: ${game.id}</p>
                            <p>Invite Code: ${game.invite_code}</p>
                            <p>
                                White Player: 
                                ${game.white_user?.name 
                                    ? formatUserName(game.white_user.name)
                                    : '<span class="loader">waiting<span class="dots">...</span></span>'}
                            </p>
                            <p>
                                Black Player: 
                                ${game.black_user?.name 
                                    ? formatUserName(game.black_user.name)
                                    : '<span class="loader">waiting<span class="dots">...</span></span>'}
                            </p>
                            <p>Status: ${game.status || 'Waiting for opponent'}</p>
                            <button class="join-game-button" data-invite-code="${game.invite_code}">Join Game</button>
                        `;
                        gamesList.appendChild(gameItem);

                        // Attach event listener to "Join Game" button
                        gameItem.querySelector('.join-game-button').addEventListener('click', () => {
                            const inviteCode = game.invite_code;
                            localStorage.setItem('inviteCode', inviteCode);
                            window.location.href = `/game/${game.id}`;
                        });
                    });
                } else {
                    gamesList.innerHTML = '<p>No games available at the moment.</p>';
                }
            })
            .catch(err => console.error('Error fetching games:', err));
    }

    // Call the function to load games when the page is loaded
    fetchChessGames();

    // Create a new game
    createGameButton.addEventListener('click', () => {
        token = localStorage.getItem('userToken');
        fetch('/api/chess/game', {
            method: 'POST',
            body: JSON.stringify({token}),
            headers: {
                'Content-Type': 'application/json',
            },
        })
        .then(res => res.json())
        .then(data => {
            if (data.response_key === "SUCCESS") {
                // window.location.href = `/game/${data.data}`;
                window.location.href = '';
            }
        })
        .catch(err => console.error('Error creating game:', err));
    });

    // Join an existing game using invite code
    joinGameButton.addEventListener('click', () => {
        const inviteCode = inviteCodeInput.value.trim();
        if (!inviteCode) {
            alert('Please enter a valid invite code');
            return;
        }

        fetch('/api/chess/game/join', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ 
                invite_code: inviteCode,
                token: localStorage.getItem('userToken'),
            }),
        })
        .then(res => res.json())
        .then(data => {
            if (data.response_key === "SUCCESS") {
                window.location.href = `/game/${data.data}`;
            } else {
                alert('Invalid invite code!');
            }
        })
        .catch(err => console.error('Error joining game:', err));
    });
});
