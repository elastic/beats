package transport

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
)

func fullAddress(host string, defaultPort int) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}

	idx := strings.Index(host, ":")
	if idx >= 0 {
		// IPv6 address detected
		return fmt.Sprintf("[%v]:%v", host, defaultPort)
	}
	return fmt.Sprintf("%v:%v", host, defaultPort)
}

// DialWith randomly dials one of a number of addresses with a given dialer.
//
// Use this to select and dial one IP being known for one host name.
func DialWith(
	dialer Dialer,
	network, host string,
	addresses []string,
	port string,
) (c net.Conn, err error) {
	switch len(addresses) {
	case 0:
		return nil, fmt.Errorf("no route to host %v", host)
	case 1:
		return dialer.Dial(network, net.JoinHostPort(addresses[0], port))
	}

	// Use randomization on DNS reported addresses combined with timeout and ACKs
	// to spread potential load when starting up large number of beats using
	// lumberjack.
	//
	// RFCs discussing reasons for ignoring order of DNS records:
	// http://www.ietf.org/rfc/rfc3484.txt
	// > is specific to locality-based address selection for multiple dns
	// > records, but exists as prior art in "Choose some different ordering for
	// > the dns records" done by a client
	//
	// https://tools.ietf.org/html/rfc1794
	// > "Clients, of course, may reorder this information" - with respect to
	// > handling order of dns records in a response.orwarded. Really required?
	for _, i := range rand.Perm(len(addresses)) {
		c, err = dialer.Dial(network, net.JoinHostPort(addresses[i], port))
		if err == nil && c != nil {
			return c, err
		}
	}

	if err == nil {
		err = fmt.Errorf("unable to connect to '%v'", host)
	}
	return nil, err
}
