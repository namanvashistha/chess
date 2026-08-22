package pkg

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// CheckWebSocketOrigin decides whether a WebSocket handshake may proceed.
//
// The previous implementation returned true unconditionally, which let any
// website on the internet open an authenticated socket to a user's game
// (cross-site WebSocket hijacking). The default is now same-origin; set
// ALLOWED_ORIGINS to a comma-separated list to permit specific others, or to
// "*" to restore the old behaviour deliberately.
func CheckWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser client (CLI tooling, health checks): no ambient
		// credentials to abuse, so there is nothing for CSWSH to steal.
		return true
	}

	allowed := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if allowed == "*" {
		return true
	}
	for _, entry := range strings.Split(allowed, ",") {
		entry = strings.TrimSpace(entry)
		if entry != "" && strings.EqualFold(entry, origin) {
			return true
		}
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	// Same-origin default: the Origin host must match the host being requested.
	return strings.EqualFold(u.Host, r.Host)
}
