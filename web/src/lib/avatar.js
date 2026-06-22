// Deterministic initials avatar as an SVG data URI (no network round-trip).
// Same hue for the same name. Ported from the original board.js.
export function avatarUrl(name) {
	const initials = (name || '?')
		.split(/\s+/)
		.map((w) => w.charAt(0))
		.slice(0, 2)
		.join('')
		.toUpperCase();
	let hash = 0;
	for (let i = 0; i < (name || '').length; i++) {
		hash = (name.charCodeAt(i) + ((hash << 5) - hash)) | 0;
	}
	const hue = Math.abs(hash) % 360;
	const svg =
		`<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80" viewBox="0 0 80 80">` +
		`<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">` +
		`<stop offset="0" stop-color="hsl(${hue},55%,52%)"/>` +
		`<stop offset="1" stop-color="hsl(${(hue + 38) % 360},55%,40%)"/>` +
		`</linearGradient></defs>` +
		`<rect width="80" height="80" rx="18" fill="url(#g)"/>` +
		`<text x="50%" y="50%" dy="0.35em" text-anchor="middle" ` +
		`font-family="Montserrat,Inter,system-ui,sans-serif" font-size="34" font-weight="700" fill="#fff">${initials}</text>` +
		`</svg>`;
	return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}
