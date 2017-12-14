package auditd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"syscall"

	"github.com/elastic/go-libaudit"
	"github.com/elastic/go-libaudit/auparse"
)

type MockNetlinkSendReceiver struct {
	messages []syscall.NetlinkMessage
	done     chan struct{}
}

func NewMock() *MockNetlinkSendReceiver {
	return &MockNetlinkSendReceiver{done: make(chan struct{})}
}

func (n *MockNetlinkSendReceiver) returnACK() *MockNetlinkSendReceiver {
	n.messages = append(n.messages, syscall.NetlinkMessage{
		Header: syscall.NlMsghdr{
			Type:  syscall.NLMSG_ERROR,
			Flags: syscall.NLM_F_ACK,
		},
		Data: make([]byte, 4), // Return code 0 (success).
	})
	return n
}

func (n *MockNetlinkSendReceiver) returnDone() *MockNetlinkSendReceiver {
	n.messages = append(n.messages, syscall.NetlinkMessage{
		Header: syscall.NlMsghdr{
			Type:  syscall.NLMSG_DONE,
			Flags: syscall.NLM_F_ACK,
		},
	})
	return n
}

func (n *MockNetlinkSendReceiver) returnStatus() *MockNetlinkSendReceiver {
	status := libaudit.AuditStatus{}
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, status); err != nil {
		panic(err)
	}

	n.messages = append(n.messages, syscall.NetlinkMessage{
		Header: syscall.NlMsghdr{Type: libaudit.AuditGet},
		Data:   buf.Bytes(),
	})
	return n
}

func (n *MockNetlinkSendReceiver) returnMessage(msg ...string) *MockNetlinkSendReceiver {
	for _, m := range msg {
		auditMsg, err := auparse.ParseLogLine(m)
		if err != nil {
			panic(err)
		}

		n.messages = append(n.messages, syscall.NetlinkMessage{
			Header: syscall.NlMsghdr{Type: uint16(auditMsg.RecordType)},
			Data:   []byte(auditMsg.RawData),
		})
	}
	return n
}

func (n *MockNetlinkSendReceiver) Close() error {
	close(n.done)
	return nil
}

func (n *MockNetlinkSendReceiver) Send(msg syscall.NetlinkMessage) (uint32, error) {
	return 0, nil
}

func (n *MockNetlinkSendReceiver) Receive(nonBlocking bool, p libaudit.NetlinkParser) ([]syscall.NetlinkMessage, error) {
	if len(n.messages) > 0 {
		msg := n.messages[0]
		n.messages = n.messages[1:]
		return []syscall.NetlinkMessage{msg}, nil
	}

	// Block until closed.
	<-n.done
	return nil, errors.New("socket closed")
}
