package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// GetLocalIPs returns all non-loopback IPv4 addresses with smart filtering
// It filters out Link-Local (169.254.x.x) addresses ONLY IF a better alternative exists.
func GetLocalIPs() []string {
	var allIPs []string
	var hasRoutableIP bool

	// 1. Collect all IPs
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return allIPs
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				allIPs = append(allIPs, ip)

				// Check if this is a "good" IP (not APIPA)
				if !strings.HasPrefix(ip, "169.254") {
					hasRoutableIP = true
				}
			}
		}
	}

	// 2. Filter based on availability
	var finalIPs []string
	for _, ip := range allIPs {
		// If we have a good IP, skip garbage (169.254)
		// If we DON'T have a good IP, keep everything (panic mode)
		if hasRoutableIP && strings.HasPrefix(ip, "169.254") {
			continue
		}
		finalIPs = append(finalIPs, ip)
	}

	return finalIPs
}

// ReportToGlobalServer reports this instance's local IPs to the global server for discovery
func ReportToGlobalServer() {
	instanceID := os.Getenv("INSTANCE_ID")
	globalURL := os.Getenv("GLOBAL_SERVER_URL")
	apiKey := os.Getenv("GLOBAL_SERVER_API_KEY")
	port := os.Getenv("PORT")
	if port == "" {
		port = "3210"
	}

	if instanceID == "" || globalURL == "" || apiKey == "" {
		fmt.Println("[Discovery] Skipping report: INSTANCE_ID, GLOBAL_SERVER_URL or API_KEY not set")
		return
	}

	payload := map[string]interface{}{
		"instanceId":      instanceID,
		"localIps":        GetLocalIPs(),
		"port":            port,
		"serverPublicKey": os.Getenv("SERVER_PUBLIC_KEY"),
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", globalURL+"/ECK/API/INTERNAL/REGISTER-INSTANCE", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("[Discovery] Failed to create request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Api-Key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[Discovery] Failed to report to global server: %v\n", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[Discovery] Global server report status: %d\n", resp.StatusCode)
}
