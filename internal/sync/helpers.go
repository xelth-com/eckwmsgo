package sync

import (
	"fmt"
	"log"
)

	"github.com/xelth-com/eckwmsgo/internal/config"
)

// MakeAuthenticatedRequest creates an authenticated HTTP request with Bearer token
func MakeAuthenticatedRequest(method, url string, body []byte, nodeID string) (http.Header, *http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, req, err
	}

	// Set Authorization header
	req.Header.Set("Authorization", "Bearer "+nodeID)

	// Set Content-Type
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}
