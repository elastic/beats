package mba

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
)

func Plugin(stability feature.Stability, deprecated bool, mm MetricsetManager) v2.Plugin {
	return v2.Plugin{
		Name:       mm.InputName,
		Stability:  stability,
		Deprecated: deprecated,
		Manager:    &mm,
	}
}
