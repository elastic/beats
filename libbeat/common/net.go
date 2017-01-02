package common

import (
	"fmt"
	"net"
)

// LocalIPAddrs finds the IP addresses of the hosts on which
// the shipper currently runs on.
func LocalIPAddrs() ([]net.IP, error) {
	var localIPAddrs []net.IP
	ipaddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range ipaddrs {
		var ip net.IP
		ok := true

		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			ok = false
		}

		if !ok {
			continue
		}

		localIPAddrs = append(localIPAddrs, ip)
	}
	return localIPAddrs, nil
}

// LocalIPAddrsAsStrings finds the IP addresses of the hosts on which
// the shipper currently runs on and returns them as an array of
// strings.
func LocalIPAddrsAsStrings(includeLoopbacks bool) ([]string, error) {
	var localIPAddrsStrings = []string{}
	var err error
	ipaddrs, err := LocalIPAddrs()
	if err != nil {
		return []string{}, err
	}
	for _, ipaddr := range ipaddrs {
		if includeLoopbacks || !ipaddr.IsLoopback() {
			localIPAddrsStrings = append(localIPAddrsStrings, ipaddr.String())
		}
	}
	return localIPAddrsStrings, err
}

// IsLoopback check if a particular IP notation corresponds
// to a loopback interface.
func IsLoopback(ipStr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("Wrong IP format %s", ipStr)
	}
	return ip.IsLoopback(), nil
}
