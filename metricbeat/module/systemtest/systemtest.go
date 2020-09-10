package systemtest

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"

	"github.com/elastic/beats/v7/metricbeat/mb/adapter/mba"
	"github.com/elastic/beats/v7/metricbeat/module/system"
	"github.com/elastic/beats/v7/metricbeat/module/system/core"
	"github.com/elastic/beats/v7/metricbeat/module/system/cpu"
	"github.com/elastic/beats/v7/metricbeat/module/system/diskio"
	"github.com/elastic/beats/v7/metricbeat/module/system/entropy"
	"github.com/elastic/beats/v7/metricbeat/module/system/filesystem"
	"github.com/elastic/beats/v7/metricbeat/module/system/fsstat"
	"github.com/elastic/beats/v7/metricbeat/module/system/load"
	"github.com/elastic/beats/v7/metricbeat/module/system/memory"
	"github.com/elastic/beats/v7/metricbeat/module/system/network"
	"github.com/elastic/beats/v7/metricbeat/module/system/network_summary"
	"github.com/elastic/beats/v7/metricbeat/module/system/process"
	"github.com/elastic/beats/v7/metricbeat/module/system/process_summary"
	"github.com/elastic/beats/v7/metricbeat/module/system/raid"
	"github.com/elastic/beats/v7/metricbeat/module/system/service"
	"github.com/elastic/beats/v7/metricbeat/module/system/socket"
	"github.com/elastic/beats/v7/metricbeat/module/system/socket_summary"
	"github.com/elastic/beats/v7/metricbeat/module/system/uptime"
	"github.com/elastic/beats/v7/metricbeat/module/system/users"
)

func Inputs() []v2.Plugin {
	// The system module registers a global CLI flag. The flag will not be
	// populated, so we have to workaround the initialization by replacing HostFS
	// pointer.
	// TODO: better allow HostFS to be configured using settings
	// system.HostFS = &hostFS

	systemModule := &mba.ModuleAdapter{Name: "system", Factory: system.NewModule}
	return []v2.Plugin{
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.cpu", "cpu", cpu.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.core", "core", core.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.diskio", "diskio", diskio.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.entropy", "entropy", entropy.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.filesystem", "filesystem", filesystem.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.fsstat", "fsstat", fsstat.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.load", "load", load.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.memory", "memory", memory.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.network", "network", network.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.network_summary", "network_summary", network_summary.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.process", "process", process.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.process_summary", "process_summary", process_summary.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.raid", "raid", raid.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.service", "service", service.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.socket", "socket", socket.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.
				MetricsetInput("system.socket_summary", "socket_summary", socket_summary.New).
				WithNamespace("system.socket.summary"),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.uptime", "uptime", uptime.New),
		),
		mba.Plugin(feature.Stable, false,
			systemModule.MetricsetInput("system.users", "users", users.New),
		),
	}
}
