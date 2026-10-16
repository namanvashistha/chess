<script>
	import { page } from '$app/stores';
	import { onDestroy } from 'svelte';
	import { currentGame, reviewPly } from '$lib/stores.js';
	import { connectSocket, disconnectSocket } from '$lib/socket.js';
	import { resultText } from '$lib/game.js';

	let id = $derived($page.params.id);

	// The socket's first message is a snapshot, so connecting is all it takes to
	// load the game; it rebinds whenever the id changes.
	$effect(() => {
		connectSocket(id);
	});

	onDestroy(() => {
		disconnectSocket();
		currentGame.set(null);
	});

	let g = $derived($currentGame?.id === id ? $currentGame : null);
	let waiting = $derived(g?.status === 'waiting' && g.you !== '');

	let statusText = $derived(
		!g ? 'Loading…'
			: g.result ? resultText(g)
			: g.status === 'waiting' ? 'Waiting for an opponent'
			: g.turn === 'w' ? 'White to move'
			: 'Black to move'
	);

	// ---- move navigation ----
	let total = $derived(g?.moves.length ?? 0);
	let cur = $derived($reviewPly ?? total); // active ply shown on the board

	let moveRows = $derived.by(() => {
		const m = g?.moves ?? [];
		const rows = [];
		for (let i = 0; i < m.length; i += 2) {
			rows.push({
				n: i / 2 + 1,
				w: { ply: i + 1, san: m[i].san },
				b: m[i + 1] ? { ply: i + 2, san: m[i + 1].san } : null
			});
		}
		return rows;
	});

	const setPly = (p) => reviewPly.set(p >= total ? null : Math.max(0, p));
	const goFirst = () => reviewPly.set(total === 0 ? null : 0);
	const goPrev = () => setPly(cur - 1);
	const goNext = () => setPly(cur + 1);
	const goLast = () => reviewPly.set(null);

	function onKey(e) {
		if (e.target instanceof HTMLInputElement) return;
		if (e.key === 'ArrowLeft') { e.preventDefault(); goPrev(); }
		else if (e.key === 'ArrowRight') { e.preventDefault(); goNext(); }
	}

	let copied = $state(false);
	function copyLink() {
		navigator.clipboard?.writeText(location.href).then(() => {
			copied = true;
			setTimeout(() => (copied = false), 1400);
		});
	}
</script>

<svelte:window onkeydown={onKey} />

<div class="rail-stack">
	{#if waiting}
		<section class="panel share">
			<span class="spinner"></span>
			<h2>Waiting for an opponent</h2>
			<p class="muted">Share this link. The game starts the moment they join.</p>
			<button class="code-btn" class:is-copied={copied} onclick={copyLink}>
				<span class="code-val">{location.host}/game/{g.id}</span>
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
					{#each moveRows as r}
						<tr>
							<td class="n">{r.n}.</td>
							<td>
								<button class="ply" class:active={cur === r.w.ply} onclick={() => setPly(r.w.ply)}>{r.w.san}</button>
							</td>
							<td>
								{#if r.b}
									<button class="ply" class:active={cur === r.b.ply} onclick={() => setPly(r.b.ply)}>{r.b.san}</button>
								{/if}
							</td>
						</tr>
					{:else}
						<tr><td colspan="3" class="empty">No moves yet.</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
		<div class="nav">
			<button onclick={goFirst} disabled={cur === 0} aria-label="First move" title="First"><i class="fa fa-angles-left"></i></button>
			<button onclick={goPrev} disabled={cur === 0} aria-label="Previous move" title="Previous"><i class="fa fa-angle-left"></i></button>
			<button onclick={goNext} disabled={cur >= total} aria-label="Next move" title="Next"><i class="fa fa-angle-right"></i></button>
			<button onclick={goLast} disabled={cur >= total} aria-label="Latest move" title="Latest"><i class="fa fa-angles-right"></i></button>
		</div>
		{#if g && g.moves.length}
			<a class="back" href="/api/games/{g.id}/pgn" download><i class="fa fa-download"></i> Download PGN</a>
		{/if}
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
		max-height: min(64vh, 560px);
	}
	.moves-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 18px;
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
		min-height: 90px;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.92rem;
		table-layout: fixed;
	}
	td {
		padding: 2px 8px;
		vertical-align: middle;
	}
	td.n {
		width: 40px;
		padding-left: 16px;
		color: var(--text-faint);
		font-variant-numeric: tabular-nums;
		font-size: 0.82rem;
	}
	tr:nth-child(odd) {
		background: color-mix(in oklab, var(--surface-2) 60%, transparent);
	}
	.ply {
		width: 100%;
		text-align: left;
		padding: 6px 8px;
		border-radius: 6px;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
		color: var(--text);
		background: transparent;
		transition: background 0.1s var(--ease);
	}
	.ply:hover {
		background: var(--surface-3);
	}
	.ply.active {
		background: var(--accent);
		color: #fff;
	}
	.empty {
		color: var(--text-faint);
		text-align: center;
		padding: 16px;
	}
	.nav {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 4px;
		padding: 10px 12px;
		border-top: 1px solid var(--border);
	}
	.nav button {
		padding: 9px 0;
		font-size: 0.95rem;
		color: var(--text-muted);
		background: var(--surface-2);
		border-radius: var(--radius-sm);
		transition: color 0.12s var(--ease), background 0.12s var(--ease);
	}
	.nav button:hover:not(:disabled) {
		color: var(--accent);
		background: var(--surface-3);
	}
	.nav button:disabled {
		opacity: 0.4;
		cursor: default;
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
