package network

import (
	"fmt"
	"net"
)

func IsPortAvailable(port int) bool {
	if port <= 0 {
		return false
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func ResolvePort(base int) int {
	if base <= 0 {
		return 0
	}

	port := base
	for {
		if IsPortAvailable(port) {
			return port
		}
		port++
	}
}
