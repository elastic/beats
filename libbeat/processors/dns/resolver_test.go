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
	"crypto/tls"
	"net"
	"strings"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/monitoring"
)

var _ PTRResolver = (*MiekgResolver)(nil)

func TestMiekgResolverLookupPTR(t *testing.T) {
	stop, addr, err := ServeDNS(FakeDNSHandler)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	reg := monitoring.NewRegistry()
	res, err := NewMiekgResolver(reg.NewRegistry(logName), 0, "udp", addr)
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

func TestMiekgResolverLookupPTRTLS(t *testing.T) {
	//Build Cert
	cert, err := tls.X509KeyPair(CertPEMBlock, KeyPEMBlock)
	if err != nil {
		t.Fatalf("unable to build certificate: %v", err)
	}
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	// serve TLS with cert
	stop, addr, err := ServeDNSTLS(FakeDNSHandler, &config)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	reg := monitoring.NewRegistry()

	res, err := NewMiekgResolver(reg.NewRegistry(logName), 0, "tls", addr)
	if err != nil {
		t.Fatal(err)
	}
	// we use a self signed certificate for localhost
	// we have to pass InsecureSSL to the DNS resolver
	res.client.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
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

func ServeDNSTLS(h dns.HandlerFunc, config *tls.Config) (cancel func() error, addr string, err error) {
	// Setup listener on ephemeral port.
	l, err := tls.Listen("tcp", "localhost:0", config)
	if err != nil {
		return nil, "", err
	}

	var s dns.Server
	s.Handler = h
	s.Listener = l
	go s.ActivateAndServe()
	return s.Shutdown, l.Addr().String(), err
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

var (
	KeyPEMBlock = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA2g2zpEtWaIUx5o6MEnWnGsf0Ba1SDc3AwgOmxeNIPBJYVCrk
sWe8Qt/5nymReVFcum76995ncr/zT+e4e8l+hXuGzTKZJpOj27Igb0/wa3j2hIcu
rnbzfwkJ+KMag2UUKdSo31ChMU+64bwziEXunF347Ot7dBLtw3PJKbabNCP+/oil
iUv2TzxxYosN+AEg4gNKLa3DMpbUnD+9Igb9KmaVp1FVhZted/AP4vn7h6Urb4ER
xMuvv3xqZvIKQ9/G1XAISYXk2feZ5yP+k1HF4ds7HJDwrP+Bv+EVyv38EKdmu9N3
Oej8wKf3Acjln/ucbg1S3Dmkyg0x2388S4c35wIDAQABAoIBAB8MnGvknmU7siNW
YPOv9R+HIWQ9jdWRWsVFp9W9y2diZVl20iHA17neErlrPd+8iiux6eKptKlOU+Mo
58gYpP9023kUn2Iy275I2v1+sIldLB0q8qa9IWcRbm4NK5VSK1DZi0JhRNK0u7Ox
DNV2v8dcSjnSPj4FA/402owqCGegBQuheYE0LDEMiNAm6hZmQ5Npf0mTfJA/OuM4
ONSR7lNncrR0pOZ3f3WWH+021eoZCgu2A64yfX5FFI7y5jvRn8KigXEDfXcdyFKO
725Slq4V2E2NmrMyRKNBLUSUC2hcy0tQsfo3+yANxA6PBNQ0EVqkF4uGn1IzNWOz
gDSyfSECgYEA2jgTpv9v0SrURdY3lOOjYZNCoJ9ZhUTxOsQQZLUJ+1/bQQ4Y0ONK
cnC/Ve76C/k+otbILAaRnOxGw5Apq25yPNoxjFFzP7tbN85IB+4db637qZNK2gfX
oEJd6wat4Urs8NbUKCE+XkbdENOIdXUiQxp9U6jXxprd5Ii4jICwRvsCgYEA/85J
1to++Td64gKfWDv4FUo5ZqVn70JdM/Knf5Pd37z/sjNowxhDz7AhismRditX02lt
T2g/raIW9Z/SpxI44VHCRJGPOvBvaMgCNGOH0FBHatFsfKwKzpMwapTfobqj3ZYa
DDDc8r9WQM8IDcLM6B7aOV46LWMEhMRSfDa9bwUCgYEAokbRVn7eSE3xTX3gF3ix
Jv67rXbSu6hpO6pSBpIaujSud9Jj4fMkibYOk3kDuaPAUJgog5Te9DNA7G1oj3Oy
wE4CSrbHXb2WOAnOxxbsDQD1BUXjhAAQ+bxg20Y8SC3Pxcn8O1t9Zd6MxtaHw9E3
iW9Jg80rqSXBnRGPK+0HKcECgYBsRYk1WjzLSTNG1CtTslZH1JnFG3+JYoKGiU9i
DVkc6Sck6uONqAiTsI4R600ZQjEzN21f7dT+Dhw/rH0B4BGZNPzP/vgrzzaol/du
6y3B+yivSqLrhfoxA1W71vVsw8217WFrBYePa3L7jWVwRaJrIRvmqj5flYiFFX+A
Ob8mbQKBgAHhlnVzoKCq4mZ7Glpc0K6L57btVZNn0TEGyVli1ECvgC3zRm1rEofG
LatVl7h6ud25ZJYnP7DelGxHsZnDXNirLFlSB0CL4F6I5xNoBvCoH0Q8ckDSh4C7
tlAyD5m9gwvgdkNFWq6/lcUPxGksTtTk8dGnhJz8pGlZvp6+dZCM
-----END RSA PRIVATE KEY-----`)

	CertPEMBlock = []byte(`-----BEGIN CERTIFICATE-----
MIIDaTCCAlGgAwIBAgIQGqg47wLgbjwwrZASuakmwjANBgkqhkiG9w0BAQsFADAy
MRQwEgYDVQQKEwtMb2cgQ291cmllcjEaMBgGA1UEAxMRYmVhdHMuZWxhc3RpYy5j
b20wHhcNMjAwNjIzMDY0NDEwWhcNMjEwNjIzMDY0NDEwWjAyMRQwEgYDVQQKEwtM
b2cgQ291cmllcjEaMBgGA1UEAxMRYmVhdHMuZWxhc3RpYy5jb20wggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDaDbOkS1ZohTHmjowSdacax/QFrVINzcDC
A6bF40g8ElhUKuSxZ7xC3/mfKZF5UVy6bvr33mdyv/NP57h7yX6Fe4bNMpkmk6Pb
siBvT/BrePaEhy6udvN/CQn4oxqDZRQp1KjfUKExT7rhvDOIRe6cXfjs63t0Eu3D
c8kptps0I/7+iKWJS/ZPPHFiiw34ASDiA0otrcMyltScP70iBv0qZpWnUVWFm153
8A/i+fuHpStvgRHEy6+/fGpm8gpD38bVcAhJheTZ95nnI/6TUcXh2zsckPCs/4G/
4RXK/fwQp2a703c56PzAp/cByOWf+5xuDVLcOaTKDTHbfzxLhzfnAgMBAAGjezB5
MA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8E
BTADAQH/MEEGA1UdEQQ6MDiCATqCCWxvY2FsaG9zdIcQAAAAAAAAAAAAAAAAAAAA
AIcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG9w0BAQsFAAOCAQEAL6px
cjflhqqewqa9cvhFNT6E7UDnA7Mf34GIQPQrORXyOnyE11mDp5sEMGaz8bDajHHc
0JL8Q/5rDyRsSfe1pIyViAOxn+V/7qXfgowI3tkJbSaqHX7SlHF0dEiuGQ1coBMx
RgW17XhPtV+fk/DiXtUEkgtB7/q0Kc9C9C2GJIbOtupZ/mnkdk/5YT4tfXywNnWC
lLjT6T5+wZgRkcnr7lYNiTdS+GtN0YspPT+YD3ZTJCYD9KPcbA6k9XXXwmU8Ij6H
waodyGzG03YJbY3l2zSt3lG3jv9Tj+Ic0kRyEzzxk8exyi6nWXue/6a884kgAjiL
bXmdL6wkIJz1U+XtuQ==
-----END CERTIFICATE-----`)
)
