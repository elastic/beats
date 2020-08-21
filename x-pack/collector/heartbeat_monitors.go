package main

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/collector/internal/adapter/registries"

	_ "github.com/elastic/beats/v7/heartbeat/include"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/defaults"
)

func makeHeartbeatRegistry(info beat.Info, sched *scheduler.Scheduler) v2.Registry {
	factory := monitors.NewFactory(info, sched, false)
	return &registries.RunnerFactoryRegistry{
		TypeField: "type",
		Factory:   factory,
		Has:       nil,
	}
}
