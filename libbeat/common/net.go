package common

import (
	"fmt"
	"net"
)

// LocalIpAddrs finds the IP addresses of the hosts on which
// the shipper currently runs on.
func LocalIpAddrs() ([]net.IP, error) {
	var localIPAddrs = []net.IP{}
	ipaddrs, err := net.InterfaceAddrs()
	if err != nil {
		return []net.IP{}, err
	}
	for _, ipaddr := range ipaddrs {
		if ipnet, ok := ipaddr.(*net.IPNet); ok {
			localIPAddrs = append(localIPAddrs, ipnet.IP)
		}
	}
	return localIPAddrs, nil
}

// LocalIpAddrsAsStrings finds the IP addresses of the hosts on which
// the shipper currently runs on and returns them as an array of
// strings.
func LocalIpAddrsAsStrings(include_loopbacks bool) ([]string, error) {
	var localIPAddrsStrings = []string{}
	var err error
	ipaddrs, err := LocalIpAddrs()
	if err != nil {
		return []string{}, err
	}
	for _, ipaddr := range ipaddrs {
		if include_loopbacks || !ipaddr.IsLoopback() {
			localIPAddrsStrings = append(localIPAddrsStrings, ipaddr.String())
		}
	}
	return localIPAddrsStrings, err
}

// IsLoopback check if a particular IP notation corresponds
// to a loopback interface.
func IsLoopback(ip_str string) (bool, error) {
	ip := net.ParseIP(ip_str)
	if ip == nil {
		return false, fmt.Errorf("Wrong IP format %s", ip_str)
	}
	return ip.IsLoopback(), nil
}
