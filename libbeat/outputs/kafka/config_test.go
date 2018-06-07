package kafka

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
)

func TestConfigAcceptValid(t *testing.T) {
	tests := map[string]common.MapStr{
		"default config is valid": common.MapStr{},
		"lz4 with 0.11": common.MapStr{
			"compression": "lz4",
			"version":     "0.11",
		},
		"lz4 with 1.0": common.MapStr{
			"compression": "lz4",
			"version":     "1.0.0",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			c, err := common.NewConfigFrom(test)
			if err != nil {
				t.Fatalf("Can not create test configuration: %v", err)
			}
			c.SetString("hosts", 0, "localhost")

			cfg := defaultConfig()
			if err := c.Unpack(&cfg); err != nil {
				t.Fatalf("Unpacking configuration failed: %v", err)
			}

			if _, err := newSaramaConfig(&cfg); err != nil {
				t.Fatalf("Failure creating sarama config: %v", err)
			}
		})
	}
}
