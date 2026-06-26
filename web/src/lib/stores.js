import { writable } from 'svelte/store';

// The position currently shown on the persistent board. null => idle/decorative.
export const currentGame = writable(null);

// The signed-in (anonymous) user, hydrated once on load.
export const user = writable(null);

// Move-review cursor: null = follow the live position; otherwise the ply count
// (0 = start) currently shown on the board.
export const reviewPly = writable(null);
