<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { currentGame, user } from '$lib/stores.js';
	import { listGames, createGame, createBotGame, joinGame } from '$lib/api.js';
	import { formatUserName } from '$lib/format.js';

	let games = $state([]);
	let tab = $state('mine');
	let botLevel = $state('medium');
	let inviteCode = $state('');
	let creating = $state(false);
	// Reactive: recomputes once the anonymous user hydrates, so grouping is correct.
	let myId = $derived($user?.id ?? null);

	onMount(() => {
		currentGame.set(null); // idle board in the lobby
		botLevel = localStorage.getItem('botLevel') || 'medium';
		refresh();
	});

	async function refresh() {
		games = await listGames();
	}

	const amPlayer = (g) =>
		myId != null && ((g.white_user && g.white_user.id === myId) || (g.black_user && g.black_user.id === myId));
	const hasOpenSeat = (g) => !g.white_user || !g.black_user;
	const isFinished = (g) => g.winner === 'w' || g.winner === 'b';

	let mine = $derived(games.filter((g) => amPlayer(g)));
	let open = $derived(games.filter((g) => !amPlayer(g) && hasOpenSeat(g) && !isFinished(g)));
	let watch = $derived(games.filter((g) => !amPlayer(g) && !(hasOpenSeat(g) && !isFinished(g))));
	let shown = $derived(tab === 'mine' ? mine : tab === 'open' ? open : watch);

	async function onCreate() {
		creating = true;
		const r = await createGame();
		creating = false;
		if (r.response_key === 'SUCCESS') goto(`/game/${r.data}`);
	}
	async function onBot() {
		creating = true;
		localStorage.setItem('botLevel', botLevel);
		const r = await createBotGame(botLevel);
		creating = false;
		if (r.response_key === 'SUCCESS') goto(`/game/${r.data}`);
	}
	async function onJoinCode() {
		if (!inviteCode.trim()) return;
		const r = await joinGame(inviteCode.trim());
		if (r.response_key === 'SUCCESS') goto(`/game/${r.data}`);
		else alert('Invalid invite code!');
	}
	async function onRow(g) {
		if (!amPlayer(g) && hasOpenSeat(g) && !isFinished(g)) {
			const r = await joinGame(g.invite_code);
			if (r.response_key === 'SUCCESS') return goto(`/game/${r.data}`);
			return alert('Could not join this game.');
		}
		goto(`/game/${g.id}`);
	}
</script>

<div class="rail-stack">
	<section class="panel console">
		<span class="eyebrow">Ready to play?</span>
		<h1 class="greet">Hey <span class="accent">{$user ? formatUserName($user.name) : 'player'}</span></h1>

		<button class="btn-create" onclick={onCreate} disabled={creating}>
			<span class="btn-create-icon"><i class="fa fa-plus"></i></span>
			<span class="btn-create-text">
				<strong>Create a game</strong>
				<small>Random side, share the code to invite</small>
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

		<div class="divider"><span>or join with a code</span></div>
		<div class="join-row">
			<input class="field" placeholder="Paste an invite code" bind:value={inviteCode}
				onkeydown={(e) => e.key === 'Enter' && onJoinCode()} />
			<button class="btn-ghost" onclick={onJoinCode}>Join</button>
		</div>
	</section>

	<section class="panel games">
		<div class="games-head">
			{#each [['mine', 'Your games', mine.length], ['open', 'Open tables', open.length], ['watch', 'Watch', watch.length]] as [key, label, n]}
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
						<span class="gr-name">{g.white_user ? formatUserName(g.white_user.name) : 'Open'}</span>
						<span class="gr-vs">vs</span>
						<span class="gr-name">{g.black_user ? formatUserName(g.black_user.name) : 'Open'}</span>
					</span>
					<i class="fa fa-chevron-right gr-go"></i>
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
	.gr-go {
		color: var(--text-faint);
	}
	.games-empty {
		padding: 30px;
		text-align: center;
		color: var(--text-faint);
		font-size: 0.9rem;
	}
</style>
