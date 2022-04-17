// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"bytes"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/metricbeat/helper/server"
)

var errInvalidPacket = errors.New("invalid statsd packet")

type metricProcessor struct {
	registry      *registry
	reservoirSize int
}

type statsdMetric struct {
	name       string
	metricType string
	sampleRate string
	value      string
	tags       map[string]string
}

func splitTags(rawTags []byte, kvSep []byte) map[string]string {
	tags := map[string]string{}
	for _, kv := range bytes.Split(rawTags, []byte(",")) {
		kvSplit := bytes.SplitN(kv, kvSep, 2)
		if len(kvSplit) != 2 {
			logger.Warnf("could not parse tags")
			continue
		}
		tags[string(kvSplit[0])] = string(kvSplit[1])
	}
	return tags
}

func parseSingle(b []byte) (statsdMetric, error) {
	// format: <metric name>:<value>|<type>[|@samplerate][|#<k>:<v>,<k>:<v>]
	// alternative: <metric name>[,<k>=<v>,<k>=<v>]:<value>|<type>[|@samplerate]
	s := statsdMetric{}

	parts := bytes.SplitN(b, []byte("|"), 4)
	if len(parts) < 2 {
		return s, errInvalidPacket
	}

	if len(parts) > 2 && len(parts[2]) > 0 && parts[2][0] == '@' {
		s.sampleRate = string(parts[2][1:])

		if len(parts) > 3 {
			parts = [][]byte{parts[0], parts[1], parts[3]}
		} else {
			parts = [][]byte{parts[0], parts[1]}
		}
	}

	if len(parts) > 2 && len(parts[2]) > 0 && parts[2][0] == '#' {
		s.tags = splitTags(parts[2][1:], []byte(":"))
	}

	nameSplit := bytes.SplitN(parts[0], []byte{':'}, 2)
	if len(nameSplit) != 2 {
		return s, errInvalidPacket
	}

	nameTagsSplit := bytes.SplitN(nameSplit[0], []byte(","), 2)
	s.name = string(nameTagsSplit[0])
	if len(nameTagsSplit) > 1 {
		s.tags = splitTags(nameTagsSplit[1], []byte("="))
	}

	s.value = string(nameSplit[1])
	s.metricType = string(parts[1])

	return s, nil
}

// parse will parse a statsd metric into its components
func parse(b []byte) ([]statsdMetric, error) {
	metrics := []statsdMetric{}
	for _, rawMetric := range bytes.Split(b, []byte("\n")) {
		if len(rawMetric) > 0 {
			metric, err := parseSingle(rawMetric)
			if err != nil {
				return metrics, err
			}
			metrics = append(metrics, metric)
		}
	}
	return metrics, nil
}

func eventMapping(metricName string, metricValue interface{}, metricSetFields common.MapStr, mappings map[string]StatsdMapping) {
	if len(mappings) == 0 {
		metricSetFields[common.DeDot(metricName)] = metricValue
		return
	}

	for _, mapping := range mappings {
		// The metricname match the one with no labels in mappings
		// Let's insert it dedotted and continue
		if metricName == mapping.Metric {
			metricSetFields[mapping.Value.Field] = metricValue
			return
		}

		res := mapping.regex.FindStringSubmatch(metricName)

		// Not all labels match
		// Skip and continue to next mapping
		if len(res) != (len(mapping.Labels) + 1) {
			logger.Debugf("not all labels match in statsd.mapping, skipped")
			continue
		}

		// Let's add the metric set fields from labels
		names := mapping.regex.SubexpNames()
		for i, _ := range res {
			for _, label := range mapping.Labels {
				if label.Attr != names[i] {
					continue
				}

				metricSetFields[label.Field] = res[i]
			}
		}

		// Let's add the metric with the value field
		metricSetFields[mapping.Value.Field] = metricValue
	}

	return
}

func newMetricProcessor(ttl time.Duration) *metricProcessor {
	return &metricProcessor{
		registry: &registry{metrics: map[string]map[string]*metric{}, ttl: ttl},
	}
}

func (p *metricProcessor) processSingle(m statsdMetric) error {
	if len(m.value) < 1 {
		return nil
	}

	// parse sample rate. Only applicable for timers and counters
	var sampleRate float64
	if m.sampleRate == "" {
		sampleRate = 1.0
	} else {
		var err error
		sampleRate, err = strconv.ParseFloat(m.sampleRate, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to process metric `%s` sample rate `%s`", m.name, m.sampleRate)
		}
		if sampleRate <= 0.0 {
			return errors.Errorf("sample rate of 0.0 is invalid for metric `%s`", m.name)
		}
	}

	switch m.metricType {
	case "c":
		c := p.registry.GetOrNewCounter(m.name, m.tags)
		v, err := strconv.ParseInt(m.value, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to process counter `%s` with value `%s`", m.name, m.value)
		}
		// apply sample rate
		v = int64(float64(v) * (1.0 / sampleRate))
		c.Inc(v)
	case "g":
		c := p.registry.GetOrNewGauge64(m.name, m.tags)
		v, err := strconv.ParseFloat(m.value, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to process gauge `%s` with value `%s`", m.name, m.value)
		}

		// inc/dec or set
		if m.value[0] == '+' || m.value[0] == '-' {
			c.Inc(v)
		} else {
			c.Set(v)
		}
	case "ms":
		c := p.registry.GetOrNewTimer(m.name, m.tags)
		v, err := strconv.ParseFloat(m.value, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to process timer `%s` with value `%s`", m.name, m.value)
		}
		c.SampledUpdate(time.Duration(v), sampleRate)
	case "h": // TODO: can these be floats?
		c := p.registry.GetOrNewHistogram(m.name, m.tags)
		v, err := strconv.ParseInt(m.value, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to process histogram `%s` with value `%s`", m.name, m.value)
		}
		c.Update(v)
	case "s":
		c := p.registry.GetOrNewSet(m.name, m.tags)
		c.Add(m.value)
	default:
		logp.NewLogger("statsd").Debugf("metric type `%s` is not supported", m.metricType)
	}
	return nil
}

func (p *metricProcessor) Process(event server.Event) error {
	bytesRaw, ok := event.GetEvent()[server.EventDataKey]
	if !ok {
		return errors.New("unable to retrieve event bytes")
	}

	b, _ := bytesRaw.([]byte)
	if len(b) == 0 {
		return errors.New("packet has no data")
	}

	metrics, err := parse(b)
	if err != nil {
		return err
	}

	for _, m := range metrics {
		if err := p.processSingle(m); err != nil {
			return err
		}
	}

	return nil
}

func (p *metricProcessor) GetAll() []metricsGroup {
	return p.registry.GetAll()
}
