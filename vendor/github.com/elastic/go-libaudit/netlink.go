// Copyright 2017 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build linux

package libaudit

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"syscall"

	"github.com/pkg/errors"
)

// Generic Netlink Client

// NetlinkSender sends a netlink message and returns the sequence number used
// in the message and an error if it occurred.
type NetlinkSender interface {
	Send(msg syscall.NetlinkMessage) (uint32, error)
}

// NetlinkReceiver receives data from the netlink socket and uses the provided
// parser to convert the raw bytes to NetlinkMessages. For most uses cases
// syscall.ParseNetlinkMessage should be used. If nonBlocking is true then
// instead of blocking when no data is available, EWOULDBLOCK is returned.
type NetlinkReceiver interface {
	Receive(nonBlocking bool, p NetlinkParser) ([]syscall.NetlinkMessage, error)
}

// NetlinkSendReceiver combines the Send and Receive into one interface.
type NetlinkSendReceiver interface {
	io.Closer
	NetlinkSender
	NetlinkReceiver
}

// NetlinkParser parses the raw bytes read from the netlink socket into
// netlink messages.
type NetlinkParser func([]byte) ([]syscall.NetlinkMessage, error)

// NetlinkClient is a generic client for sending and receiving netlink messages.
type NetlinkClient struct {
	fd         int              // File descriptor used for communication.
	src        syscall.Sockaddr // Local socket address.
	dest       syscall.Sockaddr // Remote socket address (client assumes the dest is the kernel).
	pid        uint32           // Port ID of the local socket.
	seq        uint32           // Sequence number used in outgoing messages.
	readBuf    []byte
	respWriter io.Writer
}

// NewNetlinkClient creates a new NetlinkClient. It creates a socket and binds
// it. readBuf is an optional byte buffer used for reading data from the socket.
// The size of the buffer limits the maximum message size the can be read. If no
// buffer is provided one will be allocated using the OS page size. resp is
// optional and can be used to receive a copy of all bytes read from the socket
// (this is useful for debugging).
//
// The returned NetlinkClient must be closed with Close() when finished.
func NewNetlinkClient(proto int, readBuf []byte, resp io.Writer) (*NetlinkClient, error) {
	s, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, proto)
	if err != nil {
		return nil, err
	}

	src := &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}
	if err = syscall.Bind(s, src); err != nil {
		syscall.Close(s)
		return nil, err
	}

	pid, err := getPortID(s)
	if err != nil {
		syscall.Close(s)
		return nil, err
	}

	if len(readBuf) == 0 {
		// Default size used in libnl.
		readBuf = make([]byte, os.Getpagesize())
	}

	return &NetlinkClient{
		fd:         s,
		src:        src,
		dest:       &syscall.SockaddrNetlink{},
		pid:        pid,
		readBuf:    readBuf,
		respWriter: resp,
	}, nil
}

// getPortID gets the kernel assigned port ID (PID) of the local netlink socket.
// The kernel assigns the processes PID to the first socket then assigns arbitrary values
// to any follow-on sockets. See man netlink for details.
func getPortID(fd int) (uint32, error) {
	address, err := syscall.Getsockname(fd)
	if err != nil {
		return 0, err
	}

	addr, ok := address.(*syscall.SockaddrNetlink)
	if !ok {
		return 0, errors.New("unexpected socket address type")
	}

	return addr.Pid, nil
}

// Send sends a netlink message and returns the sequence number used
// in the message and an error if it occurred. If the PID is not set then
// the value will be populated automatically (recommended).
func (c *NetlinkClient) Send(msg syscall.NetlinkMessage) (uint32, error) {
	if msg.Header.Pid == 0 {
		msg.Header.Pid = c.pid
	}

	msg.Header.Seq = atomic.AddUint32(&c.seq, 1)
	to := &syscall.SockaddrNetlink{}
	return msg.Header.Seq, syscall.Sendto(c.fd, serialize(msg), 0, to)
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

// Receive receives data from the netlink socket and uses the provided
// parser to convert the raw bytes to NetlinkMessages. See NetlinkReceiver docs.
func (c *NetlinkClient) Receive(nonBlocking bool, p NetlinkParser) ([]syscall.NetlinkMessage, error) {
	var flags int
	if nonBlocking {
		flags |= syscall.MSG_DONTWAIT
	}

	// XXX (akroh): A possible enhancement is to use the MSG_PEEK flag to
	// check the message size and increase the buffer size to handle it all.
	nr, from, err := syscall.Recvfrom(c.fd, c.readBuf, flags)
	if err != nil {
		// EAGAIN or EWOULDBLOCK will be returned for non-blocking reads where
		// the read would normally have blocked.
		return nil, err
	}
	if nr < syscall.NLMSG_HDRLEN {
		return nil, errors.Errorf("not enough bytes (%v) received to form a netlink header", nr)
	}
	fromNetlink, ok := from.(*syscall.SockaddrNetlink)
	if !ok || fromNetlink.Pid != 0 {
		// Spoofed packet received on audit netlink socket.
		return nil, errors.New("message received was not from the kernel")
	}

	buf := c.readBuf[:nr]

	// Dump raw data for inspection purposes.
	if c.respWriter != nil {
		if _, err = c.respWriter.Write(buf); err != nil {
			return nil, err
		}
	}

	msgs, err := p(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse netlink messages (bytes_received=%v): %v", nr, err)
	}

	return msgs, nil
}

// Close closes the netlink client's raw socket.
func (c *NetlinkClient) Close() error {
	return syscall.Close(c.fd)
}

// Netlink Error Code Handling

// ParseNetlinkError parses the errno from the data section of a
// syscall.NetlinkMessage. If netlinkData is less than 4 bytes an error
// describing the problem will be returned.
func ParseNetlinkError(netlinkData []byte) error {
	if len(netlinkData) >= 4 {
		errno := -binary.LittleEndian.Uint32(netlinkData[:4])
		return NetlinkErrno(errno)
	}
	return errors.New("received netlink error (data too short to read errno)")
}

// NetlinkErrno represent the error code contained in a netlink message of
// type NLMSG_ERROR.
type NetlinkErrno uint32

// Netlink error codes.
const (
	NLE_SUCCESS NetlinkErrno = iota
	NLE_FAILURE
	NLE_INTR
	NLE_BAD_SOCK
	NLE_AGAIN
	NLE_NOMEM
	NLE_EXIST
	NLE_INVAL
	NLE_RANGE
	NLE_MSGSIZE
	NLE_OPNOTSUPP
	NLE_AF_NOSUPPORT
	NLE_OBJ_NOTFOUND
	NLE_NOATTR
	NLE_MISSING_ATTR
	NLE_AF_MISMATCH
	NLE_SEQ_MISMATCH
	NLE_MSG_OVERFLOW
	NLE_MSG_TRUNC
	NLE_NOADDR
	NLE_SRCRT_NOSUPPORT
	NLE_MSG_TOOSHORT
	NLE_MSGTYPE_NOSUPPORT
	NLE_OBJ_MISMATCH
	NLE_NOCACHE
	NLE_BUSY
	NLE_PROTO_MISMATCH
	NLE_NOACCESS
	NLE_PERM
	NLE_PKTLOC_FILE
	NLE_PARSE_ERR
	NLE_NODEV
	NLE_IMMUTABLE
	NLE_DUMP_INTR
	NLE_ATTRSIZE
)

// https://github.com/thom311/libnl/blob/libnl3_2_28/lib/error.c
var netlinkErrorMsgs = map[NetlinkErrno]string{
	NLE_SUCCESS:           "Success",
	NLE_FAILURE:           "Unspecific failure",
	NLE_INTR:              "Interrupted system call",
	NLE_BAD_SOCK:          "Bad socket",
	NLE_AGAIN:             "Try again",
	NLE_NOMEM:             "Out of memory",
	NLE_EXIST:             "Object exists",
	NLE_INVAL:             "Invalid input data or parameter",
	NLE_RANGE:             "Input data out of range",
	NLE_MSGSIZE:           "Message size not sufficient",
	NLE_OPNOTSUPP:         "Operation not supported",
	NLE_AF_NOSUPPORT:      "Address family not supported",
	NLE_OBJ_NOTFOUND:      "Object not found",
	NLE_NOATTR:            "Attribute not available",
	NLE_MISSING_ATTR:      "Missing attribute",
	NLE_AF_MISMATCH:       "Address family mismatch",
	NLE_SEQ_MISMATCH:      "Message sequence number mismatch",
	NLE_MSG_OVERFLOW:      "Kernel reported message overflow",
	NLE_MSG_TRUNC:         "Kernel reported truncated message",
	NLE_NOADDR:            "Invalid address for specified address family",
	NLE_SRCRT_NOSUPPORT:   "Source based routing not supported",
	NLE_MSG_TOOSHORT:      "Netlink message is too short",
	NLE_MSGTYPE_NOSUPPORT: "Netlink message type is not supported",
	NLE_OBJ_MISMATCH:      "Object type does not match cache",
	NLE_NOCACHE:           "Unknown or invalid cache type",
	NLE_BUSY:              "Object busy",
	NLE_PROTO_MISMATCH:    "Protocol mismatch",
	NLE_NOACCESS:          "No Access",
	NLE_PERM:              "Operation not permitted",
	NLE_PKTLOC_FILE:       "Unable to open packet location file",
	NLE_PARSE_ERR:         "Unable to parse object",
	NLE_NODEV:             "No such device",
	NLE_IMMUTABLE:         "Immutable attribute",
	NLE_DUMP_INTR:         "Dump inconsistency detected, interrupted",
	NLE_ATTRSIZE:          "Attribute max length exceeded",
}

func (e NetlinkErrno) Error() string {
	if msg, found := netlinkErrorMsgs[e]; found {
		return msg
	}

	return netlinkErrorMsgs[NLE_FAILURE]
}
