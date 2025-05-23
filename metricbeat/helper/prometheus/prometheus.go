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

package prometheus

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/helper/easyops"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const acceptHeader = `text/plain;version=0.0.4;q=0.5,*/*;q=0.1`

// Prometheus helper retrieves prometheus formatted metrics
type Prometheus interface {
	// GetFamilies requests metric families from prometheus endpoint and returns them
	GetFamilies() ([]*dto.MetricFamily, error)

	GetProcessedMetrics(mapping *MetricsMapping) ([]mapstr.M, error)

	ProcessMetrics(families []*dto.MetricFamily, mapping *MetricsMapping) ([]mapstr.M, error)

	ReportProcessedMetrics(mapping *MetricsMapping, r mb.ReporterV2) error
}

type prometheus struct {
	httpfetcher
	logger *logp.Logger
	parser *Parser
}

type httpfetcher interface {
	FetchResponse() (*http.Response, error)
}

// NewPrometheusClient creates new prometheus helper
func NewPrometheusClient(base mb.BaseMetricSet) (Prometheus, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	http.SetHeaderDefault("Accept", acceptHeader)
	http.SetHeaderDefault("Accept-Encoding", "gzip")
	return &prometheus{
		http,
		base.Logger(),
		NewParser(),
	}, nil
}

// GetFamilies requests metric families from prometheus endpoint and returns them
func (p *prometheus) GetFamilies() ([]*dto.MetricFamily, error) {
	var reader io.Reader

	resp, err := p.FetchResponse()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Encoding") == "gzip" {
		greader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer greader.Close()
		reader = greader
	} else {
		reader = resp.Body
	}

	bodyBytes, err := ioutil.ReadAll(reader)
	if err == nil {
		p.logger.Debug("error received from prometheus endpoint: ", string(bodyBytes))
	}

	if resp.StatusCode > 399 {
		return nil, fmt.Errorf("unexpected status code %d from server", resp.StatusCode)
	}

	families := p.parser.Parse(bodyBytes)

	return families, nil
}

// MetricsMapping defines mapping settings for Prometheus metrics, to be used with `GetProcessedMetrics`
type MetricsMapping struct {
	// Metrics translates from prometheus metric name to Metricbeat fields
	Metrics map[string]MetricMap

	// Namespace for metrics managed by this mapping
	Namespace string

	// Labels translate from prometheus label names to Metricbeat fields
	Labels map[string]LabelMap

	// ExtraFields adds the given fields to all events coming from `GetProcessedMetrics`
	ExtraFields map[string]string

	// aggregate metrics
	AggregateMetrics []easyops.AggregateMetricMap
}

func (p *prometheus) ProcessMetrics(families []*dto.MetricFamily, mapping *MetricsMapping) ([]mapstr.M, error) {
	// 创建一个map来存储所有事件数据，key是标签组合，value是事件数据
	eventsMap := map[string]mapstr.M{}
	// 存储所有info类型的指标数据
	infoMetrics := []*infoMetricData{}
	// 创建一个集合用于存储所有infoMetrics中使用的label字段
	infoMetricLabelFields := make(map[string]struct{})

	// 遍历所有指标族（metric family）
	for _, family := range families {
		// 遍历该指标族下的所有具体指标
		for _, metric := range family.GetMetric() {
			// 从映射配置中获取该指标的处理规则
			// 例如：http_requests_total 可能被映射为 "http.requests.total"
			m, ok := mapping.Metrics[family.GetName()]
			if m == nil || !ok {
				// 如果找不到映射规则，跳过该指标
				continue
			}

			// 获取指标字段名，例如 "http.requests.total"
			field := m.GetField()
			// 获取指标值，例如 100
			value := m.GetValue(metric)

			// 如果获取值失败，跳过该指标
			if value == nil {
				continue
			}

			// 配置标签处理选项
			storeAllLabels := false
			labelsLocation := ""
			var extraFields mapstr.M
			if m != nil {
				c := m.GetConfiguration()
				// 是否存储所有未映射的标签
				storeAllLabels = c.StoreNonMappedLabels
				// 未映射标签的存储位置
				labelsLocation = c.NonMappedLabelsPlacement
				// 额外的字段配置
				extraFields = c.ExtraFields
			}

			// 获取指标的所有标签
			// 例如：{"instance": "localhost:9090", "job": "prometheus", "method": "GET", "path": "/metrics"}
			allLabels := getLabels(metric)

			// 应用额外的处理选项
			for _, option := range m.GetOptions() {
				field, value, allLabels = option.Process(field, value, allLabels)
			}

			// 处理标签映射
			labels := mapstr.M{}    // 普通标签
			keyLabels := mapstr.M{} // 关键标签（用于事件分组）
			for k, v := range allLabels {
				if l, ok := mapping.Labels[k]; ok {
					if l.IsKey() {
						// 如果是关键标签，放入keyLabels
						keyLabels.Put(l.GetField(), v)
					} else {
						// 如果是普通标签，放入labels
						labels.Put(l.GetField(), v)
					}
				} else if storeAllLabels {
					// 如果配置了存储所有标签，将未映射的标签也存储起来
					labels.Put(labelsLocation+"."+k, v)
				}
			}

			// 添加额外配置的字段到标签中
			for k, v := range extraFields {
				labels.Put(k, v)
			}

			// 处理info类型的指标
			if _, ok = m.(*infoMetric); ok {
				labels.DeepUpdate(keyLabels)
				infoMetrics = append(infoMetrics, &infoMetricData{
					Labels: keyLabels,
					Meta:   labels,
				})

				// 收集这个infoMetric中所有label字段
				for k := range keyLabels {
					infoMetricLabelFields[k] = struct{}{}
				}
				continue
			}

			// 处理普通指标
			if field != "" {
				// 获取或创建事件
				event := getEvent(eventsMap, keyLabels)
				// 创建更新数据
				update := mapstr.M{}
				update.Put(field, value)
				// 更新事件数据
				event.DeepUpdate(update)
				// 添加标签数据
				event.DeepUpdate(labels)
			}
		}
	}

	// 为所有事件添加额外字段
	for _, event := range eventsMap {
		// Add extra fields
		for k, v := range mapping.ExtraFields {
			event[k] = v
		}
	}

	// 将eventsMap转换为events数组
	events := make([]mapstr.M, 0, len(eventsMap))
	for _, event := range eventsMap {
		events = append(events, event)
	}

	// 构建标签与事件索引的映射关系，但只处理infoMetric中出现的label字段
	// 格式: labelKey -> labelValue -> eventIndices
	labelValueToEventIndices := make(map[string]map[interface{}][]int)

	// 如果有infoMetrics，才需要构建实际的映射关系
	if len(infoMetrics) > 0 {
		// 遍历所有事件，构建映射关系
		for i, event := range events {
			// 遍历事件中的所有字段
			flattenEvent := event.Flatten()
			for k, v := range flattenEvent {
				// 只处理infoMetric中出现的label字段
				if _, exists := infoMetricLabelFields[k]; exists {
					// 初始化映射关系
					if _, ok := labelValueToEventIndices[k]; !ok {
						labelValueToEventIndices[k] = make(map[interface{}][]int)
					}
					// 添加事件索引到对应的label value下
					labelValueToEventIndices[k][v] = append(labelValueToEventIndices[k][v], i)
				}
			}
		}

		// 处理infoMetrics
		for _, info := range infoMetrics {
			// 使用map实现集合，提高交集计算效率
			matchingIndices := make(map[int]struct{})
			firstLabel := true

			// 遍历info的所有label
			for k, v := range info.Labels.Flatten() {
				// 如果这个label有对应的事件索引映射
				if valueToIndices, ok := labelValueToEventIndices[k]; ok {
					if indices, ok := valueToIndices[v]; ok {
						if firstLabel {
							// 第一个标签：初始化集合
							for _, idx := range indices {
								matchingIndices[idx] = struct{}{}
							}
							firstLabel = false
						} else {
							// 后续标签：计算交集
							// 创建一个新的集合，只保留在当前标签索引列表中存在的元素
							newMatchingIndices := make(map[int]struct{})
							for _, idx := range indices {
								if _, exists := matchingIndices[idx]; exists {
									newMatchingIndices[idx] = struct{}{}
								}
							}
							matchingIndices = newMatchingIndices
						}
					} else if !firstLabel {
						// 如果后续标签没有匹配的索引，结果集将为空
						matchingIndices = make(map[int]struct{})
						break
					}
				}

				// 如果集合为空，提前退出循环
				if len(matchingIndices) == 0 && !firstLabel {
					break
				}
			}

			// 将info的meta信息更新到匹配的事件中
			for idx := range matchingIndices {
				if idx < len(events) {
					events[idx].DeepUpdate(info.Meta)
				}
			}
		}
	}

	// 处理聚合指标
	for _, am := range mapping.AggregateMetrics {
		builder := easyops.NewAggregateMetricBuilder(am)
		es := builder.Build(events)
		events = append(events, es...)
	}

	return events, nil
}

func (p *prometheus) GetProcessedMetrics(mapping *MetricsMapping) ([]mapstr.M, error) {
	families, err := p.GetFamilies()
	if err != nil {
		return nil, err
	}
	return p.ProcessMetrics(families, mapping)
}

// infoMetricData keeps data about an infoMetric
type infoMetricData struct {
	Labels mapstr.M
	Meta   mapstr.M
}

func (p *prometheus) ReportProcessedMetrics(mapping *MetricsMapping, r mb.ReporterV2) error {
	events, err := p.GetProcessedMetrics(mapping)
	if err != nil {
		return errors.Wrap(err, "error getting processed metrics")
	}
	for _, event := range events {
		r.Event(mb.Event{
			MetricSetFields: event,
			Namespace:       mapping.Namespace,
		})
	}

	return nil
}

func getEvent(m map[string]mapstr.M, labels mapstr.M) mapstr.M {
	hash := labels.String()
	res, ok := m[hash]
	if !ok {
		res = labels
		m[hash] = res
	}
	return res
}

func getLabels(metric *dto.Metric) mapstr.M {
	labels := mapstr.M{}
	for _, label := range metric.GetLabel() {
		if label.GetName() != "" && label.GetValue() != "" {
			labels.Put(label.GetName(), label.GetValue())
		}
	}
	return labels
}

// CompilePatternList compiles a pattern list and returns the list of the compiled patterns
func CompilePatternList(patterns *[]string) ([]*regexp.Regexp, error) {
	var compiledPatterns []*regexp.Regexp
	compiledPatterns = []*regexp.Regexp{}
	if patterns != nil {
		for _, pattern := range *patterns {
			r, err := regexp.Compile(pattern)
			if err != nil {
				return nil, errors.Wrapf(err, "compiling pattern '%s'", pattern)
			}
			compiledPatterns = append(compiledPatterns, r)
		}
		return compiledPatterns, nil
	}
	return []*regexp.Regexp{}, nil
}

// MatchMetricFamily checks if the given family/metric name matches any of the given patterns
func MatchMetricFamily(family string, matchMetrics []*regexp.Regexp) bool {
	for _, checkMetric := range matchMetrics {
		matched := checkMetric.MatchString(family)
		if matched {
			return true
		}
	}
	return false
}
