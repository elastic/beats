package network

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

var netService NETService
var oldNetRaw NETRaw
var newNetRaw NETRaw

func TestGetRxBytesPerSecond(t *testing.T) {
	setTime()
	// set old & new rxBytes
	oldNetRaw.RxBytes = 20
	newNetRaw.RxBytes = 120
	//WHEN
	value := netService.getRxBytesPerSecond(&newNetRaw, &oldNetRaw)
	//THEN
	assert.Equal(t, float64(50), value)
}
func TestGetRxDroppedPerSeconde(t *testing.T) {
	setTime()
	oldNetRaw.RxDropped = 40
	newNetRaw.RxDropped = 240

	value := netService.getRxDroppedPerSecond(&newNetRaw, &oldNetRaw)
	assert.Equal(t, float64(100), value)
}
func TestGetRxPacketsPerSeconde(t *testing.T) {
	setTime()
	oldNetRaw.RxPackets = 140
	newNetRaw.RxPackets = 240

	value := netService.getRxPacketsPerSecond(&newNetRaw, &oldNetRaw)
	assert.Equal(t, float64(50), value)
}
func TestGetRxErrorsPerSeconde(t *testing.T) {
	setTime()
	oldNetRaw.RxErrors = 0
	newNetRaw.RxErrors = 0

	value := netService.getRxErrorsPerSecond(&newNetRaw, &oldNetRaw)
	assert.Equal(t, float64(0), value)
}

func TestGetTxBytesPerSecond(t *testing.T) {
	setTime()
	oldNetRaw.TxPackets = 10
	newNetRaw.TxPackets = 0

	value := netService.getTxBytesPerSecond(&newNetRaw, &oldNetRaw)
	assert.Equal(t, float64(0), value)
}
func TestGetTxDroppedPerSeconde(t *testing.T) {
	setTime()
	oldNetRaw.TxDropped = 95
	newNetRaw.TxDropped = 195

	value := netService.getTxDroppedPerSecond(&newNetRaw, &oldNetRaw)
	assert.Equal(t, float64(50), value)
}
func TestGetTxPacketsPerSeconde(t *testing.T) {
	setTime()
	oldNetRaw.TxPackets = 951
	newNetRaw.TxPackets = 1951

	value := netService.getTxPacketsPerSecond(&newNetRaw, &oldNetRaw)
	assert.Equal(t, float64(500), value)
}
func TestGetTxErrorsPerSecond(t *testing.T) {
	setTime()
	oldNetRaw.TxErrors = 995
	newNetRaw.TxErrors = 1995

	value := netService.getTxErrorsPerSecond(&newNetRaw, &oldNetRaw)
	assert.Equal(t, float64(500), value)
}

func setTime() {
	oldNetRaw.Time = time.Now()
	newNetRaw.Time = oldNetRaw.Time.Add(time.Duration(2000000000))
}
func equalEvent(expectedEvent []common.MapStr, event []common.MapStr) bool {

	return reflect.DeepEqual(expectedEvent, event)

}
// Case : old interface  x of the container y !exist
/*func TestGetNetworkStatsFirstEvent(t *testing.T) {

	//GIVEN
	containerID := "containerID"
	labels := map[string]string{
		"label1": "val1",
		"label2": "val2",
	}
	container := dc.APIContainers{
		ID:         containerID,
		Image:      "image",
		Command:    "command",
		Created:    123789,
		Status:     "Up",
		Ports:      []dc.APIPort{{PrivatePort: 1234, PublicPort: 4567, Type: "portType", IP: "123.456.879.1"}},
		SizeRw:     123,
		SizeRootFs: 456,
		Names:      []string{"/container1"},
		Labels:     labels,
		Networks:   dc.NetworkList{},
	}
	networks := make(map[string]dc.NetworkStats, 2)
	networks["eth0"] = dc.NetworkStats{
		RxBytes:   100,
		RxDropped: 200,
		RxErrors:  300,
		RxPackets: 400,
		TxBytes:   500,
		TxDropped: 600,
		TxErrors:  700,
		TxPackets: 800,
	}
	networks["eth1"] = dc.NetworkStats{
		RxBytes:   900,
		RxDropped: 1000,
		RxErrors:  1100,
		RxPackets: 1200,
		TxBytes:   1300,
		TxDropped: 1400,
		TxErrors:  1500,
		TxPackets: 1600,
	}
	// create network stats
	tmp := time.Now()
	mystats := dc.Stats{}
	mystats.Networks = make(map[string]dc.NetworkStats)
	mystats.Networks = networks
	mystats.Read = tmp
	//create dockerStats
	networkStatsStruct := []docker.DockerStat{}
	networkStatsStruct = append(networkStatsStruct, docker.DockerStat{
		Container: container,
		Stats:     mystats,
	})
	//expected events
	expectedEvents := []common.MapStr{}
	expectedEvents = append(expectedEvents, common.MapStr{
		"@timestamp": tmp,
		"container": common.MapStr{
			"id":     containerID,
			"name":   "container1",
			"labels": docker.BuildLabelArray(labels),
		},
		"socket": docker.GetSocket(),
		"eth0": common.MapStr{
			"rx": common.MapStr{
				"bytes":   0,
				"dropped": 0,
				"errors":  0,
				"packets": 0,
			},
			"tx": common.MapStr{
				"bytes":   0,
				"dropped": 0,
				"errors":  0,
				"packets": 0,
			},
		}})
	expectedEvents = append(expectedEvents, common.MapStr{
		"@timestamp": tmp,
		"container": common.MapStr{
			"id":     containerID,
			"name":   "container1",
			"labels": docker.BuildLabelArray(labels),
		},
		"socket": docker.GetSocket(),
		"eth1": common.MapStr{
			"rx": common.MapStr{
				"bytes":   0,
				"dropped": 0,
				"errors":  0,
				"packets": 0,
			},
			"tx": common.MapStr{
				"bytes":   0,
				"dropped": 0,
				"errors":  0,
				"packets": 0,
			},
		},
	})

	networkService := NETService{
		NetworkStatPerContainer: make(map[string]map[string]NETRaw),
	}
	networkService.NetworkStatPerContainer[containerID] = make(map[string]NETRaw)
	//networkService.NetworkStatPerContainer[containerID]["eth0"]= NETRaw{}
	event := networkService.getNetworkStatsPerContainer(networkStatsStruct)
	formattedStats := eventsMapping(event)
	assert.True(t, equalEvent(expectedEvents, formattedStats))
	//t.Logf("expected events : %v", expectedEvents)
	//t.Logf("formated events : %v", formattedStats)
}
func getNetworkCalculatorMocked(number float64) MockNetworkCalculator {
	mockedNetworkCalculator := MockNetworkCalculator{}
	mockedNetworkCalculator.On("getRxBytesPerSecond").Return(number)
	mockedNetworkCalculator.On("getRxDroppedPerSecond").Return(number * 2)
	mockedNetworkCalculator.On("getRxErrorsPerSecond").Return(number * 3)
	mockedNetworkCalculator.On("getRxPacketsPerSecond").Return(number * 4)
	mockedNetworkCalculator.On("getTxBytesPerSecond").Return(number * 5)
	mockedNetworkCalculator.On("getTxDroppedPerSecond").Return(number * 6)
	mockedNetworkCalculator.On("getTxErrorsPerSecond").Return(number * 7)
	mockedNetworkCalculator.On("getTxPacketsPerSecond").Return(number * 8)
	return mockedNetworkCalculator

}
*/
