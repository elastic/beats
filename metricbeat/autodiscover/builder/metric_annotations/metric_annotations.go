package metric_annotations

import (
	"fmt"

	"strings"

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
	Key string
}

// Build a new metrics annotation builder
func NewMetricAnnotations(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack metric.annotations config due to error: %v", err)
	}

	return &metricAnnotations{config.Key}, nil
}

// Create configs based on hints passed from providers
func (m *metricAnnotations) CreateConfig(event bus.Event) []*common.Config {
	var config []*common.Config
	host, _ := event["host"].(string)
	if host == "" {
		return config
	}

	port, _ := event["port"].(int64)

	hints, ok := event["hints"].(common.MapStr)
	if !ok {
		return config
	}

	mod := m.getModule(hints)
	if mod == "" {
		return config
	}

	hsts := m.getHostsWithPort(hints, port)
	ns := m.getNamespace(hints)
	msets := m.getMetricSets(hints, mod)
	tout := m.getTimeout(hints)
	ival := m.getPeriod(hints)
	sslConf := m.getSSLConfig(hints)

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
	logp.Debug("metric.hints", "generated config: %v", moduleConfig.String())

	// Create config object
	cfg, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		logp.Debug("metric.hints", "config merge failed with error: %v", err)
	}
	config = append(config, cfg)

	// Apply information in event to the template to generate the final config
	// This especially helps in a scenario where endpoints are configured as:
	// co.elastic.metrics/hosts= "${data.host}:9090"
	config = template.ApplyConfigTemplate(event, config)
	return config
}

func (m *metricAnnotations) getModule(hints common.MapStr) string {
	return builder.GetHintString(hints, m.Key, module)
}

func (m *metricAnnotations) getMetricSets(hints common.MapStr, module string) []string {
	var msets []string
	msets = builder.GetHintAsList(hints, m.Key, metricsets)

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

func (m *metricAnnotations) getHostsWithPort(hints common.MapStr, port int64) []string {
	var result []string
	thosts := builder.GetHintAsList(hints, m.Key, hosts)

	// Only pick hosts that have ${data.port} or the port on current event. This will make
	// sure that incorrect meta mapping doesn't happen
	for _, h := range thosts {
		if strings.Contains(h, "data.port") || strings.Contains(h, fmt.Sprintf("%d", port)) {
			result = append(result, h)
		}
	}

	return result
}

func (m *metricAnnotations) getNamespace(hints common.MapStr) string {
	return builder.GetHintString(hints, m.Key, namespace)
}

func (m *metricAnnotations) getPeriod(hints common.MapStr) string {
	if ival := builder.GetHintString(hints, m.Key, period); ival != "" {
		return ival
	} else {
		return default_interval
	}
}

func (m *metricAnnotations) getTimeout(hints common.MapStr) string {
	if tout := builder.GetHintString(hints, m.Key, timeout); tout != "" {
		return tout
	} else {
		return default_timeout
	}
}

func (m *metricAnnotations) getSSLConfig(hints common.MapStr) common.MapStr {
	return builder.GetHintMapStr(hints, m.Key, ssl)
}
