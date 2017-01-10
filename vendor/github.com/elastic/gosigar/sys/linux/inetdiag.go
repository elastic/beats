// +build linux

package linux

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

// Enums / Constants

const (
	// AllTCPStates is a flag to request all sockets in any TCP state.
	AllTCPStates = ^uint32(0)

	// TCPDIAG_GETSOCK is the netlink message type for requesting TCP diag data.
	// https://github.com/torvalds/linux/blob/v4.0/include/uapi/linux/inet_diag.h#L7
	TCPDIAG_GETSOCK = 18

	// SOCK_DIAG_BY_FAMILY is the netlink message type for requestion socket
	// diag data by family. This is newer and can be used with inet_diag_req_v2.
	// https://github.com/torvalds/linux/blob/v4.0/include/uapi/linux/sock_diag.h#L6
	SOCK_DIAG_BY_FAMILY = 20
)

// AddressFamily is the address family of the socket.
type AddressFamily uint8

// https://github.com/torvalds/linux/blob/5924bbecd0267d87c24110cbe2041b5075173a25/include/linux/socket.h#L159
const (
	AF_INET  AddressFamily = 2
	AF_INET6               = 10
)

var addressFamilyNames = map[AddressFamily]string{
	AF_INET:  "ipv4",
	AF_INET6: "ipv6",
}

func (af AddressFamily) String() string {
	if fam, found := addressFamilyNames[af]; found {
		return fam
	}
	return fmt.Sprintf("UNKNOWN (%d)", af)
}

// TCPState represents the state of a TCP connection.
type TCPState uint8

// https://github.com/torvalds/linux/blob/5924bbecd0267d87c24110cbe2041b5075173a25/include/net/tcp_states.h#L16
const (
	TCP_ESTABLISHED TCPState = iota + 1
	TCP_SYN_SENT
	TCP_SYN_RECV
	TCP_FIN_WAIT1
	TCP_FIN_WAIT2
	TCP_TIME_WAIT
	TCP_CLOSE
	TCP_CLOSE_WAIT
	TCP_LAST_ACK
	TCP_LISTEN
	TCP_CLOSING /* Now a valid state */
)

var tcpStateNames = map[TCPState]string{
	TCP_ESTABLISHED: "ESTAB",
	TCP_SYN_SENT:    "SYN-SENT",
	TCP_SYN_RECV:    "SYN-RECV",
	TCP_FIN_WAIT1:   "FIN-WAIT-1",
	TCP_FIN_WAIT2:   "FIN-WAIT-2",
	TCP_TIME_WAIT:   "TIME-WAIT",
	TCP_CLOSE:       "UNCONN",
	TCP_CLOSE_WAIT:  "CLOSE-WAIT",
	TCP_LAST_ACK:    "LAST-ACK",
	TCP_LISTEN:      "LISTEN",
	TCP_CLOSING:     "CLOSING",
}

func (s TCPState) String() string {
	if state, found := tcpStateNames[s]; found {
		return state
	}
	return "UNKNOWN"
}

// Extensions that can be used in the InetDiagReqV2 request to ask for
// additional data.
// https://github.com/torvalds/linux/blob/v4.0/include/uapi/linux/inet_diag.h#L103
const (
	INET_DIAG_NONE    = 0
	INET_DIAG_MEMINFO = 1 << iota
	INET_DIAG_INFO
	INET_DIAG_VEGASINFO
	INET_DIAG_CONG
	INET_DIAG_TOS
	INET_DIAG_TCLASS
	INET_DIAG_SKMEMINFO
	INET_DIAG_SHUTDOWN
	INET_DIAG_DCTCPINFO
	INET_DIAG_PROTOCOL /* response attribute only */
	INET_DIAG_SKV6ONLY
	INET_DIAG_LOCALS
	INET_DIAG_PEERS
	INET_DIAG_PAD
	INET_DIAG_MARK
)

// NetlinkInetDiag sends the given netlink request parses the responses with the
// assumption that they are inet_diag_msgs. This will allocate a temporary
// buffer for reading from the socket whose size will be the length of a page
// (usually 32k). Use NetlinkInetDiagWithBuf if you want to provide your own
// buffer.
func NetlinkInetDiag(request syscall.NetlinkMessage) ([]*InetDiagMsg, error) {
	return NetlinkInetDiagWithBuf(request, nil, nil)
}

// NetlinkInetDiagWithBuf sends the given netlink request parses the responses
// with the assumption that they are inet_diag_msgs. readBuf will be used to
// hold the raw data read from the socket. If the length is not large enough to
// hold the socket contents the data will be truncated. If readBuf is nil then a
// temporary buffer will be allocated for each invocation. The resp writer, if
// non-nil, will receive a copy of all bytes read (this is useful for
// debugging).
func NetlinkInetDiagWithBuf(request syscall.NetlinkMessage, readBuf []byte, resp io.Writer) ([]*InetDiagMsg, error) {
	s, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_INET_DIAG)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(s)

	lsa := &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}
	if err := syscall.Sendto(s, serialize(request), 0, lsa); err != nil {
		return nil, err
	}

	if len(readBuf) == 0 {
		// Default size used in libnl.
		readBuf = make([]byte, os.Getpagesize())
	}

	var inetDiagMsgs []*InetDiagMsg
done:
	for {
		buf := readBuf
		nr, _, err := syscall.Recvfrom(s, buf, 0)
		if err != nil {
			return nil, err
		}
		if nr < syscall.NLMSG_HDRLEN {
			return nil, syscall.EINVAL
		}

		buf = buf[:nr]

		// Dump raw data for inspection purposes.
		if resp != nil {
			if _, err := resp.Write(buf); err != nil {
				return nil, err
			}
		}

		msgs, err := syscall.ParseNetlinkMessage(buf)
		if err != nil {
			return nil, err
		}

		for _, m := range msgs {
			if m.Header.Type == syscall.NLMSG_DONE {
				break done
			}
			if m.Header.Type == syscall.NLMSG_ERROR {
				return nil, ParseNetlinkError(m.Data)
			}

			inetDiagMsg, err := ParseInetDiagMsg(m.Data)
			if err != nil {
				return nil, err
			}
			inetDiagMsgs = append(inetDiagMsgs, inetDiagMsg)
		}
	}
	return inetDiagMsgs, nil
}

func serialize(msg syscall.NetlinkMessage) []byte {
	msg.Header.Len = uint32(syscall.SizeofNlMsghdr + len(msg.Data))
	b := make([]byte, msg.Header.Len)
	binary.LittleEndian.PutUint32(b[0:4], msg.Header.Len)
	binary.LittleEndian.PutUint16(b[4:6], msg.Header.Type)
	binary.LittleEndian.PutUint16(b[6:8], msg.Header.Flags)
	binary.LittleEndian.PutUint32(b[8:12], msg.Header.Seq)
	binary.LittleEndian.PutUint32(b[12:16], msg.Header.Pid)
	copy(b[16:], msg.Data)
	return b
}

// Request messages.

var sizeofInetDiagReq = int(unsafe.Sizeof(InetDiagReq{}))

// InetDiagReq (inet_diag_req) is used to request diagnostic data from older
// kernels.
// https://github.com/torvalds/linux/blob/v4.0/include/uapi/linux/inet_diag.h#L25
type InetDiagReq struct {
	Family uint8
	SrcLen uint8
	DstLen uint8
	Ext    uint8
	ID     InetDiagSockID
	States uint32 // States to dump.
	DBs    uint32 // Tables to dump.
}

func (r InetDiagReq) toWireFormat() []byte {
	buf := bytes.NewBuffer(make([]byte, sizeofInetDiagReq))
	buf.Reset()
	if err := binary.Write(buf, binary.LittleEndian, r); err != nil {
		// This never returns an error.
		panic(err)
	}
	return buf.Bytes()
}

// NewInetDiagReq returns a new NetlinkMessage whose payload is an InetDiagReq.
// Callers should set their own sequence number in the returned message header.
func NewInetDiagReq() syscall.NetlinkMessage {
	hdr := syscall.NlMsghdr{
		Type:  uint16(TCPDIAG_GETSOCK),
		Flags: uint16(syscall.NLM_F_DUMP | syscall.NLM_F_REQUEST),
		Pid:   uint32(0),
	}
	req := InetDiagReq{
		Family: uint8(AF_INET), // This returns both ipv4 and ipv6.
		States: AllTCPStates,
	}

	return syscall.NetlinkMessage{Header: hdr, Data: req.toWireFormat()}
}

// V2 Request

var sizeofInetDiagReqV2 = int(unsafe.Sizeof(InetDiagReqV2{}))

// InetDiagReqV2 (inet_diag_req_v2) is used to request diagnostic data.
// https://github.com/torvalds/linux/blob/v4.0/include/uapi/linux/inet_diag.h#L37
type InetDiagReqV2 struct {
	Family   uint8
	Protocol uint8
	Ext      uint8
	Pad      uint8
	States   uint32
	ID       InetDiagSockID
}

func (r InetDiagReqV2) toWireFormat() []byte {
	buf := bytes.NewBuffer(make([]byte, sizeofInetDiagReqV2))
	buf.Reset()
	if err := binary.Write(buf, binary.LittleEndian, r); err != nil {
		// This never returns an error.
		panic(err)
	}
	return buf.Bytes()
}

// NewInetDiagReqV2 returns a new NetlinkMessage whose payload is an
// InetDiagReqV2. Callers should set their own sequence number in the returned
// message header.
func NewInetDiagReqV2(af AddressFamily) syscall.NetlinkMessage {
	hdr := syscall.NlMsghdr{
		Type:  uint16(SOCK_DIAG_BY_FAMILY),
		Flags: uint16(syscall.NLM_F_DUMP | syscall.NLM_F_REQUEST),
		Pid:   uint32(0),
	}
	req := InetDiagReqV2{
		Family:   uint8(af),
		Protocol: syscall.IPPROTO_TCP,
		States:   AllTCPStates,
	}

	return syscall.NetlinkMessage{Header: hdr, Data: req.toWireFormat()}
}

// Response messages.

// InetDiagMsg (inet_diag_msg) is the base info structure. It contains socket
// identity (addrs/ports/cookie) and the information shown by netstat.
// https://github.com/torvalds/linux/blob/v4.0/include/uapi/linux/inet_diag.h#L86
type InetDiagMsg struct {
	Family  uint8 // Address family.
	State   uint8 // TCP State
	Timer   uint8
	Retrans uint8

	ID InetDiagSockID

	Expires uint32
	RQueue  uint32 // Recv-Q
	WQueue  uint32 // Send-Q
	UID     uint32 // UID
	Inode   uint32 // Inode of socket.
}

// ParseInetDiagMsg parse an InetDiagMsg from a byte slice. It assumes the
// InetDiagMsg starts at the beginning of b. Invoke this method to parse the
// payload of a netlink response.
func ParseInetDiagMsg(b []byte) (*InetDiagMsg, error) {
	r := bytes.NewReader(b)
	inetDiagMsg := &InetDiagMsg{}
	err := binary.Read(r, binary.LittleEndian, inetDiagMsg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal inet_diag_msg")
	}
	return inetDiagMsg, nil
}

// SrcPort returns the source (local) port.
func (m InetDiagMsg) SrcPort() int { return int(binary.BigEndian.Uint16(m.ID.SPort[:])) }

// DstPort returns the destination (remote) port.
func (m InetDiagMsg) DstPort() int { return int(binary.BigEndian.Uint16(m.ID.DPort[:])) }

// SrcIP returns the source (local) IP.
func (m InetDiagMsg) SrcIP() net.IP { return ip(m.ID.Src, AddressFamily(m.Family)) }

// DstIP returns the destination (remote) IP.
func (m InetDiagMsg) DstIP() net.IP { return ip(m.ID.Dst, AddressFamily(m.Family)) }

func (m InetDiagMsg) srcIPBytes() []byte { return ipBytes(m.ID.Src, AddressFamily(m.Family)) }
func (m InetDiagMsg) dstIPBytes() []byte { return ipBytes(m.ID.Dst, AddressFamily(m.Family)) }

func ip(data [16]byte, af AddressFamily) net.IP {
	if af == AF_INET {
		return net.IPv4(data[0], data[1], data[2], data[3])
	}
	return net.IP(data[:])
}

func ipBytes(data [16]byte, af AddressFamily) []byte {
	if af == AF_INET {
		return data[:4]
	}

	return data[:]
}

// FastHash returns a hash calculated using FNV-1 of the source and destination
// addresses.
func (m *InetDiagMsg) FastHash() uint64 {
	// Hash using FNV-1 algorithm.
	h := fnv.New64()
	h.Write(m.srcIPBytes()) // Must trim non-zero garbage from ipv4 buffers.
	h.Write(m.dstIPBytes())
	h.Write(m.ID.SPort[:])
	h.Write(m.ID.DPort[:])
	return h.Sum64()
}

// InetDiagSockID (inet_diag_sockid) contains the socket identity.
// https://github.com/torvalds/linux/blob/v4.0/include/uapi/linux/inet_diag.h#L13
type InetDiagSockID struct {
	SPort  [2]byte  // Source port (big-endian).
	DPort  [2]byte  // Destination port (big-endian).
	Src    [16]byte // Source IP
	Dst    [16]byte // Destination IP
	If     uint32
	Cookie [2]uint32
}
