package tcp

import (
	"testing"

	"github.com/elastic/packetbeat/config"
	"github.com/elastic/packetbeat/protos"

	"github.com/stretchr/testify/assert"
)

func Test_configToPortsMap(t *testing.T) {

	type configTest struct {
		Input  map[string]config.Protocol
		Output map[uint16]protos.Protocol
	}

	config_tests := []configTest{
		configTest{
			Input: map[string]config.Protocol{
				"http": config.Protocol{Ports: []int{80, 8080}},
			},
			Output: map[uint16]protos.Protocol{
				80:   protos.HttpProtocol,
				8080: protos.HttpProtocol,
			},
		},
		configTest{
			Input: map[string]config.Protocol{
				"http":  config.Protocol{Ports: []int{80, 8080}},
				"mysql": config.Protocol{Ports: []int{3306}},
				"redis": config.Protocol{Ports: []int{6379, 6380}},
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
			Input: map[string]config.Protocol{
				"http":  config.Protocol{Ports: []int{80, 8080, 8080}},
				"mysql": config.Protocol{Ports: []int{3306}},
			},
			Output: map[uint16]protos.Protocol{
				80:   protos.HttpProtocol,
				8080: protos.HttpProtocol,
				3306: protos.MysqlProtocol,
			},
		},
	}

	for _, test := range config_tests {
		output, err := configToPortsMap(test.Input)
		assert.Nil(t, err)
		assert.Equal(t, test.Output, output)
	}
}

func Test_configToPortsMap_negative(t *testing.T) {

	type errTest struct {
		Input map[string]config.Protocol
		Err   string
	}

	tests := []errTest{
		errTest{
			// should raise error on duplicate port
			Input: map[string]config.Protocol{
				"http":  config.Protocol{Ports: []int{80, 8080}},
				"mysql": config.Protocol{Ports: []int{3306}},
				"redis": config.Protocol{Ports: []int{6379, 6380, 3306}},
			},
			Err: "Duplicate port (3306) exists in mysql and redis protocols",
		},
	}

	for _, test := range tests {
		_, err := configToPortsMap(test.Input)
		assert.NotNil(t, err)
		assert.Equal(t, test.Err, err.Error())
	}
}
