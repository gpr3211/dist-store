package server

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ScanNetwork scans the local network and finds connected ips.
func ScanNetwork(timeout time.Duration) ([]string, error) {
	subnet, err := GetLocalSubnet()
	if err != nil {
		return nil, err
	}

	var activeHosts []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Generate all IPs in the subnet
	for ip := subnet.IP.Mask(subnet.Mask); subnet.Contains(ip); incIP(ip) {
		ipStr := ip.String()
		wg.Add(1)

		go func(host string) {
			defer wg.Done()
			if isHostActive(host, timeout) {
				mu.Lock()
				activeHosts = append(activeHosts, host)
				mu.Unlock()
			}
		}(ipStr)
	}

	wg.Wait()
	return activeHosts, nil
}

// GetLocalSubnet returns the local IP and subnet mask.
func GetLocalSubnet() (*net.IPNet, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet, nil
			}
		}
	}

	return nil, fmt.Errorf("no local subnet found")
}

// incIP increments an IP address.
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// isHostActive checks if a host is active by attempting connections to common ports.
func isHostActive(ip string, timeout time.Duration) bool {
	// Try common ports: HTTP, HTTPS, SSH, SMB
	ports := []string{"80", "443", "22", "445"}

	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), timeout)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no local network IP address found")
}
