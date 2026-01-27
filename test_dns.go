package main

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

func main() {
	fmt.Println("=== DNS Resolution Test for pda.repair ===\n")

	// 1. nslookup style (IPv4 and IPv6)
	addrs, err := net.LookupHost("pda.repair")
	if err != nil {
		fmt.Printf("❌ LookupHost error: %v\n", err)
		return
	}

	fmt.Printf("Found %d addresses:\n", len(addrs))
	for i, addr := range addrs {
		// Check IPv4 vs IPv6
		ip := net.ParseIP(addr)
		if ip == nil {
			fmt.Printf("  [%d] %s (invalid)\n", i, addr)
		} else if ip.To4() != nil {
			fmt.Printf("  [%d] %s (IPv4) ← TARGET\n", i, addr)
		} else if ip.To16() != nil {
			fmt.Printf("  [%d] %s (IPv6)\n", i, addr)
		} else {
			fmt.Printf("  [%d] %s (unknown)\n", i, addr)
		}
	}

	// 2. Test HTTP connection
	fmt.Println("\n=== Testing HTTP connection ===\n")

	// Try IPv4
	ipv4Addr := "152.53.15.15:443"
	fmt.Printf("Connecting to IPv4: %s...", ipv4Addr)
	conn, err := net.DialTimeout("tcp", ipv4Addr, 3*time.Second)
	if err != nil {
		fmt.Printf("  ❌ IPv4 connection failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ IPv4 connection SUCCESS!\n")
		conn.Close()
	}

	// Try IPv6
	ipv6Addr := "[2a02:908:2:b::1]:443"
	fmt.Printf("\nConnecting to IPv6: %s...", ipv6Addr)
	conn, err = net.DialTimeout("tcp", ipv6Addr, 3*time.Second)
	if err != nil {
		fmt.Printf("  ❌ IPv6 connection failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ IPv6 connection SUCCESS!\n")
		conn.Close()
	}

	// 3. Test actual URL
	fmt.Println("\n=== Testing actual URL ===\n")
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get("https://pda.repair/health")
	if err != nil {
		fmt.Printf("❌ HTTP GET failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✅ HTTP request successful!\n")
	fmt.Printf("   Status: %s\n", resp.Status)
	fmt.Printf("   Headers:\n")
	for k, v := range resp.Header {
		fmt.Printf("      %s: %s\n", k, v)
	}
}
