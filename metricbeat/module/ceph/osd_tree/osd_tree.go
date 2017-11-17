package osd_tree

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/api/v0.1/osd/tree"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

func init() {
	if err := mb.Registry.AddMetricSet("ceph", "osd_tree", New, hostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The ceph osd_tree metricset is beta")

	http := helper.NewHTTP(base)
	http.SetHeader("Accept", "application/json")

	return &MetricSet{
		base,
		http,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		return nil, err
	}

	events, err := eventsMapping(content)
	if err != nil {
		return nil, err
	}

	return events, nil
}
