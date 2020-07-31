package mba

import (
	"fmt"

	"github.com/elastic/go-concert/unison"
	"github.com/gofrs/uuid"
	"github.com/urso/sderr"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

type ModuleManager interface {
	ModuleName() string
	Create(base mb.BaseModule) (mb.Module, error)
	EventModifiers() []mb.EventModifier
}

type MetricsetManager struct {
	MetricsetName string
	InputName     string
	ModuleManager ModuleManager

	Factory mb.MetricSetFactory

	HostParser mb.HostParser
	Namespace  string
}

func (m MetricsetManager) WithHostParser(p mb.HostParser) MetricsetManager {
	m.HostParser = p
	return m
}

func (m MetricsetManager) WithNamespace(n string) MetricsetManager {
	m.Namespace = n
	return m
}

func (m *MetricsetManager) Init(grp unison.Group, mode v2.Mode) error { return nil }

// Creates builds a new Input instance from the given configuation, or returns
// an error if the configuation is invalid.
// The input must establish any connection for data collection yet. The Beat
// will use the Test/Run methods of the input.
func (m *MetricsetManager) Create(cfg *common.Config) (v2.Input, error) {
	var errs []error

	cfg.SetString("module", -1, m.ModuleManager.ModuleName())
	cfg.SetString("metricset", 0, m.MetricsetName)

	baseModule, err := mb.NewBaseModuleFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	module, err := m.ModuleManager.Create(baseModule)
	if err != nil {
		return nil, err
	}

	hosts := []string{""}
	if l := module.Config().Hosts; len(l) > 0 {
		hosts = l
	}

	var metricsets []mb.MetricSet
	for _, host := range hosts {
		id, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID for metricset: %w", err)
		}

		ms, err := m.createMetricSet(module, id.String(), host)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		metricsets = append(metricsets, ms)
	}

	// XXX: do we want to error the configure step, or log the error and continue with the subset of ok metricsets?
	return &metricsetInput{
		inputName:     m.InputName,
		moduleName:    m.ModuleManager.ModuleName(),
		metricsetName: m.MetricsetName,
		namespace:     m.Namespace,
		tasks:         metricsets,
		modifiers:     m.ModuleManager.EventModifiers(),
	}, sderr.WrapAll(errs, "Failed to fully initialize the input")
}

func (m *MetricsetManager) createMetricSet(module mb.Module, id, host string) (mb.MetricSet, error) {
	hd, host, err := m.parseHost(module, host)
	if err != nil {
		return nil, err
	}

	bm := mb.NewBaseMetricSet(module, id, m.MetricsetName, host, hd)
	return m.Factory(bm)
}

func (m *MetricsetManager) parseHost(module mb.Module, host string) (mb.HostData, string, error) {
	if m.HostParser == nil {
		return mb.HostData{URI: host}, host, nil
	}

	hd, err := m.HostParser(module, host)
	return hd, hd.Host, err
}
