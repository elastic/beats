package beater

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/heartbeat/watcher"
)

type MonitorManager struct {
	monitors   []Monitor
	jobControl JobControl
	client     publisher.Client
}

type Monitor struct {
	manager *MonitorManager
	watcher watcher.Watch

	name       string
	uniqueName string
	factory    monitors.Factory
	config     *common.Config

	active map[string]MonitorTask
}

type MonitorTask struct {
	job      monitors.Job
	schedule scheduler.Schedule
	cancel   JobCanceller
}

type JobControl interface {
	Add(sched scheduler.Schedule, name string, f scheduler.TaskFunc) func() error
}

type JobCanceller func() error

var defaultFilePollInterval = 5 * time.Second

func newMonitorManager(
	client publisher.Client,
	jobControl JobControl,
	registry *monitors.Registrar,
	configs []*common.Config,
) (*MonitorManager, error) {
	type watchConfig struct {
		Path string        `config:"watch.poll_file.path"`
		Poll time.Duration `config:"watch.poll_file.interval" validate:"min=1"`
	}
	defaultWatchConfig := watchConfig{
		Poll: defaultFilePollInterval,
	}

	m := &MonitorManager{
		client:     client,
		jobControl: jobControl,
	}

	if len(configs) == 0 {
		return nil, errors.New("no monitor configured")
	}

	// check monitors exist
	for _, config := range configs {
		plugin := struct {
			Type    string `config:"type" validate:"required"`
			Enabled bool   `config:"enabled"`
		}{
			Enabled: true,
		}

		if err := config.Unpack(&plugin); err != nil {
			return nil, err
		}

		if !plugin.Enabled {
			continue
		}

		info, found := registry.Query(plugin.Type)
		if !found {
			return nil, fmt.Errorf("Monitor type '%v' does not exist", plugin.Type)
		}
		logp.Info("Select (%v) monitor %v", info.Type, info.Name)

		factory := registry.GetFactory(plugin.Type)
		if factory == nil {
			return nil, fmt.Errorf("Found non-runnable monitor %v", plugin.Type)
		}

		m.monitors = append(m.monitors, Monitor{
			manager: m,
			name:    info.Name,
			factory: factory,
			config:  config,
			active:  map[string]MonitorTask{},
		})
	}

	// load watcher configs
	watchConfigs := make([]watchConfig, len(m.monitors))
	for i, monitor := range m.monitors {
		watchConfigs[i] = defaultWatchConfig
		if err := monitor.config.Unpack(&watchConfigs[i]); err != nil {
			return nil, err
		}
	}

	// load initial monitors
	for _, monitor := range m.monitors {
		err := monitor.Update([]*common.Config{monitor.config})
		if err != nil {
			logp.Err("failed to load monitor tasks: %v", err)
		}
	}

	// start monitor resource watchers if configured (will drop registered monitoring tasks and install new one if resource is available)
	for i := range m.monitors {
		monitor := &m.monitors[i]
		path := watchConfigs[i].Path
		if path == "" {
			continue
		}

		poll := watchConfigs[i].Poll
		monitor.watcher, _ = watcher.NewFilePoller(path, poll, createWatchUpdater(monitor))
	}

	return m, nil
}

func (m *Monitor) Update(configs []*common.Config) error {
	all := map[string]MonitorTask{}
	for i, upd := range configs {
		config, err := common.MergeConfigs(m.config, upd)
		if err != nil {
			logp.Err("Failed merging monitor config with updates: %v", err)
			return err
		}

		shared := struct {
			Schedule *schedule.Schedule `config:"schedule" validate:"required"`
		}{}
		if err := config.Unpack(&shared); err != nil {
			logp.Err("Failed parsing job schedule: ", err)
			return err
		}

		jobs, err := m.factory(config)
		if err != nil {
			err = fmt.Errorf("%v when initializing monitor %v(%v)", err, m.name, i)
			return err
		}

		for _, job := range jobs {
			all[job.Name()] = MonitorTask{
				job:      job,
				schedule: shared.Schedule,
			}
		}
	}

	// stop all active jobs
	for _, job := range m.active {
		job.cancel()
	}
	m.active = map[string]MonitorTask{}

	// start new and reconfigured tasks
	for name, t := range all {
		job := createJob(m.manager.client, name, t.job)
		t.cancel = m.manager.jobControl.Add(t.schedule, name, job)
		m.active[name] = t
	}

	return nil
}

func createWatchUpdater(monitor *Monitor) func(content []byte) {
	return func(content []byte) {
		defer logp.Recover("Failed applying monitor watch")

		// read multiple json objects from content
		dec := json.NewDecoder(bytes.NewBuffer(content))
		var configs []*common.Config
		for dec.More() {
			var obj map[string]interface{}
			err := dec.Decode(&obj)
			if err != nil {
				logp.Err("Failed parsing json object: %v", err)
				return
			}

			logp.Info("load watch object: %v", obj)

			cfg, err := common.NewConfigFrom(obj)
			if err != nil {
				logp.Err("Failed normalizing json input: %v", err)
				return
			}

			configs = append(configs, cfg)
		}

		// apply read configurations
		if err := monitor.Update(configs); err != nil {
			logp.Err("Failed applying configuration: %v", err)
		}
	}
}

func createJob(
	client publisher.Client,
	name string,
	r monitors.Job,
) scheduler.TaskFunc {
	return createJobTask(client, name, r)
}

func createJobTask(
	client publisher.Client,
	name string,
	r monitors.TaskRunner,
) scheduler.TaskFunc {
	return func() []scheduler.TaskFunc {
		event, next, err := r.Run()
		if err != nil {
			logp.Err("Job %v failed with: ", err)
		}

		if event != nil {
			event["monitor"] = name
			client.PublishEvent(event)
		}

		if len(next) == 0 {
			return nil
		}

		cont := make([]scheduler.TaskFunc, len(next))
		for i, n := range next {
			cont[i] = createJobTask(client, name, n)
		}
		return cont
	}
}
