package socket

import (
	"net"
)

// Direction indicates how a socket was initiated.
type Direction uint8

const (
	_ Direction = iota
	// Incoming indicates a connection was established from the outside to
	// listening socket on this host.
	Incoming
	// Outgoing indicates a connection was established from this socket to an
	// external listening socket.
	Outgoing
	// Listening indicates a socket that is listening.
	Listening
)

var directionNames = map[Direction]string{
	Incoming:  "incoming",
	Outgoing:  "outgoing",
	Listening: "listening",
}

func (d Direction) String() string {
	if name, exists := directionNames[d]; exists {
		return name
	}
	return "unknown"
}

// ipList is a list of IP addresses.
type ipList struct {
	ips []net.IP
}

func (l *ipList) put(ip net.IP) { l.ips = append(l.ips, ip) }

// portTable is a mapping of port number to listening IP addresses.
type portTable map[int]*ipList

// protocolTable is a mapping of protocol numbers to listening ports.
type protocolTable map[uint8]portTable

// ListenerTable tracks sockets that are listening. It can then be used to
// identify if a socket is listening, incoming, or outgoing.
type ListenerTable struct {
	data protocolTable
}

// NewListenerTable returns a new ListenerTable.
func NewListenerTable() *ListenerTable {
	return &ListenerTable{
		data: protocolTable{},
	}
}

// Reset resets all data in the table.
func (t *ListenerTable) Reset() {
	for _, ports := range t.data {
		for port := range ports {
			delete(ports, port)
		}
	}
}

// Put puts a new listening address into the table.
func (t *ListenerTable) Put(proto uint8, ip net.IP, port int) {
	ports, exists := t.data[proto]
	if !exists {
		ports = portTable{}
		t.data[proto] = ports
	}

	// Add port + addr to table.
	interfaces, exists := ports[port]
	if !exists {
		interfaces = &ipList{}
		ports[port] = interfaces
	}
	interfaces.put(ip)
}

// Direction returns whether the connection was incoming or outgoing based on
// the protocol and local address. It compares the given local address to the
// listeners in the table for the protocol and returns Incoming if there is a
// match. If remotePort is 0 then Listening is returned.
func (t *ListenerTable) Direction(
	proto uint8,
	localIP net.IP, localPort int,
	remoteIP net.IP, remotePort int,
) Direction {
	if remotePort == 0 {
		return Listening
	}

	// Are there any listeners on the given protocol?
	ports, exists := t.data[proto]
	if !exists {
		return Outgoing
	}

	// Is there any listener on the port?
	interfaces, exists := ports[localPort]
	if !exists {
		return Outgoing
	}

	// Is there a listener that specific interface? OR
	// Is there a listener on the "any" address (0.0.0.0 or ::)?
	isIPv4 := localIP.To4() != nil
	for _, ip := range interfaces.ips {
		switch {
		case ip.Equal(localIP):
			return Incoming
		case ip.Equal(net.IPv4zero) && isIPv4:
			return Incoming
		case ip.Equal(net.IPv6zero) && !isIPv4:
			return Incoming
		}
	}

	return Outgoing
}
