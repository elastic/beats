// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package interfaces

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
	"github.com/elastic/elastic-agent-libs/logp"
)

type mockPanwClient struct {
	response []byte
	err      error
}

func (m *mockPanwClient) Op(req any, vsys string, extras, ans any) ([]byte, error) {
	return m.response, m.err
}

const tunnelXMLNoState = `<response status="success"><result>
  <ntun>1</ntun>
  <entries>
    <entry>
      <gw>gw_DC02401_008842100459123_WS02019834</gw>
      <kb>0</kb>
      <life>3600</life>
      <TSr_port>0</TSr_port>
      <hash>SHA256</hash>
      <TSi_prefix>0</TSi_prefix>
      <TSi_ip>0.0.0.0</TSi_ip>
      <proto>ESP</proto>
      <TSr_proto>0</TSr_proto>
      <enc>AES256-GCM16</enc>
      <TSr_prefix>0</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>0</TSi_port>
      <TSr_ip>0.0.0.0</TSr_ip>
      <dh>DH20</dh>
      <id>5</id>
      <TSi_proto>0</TSi_proto>
      <name>tl_DC02401_008842100459123_WS02019834</name>
    </entry>
  </entries>
</result></response>`

const tunnelXMLWithState = `<response status="success"><result>
  <ntun>1</ntun>
  <entries>
    <entry>
      <gw>gw_NY01502_009953200587246_CA03028945</gw>
      <kb>512</kb>
      <life>7200</life>
      <TSr_port>443</TSr_port>
      <hash>SHA512</hash>
      <TSi_prefix>24</TSi_prefix>
      <TSi_ip>192.168.100.0</TSi_ip>
      <proto>ESP</proto>
      <TSr_proto>6</TSr_proto>
      <enc>AES128-CBC</enc>
      <TSr_prefix>32</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>0</TSi_port>
      <TSr_ip>10.50.0.0</TSr_ip>
      <dh>DH14</dh>
      <id>8</id>
      <TSi_proto>0</TSi_proto>
      <name>tl_NY01502_009953200587246_CA03028945</name>
      <state>active</state>
    </entry>
  </entries>
</result></response>`

const tunnelXMLMultipleEntries = `<response status="success"><result>
  <ntun>4</ntun>
  <entries>
    <entry>
      <gw>gw_LA03601_001122334455667_TX04037856</gw>
      <kb>1024</kb>
      <life>1800</life>
      <TSr_port>0</TSr_port>
      <hash>SHA256</hash>
      <TSi_prefix>16</TSi_prefix>
      <TSi_ip>172.16.0.0</TSi_ip>
      <proto>ESP</proto>
      <TSr_proto>0</TSr_proto>
      <enc>AES256-GCM16</enc>
      <TSr_prefix>16</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>0</TSi_port>
      <TSr_ip>172.17.0.0</TSr_ip>
      <dh>DH19</dh>
      <id>1</id>
      <TSi_proto>0</TSi_proto>
      <name>tl_LA03601_001122334455667_TX04037856</name>
      <state>active</state>
    </entry>
    <entry>
      <gw>gw_SF02701_009988776655443_OR05048967</gw>
      <kb>0</kb>
      <life>3600</life>
      <TSr_port>0</TSr_port>
      <hash>SHA384</hash>
      <TSi_prefix>0</TSi_prefix>
      <TSi_ip>0.0.0.0</TSi_ip>
      <proto>ESP</proto>
      <TSr_proto>0</TSr_proto>
      <enc>AES192-CBC</enc>
      <TSr_prefix>0</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>0</TSi_port>
      <TSr_ip>0.0.0.0</TSr_ip>
      <dh>DH21</dh>
      <id>2</id>
      <TSi_proto>0</TSi_proto>
      <name>tl_SF02701_009988776655443_OR05048967</name>
    </entry>
    <entry>
      <gw>gw_CH04801_005566778899001_MI06059078</gw>
      <kb>2048</kb>
      <life>7200</life>
      <TSr_port>8080</TSr_port>
      <hash>SHA256</hash>
      <TSi_prefix>24</TSi_prefix>
      <TSi_ip>10.100.50.0</TSi_ip>
      <proto>ESP</proto>
      <TSr_proto>6</TSr_proto>
      <enc>AES256-CBC</enc>
      <TSr_prefix>24</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>443</TSi_port>
      <TSr_ip>10.200.75.0</TSr_ip>
      <dh>DH20</dh>
      <id>3</id>
      <TSi_proto>6</TSi_proto>
      <name>tl_CH04801_005566778899001_MI06059078</name>
      <state>init</state>
    </entry>
    <entry>
      <gw>gw_SE05901_003344556677889_WA07060189</gw>
      <kb>256</kb>
      <life>900</life>
      <TSr_port>22</TSr_port>
      <hash>MD5</hash>
      <TSi_prefix>8</TSi_prefix>
      <TSi_ip>192.0.0.0</TSi_ip>
      <proto>AH</proto>
      <TSr_proto>17</TSr_proto>
      <enc>3DES</enc>
      <TSr_prefix>8</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>53</TSi_port>
      <TSr_ip>193.0.0.0</TSr_ip>
      <dh>DH5</dh>
      <id>4</id>
      <TSi_proto>17</TSi_proto>
      <name>tl_SE05901_003344556677889_WA07060189</name>
      <state>down</state>
    </entry>
  </entries>
</result></response>`

const tunnelXMLEmptyEntries = `<response status="success"><result>
  <ntun>0</ntun>
  <entries>
  </entries>
</result></response>`

const tunnelXMLWithExtraFields = `<response status="success"><result>
  <ntun>2</ntun>
  <entries>
    <entry>
      <gw>gw_BOS03701_002233445566778_PHX08071290</gw>
      <kb>128</kb>
      <life>3600</life>
      <TSr_port>0</TSr_port>
      <hash>SHA256</hash>
      <TSi_prefix>0</TSi_prefix>
      <TSi_ip>0.0.0.0</TSi_ip>
      <proto>ESP</proto>
      <TSr_proto>0</TSr_proto>
      <enc>AES256-GCM16</enc>
      <TSr_prefix>0</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>0</TSi_port>
      <TSr_ip>0.0.0.0</TSr_ip>
      <dh>DH20</dh>
      <id>7</id>
      <TSi_proto>0</TSi_proto>
      <name>tl_BOS03701_002233445566778_PHX08071290</name>
      <state>active</state>
      <peerip>203.0.113.45</peerip>
      <localip>198.51.100.12</localip>
      <outer-if>ae2.200</outer-if>
      <inner-if>tunnel.501</inner-if>
      <mon>up</mon>
      <owner>1</owner>
      <gwid>7</gwid>
    </entry>
    <entry>
      <gw>gw_ATL04801_003344556677889_DEN09082301</gw>
      <kb>256</kb>
      <life>1800</life>
      <TSr_port>443</TSr_port>
      <hash>SHA384</hash>
      <TSi_prefix>24</TSi_prefix>
      <TSi_ip>10.20.30.0</TSi_ip>
      <proto>ESP</proto>
      <TSr_proto>6</TSr_proto>
      <enc>AES128-GCM16</enc>
      <TSr_prefix>24</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>0</TSi_port>
      <TSr_ip>10.40.50.0</TSr_ip>
      <dh>DH19</dh>
      <id>9</id>
      <TSi_proto>0</TSi_proto>
      <name>tl_ATL04801_003344556677889_DEN09082301</name>
      <state>init</state>
      <peerip>192.0.2.78</peerip>
      <localip>198.51.100.99</localip>
      <outer-if>ae3.300</outer-if>
      <inner-if>tunnel.602</inner-if>
      <mon>down</mon>
      <owner>2</owner>
      <gwid>9</gwid>
    </entry>
  </entries>
</result></response>`

const tunnelXMLWithMonitor = `<response status="success"><result>
  <ntun>1</ntun>
  <entries>
    <entry>
      <gw>gw_SEA05901_004455667788990_PDX10093412</gw>
      <kb>512</kb>
      <life>7200</life>
      <TSr_port>0</TSr_port>
      <hash>SHA256</hash>
      <TSi_prefix>0</TSi_prefix>
      <TSi_ip>0.0.0.0</TSi_ip>
      <proto>ESP</proto>
      <TSr_proto>0</TSr_proto>
      <enc>AES256-GCM16</enc>
      <TSr_prefix>0</TSr_prefix>
      <mode>tunl</mode>
      <TSi_port>0</TSi_port>
      <TSr_ip>0.0.0.0</TSr_ip>
      <dh>DH20</dh>
      <id>12</id>
      <TSi_proto>0</TSi_proto>
      <name>tl_SEA05901_004455667788990_PDX10093412</name>
      <state>active</state>
      <pkt-decap>198892459</pkt-decap>
      <remote-spi>A1B2C3D4</remote-spi>
      <enable-gre-encap>False</enable-gre-encap>
      <keytype>auto key</keytype>
      <anti-replay>True</anti-replay>
      <last-rekey>862</last-rekey>
      <dec-err>0</dec-err>
      <inner-warn>0</inner-warn>
      <owner>1</owner>
      <anti-replay-window>1024</anti-replay-window>
      <softtime>3493</softtime>
      <monitor>
        <status>True</status>
        <on>True</on>
        <ka-status>31</ka-status>
        <pkt-recv>3169638</pkt-recv>
        <src>10.255.254.69</src>
        <dst>10.255.254.53</dst>
        <interval>3</interval>
        <bitmap>31</bitmap>
        <rtt>1.5</rtt>
        <pkt-reply>6340818</pkt-reply>
        <threshold>5</threshold>
        <pkt-seen>3170409</pkt-seen>
        <pkt-sent>3247703</pkt-sent>
      </monitor>
      <localip>198.51.100.55</localip>
      <copy-tos>False</copy-tos>
      <remainsize>N/A</remainsize>
      <natt>False</natt>
      <start>9947156</start>
      <pkt-lifesize>0</pkt-lifesize>
      <inner-if>tunnel.902</inner-if>
      <remaintime>2738</remaintime>
      <sid>52002</sid>
      <type>IPSec</type>
      <seq-send>20957</seq-send>
      <byte-decap>28346038120</byte-decap>
      <peerip>203.0.113.101</peerip>
      <owner-cpuid>0</owner-cpuid>
      <seq-recv>16585</seq-recv>
      <timestamp>9947156</timestamp>
      <acquire>38863</acquire>
      <hardtime>3600</hardtime>
      <ts>
        <remote>
          <eip>255.255.255.255</eip>
          <sip>0.0.0.0</sip>
          <sport>0</sport>
          <eport>65535</eport>
          <proto>0</proto>
        </remote>
        <local>
          <eip>255.255.255.255</eip>
          <sip>0.0.0.0</sip>
          <sport>0</sport>
          <eport>65535</eport>
          <proto>0</proto>
        </local>
      </ts>
      <auth>null</auth>
      <pkt-encap>231300323</pkt-encap>
      <pkt-replay>0</pkt-replay>
      <natt-rp>0</natt-rp>
      <local-spi>E5F6A7B8</local-spi>
      <natt-lp>0</natt-lp>
      <initiator>True</initiator>
      <outer-if>ae1.101</outer-if>
      <auth-err>0</auth-err>
      <owner-state>0</owner-state>
      <gwid>12</gwid>
      <mtu>1431</mtu>
      <subtype>None</subtype>
      <byte-encap>144622057752</byte-encap>
      <context>3191</context>
      <pkt-lifetime>0</pkt-lifetime>
    </entry>
  </entries>
</result></response>`

func newTestMetricSet(client panw.PanwClient) *MetricSet {
	return &MetricSet{
		config: &panw.Config{
			HostIp: "10.0.0.1",
		},
		logger: logp.NewLogger("panw_test"),
		client: client,
	}
}

func TestGetIPSecTunnelEvents_WithState(t *testing.T) {
	client := &mockPanwClient{
		response: []byte(tunnelXMLWithState),
	}
	m := newTestMetricSet(client)

	events, err := getIPSecTunnelEvents(m)
	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, 8, event.MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "tl_NY01502_009953200587246_CA03028945", event.MetricSetFields["ipsec_tunnel.name"])
	assert.Equal(t, "active", event.MetricSetFields["ipsec_tunnel.state"])
	assert.Equal(t, "gw_NY01502_009953200587246_CA03028945", event.MetricSetFields["ipsec_tunnel.gw"])
	assert.Equal(t, "ESP", event.MetricSetFields["ipsec_tunnel.proto"])
	assert.Equal(t, "AES128-CBC", event.MetricSetFields["ipsec_tunnel.enc"])
	assert.Equal(t, "SHA512", event.MetricSetFields["ipsec_tunnel.hash"])
	assert.Equal(t, 7200, event.MetricSetFields["ipsec_tunnel.life.sec"])
}

func TestGetIPSecTunnelEvents_NoState(t *testing.T) {
	client := &mockPanwClient{
		response: []byte(tunnelXMLNoState),
	}
	m := newTestMetricSet(client)

	events, err := getIPSecTunnelEvents(m)
	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, 5, event.MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "tl_DC02401_008842100459123_WS02019834", event.MetricSetFields["ipsec_tunnel.name"])
	assert.Equal(t, "", event.MetricSetFields["ipsec_tunnel.state"], "State should be empty when not in XML")
}

func TestGetIPSecTunnelEvents_MultipleEntries(t *testing.T) {
	client := &mockPanwClient{
		response: []byte(tunnelXMLMultipleEntries),
	}
	m := newTestMetricSet(client)

	events, err := getIPSecTunnelEvents(m)
	require.NoError(t, err)
	require.Len(t, events, 4)

	// Entry 1: state = "active"
	assert.Equal(t, 1, events[0].MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "active", events[0].MetricSetFields["ipsec_tunnel.state"])

	// Entry 2: no state field
	assert.Equal(t, 2, events[1].MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "", events[1].MetricSetFields["ipsec_tunnel.state"])

	// Entry 3: state = "init"
	assert.Equal(t, 3, events[2].MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "init", events[2].MetricSetFields["ipsec_tunnel.state"])

	// Entry 4: state = "down"
	assert.Equal(t, 4, events[3].MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "down", events[3].MetricSetFields["ipsec_tunnel.state"])
}

func TestGetIPSecTunnelEvents_EmptyEntries(t *testing.T) {
	client := &mockPanwClient{
		response: []byte(tunnelXMLEmptyEntries),
	}
	m := newTestMetricSet(client)

	events, err := getIPSecTunnelEvents(m)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestGetIPSecTunnelEvents_ClientError(t *testing.T) {
	client := &mockPanwClient{
		err: errors.New("connection refused"),
	}
	m := newTestMetricSet(client)

	events, err := getIPSecTunnelEvents(m)
	require.Error(t, err)
	assert.Nil(t, events)
	assert.Contains(t, err.Error(), "error querying IPSec tunnels")
}

func TestGetIPSecTunnelEvents_InvalidXML(t *testing.T) {
	client := &mockPanwClient{
		response: []byte("<invalid><xml>"),
	}
	m := newTestMetricSet(client)

	events, err := getIPSecTunnelEvents(m)
	require.Error(t, err)
	assert.Nil(t, events)
	assert.Contains(t, err.Error(), "error unmarshaling IPSec tunnels response")
}

func TestGetIPSecTunnelEvents_WithExtraFields(t *testing.T) {
	client := &mockPanwClient{
		response: []byte(tunnelXMLWithExtraFields),
	}
	m := newTestMetricSet(client)

	events, err := getIPSecTunnelEvents(m)
	require.NoError(t, err)
	require.Len(t, events, 2)

	// Entry 1: verify known fields are parsed correctly
	assert.Equal(t, 7, events[0].MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "tl_BOS03701_002233445566778_PHX08071290", events[0].MetricSetFields["ipsec_tunnel.name"])
	assert.Equal(t, "gw_BOS03701_002233445566778_PHX08071290", events[0].MetricSetFields["ipsec_tunnel.gw"])
	assert.Equal(t, "active", events[0].MetricSetFields["ipsec_tunnel.state"])
	assert.Equal(t, "ESP", events[0].MetricSetFields["ipsec_tunnel.proto"])
	assert.Equal(t, "AES256-GCM16", events[0].MetricSetFields["ipsec_tunnel.enc"])
	assert.Equal(t, 3600, events[0].MetricSetFields["ipsec_tunnel.life.sec"])

	// Entry 2: verify state is correctly parsed
	assert.Equal(t, 9, events[1].MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "tl_ATL04801_003344556677889_DEN09082301", events[1].MetricSetFields["ipsec_tunnel.name"])
	assert.Equal(t, "init", events[1].MetricSetFields["ipsec_tunnel.state"])
}

func TestGetIPSecTunnelEvents_WithMonitor(t *testing.T) {
	client := &mockPanwClient{
		response: []byte(tunnelXMLWithMonitor),
	}
	m := newTestMetricSet(client)

	events, err := getIPSecTunnelEvents(m)
	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, 12, event.MetricSetFields["ipsec_tunnel.id"])
	assert.Equal(t, "tl_SEA05901_004455667788990_PDX10093412", event.MetricSetFields["ipsec_tunnel.name"])
	assert.Equal(t, "gw_SEA05901_004455667788990_PDX10093412", event.MetricSetFields["ipsec_tunnel.gw"])
	assert.Equal(t, "active", event.MetricSetFields["ipsec_tunnel.state"])
	assert.Equal(t, "ESP", event.MetricSetFields["ipsec_tunnel.proto"])
	assert.Equal(t, "AES256-GCM16", event.MetricSetFields["ipsec_tunnel.enc"])
	assert.Equal(t, "SHA256", event.MetricSetFields["ipsec_tunnel.hash"])
	assert.Equal(t, 7200, event.MetricSetFields["ipsec_tunnel.life.sec"])
	assert.Equal(t, 512, event.MetricSetFields["ipsec_tunnel.kb"])
}
