package sync

import (
	"bytes"
	"net/http"
)

// MakeAuthenticatedRequest creates an authenticated HTTP request with Bearer token
func MakeAuthenticatedRequest(method, url string, body []byte, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// Set Authorization header
	req.Header.Set("Authorization", "Bearer "+token)

	// Set Content-Type
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}
