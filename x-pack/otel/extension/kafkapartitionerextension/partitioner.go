// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kafkapartitionerextension

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/elastic/beats/v7/libbeat/outputs/kafka"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type partitionBuilder func(*logp.Logger, *config.C, bool) (kgo.Partitioner, error)

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
		return nil, fmt.Errorf("unable to get config %w", err)
	}
	// parse shared config
	reachable := struct {
		Reachable bool `config:"reachable_only"`
	}{
		Reachable: false,
	}
	err = cfg.Unpack(&reachable)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack config %w", err)
	}
	builder := partitioners[name]
	if builder == nil {
		return nil, fmt.Errorf("unknown kafka partition mode %v", name)
	}

	return builder(log, cfg, reachable.Reachable)
}

func cfgRandomPartitioner(_ *logp.Logger, cfg *config.C, reachable bool) (kgo.Partitioner, error) {
	conf := struct {
		GroupEvents int `config:"group_events" validate:"min=1"`
	}{
		GroupEvents: 1,
	}

	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	return partitionerFn(func(topic string) func(*kgo.Record, int) int {
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
	}, reachable), nil
}

func cfgRoundRobinPartitioner(_ *logp.Logger, cfg *config.C, reachable bool) (kgo.Partitioner, error) {
	conf := struct {
		GroupEvents int `config:"group_events" validate:"min=1"`
	}{
		GroupEvents: 1,
	}

	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	return partitionerFn(func(topic string) func(*kgo.Record, int) int {
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
	}, reachable), nil
}

func cfgHashPartitioner(log *logp.Logger, cfg *config.C, reachable bool) (kgo.Partitioner, error) {
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

	return partitionerFn(func(topic string) func(*kgo.Record, int) int {
		return makeFieldsHashPartitioner(log, conf.Hash, !conf.Random)
	}, reachable), nil
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

			return int(kafka.Hash2Partition(hasher.Sum32(), int32(numPartitions))) //nolint:gosec // Conversion from int to int32 is safe here.
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
			if dropFail {
				return -1
			}
			return int(kafka.Hash2Partition(rand.Uint32(), int32(numPartitions))) //nolint:gosec // Conversion from int to int32 is safe here.
		}

		hasher.Reset()
		for _, field := range fields {
			err = kafka.HashFieldValue(hasher, unmarshaled, field)
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

		return int(kafka.Hash2Partition(hash, int32(numPartitions))) //nolint:gosec // Conversion from int to int32 is safe here.
	}
}

func partitionerFn(partition func(string) func(r *kgo.Record, n int) int, requireConsistency bool) kgo.Partitioner {
	return &basicPartitioner{fn: partition, requireConsistency: requireConsistency}
}

type (
	basicPartitioner struct {
		fn                 func(string) func(*kgo.Record, int) int
		requireConsistency bool
	}

	basicTopicPartitioner struct {
		fn                 func(*kgo.Record, int) int
		requireConsistency bool
	}
)

func (b *basicPartitioner) ForTopic(t string) kgo.TopicPartitioner {
	return &basicTopicPartitioner{fn: b.fn(t), requireConsistency: b.requireConsistency}
}

func (b *basicTopicPartitioner) RequiresConsistency(*kgo.Record) bool { return !b.requireConsistency }
func (b *basicTopicPartitioner) Partition(r *kgo.Record, n int) int   { return b.fn(r, n) }
