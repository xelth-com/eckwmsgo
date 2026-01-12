package middleware

import (
	"net/http"
	"strings"
)

// CaseInsensitiveMiddleware converts all URL paths to lowercase
// This allows API endpoints to be accessed regardless of case
// Example: /API/status and /api/status both work
// Useful for QR codes where uppercase letters are more efficient
func CaseInsensitiveMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Convert path to lowercase
		r.URL.Path = strings.ToLower(r.URL.Path)

		// Pass to next handler with modified path
		next.ServeHTTP(w, r)
	})
}
