import { writable } from 'svelte/store';

// The position currently shown on the persistent board. null => idle/decorative.
export const currentGame = writable(null);

// The signed-in (anonymous) user, hydrated once on load.
export const user = writable(null);
