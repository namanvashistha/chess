<script>
	// Persistent board. Renders from a game's current_state (or a decorative
	// start position when idle). Click-to-move and smooth pointer drag (the
	// dragged piece follows the cursor) for the side to move; emits
	// (piece, source, destination) via onmove.
	let {
		game = null,
		myId = null,
		pov = 'w',
		onmove = () => {},
		position = null, // override board map (move review); null = live current_state
		reviewing = false, // viewing a past ply -> read-only
		lastMoveStr = null, // override last-move highlight for the viewed ply
		local = false // pass & play: one client controls both sides
	} = $props();

	const FILES = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'];
	const PIECE = {
		P: 'wP', R: 'wR', N: 'wN', B: 'wB', Q: 'wQ', K: 'wK',
		p: 'bP', r: 'bR', n: 'bN', b: 'bB', q: 'bQ', k: 'bK'
	};
	const url = (code) => `/static/images/${PIECE[code]}.svg`;

	function idleState() {
		const m = {};
		const back = ['R', 'N', 'B', 'Q', 'K', 'B', 'N', 'R'];
		for (let f = 0; f < 8; f++) {
			m[FILES[f] + '8'] = back[f].toLowerCase();
			m[FILES[f] + '7'] = 'p';
			m[FILES[f] + '2'] = 'P';
			m[FILES[f] + '1'] = back[f];
		}
		m['e4'] = 'P';
		delete m['e2'];
		return m;
	}

	let gridEl;
	let selected = $state(null);
	let drag = $state(null); // { from, code, x, y, moved, size }

	// Reviewing a past ply is read-only: never carry a selection/drag into it.
	$effect(() => {
		if (reviewing) {
			selected = null;
			drag = null;
		}
	});

	let state = $derived(position ?? game?.current_state ?? idleState());
	let legal = $derived(reviewing ? {} : (game?.legal_moves ?? {}));
	let myColor = $derived(
		!game || myId == null
			? null
			: game.white_user && game.white_user.id === myId
				? 'w'
				: game.black_user && game.black_user.id === myId
					? 'b'
					: null
	);
	// In pass & play either side may move; otherwise only your own colour on your turn.
	let isMyTurn = $derived(
		!!(game && !reviewing && !game.winner && (local || (myColor && game.state?.turn === myColor)))
	);

	let lastMove = $derived.by(() => {
		const lm = reviewing ? lastMoveStr : game?.state?.last_move;
		if (!lm || lm.length < 5) return new Set();
		return new Set([lm.substring(1, 3), lm.substring(3, 5)]);
	});
	let targets = $derived(selected && legal[selected] ? new Set(legal[selected]) : new Set());

	let cells = $derived.by(() => {
		const out = [];
		for (let r = 0; r < 8; r++) {
			for (let c = 0; c < 8; c++) {
				const file = pov === 'w' ? FILES[c] : FILES[7 - c];
				const rank = pov === 'w' ? 8 - r : r + 1;
				const sq = file + rank;
				out.push({
					sq,
					file,
					rank,
					dark: (file.charCodeAt(0) - 97 + rank) % 2 === 1,
					piece: state[sq] || null,
					last: lastMove.has(sq),
					showFile: r === 7,
					showRank: c === 0
				});
			}
		}
		return out;
	});

	const movableFrom = (sq) => isMyTurn && Array.isArray(legal[sq]) && legal[sq].length > 0;

	function doMove(from, to) {
		const piece = state[from];
		selected = null;
		drag = null;
		if (piece) onmove(piece, from, to);
	}

	function squareAt(x, y) {
		const el = document.elementFromPoint(x, y);
		return el?.closest('[data-sq]')?.getAttribute('data-sq') ?? null;
	}

	function onPointerDown(cell, e) {
		if (!e.isPrimary || e.button > 0) return;
		if (!isMyTurn) {
			selected = null;
			return;
		}
		// A piece is selected and we tapped a legal target -> move.
		if (selected && selected !== cell.sq && targets.has(cell.sq)) {
			doMove(selected, cell.sq);
			return;
		}
		if (movableFrom(cell.sq)) {
			e.preventDefault();
			selected = cell.sq;
			const size = gridEl ? gridEl.clientWidth / 8 : 64;
			drag = {
				from: cell.sq,
				code: state[cell.sq],
				x: e.clientX,
				y: e.clientY,
				startX: e.clientX,
				startY: e.clientY,
				moved: false,
				size
			};
		} else {
			selected = null;
		}
	}

	function onWindowMove(e) {
		if (!drag) return;
		// Only treat it as a drag once the pointer leaves the origin square's
		// rough footprint, so a jittery click isn't mistaken for a drag.
		const threshold = drag.size ? drag.size * 0.35 : 10;
		if (!drag.moved && Math.hypot(e.clientX - drag.startX, e.clientY - drag.startY) < threshold) {
			drag.x = e.clientX;
			drag.y = e.clientY;
			return;
		}
		drag.moved = true;
		drag.x = e.clientX;
		drag.y = e.clientY;
	}

	function onWindowUp(e) {
		if (!drag) return;
		const d = drag;
		drag = null;
		if (!d.moved) return; // a tap: keep the selection for click-to-move
		const sq = squareAt(e.clientX, e.clientY);
		if (sq && sq !== d.from && legal[d.from]?.includes(sq)) {
			doMove(d.from, sq);
		}
		// Dropped off-target (or jittered back onto the source): keep `selected`
		// so clicking a destination square still completes the move.
	}
</script>

<svelte:window onpointermove={onWindowMove} onpointerup={onWindowUp} />

<div class="board-frame">
	<div class="board-grid" bind:this={gridEl}>
		{#each cells as cell (cell.sq)}
			<div
				class="sq {cell.dark ? 'sq--d' : 'sq--l'}"
				data-sq={cell.sq}
				class:is-last={cell.last}
				class:is-select={!reviewing && selected === cell.sq}
				class:is-move={targets.has(cell.sq)}
				class:has-piece={targets.has(cell.sq) && cell.piece}
				class:can-move={movableFrom(cell.sq)}
				role="button"
				tabindex="-1"
				onpointerdown={(e) => onPointerDown(cell, e)}
			>
				{#if cell.piece}
					<img
						class="piece-img"
						class:is-dragging={drag?.from === cell.sq && drag?.moved}
						src={url(cell.piece)}
						alt=""
						draggable="false"
					/>
				{/if}
				{#if cell.showFile}<span class="sq-coord file">{cell.file}</span>{/if}
				{#if cell.showRank}<span class="sq-coord rank">{cell.rank}</span>{/if}
			</div>
		{/each}
	</div>
</div>

{#if drag?.moved}
	<img
		class="drag-ghost"
		src={url(drag.code)}
		alt=""
		style="left:{drag.x}px; top:{drag.y}px; width:{drag.size}px; height:{drag.size}px;"
	/>
{/if}

<style>
	.piece-img.is-dragging {
		opacity: 0;
	}
	.drag-ghost {
		position: fixed;
		z-index: 1000;
		transform: translate(-50%, -50%);
		pointer-events: none;
		object-fit: contain;
		filter: drop-shadow(0 6px 10px rgba(0, 0, 0, 0.4));
		cursor: grabbing;
	}
</style>
