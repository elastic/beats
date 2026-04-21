// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kafkapartitionerextension

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"math/rand/v2"
	"strconv"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type partitionBuilder func(*logp.Logger, *config.C) (kgo.Partitioner, error)

var partitioners = map[string]partitionBuilder{
	"random":      cfgRandomPartitioner,
	"round_robin": cfgRoundRobinPartitioner,
	"hash":        cfgHashPartitioner,
}

func makePartitioner(
	log *logp.Logger,
	partition map[string]any,
) (kgo.Partitioner, error) {

	if len(partition) == 0 {
		return makeHashKgoPartitioner(), nil
	}

	if len(partition) > 1 {
		return nil, fmt.Errorf("too many partitioners configured")
	}

	var name string
	var cfg *config.C
	var err error
	for n, c := range partition {
		cfg, err = config.NewConfigFrom(c)
		name = n
	}

	if err != nil {
		return nil, fmt.Errorf("unable to marshal config %w", err)
	}

	builder := partitioners[name]
	if builder == nil {
		return nil, fmt.Errorf("unknown kafka partition mode %v", name)
	}

	return builder(log, cfg)
}

func cfgRandomPartitioner(_ *logp.Logger, cfg *config.C) (kgo.Partitioner, error) {
	conf := struct {
		GroupEvents int `config:"group_events" validate:"min=1"`
	}{
		GroupEvents: 1,
	}

	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	return kgo.BasicConsistentPartitioner(func(topic string) func(*kgo.Record, int) int {
		N := conf.GroupEvents
		count := N
		partition := 0

		return func(_ *kgo.Record, numPartitions int) int {
			if N == count {
				count = 0
				partition = rand.IntN(numPartitions)
			}
			count++
			return partition
		}
	}), nil
}

func cfgRoundRobinPartitioner(_ *logp.Logger, cfg *config.C) (kgo.Partitioner, error) {
	conf := struct {
		GroupEvents int `config:"group_events" validate:"min=1"`
	}{
		GroupEvents: 1,
	}

	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	return kgo.BasicConsistentPartitioner(func(topic string) func(*kgo.Record, int) int {
		N := conf.GroupEvents
		count := N
		partition := rand.IntN(1<<31 - 1)

		return func(_ *kgo.Record, numPartitions int) int {
			if N == count {
				count = 0
				partition++
				if partition >= numPartitions {
					partition = 0
				}
			}
			count++
			return partition
		}
	}), nil
}

func cfgHashPartitioner(log *logp.Logger, cfg *config.C) (kgo.Partitioner, error) {
	conf := struct {
		Hash   []string `config:"hash"`
		Random bool     `config:"random"`
	}{
		Random: true,
	}

	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	if len(conf.Hash) == 0 {
		return makeHashKgoPartitioner(), nil
	}

	return kgo.BasicConsistentPartitioner(func(topic string) func(*kgo.Record, int) int {
		return makeFieldsHashPartitioner(log, conf.Hash, !conf.Random)
	}), nil
}

func makeHashKgoPartitioner() kgo.Partitioner {
	return kgo.BasicConsistentPartitioner(func(topic string) func(*kgo.Record, int) int {
		hasher := fnv.New32a()

		return func(msg *kgo.Record, numPartitions int) int {
			if msg.Key == nil {
				return rand.IntN(numPartitions)
			}

			hasher.Reset()
			_, _ = hasher.Write(msg.Key)

			return hash2Partition(hasher.Sum32(), numPartitions)
		}
	})
}

func makeFieldsHashPartitioner(
	log *logp.Logger,
	fields []string,
	dropFail bool,
) func(*kgo.Record, int) int {

	hasher := fnv.New32a()

	return func(msg *kgo.Record, numPartitions int) int {
		unmarshaled, err := unmarshalLogs(msg.Value)
		if err != nil {
			log.Errorf("failed to unmarshal logs into map: %v", err)
			return hash2Partition(rand.Uint32(), numPartitions)
		}

		hasher.Reset()
		for _, field := range fields {
			err = hashFieldValue(hasher, unmarshaled, field)
			if err != nil {
				break
			}
		}
		var hash uint32
		if err != nil {
			if dropFail {
				log.Errorf("Hashing partition key failed: %+v", err)
				return -1
			}
			hash = rand.Uint32()
		} else {
			hash = hasher.Sum32()
		}

		return hash2Partition(hash, numPartitions)
	}
}

func hash2Partition(hash uint32, numPartitions int) int {
	return int(hash&0x7fffffff) % numPartitions
}

func hashFieldValue(h hash.Hash32, event mapstr.M, field string) error {
	type stringer interface {
		String() string
	}

	type hashable interface {
		Hash32(h hash.Hash32) error
	}

	v, err := event.GetValue(field)
	if err != nil {
		return err
	}

	switch s := v.(type) {
	case hashable:
		return s.Hash32(h)

	case string:
		_, err = h.Write([]byte(s))

	case []byte:
		_, err = h.Write(s)

	case stringer:
		_, err = h.Write([]byte(s.String()))

	case int8, int16, int32, int64, int,
		uint8, uint16, uint32, uint64, uint:
		err = binary.Write(h, binary.LittleEndian, v)

	case float32:
		tmp := strconv.FormatFloat(float64(s), 'g', -1, 32)
		_, err = h.Write([]byte(tmp))

	case float64:
		tmp := strconv.FormatFloat(s, 'g', -1, 64)
		_, err = h.Write([]byte(tmp))

	default:
		err = binary.Write(h, binary.LittleEndian, v)
		if err != nil {
			return fmt.Errorf("cannot hash key '%v' of unknown type", field)
		}
	}

	return err
}
