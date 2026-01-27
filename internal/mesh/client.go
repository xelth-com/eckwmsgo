package mesh

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
)

// NewHTTPClient создает HTTP клиент с принудительным IPv4
func NewHTTPClient() *http.Client {
	// Create IPv4-only dialer
	ipv4Dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Force IPv4 dialing
				return ipv4Dialer.DialContext(ctx, "tcp4", addr)
			},
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true,
		},
	}
}

// CreateDiscoveryClient создает клиент для mesh discovery с IPv4
func CreateDiscoveryClient(cfg *config.Config) *http.Client {
	return NewHTTPClient()
}
