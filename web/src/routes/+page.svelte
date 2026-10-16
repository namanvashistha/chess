<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { currentGame, user } from '$lib/stores.js';
	import { lobby, createGame, joinGame } from '$lib/api.js';
	import { formatUserName } from '$lib/format.js';

	let games = $state({ open: [], live: [], mine: [] });
	let tab = $state('mine');
	let botLevel = $state('medium');
	let link = $state('');
	let creating = $state(false);
	let error = $state('');

	onMount(() => {
		currentGame.set(null); // idle board in the lobby
		botLevel = localStorage.getItem('botLevel') || 'medium';
		refresh();
	});

	// The lobby depends on who is asking, so reload once the session exists.
	$effect(() => {
		if ($user) refresh();
	});

	async function refresh() {
		try {
			games = await lobby();
		} catch {
			// keep the last list
		}
	}

	let shown = $derived(tab === 'mine' ? games.mine : tab === 'open' ? games.open : games.live);

	async function start(options) {
		creating = true;
		error = '';
		try {
			goto(`/game/${await createGame(options)}`);
		} catch (e) {
			error = e.message;
		} finally {
			creating = false;
		}
	}
	const onCreate = () => start({ mode: 'online' });
	function onBot() {
		localStorage.setItem('botLevel', botLevel);
		start({ mode: 'bot', level: botLevel });
	}
	const onLocal = () => start({ mode: 'local' });

	// Accepts a full game link or just its id.
	async function onJoinLink() {
		const id = link.trim().split('/').filter(Boolean).pop();
		if (!id) return;
		await join(id);
	}
	async function join(id) {
		error = '';
		try {
			await joinGame(id);
			goto(`/game/${id}`);
		} catch (e) {
			error = e.message;
		}
	}
	function onRow(g) {
		if (tab === 'open') join(g.id);
		else goto(`/game/${g.id}`);
	}

	const statusLabel = (g) =>
		g.status === 'finished' ? g.result : g.status === 'waiting' ? 'Waiting' : `Move ${Math.floor(g.plies / 2) + 1}`;
</script>

<div class="rail-stack">
	<section class="panel console">
		<span class="eyebrow">Ready to play?</span>
		<h1 class="greet">Hey <span class="accent">{$user ? formatUserName($user.name) : 'player'}</span></h1>

		<button class="btn-create" onclick={onCreate} disabled={creating}>
			<span class="btn-create-icon"><i class="fa fa-plus"></i></span>
			<span class="btn-create-text">
				<strong>Create a game</strong>
				<small>Random side, share the link to invite</small>
			</span>
			<i class="fa fa-arrow-right btn-create-go"></i>
		</button>

		<button class="btn-bot" onclick={onBot} disabled={creating}>
			<i class="fa fa-robot"></i><span>Play against the bot</span>
		</button>
		<div class="bot-levels" role="group" aria-label="Bot difficulty">
			{#each ['easy', 'medium', 'hard'] as lvl}
				<button class="bot-level" class:is-active={botLevel === lvl} onclick={() => (botLevel = lvl)}>
					{lvl[0].toUpperCase() + lvl.slice(1)}
				</button>
			{/each}
		</div>

		<button class="btn-bot" onclick={onLocal} disabled={creating}>
			<i class="fa fa-users"></i><span>Pass &amp; play</span>
		</button>

		<div class="divider"><span>or join with a link</span></div>
		<div class="join-row">
			<input class="field" placeholder="Paste a game link" bind:value={link}
				onkeydown={(e) => e.key === 'Enter' && onJoinLink()} />
			<button class="btn-ghost" onclick={onJoinLink}>Join</button>
		</div>
		{#if error}<p class="form-error" role="alert">{error}</p>{/if}
	</section>

	<section class="panel games">
		<div class="games-head">
			{#each [['mine', 'Your games', games.mine.length], ['open', 'Open tables', games.open.length], ['watch', 'Watch', games.live.length]] as [key, label, n]}
				<button class="tab" class:is-active={tab === key} onclick={() => (tab = key)}>
					{label} <span class="tab-count">{n}</span>
				</button>
			{/each}
			<button class="icon-btn tab-refresh" onclick={refresh} title="Refresh" aria-label="Refresh"><i class="fa fa-rotate-right"></i></button>
		</div>
		<div class="games-list">
			{#each shown as g (g.id)}
				<button class="game-row" onclick={() => onRow(g)}>
					<span class="gr-players">
						<span class="gr-name">{g.white ? formatUserName(g.white.name) : 'Open'}</span>
						<span class="gr-vs">vs</span>
						<span class="gr-name">{g.black ? formatUserName(g.black.name) : 'Open'}</span>
					</span>
					<span class="gr-status">{statusLabel(g)}</span>
				</button>
			{:else}
				<p class="games-empty">Nothing here yet.</p>
			{/each}
		</div>
	</section>
</div>

<style>
	.rail-stack {
		display: flex;
		flex-direction: column;
		gap: 18px;
	}
	.console {
		display: flex;
		flex-direction: column;
		gap: 14px;
		padding: 26px;
	}
	.greet {
		font-family: var(--font-display);
		font-size: 1.6rem;
		font-weight: 800;
		letter-spacing: -0.01em;
	}
	.accent {
		color: var(--accent);
	}
	.join-row {
		display: flex;
		gap: 10px;
	}
	.games {
		display: flex;
		flex-direction: column;
		max-height: 50vh;
	}
	.games-head {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 10px 12px;
		border-bottom: 1px solid var(--border);
	}
	.tab {
		padding: 7px 11px;
		font-size: 0.82rem;
		font-weight: 600;
		color: var(--text-muted);
		background: transparent;
		border-radius: var(--radius-sm);
	}
	.tab.is-active {
		color: var(--accent);
		background: var(--surface-2);
	}
	.tab-count {
		font-variant-numeric: tabular-nums;
		opacity: 0.7;
	}
	.tab-refresh {
		margin-left: auto;
		width: 32px;
		height: 32px;
		font-size: 12px;
	}
	.games-list {
		flex: 1;
		overflow-y: auto;
		padding: 10px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.game-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 14px;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		text-align: left;
		transition: border-color 0.15s var(--ease), transform 0.15s var(--ease);
	}
	.game-row:hover {
		border-color: var(--accent);
		transform: translateY(-1px);
	}
	.gr-players {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 0.9rem;
		font-weight: 600;
	}
	.gr-vs {
		color: var(--text-faint);
		font-weight: 500;
		font-size: 0.8rem;
	}
	.gr-status {
		color: var(--text-faint);
		font-size: 0.8rem;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}
	.form-error {
		color: var(--danger);
		font-size: 0.85rem;
	}
	.games-empty {
		padding: 30px;
		text-align: center;
		color: var(--text-faint);
		font-size: 0.9rem;
	}
</style>
