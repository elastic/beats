package dhcpv4

import (
	"fmt"
	"net"
)

// This option implements the domain name server option
// https://tools.ietf.org/html/rfc2132

// OptDomainNameServer represents an option encapsulating the domain name
// servers.
type OptDomainNameServer struct {
	NameServers []net.IP
}

// ParseOptDomainNameServer returns a new OptDomainNameServer from a byte
// stream, or error if any.
func ParseOptDomainNameServer(data []byte) (*OptDomainNameServer, error) {
	if len(data) < 2 {
		return nil, ErrShortByteStream
	}
	code := OptionCode(data[0])
	if code != OptionDomainNameServer {
		return nil, fmt.Errorf("expected code %v, got %v", OptionDomainNameServer, code)
	}
	length := int(data[1])
	if length == 0 || length%4 != 0 {
		return nil, fmt.Errorf("Invalid length: expected multiple of 4 larger than 4, got %v", length)
	}
	if len(data) < 2+length {
		return nil, ErrShortByteStream
	}
	nameservers := make([]net.IP, 0, length%4)
	for idx := 0; idx < length; idx += 4 {
		b := data[2+idx : 2+idx+4]
		nameservers = append(nameservers, net.IPv4(b[0], b[1], b[2], b[3]))
	}
	return &OptDomainNameServer{NameServers: nameservers}, nil
}

// Code returns the option code.
func (o *OptDomainNameServer) Code() OptionCode {
	return OptionDomainNameServer
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *OptDomainNameServer) ToBytes() []byte {
	ret := []byte{byte(o.Code()), byte(o.Length())}
	for _, ns := range o.NameServers {
		ret = append(ret, ns...)
	}
	return ret
}

// String returns a human-readable string.
func (o *OptDomainNameServer) String() string {
	var servers string
	for idx, ns := range o.NameServers {
		servers += ns.String()
		if idx < len(o.NameServers)-1 {
			servers += ", "
		}
	}
	return fmt.Sprintf("Domain Name Servers -> %v", servers)
}

// Length returns the length of the data portion (excluding option code an byte
// length).
func (o *OptDomainNameServer) Length() int {
	return len(o.NameServers) * 4
}
