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

package dns

import (
	"net"
	"strings"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/monitoring"
)

var _ PTRResolver = (*MiekgResolver)(nil)

func TestMiekgResolverLookupPTR(t *testing.T) {
	stop, addr, err := ServeDNS(FakeDNSHandler)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	reg := monitoring.NewRegistry()
	res, err := NewMiekgResolver(reg.NewRegistry(logName), 0, addr)
	if err != nil {
		t.Fatal(err)
	}

	// Success
	ptr, err := res.LookupPTR("8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, "google-public-dns-a.google.com", ptr.Host)
	assert.EqualValues(t, 19273, ptr.TTL)

	// NXDOMAIN
	_, err = res.LookupPTR("1.1.1.1")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "NXDOMAIN")
	}

	// Validate that our metrics exist.
	var metricCount int
	reg.Do(monitoring.Full, func(name string, v interface{}) {
		if strings.Contains(name, "processor.dns") {
			metricCount++
		}
		t.Logf("%v: %+v", name, v)
	})
	assert.Equal(t, 12, metricCount)
}

func ServeDNS(h dns.HandlerFunc) (cancel func() error, addr string, err error) {
	// Setup listener on ephemeral port.
	a, err := net.ResolveUDPAddr("udp4", "localhost:0")
	if err != nil {
		return nil, "", err
	}
	l, err := net.ListenUDP("udp4", a)
	if err != nil {
		return nil, "", err
	}

	var s dns.Server
	s.PacketConn = l
	s.Handler = h
	go s.ActivateAndServe()
	return s.Shutdown, s.PacketConn.LocalAddr().String(), err
}

func FakeDNSHandler(w dns.ResponseWriter, msg *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(msg)
	switch {
	case strings.HasPrefix(msg.Question[0].Name, "8.8.8.8"):
		m.Answer = make([]dns.RR, 1)
		m.Answer[0], _ = dns.NewRR("8.8.8.8.in-addr.arpa.	19273	IN	PTR	google-public-dns-a.google.com.")
	default:
		m.SetRcode(msg, dns.RcodeNameError)
	}
	w.WriteMsg(m)
}
