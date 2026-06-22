// Turn a generated handle like "creative-raccoon" into "Creative Raccoon".
export function formatUserName(name) {
	if (!name) return 'Anonymous';
	return name
		.split('-')
		.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
		.slice(0, 2)
		.join(' ');
}
