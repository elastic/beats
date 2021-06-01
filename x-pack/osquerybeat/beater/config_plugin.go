// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	configName           = "osq_config"
	osqueryConfigFile    = "osquery.conf"
	scheduleSplayPercent = 10
)

type QueryConfig struct {
	Name     string
	Query    string
	Interval int
	Platform string
	Version  string
}

type ConfigPlugin struct {
	dataPath string

	log *logp.Logger

	mx sync.RWMutex

	newQueryConfigs []QueryConfig

	dirty    bool
	schedule map[string]osqueryConfigInfo
}

func NewConfigPlugin(log *logp.Logger, dataPath string) *ConfigPlugin {
	p := &ConfigPlugin{
		dataPath: dataPath,
		log:      log.With("ctx", "config"),
	}

	// load queries config from the file if it was previously persisted
	// the errors are logged
	p.loadConfig()
	return p
}

func (p *ConfigPlugin) Set(configs []QueryConfig) {
	p.mx.Lock()
	defer p.mx.Unlock()

	p.newQueryConfigs = configs
	p.dirty = true
}

func (p *ConfigPlugin) Count() int {
	return len(p.schedule)
}

func (p *ConfigPlugin) ResolveName(name string) (sql string, ok bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	sc, ok := p.schedule[name]

	return sc.Query, ok
}

func (p *ConfigPlugin) GenerateConfig(ctx context.Context) (map[string]string, error) {
	p.log.Debug("configPlugin GenerateConfig is called")

	p.mx.Lock()
	defer p.mx.Unlock()

	c, err := p.render()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		configName: c,
	}, nil
}

type osqueryConfigInfo struct {
	Query    string `json:"query"`
	Interval int    `json:"interval,omitempty"`
	Platform string `json:"platform,omitempty"`
	Version  string `json:"version,omitempty"`
	Snapshot bool   `json:"snapshot,omitempty"`
}

type osqueryConfig struct {
	Options  map[string]interface{}       `json:"options"`
	Schedule map[string]osqueryConfigInfo `json:"schedule,omitempty"`
}

func newOsqueryConfig(schedule map[string]osqueryConfigInfo) osqueryConfig {
	return osqueryConfig{
		Options: map[string]interface{}{
			"schedule_splay_percent": scheduleSplayPercent,
		},
		Schedule: schedule,
	}
}

func (c osqueryConfig) render() ([]byte, error) {
	return json.MarshalIndent(c, "", "    ")
}

func (p *ConfigPlugin) render() (string, error) {
	save := false
	if p.dirty {
		save = true
		p.schedule = make(map[string]osqueryConfigInfo)

		for _, qc := range p.newQueryConfigs {
			p.schedule[qc.Name] = osqueryConfigInfo{
				Query:    qc.Query,
				Interval: qc.Interval,
				Platform: qc.Platform,
				Version:  qc.Version,
				Snapshot: true, // enforce snapshot for all queries
			}
		}
		p.dirty = false
	}

	raw, err := newOsqueryConfig(p.schedule).render()
	if err != nil {
		return "", err
	}
	if save {
		err := p.saveConfig(p.getConfigFilePath(), raw)
		if err != nil {
			p.log.Errorf("failed to persist config file: %v", err)
			return "", err
		}
	}
	return string(raw), err
}

func (p *ConfigPlugin) loadConfig() {
	p.log.Debug("try load config from file")
	f, err := os.Open(p.getConfigFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			p.log.Debug("config file is not found")
			return
		}
		p.log.Errorf("failed to load the config file: %v", err)
		return
	}
	defer f.Close()

	var c osqueryConfig
	d := json.NewDecoder(f)
	err = d.Decode(&c)
	if err != nil {
		p.log.Errorf("failed to decode config file: %v", err)
		return
	}

	sz := len(c.Schedule)
	if sz == 0 {
		return
	}

	p.newQueryConfigs = make([]QueryConfig, 0, sz)
	p.dirty = true

	for name, qi := range c.Schedule {
		p.newQueryConfigs = append(p.newQueryConfigs, QueryConfig{
			Name:     name,
			Query:    qi.Query,
			Interval: qi.Interval,
			Platform: qi.Platform,
			Version:  qi.Version,
		})
	}
	return
}

func (p *ConfigPlugin) getConfigFilePath() string {
	return filepath.Join(p.dataPath, osqueryConfigFile)
}

func (p *ConfigPlugin) saveConfig(fp string, data []byte) error {

	tmpFilePath := p.getConfigFilePath() + ".tmp"

	err := ioutil.WriteFile(tmpFilePath, data, 0644)
	if err != nil {
		return err
	}

	defer func() {
		os.Remove(tmpFilePath)
	}()

	err = os.Rename(tmpFilePath, p.getConfigFilePath())
	if err != nil {
		return err
	}

	return nil
}
