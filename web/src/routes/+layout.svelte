<script>
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { ensureUser } from '$lib/api.js';
	import { currentGame, reviewPly, moveError } from '$lib/stores.js';
	import { formatUserName } from '$lib/format.js';
	import { avatarUrl } from '$lib/avatar.js';
	import { computeCaptured } from '$lib/captured.js';
	import { positionsFrom } from '$lib/replay.js';
	import { uciFor, resultText } from '$lib/game.js';
	import { sendMove } from '$lib/socket.js';
	import Board from '$lib/components/Board.svelte';

	let { children } = $props();
	let povOverride = $state(null);

	onMount(() => {
		ensureUser();
	});

	function toggleTheme() {
		const root = document.documentElement;
		const next = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
		root.setAttribute('data-theme', next);
		localStorage.setItem('theme', next);
	}

	let g = $derived($currentGame);

	// Reset a manual flip and the review cursor whenever the game changes.
	let lastGameId = null;
	$effect(() => {
		const gid = g?.id ?? null;
		if (gid !== lastGameId) {
			lastGameId = gid;
			povOverride = null;
			reviewPly.set(null);
		}
	});

	// Move review: show a past position when reviewPly points before the latest.
	let moveCount = $derived(g?.moves.length ?? 0);
	let liveMode = $derived($reviewPly === null || $reviewPly >= moveCount);
	let viewPosition = $derived(liveMode ? (g?.board ?? null) : positionsFrom(g.moves)[$reviewPly]);
	let viewLastMove = $derived(
		liveMode ? (g?.lastMove ?? null) : $reviewPly > 0 ? g.moves[$reviewPly - 1].uci : null
	);

	// Pass & play turns the board to whoever is to move; otherwise Black players
	// see the board from Black's side.
	let basePov = $derived(!g ? 'w' : g.you === 'both' ? g.turn : g.you === 'b' ? 'b' : 'w');
	let pov = $derived(povOverride ?? basePov);
	function flip() {
		povOverride = pov === 'w' ? 'b' : 'w';
	}

	function onMove(from, to) {
		const uci = uciFor(g, from, to);
		if (uci) sendMove(g.moves.length, uci);
	}

	let topColor = $derived(pov === 'w' ? 'b' : 'w');
	let bottomColor = $derived(pov === 'w' ? 'w' : 'b');
	let cap = $derived(computeCaptured(g?.board));

	const seat = (color) => (!g ? null : color === 'w' ? g.white : g.black);

	const CAP_IMG = {
		P: 'wP', R: 'wR', N: 'wN', B: 'wB', Q: 'wQ',
		p: 'bP', r: 'bR', n: 'bN', b: 'bB', q: 'bQ'
	};
	const capUrl = (code) => `/static/images/${CAP_IMG[code]}.svg`;
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
					position={viewPosition}
					targets={g?.targets ?? {}}
					canMove={liveMode && !!g?.canMove}
					lastMove={viewLastMove}
					{pov}
					onmove={onMove}
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

{#if $moveError}
	<div class="toast" role="status">{$moveError}</div>
{/if}

{#if g?.result}
	<div class="winner">
		<div class="winner-card">
			<span class="winner-icon">♔</span>
			<h2>Game over</h2>
			<p class="winner-msg">{resultText(g)}</p>
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
	{@const label = g?.you === 'both'
		? color === 'w' ? 'White' : 'Black'
		: u
			? formatUserName(u.name)
			: g
				? 'Waiting for opponent…'
				: color === bottomColor ? 'You' : 'Opponent'}
	<div class="player-bar" class:is-turn={g && g.status === 'active' && g.turn === color}>
		<div class="player-id">
			<img class="player-dp" src={avatarUrl(label)} alt="" />
			<div class="player-meta">
				<span class="player-name" class:muted={g && !u}>{label}</span>
				{#if g && (taken.length || adv > 0)}
					<span class="player-captured">
						{#each taken as code, i (i)}<img src={capUrl(code)} alt="" />{/each}
						{#if adv > 0}<span class="material">+{adv}</span>{/if}
					</span>
				{/if}
			</div>
		</div>
		{#if g && g.status === 'active' && g.turn === color}
			<span class="turn-pill">{u?.bot ? 'Thinking' : 'To move'}</span>
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
	.toast {
		position: fixed;
		left: 50%;
		bottom: 28px;
		transform: translateX(-50%);
		z-index: 120;
		padding: 10px 16px;
		border-radius: var(--radius);
		background: var(--surface);
		border: 1px solid var(--border);
		box-shadow: var(--shadow-md);
		font-size: 0.9rem;
		font-weight: 600;
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
