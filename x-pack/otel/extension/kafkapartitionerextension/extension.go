// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kafkapartitionerextension

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"

	"github.com/elastic/elastic-agent-libs/logp"
)

var _ extension.Extension = (*kafkaPartitioner)(nil)
var _ kafkaexporter.RecordPartitionerExtension = (*kafkaPartitioner)(nil)

type kafkaPartitioner struct {
	cfg         *Config
	logger      *zap.Logger
	partitioner kgo.Partitioner
}

func (k *kafkaPartitioner) Start(context.Context, component.Host) error {
	logger, err := logp.NewZapLogger(k.logger)
	if err != nil {
		return fmt.Errorf("error creating logger: %w", err)
	}
	partitioner, err := makePartitioner(logger, k.cfg.PartitionerConfig)
	if err != nil {
		return fmt.Errorf("error configuring the partitioner: %w", err)
	}
	k.partitioner = partitioner
	return nil
}

func (*kafkaPartitioner) Shutdown(context.Context) error {
	return nil
}

func (k *kafkaPartitioner) GetPartitioner() kgo.Partitioner {
	return k.partitioner
}
