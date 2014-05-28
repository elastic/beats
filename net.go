package main

import (
    "net"
)

func LocalAddrs() ([]string, error) {
    var localAddrs = []string{}
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return nil, err
    }
    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            localAddrs = append(localAddrs, ipnet.IP.String())
        }
    }
    return localAddrs, nil
}
