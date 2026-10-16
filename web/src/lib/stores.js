import { writable } from 'svelte/store';

// The game shown on the persistent board (see game.js for its shape), or null
// for the idle board in the lobby.
export const currentGame = writable(null);

// The signed-in (anonymous) user, hydrated once on load.
export const user = writable(null);

// Move-review cursor: null follows the live position; otherwise the ply count
// (0 = start) currently shown on the board.
export const reviewPly = writable(null);

// The last error the server sent for this client's move, shown briefly.
export const moveError = writable(null);
