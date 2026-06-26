<script>
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { ensureUser, userId } from '$lib/api.js';
	import { currentGame, reviewPly } from '$lib/stores.js';
	import { formatUserName } from '$lib/format.js';
	import { avatarUrl } from '$lib/avatar.js';
	import { computeCaptured } from '$lib/captured.js';
	import { positionsFrom } from '$lib/replay.js';
	import { sendMove } from '$lib/socket.js';
	import Board from '$lib/components/Board.svelte';

	let { children } = $props();
	let myId = $state(null);
	let povOverride = $state(null);

	// Cosmetic clocks: count down the side to move. Not server-enforced (no flag
	// loss); a visual stand-in until real time controls land.
	const START_SECONDS = 600;
	let clockW = $state(START_SECONDS);
	let clockB = $state(START_SECONDS);
	const fmt = (s) => `${Math.floor(Math.max(0, s) / 60)}:${String(Math.max(0, s) % 60).padStart(2, '0')}`;

	onMount(async () => {
		await ensureUser();
		myId = userId();
		setInterval(() => {
			const g = $currentGame;
			if (!g || g.winner || !g.white_user || !g.black_user) return;
			if (g.state?.turn === 'w' && clockW > 0) clockW -= 1;
			else if (g.state?.turn === 'b' && clockB > 0) clockB -= 1;
		}, 1000);
	});

	function toggleTheme() {
		const root = document.documentElement;
		const next = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
		root.setAttribute('data-theme', next);
		localStorage.setItem('theme', next);
	}

	// Reset a manual flip whenever the game changes.
	let lastGameId = null;
	$effect(() => {
		const gid = $currentGame?.id ?? null;
		if (gid !== lastGameId) {
			lastGameId = gid;
			povOverride = null;
			clockW = START_SECONDS;
			clockB = START_SECONDS;
			reviewPly.set(null);
		}
	});

	// Move-review: show a past position when reviewPly points before the latest.
	let moveCount = $derived($currentGame?.moves?.length ?? 0);
	let liveMode = $derived($reviewPly === null || $reviewPly >= moveCount);
	let positions = $derived(liveMode ? null : positionsFrom($currentGame?.moves));
	let viewPosition = $derived(liveMode ? null : positions[$reviewPly]);
	let viewLastMove = $derived(
		liveMode ? null : $reviewPly > 0 ? $currentGame.moves[$reviewPly - 1].move : null
	);

	// Pass & play: same user seated as both colours -> control both sides, and
	// flip the board to whoever is to move.
	let isLocal = $derived(
		!!(
			$currentGame?.white_user &&
			$currentGame?.black_user &&
			$currentGame.white_user.id === $currentGame.black_user.id
		)
	);
	let basePov = $derived(
		isLocal
			? $currentGame.state?.turn ?? 'w'
			: $currentGame && myId != null && $currentGame.black_user && $currentGame.black_user.id === myId
				? 'b'
				: 'w'
	);
	let pov = $derived(povOverride ?? basePov);
	function flip() {
		povOverride = pov === 'w' ? 'b' : 'w';
	}

	let topColor = $derived(pov === 'w' ? 'b' : 'w');
	let bottomColor = $derived(pov === 'w' ? 'w' : 'b');
	let turn = $derived($currentGame?.state?.turn ?? null);
	let cap = $derived(computeCaptured($currentGame?.current_state));

	const seat = (color) =>
		!$currentGame ? null : color === 'w' ? $currentGame.white_user : $currentGame.black_user;

	const CAP_IMG = {
		P: 'wP', R: 'wR', N: 'wN', B: 'wB', Q: 'wQ',
		p: 'bP', r: 'bR', n: 'bN', b: 'bB', q: 'bQ'
	};
	const capUrl = (code) => `/static/images/${CAP_IMG[code]}.svg`;

	let winnerText = $derived(
		!$currentGame?.winner ? '' : $currentGame.winner === 'w' ? 'White wins' : $currentGame.winner === 'b' ? 'Black wins' : 'Draw'
	);
</script>

<div class="app">
	<header class="topbar">
		<a class="brand" href="/">
			<span class="brand-mark">♞</span>
			<span class="brand-name">Chess</span>
		</a>
		<div class="topbar-actions">
			{#if $currentGame}
				<button class="icon-btn" onclick={flip} aria-label="Flip board" title="Flip board">
					<i class="fa fa-rotate"></i>
				</button>
			{/if}
			<button class="icon-btn" onclick={toggleTheme} aria-label="Toggle theme" title="Toggle theme">
				<i class="fa fa-moon"></i>
			</button>
		</div>
	</header>

	<main class="shell">
		<section class="board-col">
			<div class="board-stack">
				{@render bar(topColor)}
				<Board
				game={$currentGame}
				{myId}
				{pov}
				local={isLocal}
				onmove={sendMove}
				position={viewPosition}
				reviewing={!liveMode}
				lastMoveStr={viewLastMove}
			/>
				{@render bar(bottomColor)}
				{#if !$currentGame}<p class="board-cap">After 1. e4</p>{/if}
			</div>
		</section>

		<aside class="rail">
			{@render children()}
		</aside>
	</main>
</div>

{#if $currentGame?.winner}
	<div class="winner">
		<div class="winner-card">
			<span class="winner-icon">♔</span>
			<h2>Game over</h2>
			<p class="winner-msg">{winnerText}</p>
			<button class="btn-create winner-home" onclick={() => goto('/')}>
				<span class="btn-create-text"><strong>New game</strong></span>
			</button>
		</div>
	</div>
{/if}

{#snippet bar(color)}
	{@const u = seat(color)}
	{@const taken = color === 'w' ? cap.whiteCaptured : cap.blackCaptured}
	{@const adv = color === 'w' ? cap.adv : -cap.adv}
	{@const label = isLocal
		? color === 'w' ? 'White' : 'Black'
		: u
			? formatUserName(u.name)
			: $currentGame
				? 'Waiting for opponent…'
				: color === bottomColor ? 'You' : 'Opponent'}
	{@const seed = isLocal ? label : u ? formatUserName(u.name) : color === bottomColor ? 'You' : 'Opponent'}
	<div class="player-bar" class:is-turn={$currentGame && turn === color && !$currentGame.winner}>
		<div class="player-id">
			<img class="player-dp" src={avatarUrl(seed)} alt="" />
			<div class="player-meta">
				<span class="player-name" class:muted={$currentGame && !u}>{label}</span>
				{#if $currentGame && (taken.length || adv > 0)}
					<span class="player-captured">
						{#each taken as code, i (i)}<img src={capUrl(code)} alt="" />{/each}
						{#if adv > 0}<span class="material">+{adv}</span>{/if}
					</span>
				{/if}
			</div>
		</div>
		{#if $currentGame}
			{@const secs = color === 'w' ? clockW : clockB}
			<span class="clock" class:running={turn === color && !$currentGame.winner} class:flagged={secs <= 0}>
				{fmt(secs)}
			</span>
		{/if}
	</div>
{/snippet}

<style>
	.player-meta {
		display: flex;
		flex-direction: column;
		gap: 3px;
		min-width: 0;
	}
	.player-captured {
		display: flex;
		align-items: center;
		gap: 1px;
		height: 15px;
	}
	.player-captured img {
		width: 14px;
		height: 14px;
		object-fit: contain;
		opacity: 0.85;
		margin-left: -3px;
	}
	.player-captured img:first-child {
		margin-left: 0;
	}
	.material {
		margin-left: 6px;
		font-size: 0.72rem;
		font-weight: 700;
		color: var(--text-muted);
		font-variant-numeric: tabular-nums;
	}
	.turn-pill {
		font-size: 0.66rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--good);
		background: var(--good-soft);
		padding: 4px 9px;
		border-radius: 99px;
	}
	.winner {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(8, 11, 16, 0.6);
		backdrop-filter: blur(6px);
	}
	.winner-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		text-align: center;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-lg);
		padding: 34px 46px;
	}
	.winner-icon {
		font-size: 2.6rem;
		color: var(--accent);
	}
	.winner-card h2 {
		font-size: 0.78rem;
		text-transform: uppercase;
		letter-spacing: 0.1em;
		color: var(--text-muted);
		font-weight: 700;
	}
	.winner-msg {
		font-size: 1.7rem;
		font-weight: 800;
		letter-spacing: -0.02em;
	}
	.winner-home {
		width: auto;
		margin-top: 12px;
		padding: 11px 22px;
	}
</style>
