// +build !integration

package kafka

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

type partTestScenario func(*testing.T, bool, sarama.Partitioner) error

func TestPartitioners(t *testing.T) {
	type obj map[string]interface{}
	type arr []interface{}

	nonHashScenarios := []partTestScenario{
		partTestSimple(100, false),
	}

	hashScenarios := []partTestScenario{
		partTestSimple(100, true),
		partTestHashInvariant(1),
	}

	tests := []struct {
		title         string
		reachableOnly bool
		scenarios     []partTestScenario
		config        obj
	}{
		{
			"random every event, non-consistent ",
			true,
			nonHashScenarios,
			obj{"partition.random": obj{
				"reachable_only": true,
				"group_events":   1,
			}},
		},
		{
			"random every event, consistent",
			false,
			nonHashScenarios,
			obj{"partition.random": obj{
				"reachable_only": false,
				"group_events":   1,
			}},
		},
		{
			"random every 3rd event, non-consistent",
			true,
			nonHashScenarios,
			obj{"partition.random": obj{
				"reachable_only": true,
				"group_events":   3,
			}},
		},
		{
			"random every 3rd event, consistent",
			false,
			nonHashScenarios,
			obj{"partition.random": obj{
				"reachable_only": false,
				"group_events":   3,
			}},
		},
		{
			"round-robin every event, non-consistent",
			true,
			nonHashScenarios,
			obj{"partition.round_robin": obj{
				"reachable_only": true,
				"group_events":   1,
			}},
		},
		{
			"round-robin every event, consistent",
			false,
			nonHashScenarios,
			obj{"partition.round_robin": obj{
				"reachable_only": false,
				"group_events":   1,
			}},
		},
		{
			"round-robin every 3rd event, non-consistent",
			true,
			nonHashScenarios,
			obj{"partition.round_robin": obj{
				"reachable_only": true,
				"group_events":   3,
			}},
		},
		{
			"round-robin every 3rd event, consistent",
			false,
			nonHashScenarios,
			obj{"partition.round_robin": obj{
				"reachable_only": false,
				"group_events":   3,
			}},
		},
		{
			"hash without key, fallback random, non-consistent",
			true,
			nonHashScenarios,
			obj{"partition.hash": obj{
				"reachable_only": true,
			}},
		},
		{
			"hash without key, fallback random, consistent",
			false,
			nonHashScenarios,
			obj{"partition.hash": obj{
				"reachable_only": false,
			}},
		},
		{
			"hash with key, consistent",
			true,
			hashScenarios,
			obj{"partition.hash": obj{
				"reachable_only": true,
			}},
		},
		{
			"hash with key, non-consistent",
			false,
			hashScenarios,
			obj{"partition.hash": obj{
				"reachable_only": false,
			}},
		},
		{
			"hash message field, non-consistent",
			true,
			hashScenarios,
			obj{"partition.hash": obj{
				"reachable_only": true,
				"hash":           arr{"message"},
			}},
		},
		{
			"hash message field, consistent",
			false,
			hashScenarios,
			obj{"partition.hash": obj{
				"reachable_only": false,
				"hash":           arr{"message"},
			}},
		},
	}

	for i, test := range tests {
		t.Logf("run test(%v): %v", i, test.title)

		cfg, err := common.NewConfigFrom(test.config)
		if err != nil {
			t.Error(err)
			continue
		}

		pcfg := struct {
			Partition map[string]*common.Config `config:"partition"`
		}{}
		err = cfg.Unpack(&pcfg)
		if err != nil {
			t.Error(err)
			continue
		}

		constr, err := makePartitioner(pcfg.Partition)
		if err != nil {
			t.Error(err)
			continue
		}

		for _, runner := range test.scenarios {
			partitioner := constr("test")
			err := runner(t, test.reachableOnly, partitioner)
			if err != nil {
				t.Error(err)
				break
			}
		}
	}
}

func partTestSimple(N int, makeKey bool) partTestScenario {
	numPartitions := int32(15)

	return func(t *testing.T, reachableOnly bool, part sarama.Partitioner) error {
		t.Logf("  simple test with %v partitions", numPartitions)

		partitions := make([]int, numPartitions)

		requiresConsistency := !reachableOnly
		assert.Equal(t, requiresConsistency, part.RequiresConsistency())

		for i := 0; i <= N; i++ {
			ts := time.Now()

			event := common.MapStr{
				"@timestamp": common.Time(ts),
				"type":       "test",
				"message":    randString(20),
			}

			jsonEvent, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("json encoding failed with %v", err)
			}

			msg := &message{partition: -1}
			msg.data = outputs.Data{event, nil}
			msg.topic = "test"
			if makeKey {
				msg.key = randASCIIBytes(10)
			}
			msg.value = jsonEvent
			msg.ts = ts
			msg.initProducerMessage()

			p, err := part.Partition(&msg.msg, numPartitions)
			if err != nil {
				return err
			}

			assert.True(t, 0 <= p && p < numPartitions)
			partitions[p]++
		}

		// count number of partitions being used
		nPartitions := 0
		for _, p := range partitions {
			if p > 0 {
				nPartitions++
			}
		}
		t.Logf("    partitions used: %v/%v", nPartitions, numPartitions)
		assert.True(t, nPartitions > 3)

		return nil
	}
}

func partTestHashInvariant(N int) partTestScenario {
	numPartitions := int32(15)

	return func(t *testing.T, reachableOnly bool, part sarama.Partitioner) error {
		t.Logf("  hash invariant test with %v partitions", numPartitions)

		for i := 0; i <= N; i++ {
			ts := time.Now()

			event := common.MapStr{
				"@timestamp": common.Time(ts),
				"type":       "test",
				"message":    randString(20),
			}

			jsonEvent, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("json encoding failed with %v", err)
			}

			msg := &message{partition: -1}
			msg.data = outputs.Data{event, nil}
			msg.topic = "test"
			msg.key = randASCIIBytes(10)
			msg.value = jsonEvent
			msg.ts = ts
			msg.initProducerMessage()

			p1, err := part.Partition(&msg.msg, numPartitions)
			if err != nil {
				return err
			}

			// reset message state
			msg.hash = 0
			msg.partition = -1

			p2, err := part.Partition(&msg.msg, numPartitions)
			if err != nil {
				return err
			}

			assert.True(t, 0 <= p1 && p1 < numPartitions)
			assert.True(t, 0 <= p2 && p2 < numPartitions)
			assert.Equal(t, p1, p2)
		}

		return nil
	}
}
