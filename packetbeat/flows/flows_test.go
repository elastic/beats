// +build !integration

package flows

import (
	"net"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/config"
	"github.com/stretchr/testify/assert"
)

type flowsChan struct {
	ch chan []common.MapStr
}

func (f *flowsChan) PublishFlows(events []common.MapStr) bool {
	f.ch <- events
	return true
}

func TestFlowsCounting(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}
	mac1 := []byte{1, 2, 3, 4, 5, 6}
	mac2 := []byte{6, 5, 4, 3, 2, 1}
	ip1 := []byte{127, 0, 0, 1}
	ip2 := []byte{128, 0, 1, 2}
	port1 := []byte{0, 1}
	port2 := []byte{0, 2}

	module, err := NewFlows(nil, &config.Flows{})
	assert.NoError(t, err)

	uint1, err := module.NewUint("uint1")
	uint2, err := module.NewUint("uint2")
	int1, err := module.NewInt("int1")
	int2, err := module.NewInt("int2")
	float1, err := module.NewFloat("float1")
	float2, err := module.NewFloat("float2")

	assert.NoError(t, err)

	pub := &flowsChan{make(chan []common.MapStr, 1)}

	processor := &flowsProcessor{
		table:    module.table,
		counters: module.counterReg,
		timeout:  20 * time.Millisecond,
	}
	processor.spool.init(pub, 1)

	worker, err := makeWorker(
		processor,
		10*time.Millisecond,
		1,
		-1,
		0)
	if err != nil {
		t.Fatalf("Failed to create flow worker: %v", err)
	}

	worker.Start()
	defer worker.Stop()

	idForward := newFlowID()
	addrForward := addAll(
		addEther(mac1, mac2),
		addIP(ip1, ip2),
		addTCP(port1, port2),
	)
	addrForward(idForward)

	idRev := newFlowID()
	addrRev := addAll(
		addEther(mac2, mac1),
		addIP(ip2, ip1),
		addTCP(port2, port1),
	)
	addrRev(idRev)
	assert.True(t, FlowIDsEqual(idForward, idRev))

	{
		module.Lock()

		flow := module.Get(idForward)
		flowRev := module.Get(idRev)

		int1.Add(flow, -1)
		uint1.Add(flow, 1)
		float1.Add(flow, 3.14)

		int2.Set(flowRev, -1)
		uint2.Set(flowRev, 5)
		float2.Set(flowRev, 1.4142)

		module.Unlock()
	}

	var events []common.MapStr
	select {
	case events = <-pub.ch:
	case <-time.After(5 * time.Second):
	}

	if events == nil {
		t.Fatalf("no event received in time")
	}
	event := events[0]
	t.Logf("event: %v", event)

	source := event["source"].(common.MapStr)
	dest := event["dest"].(common.MapStr)

	// validate generated event
	assert.Equal(t, net.HardwareAddr(mac1).String(), source["mac"])
	assert.Equal(t, net.HardwareAddr(mac2).String(), dest["mac"])
	assert.Equal(t, net.IP(ip1).String(), source["ip"])
	assert.Equal(t, net.IP(ip2).String(), dest["ip"])
	assert.Equal(t, uint16(256), source["port"])
	assert.Equal(t, uint16(512), dest["port"])
	assert.Equal(t, "tcp", event["transport"])

	stat := source["stats"].(map[string]interface{})
	assert.Equal(t, int64(-1), stat["int1"])
	assert.Equal(t, nil, stat["int2"])
	assert.Equal(t, uint64(1), stat["uint1"])
	assert.Equal(t, nil, stat["uint2"])
	assert.Equal(t, 3.14, stat["float1"])
	assert.Equal(t, nil, stat["float2"])

	stat = dest["stats"].(map[string]interface{})
	assert.Equal(t, nil, stat["int1"])
	assert.Equal(t, int64(-1), stat["int2"])
	assert.Equal(t, nil, stat["uint1"])
	assert.Equal(t, uint64(5), stat["uint2"])
	assert.Equal(t, nil, stat["float1"])
	assert.Equal(t, 1.4142, stat["float2"])
}
