package metric_annotations

import (
	"fmt"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	autodiscover.Registry.AddBuilder("metric.annotations", NewMetricAnnotations)
}

const (
	module     = "module"
	namespace  = "namespace"
	hosts      = "hosts"
	metricsets = "metricsets"
	period     = "period"
	timeout    = "timeout"
	ssl        = "ssl"

	default_timeout  = "3s"
	default_interval = "1m"
)

type metricAnnotations struct {
	Prefix string
}

func NewMetricAnnotations(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack metric.annotations config due to error: %v", err)
	}

	return &metricAnnotations{config.Prefix}, nil
}

func (m *metricAnnotations) CreateConfig(event bus.Event) []*common.Config {
	var config []*common.Config
	host, _ := event["host"].(string)
	if host == "" {
		return config
	}

	annotations, ok := event["annotations"].(map[string]string)
	if !ok {
		return config
	}

	container, ok := event["container"].(common.MapStr)
	if !ok {
		return config
	}

	name := builder.GetContainerName(container)

	mod := builder.GetContainerAnnotationAsString(annotations, m.Prefix, name, module)
	if mod == "" {
		return config
	}

	hsts := builder.GetContainerAnnotationAsString(annotations, m.Prefix, name, hosts)
	ns := builder.GetContainerAnnotationAsString(annotations, m.Prefix, name, namespace)
	msets := m.getMetricSets(annotations, name, mod)
	tout := m.getTimeout(annotations, name)
	ival := m.getPeriod(annotations, name)

	sslConf := builder.GetContainerAnnotationsWithPrefix(annotations, m.Prefix, name, ssl)

	moduleConfig := common.MapStr{
		"module":     mod,
		"metricsets": msets,
		"hosts":      hsts,
		"timeout":    tout,
		"period":     ival,
		"enabled":    true,
	}

	if ns != "" {
		moduleConfig["namespace"] = ns
	}

	for k, v := range sslConf {
		moduleConfig.Put(k, v)
	}
	logp.Debug("metric.annotations", "generated config: %v", moduleConfig.String())

	// Create config object
	cfg, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		logp.Debug("metric.annotations", "config merge failed with error: %v", err)
	}
	config = append(config, cfg)

	// Apply information in event to the template to generate the final config
	// This especially helps in a scenario where endpoints are configured as:
	// co.elastic.metrics/hosts= "${data.host}:9090"
	config = template.ApplyConfigTemplate(event, config)
	return config
}

func (m *metricAnnotations) getMetricSets(annotations map[string]string, container, module string) []string {
	var msets []string
	msets = builder.GetContainerAnnotationsAsList(annotations, m.Prefix, container, metricsets)

	if len(msets) == 0 {
		// Special handling for prometheus as most use cases rely on exporters/instrumentation.
		// Prometheus stats can be explicitly configured if need be.
		if module == "prometheus" {
			return []string{"collector"}
		} else {
			msets = mb.Registry.MetricSets(module)
		}
	}
	return msets
}

func (m *metricAnnotations) getPeriod(annotations map[string]string, container string) string {
	if ival := builder.GetContainerAnnotationAsString(annotations, m.Prefix, container, period); ival != "" {
		return ival
	} else {
		return default_interval
	}
}

func (m *metricAnnotations) getTimeout(annotations map[string]string, container string) string {
	if tout := builder.GetContainerAnnotationAsString(annotations, m.Prefix, container, timeout); tout != "" {
		return tout
	} else {
		return default_timeout
	}
}
