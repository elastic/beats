package spool

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/publisher/queue"
	"github.com/elastic/go-txfile"
)

func init() {
	queue.RegisterType("spool", create)
}

func create(eventer queue.Eventer, cfg *common.Config) (queue.Queue, error) {
	cfgwarn.Beta("Spooling to disk is beta")

	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	path := config.File.Path
	if path == "" {
		path = paths.Resolve(paths.Data, "spool.dat")
	}

	flushEvents := uint(0)
	if count := config.Write.FlushEvents; count > 0 {
		flushEvents = uint(count)
	}

	return NewSpool(defaultLogger(), path, Settings{
		Eventer:           eventer,
		Mode:              config.File.Permissions,
		WriteBuffer:       uint(config.Write.BufferSize),
		WriteFlushTimeout: config.Write.FlushTimeout,
		WriteFlushEvents:  flushEvents,
		ReadFlushTimeout:  config.Read.FlushTimeout,
		Codec:             config.Write.Codec,
		File: txfile.Options{
			MaxSize:  uint64(config.File.MaxSize),
			PageSize: uint32(config.File.PageSize),
			Prealloc: config.File.Prealloc,
			Readonly: false,
		},
	})
}
