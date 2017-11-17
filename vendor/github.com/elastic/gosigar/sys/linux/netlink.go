// +build linux

package linux

import (
	"encoding/binary"
	"errors"
)

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
