<script>
	import { page } from '$app/stores';
	import { onDestroy } from 'svelte';
	import { getGame } from '$lib/api.js';
	import { currentGame, user } from '$lib/stores.js';
	import { connectSocket, disconnectSocket } from '$lib/socket.js';

	let id = $derived($page.params.id);
	let myId = $derived($user?.id ?? null);

	// Load initial state and (re)bind the socket whenever the game id changes.
	// Live updates arrive over the socket; this is the only fetch in steady state.
	$effect(() => {
		const gid = id;
		getGame(gid).then((g) => currentGame.set(g));
		connectSocket(gid);
	});

	onDestroy(() => {
		disconnectSocket();
		currentGame.set(null); // board returns to idle when leaving the game
	});

	let g = $derived($currentGame);
	const amPlayer = $derived(
		g && myId != null && ((g.white_user && g.white_user.id === myId) || (g.black_user && g.black_user.id === myId))
	);
	let waiting = $derived(g && amPlayer && (!g.white_user || !g.black_user) && !g.winner);

	// A join isn't pushed over the socket, so poll only while waiting for one.
	let waitPoll;
	$effect(() => {
		clearInterval(waitPoll);
		if (waiting) waitPoll = setInterval(() => getGame(id).then((x) => currentGame.set(x)), 3000);
		return () => clearInterval(waitPoll);
	});
	let statusText = $derived(
		!g ? 'Loading…'
			: g.winner === 'w' ? 'White wins'
			: g.winner === 'b' ? 'Black wins'
			: g.state?.turn === 'w' ? 'White to move'
			: 'Black to move'
	);

	let movePairs = $derived.by(() => {
		const m = g?.moves ?? [];
		const rows = [];
		for (let i = 0; i < m.length; i += 2) {
			rows.push({ n: i / 2 + 1, w: m[i]?.move ?? '', b: m[i + 1]?.move ?? '' });
		}
		return rows;
	});

	let copied = $state(false);
	function copyCode() {
		navigator.clipboard?.writeText(g.invite_code).then(() => {
			copied = true;
			setTimeout(() => (copied = false), 1400);
		});
	}
</script>

<div class="rail-stack">
	{#if waiting}
		<section class="panel share">
			<span class="spinner"></span>
			<h2>Waiting for an opponent</h2>
			<p class="muted">Share this code. The game starts the moment they join.</p>
			<button class="code-btn" class:is-copied={copied} onclick={copyCode}>
				<span class="code-val">{g.invite_code}</span>
				<span class="code-copy"><i class="fa fa-{copied ? 'check' : 'copy'}"></i> {copied ? 'Copied' : 'Copy'}</span>
			</button>
		</section>
	{/if}

	<section class="panel moves">
		<div class="moves-head">
			<span class="eyebrow">Moves</span>
			<span class="status">{statusText}</span>
		</div>
		<div class="moves-body">
			<table>
				<tbody>
					{#each movePairs as r}
						<tr><td class="n">{r.n}.</td><td>{r.w}</td><td>{r.b}</td></tr>
					{:else}
						<tr><td colspan="3" class="empty">No moves yet.</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
		<a class="back" href="/"><i class="fa fa-plus"></i> New game</a>
	</section>
</div>

<style>
	.rail-stack {
		display: flex;
		flex-direction: column;
		gap: 18px;
	}
	.share {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 10px;
		padding: 26px;
	}
	.share h2 {
		font-size: 1.1rem;
		font-weight: 800;
	}
	.muted {
		color: var(--text-muted);
		font-size: 0.9rem;
		max-width: 30ch;
	}
	.spinner {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		border: 3px solid var(--surface-3);
		border-top-color: var(--accent);
		animation: spin 0.8s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	.code-btn {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		margin-top: 6px;
		padding: 12px 16px;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}
	.code-btn.is-copied {
		border-color: var(--good);
	}
	.code-val {
		flex: 1;
		text-align: left;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
		letter-spacing: 0.06em;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.code-copy {
		flex-shrink: 0;
		font-size: 0.82rem;
		font-weight: 700;
		color: var(--accent);
	}
	.is-copied .code-copy {
		color: var(--good);
	}
	.moves {
		display: flex;
		flex-direction: column;
		max-height: 60vh;
	}
	.moves-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 18px;
		border-bottom: 1px solid var(--border);
	}
	.status {
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--text-muted);
	}
	.moves-body {
		flex: 1;
		overflow-y: auto;
		min-height: 80px;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.9rem;
	}
	td {
		padding: 7px 18px;
		font-variant-numeric: tabular-nums;
	}
	td.n {
		color: var(--text-faint);
		width: 46px;
	}
	tr:nth-child(odd) {
		background: var(--surface-2);
	}
	.empty {
		color: var(--text-faint);
		text-align: center;
	}
	.back {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 13px;
		border-top: 1px solid var(--border);
		font-weight: 700;
		font-size: 0.9rem;
		color: var(--accent);
	}
</style>
