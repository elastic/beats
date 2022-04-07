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

package server

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

type template struct {
	Namespace string
	Delimiter string
	Parts     []string
	Tags      map[string]string
}

type metricProcessor struct {
	templates       *tree
	defaultTemplate template
	sync.RWMutex
}

func NewMetricProcessor(templates []TemplateConfig, defaultTemplate TemplateConfig) *metricProcessor {
	templateTree := NewTree(getTemplateFromConfig(defaultTemplate))
	for _, t := range templates {
		templateTree.Insert(t.Filter, getTemplateFromConfig(t))
	}

	return &metricProcessor{
		templates:       templateTree,
		defaultTemplate: getTemplateFromConfig(defaultTemplate),
	}
}

func getTemplateFromConfig(config TemplateConfig) template {
	return template{
		Namespace: config.Namespace,
		Tags:      config.Tags,
		Delimiter: config.Delimiter,
		Parts:     strings.Split(config.Template, "."),
	}
}

func (m *metricProcessor) AddTemplate(t TemplateConfig) {
	m.Lock()
	template := getTemplateFromConfig(t)
	m.templates.Insert(t.Filter, template)
	m.Unlock()
}

func (m *metricProcessor) RemoveTemplate(template TemplateConfig) {
	m.Lock()
	m.templates.Delete(template.Filter)
	m.Unlock()
}

func (m *metricProcessor) Process(message string) (common.MapStr, error) {
	metric, timestamp, value, err := m.splitMetric(message)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(metric, ".")
	t := m.FindTemplate(parts)

	var name, namespace string
	var tags common.MapStr
	if t == nil {
		name, tags = m.defaultTemplate.Apply(parts)
		namespace = m.defaultTemplate.Namespace
	} else {
		name, tags = t.Apply(parts)
		namespace = t.Namespace
	}

	event := common.MapStr{
		"@timestamp":    timestamp,
		name:            value,
		mb.NamespaceKey: namespace,
	}
	if len(tags) != 0 {
		event["tag"] = tags
	}
	return event, nil
}

func (m *metricProcessor) FindTemplate(metric []string) *template {
	return m.templates.Search(metric)
}

func (m *metricProcessor) splitMetric(metric string) (string, common.Time, float64, error) {
	var metricName string
	var timestamp common.Time
	var value float64

	parts := strings.Fields(metric)
	currentTime := common.Time(time.Now())
	if len(parts) < 2 {
		return "", currentTime, 0, errors.New("Message not in expected format")
	} else {
		metricName = parts[0]
		val, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return "", currentTime, 0, errors.New("Unable to parse metric value")
		} else {
			value = val
		}
	}

	if len(parts) == 3 {
		if parts[2] == "N" {
			timestamp = currentTime
		}
		ts, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return "", currentTime, 0, errors.New("Unable to parse timestamp")
		}

		if ts != -1 {
			timestamp = common.Time(time.Unix(int64(ts), int64((ts-math.Floor(ts))*float64(time.Second))))
		} else {
			timestamp = currentTime
		}

	} else {
		timestamp = currentTime
	}

	return metricName, timestamp, value, nil
}

func (t *template) Apply(parts []string) (string, common.MapStr) {
	tags := make(common.MapStr)

	metric := make([]string, 0)
	for tagKey, tagVal := range t.Tags {
		tags[tagKey] = tagVal
	}

	tagsMap := make(map[string][]string)
	for i := 0; i < len(t.Parts); i++ {
		if t.Parts[i] == "metric" {
			metric = append(metric, parts[i])
		} else if t.Parts[i] == "metric*" {
			metric = append(metric, parts[i:]...)
		} else if t.Parts[i] != "" {
			tagsMap[t.Parts[i]] = append(tagsMap[t.Parts[i]], parts[i])
		}
	}

	for key, value := range tagsMap {
		tags[key] = strings.Join(value, t.Delimiter)
	}

	if len(metric) == 0 {
		return "", tags
	} else {
		return strings.Join(metric, t.Delimiter), tags
	}
}
