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

	// iOS Safari often blocks 307 redirects to custom schemes from within in-app browsers (like Discord).
	// We serve a lightweight landing page with a manual button to ensure the action is "user-initiated".
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Opening Terminal...</title>
    <style>
        body {
            background-color: #121212;
            color: #ffffff;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            height: 100vh;
            margin: 0;
            text-align: center;
        }
        .btn {
            background-color: #03dac6;
            color: #000000;
            border: none;
            padding: 16px 32px;
            text-align: center;
            text-decoration: none;
            display: inline-block;
            font-size: 18px;
            font-weight: bold;
            border-radius: 8px;
            cursor: pointer;
            box-shadow: 0 4px 6px rgba(0,0,0,0.3);
            transition: transform 0.1s;
        }
        .btn:active {
            transform: scale(0.98);
        }
        p {
            margin-top: 20px;
            color: #888;
            font-size: 14px;
        }
        .url {
            color: #555;
            font-family: monospace;
            font-size: 12px;
            max-width: 90vw;
            word-break: break-all;
        }
    </style>
</head>
<body>
    <a id="launch-btn" href="%s" class="btn">Open Terminal</a>
    <p>If the app doesn't open automatically, tap the button.</p>
    <div class="url">%s</div>

    <script>
        // Attempt automatic redirect after a short delay
        setTimeout(() => {
            window.location.href = "%s";
        }, 500);
    </script>
</body>
</html>
`, target, target, target)

	_, _ = w.Write([]byte(html))
}
