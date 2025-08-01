// Package util provides utility functions and helpers for common operations.
//
//nolint:revive,nolintlint // util is an established package name in this codebase
package util

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// GetIP retrieves the client's IP address from an HTTP request.
// It checks for common proxy headers and falls back to the remote address.
func GetIP(r *http.Request) string {
	// 1. Check for X-Forwarded-For header
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// The X-Forwarded-For header can contain a comma-separated list of IPs.
		// The first one is the original client.
		ips := strings.Split(xForwardedFor, ",")
		for i, ip := range ips {
			ips[i] = strings.TrimSpace(ip)
		}
		// It's important to return the first IP in the list.
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// 2. Check for X-Real-IP header
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// 3. Fallback to RemoteAddr
	// RemoteAddr contains IP and port, so we need to split it.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If splitting fails, it might be just an IP address without a port.
		return r.RemoteAddr
	}

	return ip
}

// GetLocalIP convenience method that obtains the non localhost ip address for machine running app.
func GetLocalIP() string {
	addrs, _ := net.InterfaceAddrs()

	currentIP := ""

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			currentIP = ipnet.IP.String()
			break
		}
		if ipnet, ok := address.(*net.IPNet); ok {
			currentIP = ipnet.IP.String()
		}
	}

	return currentIP
}

// GetMacAddress convenience method to get some unique address based on the network interfaces the application is running on.
func GetMacAddress() string {
	currentIP := GetLocalIP()

	interfaces, _ := net.Interfaces()
	for _, interf := range interfaces {
		if addrs, err := interf.Addrs(); err == nil {
			for _, addr := range addrs {
				// only interested in the name with current IP address
				if strings.Contains(addr.String(), currentIP) {
					return fmt.Sprintf("%s:%s", interf.Name, interf.HardwareAddr.String())
				}
			}
		}
	}
	return ""
}
