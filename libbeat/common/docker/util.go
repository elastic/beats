package docker

import (
	"net"
	"os"
	"strings"
)

// GetHostIP can look at the hostname and give the container gateway/host IP based on where the given
// beat is running.
func GetHostIP(watcher Watcher) string {
	hostname, err := os.Hostname()
	if watcher == nil || err != nil {
		return ""
	}

	// Assume that hostname is the container ID and attempt to do docker inspect to get its Gateway address.
	// Gateway address of a container is one of the interfaces on the host that can be used to connect to
	// containers that are being run on host network.
	for cid, container := range watcher.Containers() {
		if strings.Index(cid, hostname) == 0 && container != nil {
			if len(container.gateways) != 0 {
				return container.gateways[0]
			}
		}
	}

	// If the process is not running in a container then return a valid IPv4 address that can be used to access
	// the service as it would be running on host network.
	return getLocalIP()
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// Check the address type and if it is not a loop back then return it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
