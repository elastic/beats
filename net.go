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
        // a bit wtf'ish.. Don't know how to do this otherwise
        ip, _, err := net.ParseCIDR(addr.String())
        if ip.IsLoopback() {
            continue
        }
        if err == nil && ip != nil {
            localAddrs = append(localAddrs, ip.String())
        }
    }
    return localAddrs, nil
}
