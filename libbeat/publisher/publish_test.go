// +build !integration

package publisher

import (
	"runtime"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

const (
	shipperName   = "testShipper"
	hostOnNetwork = "someHost"
)

type testTopology struct {
	hostname string // Hostname returned by GetNameByIP.

	// Parameters from PublishIPs invocation.
	publishName       chan string
	publishLocalAddrs chan []string
}

var _ outputs.TopologyOutputer = testTopology{}

func (topo testTopology) PublishIPs(name string, localAddrs []string) error {
	topo.publishName <- name
	topo.publishLocalAddrs <- localAddrs
	return nil
}

func (topo testTopology) GetNameByIP(ip string) string {
	return topo.hostname
}

// Test GetServerName.
func TestPublisherTypeGetServerName(t *testing.T) {
	pt := &BeatPublisher{name: shipperName}
	assert.Equal(t, shipperName, pt.GetServerName("127.0.0.1"))

	// Unknown hosts return empty string.
	assert.Equal(t, "", pt.GetServerName("172.0.0.1"))

	// Hostname is returned when topology knows the IP.
	pt.TopologyOutput = testTopology{hostname: hostOnNetwork}
	assert.Equal(t, hostOnNetwork, pt.GetServerName("172.0.0.1"))
}

// Test the PublisherType UpdateTopologyPeriodically() method.
func TestPublisherTypeUpdateTopologyPeriodically(t *testing.T) {
	// Setup.
	c := make(chan time.Time, 1)
	topo := testTopology{
		hostname:          hostOnNetwork,
		publishName:       make(chan string, 1),
		publishLocalAddrs: make(chan []string, 1),
	}
	pt := &BeatPublisher{
		name:                 shipperName,
		RefreshTopologyTimer: c,
		TopologyOutput:       topo,
	}

	// Simulate a single clock tick and close the channel.
	c <- time.Now()
	close(c)
	pt.UpdateTopologyPeriodically()

	// Validate that PublishTopology was invoked.
	assert.Equal(t, shipperName, <-topo.publishName)
	switch runtime.GOOS {
	default:
		assert.True(t, len(<-topo.publishLocalAddrs) > 0)
	case "nacl", "plan9", "solaris":
		t.Skipf("Golang's net.InterfaceAddrs is a stub on %s", runtime.GOOS)
	}
}
