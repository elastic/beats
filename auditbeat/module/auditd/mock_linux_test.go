// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package auditd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"syscall"

	"github.com/elastic/go-libaudit/v2"
	"github.com/elastic/go-libaudit/v2/auparse"
)

type MockNetlinkSendReceiver struct {
	messages []syscall.NetlinkMessage
	sendRet  []uint32
	errors   []error
	done     chan struct{}
}

func NewMock() *MockNetlinkSendReceiver {
	return &MockNetlinkSendReceiver{done: make(chan struct{})}
}

func (n *MockNetlinkSendReceiver) returnACK() *MockNetlinkSendReceiver {
	return n.returnReceiveAckWithSeq(0)
}

func (n *MockNetlinkSendReceiver) returnReceiveAckWithSeq(seq uint32) *MockNetlinkSendReceiver {
	n.messages = append(n.messages, syscall.NetlinkMessage{
		Header: syscall.NlMsghdr{
			Type:  syscall.NLMSG_ERROR,
			Flags: syscall.NLM_F_ACK,
			Seq:   seq,
		},
		Data: make([]byte, 4), // Return code 0 (success).
	})
	return n
}

func (n *MockNetlinkSendReceiver) returnSendValue(ret uint32) *MockNetlinkSendReceiver {
	n.sendRet = append(n.sendRet, ret)
	return n
}

func (n *MockNetlinkSendReceiver) returnReceiveError(err error) *MockNetlinkSendReceiver {
	n.errors = append(n.errors, err)
	return n
}

//nolint:unused // it still might be useful for tests in the future
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
	if len(n.sendRet) > 0 {
		ret := n.sendRet[0]
		n.sendRet = n.sendRet[1:]
		return ret, nil
	}
	return 0, nil
}

func (n *MockNetlinkSendReceiver) SendNoWait(msg syscall.NetlinkMessage) (uint32, error) {
	return 0, nil
}

func (n *MockNetlinkSendReceiver) Receive(nonBlocking bool, p libaudit.NetlinkParser) ([]syscall.NetlinkMessage, error) {
	if len(n.errors) > 0 {
		err := n.errors[0]
		n.errors = n.errors[1:]
		return nil, err
	}
	if len(n.messages) > 0 {
		msg := n.messages[0]
		n.messages = n.messages[1:]
		return []syscall.NetlinkMessage{msg}, nil
	}

	// Block until closed.
	<-n.done
	return nil, errors.New("socket closed")
}
