// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kafkapartitionerextension

import (
	"context"
	"hash/fnv"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ extension.Extension = (*kafkaPartitioner)(nil)
var _ kafkaexporter.RecordPartitionerExtension = (*kafkaPartitioner)(nil)

type kafkaPartitioner struct {
	cfg    *Config
	logger *zap.Logger
}

func (*kafkaPartitioner) Start(context.Context, component.Host) error {
	return nil
}

func (*kafkaPartitioner) Shutdown(context.Context) error {
	return nil
}

func (k *kafkaPartitioner) GetPartitioner() kgo.Partitioner {
	partitioner, err := makePartitioner(logp.NewLogger("", zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return k.logger.Core()
	})), k.cfg.PartitionerConfig)
	if err != nil {
		k.logger.Error("error creating partitioner, defaulting to sticky key partitioner with fnv32 hasher", zap.Error(err))
		return kgo.StickyKeyPartitioner(kgo.SaramaCompatHasher(fnv32a))
	}
	return partitioner
}

func fnv32a(b []byte) uint32 {
	h := fnv.New32a()
	h.Reset()
	h.Write(b)
	return h.Sum32()
}
