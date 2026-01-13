package middleware

import (
	"net/http"
	"strings"
)

// CaseInsensitiveMiddleware converts API URL paths to lowercase
// This allows API endpoints to be accessed regardless of case
// Example: /API/status and /api/status both work
// Useful for QR codes where uppercase letters are more efficient
// Static files (/i/, files with extensions) are left unchanged
func CaseInsensitiveMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Don't modify paths for static files
		// - Paths containing /i/ (SvelteKit assets - e.g., /i/ or /E/i/)
		// - Paths with file extensions (e.g., .js, .css, .png)
		if strings.Contains(path, "/i/") ||
		   strings.Contains(path, ".") && !strings.HasSuffix(path, "/") {
			// Pass through unchanged for static files
			next.ServeHTTP(w, r)
			return
		}

		// Convert API/auth paths to lowercase
		r.URL.Path = strings.ToLower(path)

		// Pass to next handler with modified path
		next.ServeHTTP(w, r)
	})
}
