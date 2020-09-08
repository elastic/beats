package lsout

import (
	"errors"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/logstash"
	beatpipe "github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/collector/internal/adapter/beatsout"
	"github.com/elastic/beats/v7/x-pack/collector/internal/publishing"
)

func Plugin(info beat.Info) publishing.Plugin {
	return publishing.Plugin{
		Name:       "console",
		Stability:  feature.Stable,
		Deprecated: false,
		Configure: func(log *logp.Logger, cfg *common.Config) (publishing.Output, error) {
			return configure(info, log, cfg)
		},
	}
}

func configure(info beat.Info, log *logp.Logger, cfg *common.Config) (publishing.Output, error) {
	// We just take over the output settings as is, but disallow users from
	// configuring the queue for the logstash output. Instead we compute an optimal sizing
	// for the memory queue and flushing policy, such that all output workers
	// can be satisfied with new batches of events immediately.
	// The total number of workers is given by the number of hosts configured times the number of `worker`.
	// For each worker we want configure `pipelining + 1` batches. One extra
	// batch in memory that can be picked up immediately once an older batch has
	// been ACKed.

	// 1. parse pipeline + queue config
	var pipeConfig beatpipe.Config
	if err := cfg.Unpack(&pipeConfig); err != nil {
		return nil, err
	}
	if pipeConfig.Queue.IsSet() {
		return nil, errors.New("`queue` setting is invalid for the Logstash output")
	}

	// 2. parse output settings
	outputSettings := struct {
		Worker       int             `config:"worker"`
		Config       logstash.Config `config:",inline"`
		FlushTimeout time.Duration   `config:"flush.timeout"`
	}{
		1,
		logstash.DefaultConfig(),
		1 * time.Second,
	}
	if err := cfg.Unpack(&outputSettings); err != nil {
		return nil, err
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return nil, err
	}

	// 3. compute memory queue settings based on output settings.
	batchesPerWorker := outputSettings.Config.Pipelining + 1
	if batchesPerWorker == 1 {
		// pipelining was disabled. We still want to keep on batch in memory while
		// we are waiting for ACK
		batchesPerWorker = 2
	}

	numHosts := len(hosts)
	batchesPerHost := batchesPerWorker * outputSettings.Worker
	activeBatches := batchesPerHost * numHosts
	preparedBatches := numHosts * outputSettings.Worker // number of batches waiting to be picked up by an output
	totalBatches := activeBatches + preparedBatches     // total number of batches prepared in memory

	batchSize := outputSettings.Config.BulkMaxSize
	totalEvents := batchSize * totalBatches

	// 4. merge computed queue settings with already parsed pipeline settings
	queueConfig := common.MustNewConfigFrom(map[string]interface{}{
		"queue.memqueue.events":           totalEvents,
		"queue.memqueue.flush.min_events": batchSize,
		"queue.memqueue.flush.timeout":    outputSettings.FlushTimeout,
	})
	if err := queueConfig.Unpack(&pipeConfig); err != nil {
		return nil, err
	}

	// 5. Create beats pipeline based output with computed pipeline + queue settings
	return beatsout.NewPipelineOutput(info, pipeConfig, "logstash", cfg), nil
}
