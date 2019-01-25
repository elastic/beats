package server

import (
	"bytes"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

var srvrTestInput = `Zookeeper version: 3.4.13-2d71af4dbe22557fda74f9a9b4309b15a7487f03, built on 06/29/2018 04:05 GMT
Latency min/avg/max: 1/2/3
Received: 46
Sent: 45
Connections: 1
Outstanding: 0
Zxid: 0x700601132
Mode: standalone
Node count: 4
Proposal sizes last/min/max: -3/-999/-1
`

func TestParser(t *testing.T) {
	mapStr, err := parseSrvr(bytes.NewReader([]byte(srvrTestInput)))
	if err != nil {
		t.Fatal(err)
	}

	version := mapStr["version"].(common.MapStr)
	assert.Equal(t, "06/29/2018 04:05 GMT", version["date"])
	assert.Equal(t, "3.4.13-2d71af4dbe22557fda74f9a9b4309b15a7487f03", version["id"])

	latency := mapStr["latency"].(common.MapStr)
	assert.Equal(t, int64(1), latency["min"])
	assert.Equal(t, int64(2), latency["avg"])
	assert.Equal(t, int64(3), latency["max"])

	assert.Equal(t, int64(46), mapStr["received"])
	assert.Equal(t, int64(45), mapStr["sent"])
	assert.Equal(t, int64(1), mapStr["connections"])
	assert.Equal(t, int64(0), mapStr["outstanding"])
	assert.Equal(t, "standalone", mapStr["mode"])
	assert.Equal(t, int64(4), mapStr["node_count"])

	proposalSizes := mapStr["proposal_sizes"].(common.MapStr)
	assert.Equal(t, int64(-3), proposalSizes["last"])
	assert.Equal(t, int64(-999), proposalSizes["min"])
	assert.Equal(t, int64(-1), proposalSizes["max"])
}
