// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package remote_write

import (
	"math"
	"strconv"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
	xcollector "github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus/collector"
)

type histogram struct {
	timestamp  time.Time
	buckets    []*dto.Bucket
	labels     common.MapStr
	metricName string
}

func remoteWriteEventsGeneratorFactory(base mb.BaseMetricSet) (remote_write.RemoteWriteEventsGenerator, error) {
	config := config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.UseTypes {
		// use a counter cache with a timeout of 5x the period, as a safe value
		// to make sure that all counters are available between fetches
		counters := xcollector.NewCounterCache(base.Module().Config().Period * 5)

		g := RemoteWriteTypedGenerator{
			CounterCache: counters,
			RateCounters: config.RateCounters,
		}

		return &g, nil
	}

	return remote_write.DefaultRemoteWriteEventsGeneratorFactory(base)
}

type RemoteWriteTypedGenerator struct {
	CounterCache xcollector.CounterCache
	RateCounters bool
}

func (g *RemoteWriteTypedGenerator) Start() {
	cfgwarn.Beta("Prometheus 'use_types' setting is beta")

	if g.RateCounters {
		cfgwarn.Experimental("Prometheus 'rate_counters' setting is experimental")
	}

	g.CounterCache.Start()
}

func (g *RemoteWriteTypedGenerator) Stop() {
	logp.Debug("prometheus.remote_write.cache", "stopping CounterCache")
	g.CounterCache.Stop()
}

func (g RemoteWriteTypedGenerator) GenerateEvents(metrics model.Samples) map[string]mb.Event {
	var data common.MapStr
	var histograms map[string]histogram
	eventList := map[string]mb.Event{}

	for _, metric := range metrics {
		labels := common.MapStr{}

		if metric == nil {
			continue
		}
		name := string(metric.Metric["__name__"])
		delete(metric.Metric, "__name__")

		for k, v := range metric.Metric {
			labels[string(k)] = v
		}

		promType := findType(name, labels)
		val := float64(metric.Value)
		if !math.IsNaN(val) && !math.IsInf(val, 0) {
			// join metrics with same labels in a single event
			labelsHash := labels.String()
			if _, ok := eventList[labelsHash]; !ok {
				eventList[labelsHash] = mb.Event{
					ModuleFields: common.MapStr{
						"metrics": common.MapStr{},
					},
				}

				// Add labels
				if len(labels) > 0 {
					eventList[labelsHash].ModuleFields["labels"] = labels
				}
			}

			e := eventList[labelsHash]
			e.Timestamp = metric.Timestamp.Time()
			switch promType {
			case "counter_float":
				data = common.MapStr{
					name: g.rateCounterFloat64(name, labels, val),
				}
			case "counter_int":
				//events = append(events, collector.PromEvent{
				//	Data: common.MapStr{
				//		name: g.rateCounterUint64(name, labels, uint64(val)),
				//	},
				//	Labels: labels,
				//	Timestamp: metric.Timestamp.Time(),
				//})
				data = common.MapStr{
					name: g.rateCounterUint64(name, labels, uint64(val)),
				}
			case "other":
				//events = append(events, collector.PromEvent{
				//	Data: common.MapStr{
				//		name: common.MapStr{
				//			"value": val,
				//		},
				//	},
				//	Labels: labels,
				//	Timestamp: metric.Timestamp.Time(),
				//})
				data = common.MapStr{
					name: common.MapStr{
						"value": val,
					},
				}
			case "histogram":
				labelsClone := labels.Clone()
				labelsClone.Delete("le")

				histKey := name + labelsClone.String()

				le, _ := labels.GetValue("le")
				upperBound := le.(string)

				bucket, err := strconv.ParseFloat(upperBound, 64)
				if err != nil {
					continue
				}
				v := uint64(val)
				b := &dto.Bucket{
					CumulativeCount: &v,
					UpperBound:      &bucket,
				}
				hist, ok := histograms[histKey]
				if !ok {
					hist = histogram{}
				}
				hist.buckets = append(hist.buckets, b)
				hist.timestamp = metric.Timestamp.Time()
				hist.labels = labels
				hist.metricName = name
				continue
			}
			e.ModuleFields["metrics"].(common.MapStr).Update(data)
		}
	}

	g.processPromHistograms(eventList, histograms)
	return eventList
}

// rateCounterUint64 fills a counter value and optionally adds the rate if rate_counters is enabled
func (g *RemoteWriteTypedGenerator) rateCounterUint64(name string, labels common.MapStr, value uint64) common.MapStr {
	d := common.MapStr{
		"counter": value,
	}

	if g.RateCounters {
		d["rate"], _ = g.CounterCache.RateUint64(name+labels.String(), value)
	}

	return d
}

// rateCounterFloat64 fills a counter value and optionally adds the rate if rate_counters is enabled
func (g *RemoteWriteTypedGenerator) rateCounterFloat64(name string, labels common.MapStr, value float64) common.MapStr {
	d := common.MapStr{
		"counter": value,
	}

	if g.RateCounters {
		d["rate"], _ = g.CounterCache.RateFloat64(name+labels.String(), value)
	}

	return d
}

func (g *RemoteWriteTypedGenerator) processPromHistograms(eventList map[string]mb.Event, histograms map[string]histogram) {
	for name, histogram := range histograms {
		labelsHash := histogram.labels.String()
		if _, ok := eventList[labelsHash]; !ok {
			eventList[labelsHash] = mb.Event{
				ModuleFields: common.MapStr{
					"metrics": common.MapStr{},
				},
			}

			// Add labels
			if len(histogram.labels) > 0 {
				eventList[labelsHash].ModuleFields["labels"] = histogram.labels
			}
		}

		e := eventList[labelsHash]
		e.Timestamp = histogram.timestamp


		hist := dto.Histogram{
			Bucket: histogram.buckets,
		}

		//events = append(events, collector.PromEvent{
		//	Data: common.MapStr{
		//		name: common.MapStr{
		//			"histogram": xcollector.PromHistogramToES(g.CounterCache, histogram.metricName, histogram.labels, &hist),
		//		},
		//	},
		//	Labels:    histogram.labels,
		//	Timestamp: histogram.timestamp,
		//})

		data := common.MapStr{
			name: common.MapStr{
				"histogram": xcollector.PromHistogramToES(g.CounterCache, histogram.metricName, histogram.labels, &hist),
			},
		}
		e.ModuleFields["metrics"].(common.MapStr).Update(data)
	}
}

func findType(metricName string, labels common.MapStr) string {
	leLabel := false
	if _, ok := labels["le"]; ok {
		leLabel = true
	}
	if strings.Contains(metricName, "_total") || strings.Contains(metricName, "_sum") {
		return "counter_float"
	} else if strings.Contains(metricName, "_sum") {
		return "counter_int"
	} else if strings.Contains(metricName, "_bucket") && leLabel {
		return "histogram"
	}
	return "other"
}
