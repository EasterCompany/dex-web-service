package endpoints

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// OpenHandler handles redirects to non-standard protocols (ssh, mosh, etc.)
// enabling deep linking from apps like Discord that filter them.
func OpenHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "Target URL required", http.StatusBadRequest)
		return
	}

	// Security: Only allow safe/expected schemes
	u, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}

	allowedSchemes := map[string]bool{
		"ssh":  true,
		"mosh": true,
		"vnc":  true,
		"sftp": true,
		"ish":  true, // Just in case they add support
	}

	if !allowedSchemes[strings.ToLower(u.Scheme)] {
		http.Error(w, fmt.Sprintf("Scheme '%s' not allowed. Only ssh, mosh, vnc, sftp are supported.", u.Scheme), http.StatusForbidden)
		return
	}

	// 307 Temporary Redirect preserves the method and body, though irrelevant for GET
	// It tells the browser "Go here now", which triggers the OS app handler.
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}
