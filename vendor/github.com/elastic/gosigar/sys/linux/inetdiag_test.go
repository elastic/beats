// +build linux

package linux

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseInetDiagMsgs reads netlink messages stored in a file (these can be
// captured with ss -diag <file>).
func TestParseInetDiagMsgs(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/inet-dump-rhel6-2.6.32-504.3.3.el6.x86_64.bin")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Netlink data length: ", len(data))
	netlinkMsgs, err := syscall.ParseNetlinkMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Parsed %d netlink messages", len(netlinkMsgs))
	done := false
	for _, m := range netlinkMsgs {
		if m.Header.Type == syscall.NLMSG_DONE {
			done = true
			break
		}

		inetDiagMsg, err := ParseInetDiagMsg(m.Data)
		if err != nil {
			t.Fatal("parse error", err)
		}

		if inetDiagMsg.DstPort() == 0 {
			assert.EqualValues(t, TCP_LISTEN, inetDiagMsg.State)
		} else {
			assert.EqualValues(t, TCP_ESTABLISHED, inetDiagMsg.State)
		}
	}

	assert.True(t, done, "missing NLMSG_DONE message")
}

// TestNetlinkInetDiag sends a inet_diag_req to the kernel, checks for errors,
// and inspects the responses based on some invariant rules.
func TestNetlinkInetDiag(t *testing.T) {
	req := NewInetDiagReq()
	req.Header.Seq = 12345

	dump := new(bytes.Buffer)
	msgs, err := NetlinkInetDiagWithBuf(req, nil, dump)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Received %d messages decoded from %d bytes", len(msgs), dump.Len())
	for _, m := range msgs {
		if m.Family != uint8(AF_INET) && m.Family != uint8(AF_INET6) {
			t.Errorf("invalid Family (%v)", m.Family)
		}

		if m.DstPort() == 0 {
			assert.True(t, m.DstIP().IsUnspecified(), "dport is 0, dst ip should be unspecified")
			assert.EqualValues(t, m.State, TCP_LISTEN)
		}
	}

	if t.Failed() {
		t.Log("Raw newlink response:\n", hex.Dump(dump.Bytes()))
	}
}
