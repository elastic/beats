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
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/elastic/go-libaudit/auparse"
)

const (
	// AuditMessageMaxLength is the maximum length of an audit message (data
	// portion of a NetlinkMessage).
	// https://github.com/linux-audit/audit-userspace/blob/990aa27ccd02f9743c4f4049887ab89678ab362a/lib/libaudit.h#L435
	AuditMessageMaxLength = 8970
)

// Audit command and control message types.
const (
	AuditGet uint16 = iota + 1000
	AuditSet
)

// WaitMode is a flag to control the behavior of methods that abstract
// asynchronous communication for the caller.
type WaitMode uint8

const (
	// WaitForReply mode causes a call to wait for a reply message.
	WaitForReply WaitMode = iota + 1
	// NoWait mode causes a call to return without waiting for a reply message.
	NoWait
)

// AuditClient is a client for communicating with the Linux kernels audit
// interface over netlink.
type AuditClient struct {
	Netlink NetlinkSendReceiver
}

// NewAuditClient creates a new AuditClient. The resp parameter is optional. If
// provided resp will receive a copy of all data read from the netlink socket.
// This is useful for debugging purposes.
func NewAuditClient(resp io.Writer) (*AuditClient, error) {
	buf := make([]byte, syscall.NLMSG_HDRLEN+AuditMessageMaxLength)

	netlink, err := NewNetlinkClient(syscall.NETLINK_AUDIT, buf, resp)
	if err != nil {
		return nil, err
	}

	return &AuditClient{Netlink: netlink}, nil
}

// GetStatus returns the current status of the kernel's audit subsystem.
func (c *AuditClient) GetStatus() (*AuditStatus, error) {
	msg := syscall.NetlinkMessage{
		Header: syscall.NlMsghdr{
			Type:  AuditGet,
			Flags: syscall.NLM_F_REQUEST | syscall.NLM_F_ACK,
		},
		Data: nil,
	}

	// Send AUDIT_GET message to the kernel.
	seq, err := c.Netlink.Send(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending request")
	}

	// Get the ack message which is a NLMSG_ERROR type whose error code is SUCCESS.
	ack, err := c.getReply(seq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit status ack")
	}

	if ack.Header.Type != syscall.NLMSG_ERROR {
		return nil, errors.Errorf("unexpected ACK to GET, type=%d", ack.Header.Type)
	}

	if err = ParseNetlinkError(ack.Data); err != NLE_SUCCESS {
		if len(ack.Data) >= 4+12 {
			status := &AuditStatus{}
			if err = status.fromWireFormat(ack.Data[4:]); err == nil {
				return nil, syscall.Errno(status.Failure)
			}
		}
		return nil, err
	}

	// Get the audit_status reply message. It has the same sequence number as
	// our original request.
	reply, err := c.getReply(seq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get audit status reply")
	}

	if reply.Header.Type != AuditGet {
		return nil, errors.Errorf("unexpected reply to GET, type%d", reply.Header.Type)
	}

	replyStatus := &AuditStatus{}
	if err := replyStatus.fromWireFormat(reply.Data); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal reply")
	}

	return replyStatus, nil
}

// SetPID sends a netlink message to the kernel telling it the PID of the
// client that should receive audit messages.
// https://github.com/linux-audit/audit-userspace/blob/990aa27ccd02f9743c4f4049887ab89678ab362a/lib/libaudit.c#L432-L464
func (c *AuditClient) SetPID(wm WaitMode) error {
	status := AuditStatus{
		Mask: AuditStatusPID,
		PID:  uint32(os.Getpid()),
	}
	return c.set(status, wm)
}

// SetRateLimit will set the maximum number of messages that the kernel will
// send per second. This can be used to throttle the rate if systems become
// unresponsive. Of course the trade off is that events will be dropped.
// The default value is 0, meaning no limit.
func (c *AuditClient) SetRateLimit(perSecondLimit uint32, wm WaitMode) error {
	status := AuditStatus{
		Mask:      AuditStatusRateLimit,
		RateLimit: perSecondLimit,
	}
	return c.set(status, wm)
}

// SetBacklogLimit sets the queue length for audit events awaiting transfer to
// the audit daemon. The default value is 64 which can potentially be overrun by
// bursts of activity. When the backlog limit is reached, the kernel consults
// the failure_flag to see what action to take.
func (c *AuditClient) SetBacklogLimit(limit uint32, wm WaitMode) error {
	status := AuditStatus{
		Mask:         AuditStatusBacklogLimit,
		BacklogLimit: limit,
	}
	return c.set(status, wm)
}

// SetEnabled is used to control whether or not the audit system is
// active. When the audit system is enabled (enabled set to 1), every syscall
// will pass through the audit system to collect information and potentially
// trigger an event.
func (c *AuditClient) SetEnabled(enabled bool, wm WaitMode) error {
	var e uint32
	if enabled {
		e = 1
	}

	status := AuditStatus{
		Mask:    AuditStatusEnabled,
		Enabled: e,
	}
	return c.set(status, wm)
}

// RawAuditMessage is a raw audit message received from the kernel.
type RawAuditMessage struct {
	Type auparse.AuditMessageType
	Data []byte // RawData is backed by the read buffer so make a copy.
}

// Receive reads an audit message from the netlink socket. If you are going to
// use the returned message then you should make a copy of the raw data before
// calling receive again because the raw data is backed by the read buffer.
func (c *AuditClient) Receive(nonBlocking bool) (*RawAuditMessage, error) {
	msgs, err := c.Netlink.Receive(nonBlocking, parseNetlinkAuditMessage)
	if err != nil {
		return nil, err
	}

	// ParseNetlinkAuditMessage always return a slice with 1 item.
	return &RawAuditMessage{
		Type: auparse.AuditMessageType(msgs[0].Header.Type),
		Data: msgs[0].Data,
	}, nil
}

// Close closes the AuditClient and frees any associated resources.
func (c *AuditClient) Close() error {
	return c.Netlink.Close()
}

// getReply reads from the netlink socket and find the message with the given
// sequence number. The caller should inspect the returned message's type,
// flags, and error code.
func (c *AuditClient) getReply(seq uint32) (*syscall.NetlinkMessage, error) {
	var msgs []syscall.NetlinkMessage
	var err error

	// Retry the non-blocking read multiple times until a response is received.
	for i := 0; i < 10; i++ {
		msgs, err = c.Netlink.Receive(true, parseNetlinkAuditMessage)
		if err != nil {
			switch err {
			case syscall.EINTR:
				continue
			case syscall.EAGAIN:
				time.Sleep(50 * time.Millisecond)
				continue
			default:
				return nil, errors.Wrap(err, "error receiving audit reply")
			}
		}

		break
	}

	if len(msgs) == 0 {
		return nil, errors.New("no reply received")
	}
	msg := msgs[0]

	if msg.Header.Seq != seq {
		return nil, errors.Errorf("unexpected sequence number for reply (expected %v but got %v)",
			seq, msg.Header.Seq)
	}
	return &msg, nil
}

func (c *AuditClient) set(status AuditStatus, mode WaitMode) error {
	msg := syscall.NetlinkMessage{
		Header: syscall.NlMsghdr{
			Type:  AuditSet,
			Flags: syscall.NLM_F_REQUEST | syscall.NLM_F_ACK,
		},
		Data: status.toWireFormat(),
	}

	seq, err := c.Netlink.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed sending request")
	}

	if mode == NoWait {
		return nil
	}

	ack, err := c.getReply(seq)
	if err != nil {
		return err
	}

	if ack.Header.Type != syscall.NLMSG_ERROR {
		return errors.Errorf("unexpected ACK to SET, type=%d", ack.Header.Type)
	}

	if err := ParseNetlinkError(ack.Data); err != NLE_SUCCESS {
		if len(ack.Data) >= 4+12 {
			status := &AuditStatus{}
			if err = status.fromWireFormat(ack.Data[4:]); err == nil {
				return syscall.Errno(status.Failure)
			}
		}
		return err
	}

	return nil
}

// parseNetlinkAuditMessage parses an audit message received from the kernel.
// Audit messages differ significantly from typical netlink messages in that
// a single message is sent and the length in the header should be ignored.
// This is why syscall.ParseNetlinkMessage is not used.
func parseNetlinkAuditMessage(buf []byte) ([]syscall.NetlinkMessage, error) {
	if len(buf) < syscall.NLMSG_HDRLEN {
		return nil, syscall.EINVAL
	}

	r := bytes.NewReader(buf)
	m := syscall.NetlinkMessage{}
	if err := binary.Read(r, binary.LittleEndian, &m.Header); err != nil {
		return nil, err
	}
	m.Data = buf[syscall.NLMSG_HDRLEN:]

	return []syscall.NetlinkMessage{m}, nil
}

// audit_status message

// AuditStatusMask is a bitmask used to convey the fields used in AuditStatus.
// https://github.com/linux-audit/audit-kernel/blob/v4.7/include/uapi/linux/audit.h#L318-L325
type AuditStatusMask uint32

// Mask types for AuditStatus.
const (
	AuditStatusEnabled AuditStatusMask = 1 << iota
	AuditStatusFailure
	AuditStatusPID
	AuditStatusRateLimit
	AuditStatusBacklogLimit
	AuditStatusBacklogWaitTime
)

var sizeofAuditStatus = int(unsafe.Sizeof(AuditStatus{}))

// AuditStatus is a status message and command and control message exchanged
// between the kernel and user-space.
// https://github.com/linux-audit/audit-kernel/blob/v4.7/include/uapi/linux/audit.h#L413-L427
type AuditStatus struct {
	Mask            AuditStatusMask // Bit mask for valid entries.
	Enabled         uint32          // 1 = enabled, 0 = disabled
	Failure         uint32          // Failure-to-log action.
	PID             uint32          // PID of auditd process.
	RateLimit       uint32          // Messages rate limit (per second).
	BacklogLimit    uint32          // Waiting messages limit.
	Lost            uint32          // Messages lost.
	Backlog         uint32          // Messages waiting in queue.
	FeatureBitmap   uint32          // Bitmap of kernel audit features (previously to 3.19 it was the audit api version number).
	BacklogWaitTime uint32          // Message queue wait timeout.
}

func (s AuditStatus) toWireFormat() []byte {
	buf := bytes.NewBuffer(make([]byte, sizeofAuditStatus))
	buf.Reset()
	if err := binary.Write(buf, binary.LittleEndian, s); err != nil {
		// This never returns an error.
		panic(err)
	}
	return buf.Bytes()
}

// fromWireFormat unmarshals the given buffer to an AuditStatus object. Due to
// changes in the audit_status struct in the kernel source this method does
// not return an error if the buffer is smaller than the sizeof our AuditStatus
// struct.
func (s *AuditStatus) fromWireFormat(buf []byte) error {
	fields := []interface{}{
		&s.Mask,
		&s.Enabled,
		&s.Failure,
		&s.PID,
		&s.RateLimit,
		&s.BacklogLimit,
		&s.Lost,
		&s.Backlog,
		&s.FeatureBitmap,
		&s.BacklogWaitTime,
	}

	if len(buf) == 0 {
		return io.EOF
	}

	r := bytes.NewReader(buf)
	for _, f := range fields {
		if r.Len() == 0 {
			return nil
		}

		if err := binary.Read(r, binary.LittleEndian, f); err != nil {
			return err
		}
	}

	return nil
}
