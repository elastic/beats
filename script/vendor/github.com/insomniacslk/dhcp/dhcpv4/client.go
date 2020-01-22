// +build !windows

package dhcpv4

import (
	"encoding/binary"
	"errors"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/net/ipv4"
)

// MaxUDPReceivedPacketSize is the (arbitrary) maximum UDP packet size supported
// by this library. Theoretically could be up to 65kb.
const (
	MaxUDPReceivedPacketSize = 8192
)

var (
	// DefaultReadTimeout is the time to wait after listening in which the
	// exchange is considered failed.
	DefaultReadTimeout = 3 * time.Second

	// DefaultWriteTimeout is the time to wait after sending in which the
	// exchange is considered failed.
	DefaultWriteTimeout = 3 * time.Second
)

// Client is the object that actually performs the DHCP exchange. It currently
// only has read and write timeout values.
type Client struct {
	ReadTimeout, WriteTimeout time.Duration
}

// NewClient generates a new client to perform a DHCP exchange with, setting the
// read and write timeout fields to defaults.
func NewClient() *Client {
	return &Client{
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
	}
}

// MakeRawBroadcastPacket converts payload (a serialized DHCPv4 packet) into a
// raw packet suitable for UDP broadcast.
func MakeRawBroadcastPacket(payload []byte) ([]byte, error) {
	udp := make([]byte, 8)
	binary.BigEndian.PutUint16(udp[:2], ClientPort)
	binary.BigEndian.PutUint16(udp[2:4], ServerPort)
	binary.BigEndian.PutUint16(udp[4:6], uint16(8+len(payload)))
	binary.BigEndian.PutUint16(udp[6:8], 0) // try to offload the checksum

	h := ipv4.Header{
		Version:  4,
		Len:      20,
		TotalLen: 20 + len(udp) + len(payload),
		TTL:      64,
		Protocol: 17, // UDP
		Dst:      net.IPv4bcast,
		Src:      net.IPv4zero,
	}
	ret, err := h.Marshal()
	if err != nil {
		return nil, err
	}
	ret = append(ret, udp...)
	ret = append(ret, payload...)
	return ret, nil
}

// MakeBroadcastSocket creates a socket that can be passed to syscall.Sendto
// that will send packets out to the broadcast address.
func MakeBroadcastSocket(ifname string) (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return fd, err
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return fd, err
	}
	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1)
	if err != nil {
		return fd, err
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	if err != nil {
		return fd, err
	}
	err = BindToInterface(fd, ifname)
	if err != nil {
		return fd, err
	}
	return fd, nil
}

// MakeListeningSocket creates a listening socket on 0.0.0.0 for the DHCP client
// port and returns it.
func MakeListeningSocket(ifname string) (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return fd, err
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return fd, err
	}
	var addr [4]byte
	copy(addr[:], net.IPv4zero.To4())
	if err = syscall.Bind(fd, &syscall.SockaddrInet4{Port: ClientPort, Addr: addr}); err != nil {
		return fd, err
	}
	err = BindToInterface(fd, ifname)
	if err != nil {
		return fd, err
	}
	return fd, nil
}

// Exchange runs a full DORA transaction: Discover, Offer, Request, Acknowledge,
// over UDP. Does not retry in case of failures. Returns a list of DHCPv4
// structures representing the exchange. It can contain up to four elements,
// ordered as Discovery, Offer, Request and Acknowledge. In case of errors, an
// error is returned, and the list of DHCPv4 objects will be shorted than 4,
// containing all the sent and received DHCPv4 messages.
func (c *Client) Exchange(ifname string, discover *DHCPv4) ([]DHCPv4, error) {
	conversation := make([]DHCPv4, 1)
	var err error

	// Get our file descriptor for the broadcast socket.
	sfd, err := MakeBroadcastSocket(ifname)
	if err != nil {
		return conversation, err
	}
	rfd, err := MakeListeningSocket(ifname)
	if err != nil {
		return conversation, err
	}

	// Discover
	if discover == nil {
		discover, err = NewDiscoveryForInterface(ifname)
		if err != nil {
			return conversation, err
		}
	}
	conversation[0] = *discover

	// Offer
	offer, err := BroadcastSendReceive(sfd, rfd, discover, c.ReadTimeout, c.WriteTimeout)
	if err != nil {
		return conversation, err
	}
	conversation = append(conversation, *offer)

	// Request
	request, err := RequestFromOffer(*offer)
	if err != nil {
		return conversation, err
	}
	conversation = append(conversation, *request)

	// Ack
	ack, err := BroadcastSendReceive(sfd, rfd, discover, c.ReadTimeout, c.WriteTimeout)
	if err != nil {
		return conversation, err
	}
	conversation = append(conversation, *ack)
	return conversation, nil
}

// BroadcastSendReceive broadcasts packet (with some write timeout) and waits for a
// response up to some read timeout value.
func BroadcastSendReceive(sendFd, recvFd int, packet *DHCPv4, readTimeout, writeTimeout time.Duration) (*DHCPv4, error) {
	packetBytes, err := MakeRawBroadcastPacket(packet.ToBytes())
	if err != nil {
		return nil, err
	}

	// Create a goroutine to perform the blocking send, and time it out after
	// a certain amount of time.
	var destination [4]byte
	copy(destination[:], net.IPv4bcast.To4())
	remoteAddr := syscall.SockaddrInet4{Port: ClientPort, Addr: destination}
	recvErrors := make(chan error, 1)
	var response *DHCPv4
	go func(errs chan<- error) {
		conn, err := net.FileConn(os.NewFile(uintptr(recvFd), ""))
		if err != nil {
			errs <- err
			return
		}
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(readTimeout))

		buf := make([]byte, MaxUDPReceivedPacketSize)
		n, _, _, _, err := conn.(*net.UDPConn).ReadMsgUDP(buf, []byte{})
		if err != nil {
			errs <- err
			return
		}

		response, err = FromBytes(buf[:n])
		if err != nil {
			errs <- err
			return
		}
		recvErrors <- nil
	}(recvErrors)
	if err = syscall.Sendto(sendFd, packetBytes, 0, &remoteAddr); err != nil {
		return nil, err
	}

	select {
	case err = <-recvErrors:
		if err != nil {
			return nil, err
		}
	case <-time.After(readTimeout):
		return nil, errors.New("timed out while listening for replies")
	}

	return response, nil
}
