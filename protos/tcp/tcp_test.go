package tcp

import (
	"testing"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/packetbeat/protos"

	"github.com/stretchr/testify/assert"
)

type TestProtocol struct {
	Ports []int
}

func (proto *TestProtocol) Init(test_mode bool, results chan common.MapStr) error {
	return nil
}

func (proto *TestProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto *TestProtocol) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {
	return private
}

func (proto *TestProtocol) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return private
}

func (proto *TestProtocol) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	return private, true
}

func Test_configToPortsMap(t *testing.T) {

	type configTest struct {
		Input  map[protos.Protocol]protos.ProtocolPlugin
		Output map[uint16]protos.Protocol
	}

	config_tests := []configTest{
		configTest{
			Input: map[protos.Protocol]protos.ProtocolPlugin{
				protos.HttpProtocol: &TestProtocol{Ports: []int{80, 8080}},
			},
			Output: map[uint16]protos.Protocol{
				80:   protos.HttpProtocol,
				8080: protos.HttpProtocol,
			},
		},
		configTest{
			Input: map[protos.Protocol]protos.ProtocolPlugin{
				protos.HttpProtocol:  &TestProtocol{Ports: []int{80, 8080}},
				protos.MysqlProtocol: &TestProtocol{Ports: []int{3306}},
				protos.RedisProtocol: &TestProtocol{Ports: []int{6379, 6380}},
			},
			Output: map[uint16]protos.Protocol{
				80:   protos.HttpProtocol,
				8080: protos.HttpProtocol,
				3306: protos.MysqlProtocol,
				6379: protos.RedisProtocol,
				6380: protos.RedisProtocol,
			},
		},

		// should ignore duplicate ports in the same protocol
		configTest{
			Input: map[protos.Protocol]protos.ProtocolPlugin{
				protos.HttpProtocol:  &TestProtocol{Ports: []int{80, 8080, 8080}},
				protos.MysqlProtocol: &TestProtocol{Ports: []int{3306}},
			},
			Output: map[uint16]protos.Protocol{
				80:   protos.HttpProtocol,
				8080: protos.HttpProtocol,
				3306: protos.MysqlProtocol,
			},
		},
	}

	for _, test := range config_tests {
		output, err := buildPortsMap(test.Input)
		assert.Nil(t, err)
		assert.Equal(t, test.Output, output)
	}
}

func Test_configToPortsMap_negative(t *testing.T) {

	type errTest struct {
		Input map[protos.Protocol]protos.ProtocolPlugin
		Err   string
	}

	tests := []errTest{
		errTest{
			// should raise error on duplicate port
			Input: map[protos.Protocol]protos.ProtocolPlugin{
				protos.HttpProtocol:  &TestProtocol{Ports: []int{80, 8080}},
				protos.MysqlProtocol: &TestProtocol{Ports: []int{3306}},
				protos.RedisProtocol: &TestProtocol{Ports: []int{6379, 6380, 3306}},
			},
			Err: "Duplicate port (3306) exists",
		},
	}

	for _, test := range tests {
		_, err := buildPortsMap(test.Input)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), test.Err)
	}
}

func Test_portsToBpfFilter(t *testing.T) {
	type io struct {
		Ports     []int
		WithVlans bool
		Output    string
	}

	tests := []io{
		io{
			Ports:  []int{2, 3, 4},
			Output: "port 2 or port 3 or port 4",
		},
		io{
			Ports:  []int{80, 8080},
			Output: "port 80 or port 8080",
		},
		io{
			Ports:     []int{2, 3, 4},
			WithVlans: true,
			Output:    "port 2 or port 3 or port 4 or (vlan and (port 2 or port 3 or port 4))",
		},
		io{
			Ports:     []int{80, 8080},
			WithVlans: true,
			Output:    "port 80 or port 8080 or (vlan and (port 80 or port 8080))",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, portsToBpfFilter(test.Ports, test.WithVlans))
	}
}
