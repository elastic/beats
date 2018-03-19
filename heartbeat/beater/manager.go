package beater

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/heartbeat/watcher"
)

type monitorManager struct {
	monitors   []monitor
	jobControl jobControl
}

type monitor struct {
	manager *monitorManager
	watcher watcher.Watch

	name       string
	uniqueName string
	factory    monitors.Factory
	config     *common.Config

	active map[string]monitorTask

	pipeline beat.Pipeline
}

type monitorTask struct {
	job    monitors.Job
	cancel jobCanceller

	config monitorTaskConfig
}

type monitorTaskConfig struct {
	Name     string             `config:"name"`
	Type     string             `config:"type"`
	Schedule *schedule.Schedule `config:"schedule" validate:"required"`

	// Fields and tags to add to monitor.
	EventMetadata common.EventMetadata    `config:",inline"`
	Processors    processors.PluginConfig `config:"processors"`
}

type jobControl interface {
	Add(sched scheduler.Schedule, name string, f scheduler.TaskFunc) func() error
}

type jobCanceller func() error

var defaultFilePollInterval = 5 * time.Second

const defaultEventType = "monitor"

func newMonitorManager(
	pipeline beat.Pipeline,
	jobControl jobControl,
	registry *monitors.Registrar,
	configs []*common.Config,
) (*monitorManager, error) {
	type watchConfig struct {
		Path string        `config:"watch.poll_file.path"`
		Poll time.Duration `config:"watch.poll_file.interval" validate:"min=1"`
	}
	defaultWatchConfig := watchConfig{
		Poll: defaultFilePollInterval,
	}

	m := &monitorManager{
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

		m.monitors = append(m.monitors, monitor{
			manager:  m,
			name:     info.Name,
			factory:  factory,
			config:   config,
			active:   map[string]monitorTask{},
			pipeline: pipeline,
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

func (m *monitorManager) Stop() {
	for _, m := range m.monitors {
		m.Close()
	}
}

func (m *monitor) Update(configs []*common.Config) error {
	all := map[string]monitorTask{}
	for i, upd := range configs {
		config, err := common.MergeConfigs(m.config, upd)
		if err != nil {
			logp.Err("Failed merging monitor config with updates: %v", err)
			return err
		}

		shared := monitorTaskConfig{}
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
			all[job.Name()] = monitorTask{
				job:    job,
				config: shared,
			}
		}
	}

	// stop all active jobs
	for _, job := range m.active {
		job.cancel()
	}
	m.active = map[string]monitorTask{}

	// start new and reconfigured tasks
	for id, t := range all {
		processors, err := processors.New(t.config.Processors)
		if err != nil {
			logp.Critical("Fail to load monitor processors: %v", err)
			continue
		}

		// create connection per monitorTask
		client, err := m.pipeline.ConnectWith(beat.ClientConfig{
			EventMetadata: t.config.EventMetadata,
			Processor:     processors,
		})
		if err != nil {
			logp.Critical("Fail to connect job '%v' to publisher pipeline: %v", id, err)
			continue
		}

		job := t.createJob(client)
		jobCancel := m.manager.jobControl.Add(t.config.Schedule, id, job)
		t.cancel = func() error {
			client.Close()
			return jobCancel()
		}

		m.active[id] = t
	}

	return nil
}

func (m *monitor) Close() {
	for _, mt := range m.active {
		mt.cancel()
	}
}

func createWatchUpdater(monitor *monitor) func(content []byte) {
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

func (m *monitorTask) createJob(client beat.Client) scheduler.TaskFunc {
	name := m.config.Name
	if name == "" {
		name = m.config.Type
	}

	meta := common.MapStr{
		"monitor": common.MapStr{
			"name": name,
			"type": m.config.Type,
		},
	}
	return m.prepareSchedulerJob(client, meta, m.job.Run)
}

func (m *monitorTask) prepareSchedulerJob(client beat.Client, meta common.MapStr, run monitors.JobRunner) scheduler.TaskFunc {
	return func() []scheduler.TaskFunc {
		event, next, err := run()
		if err != nil {
			logp.Err("Job %v failed with: ", err)
		}

		if event.Fields != nil {
			event.Fields.DeepUpdate(meta)
			if _, exists := event.Fields["type"]; !exists {
				event.Fields["type"] = defaultEventType
			}

			client.Publish(event)
		}

		if len(next) == 0 {
			return nil
		}

		cont := make([]scheduler.TaskFunc, len(next))
		for i, n := range next {
			cont[i] = m.prepareSchedulerJob(client, meta, n)
		}
		return cont
	}
}
