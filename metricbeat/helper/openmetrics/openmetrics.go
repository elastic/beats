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

package openmetrics

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/common/model"

	"github.com/prometheus/prometheus/pkg/exemplar"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/timestamp"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/helper"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

const acceptHeader = `application/openmetrics-text; version=1.0.0; charset=utf-8,text/plain`

var errNameLabelMandatory = fmt.Errorf("missing metric name (%s label)", labels.MetricName)

type Gauge struct {
	Value *float64
}

func (m *Gauge) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Info struct {
	Value *int64
}

func (m *Info) GetValue() int64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}
func (m *Info) HasValidValue() bool {
	return m != nil && *m.Value == 1
}

type Stateset struct {
	Value *int64
}

func (m *Stateset) GetValue() int64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}
func (m *Stateset) HasValidValue() bool {
	return m != nil && (*m.Value == 0 || *m.Value == 1)
}

type Counter struct {
	Value *float64
}

func (m *Counter) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Quantile struct {
	Quantile *float64
	Value    *float64
	Exemplar *exemplar.Exemplar
}

func (m *Quantile) GetQuantile() float64 {
	if m != nil && m.Quantile != nil {
		return *m.Quantile
	}
	return 0
}

func (m *Quantile) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Summary struct {
	SampleCount *uint64
	SampleSum   *float64
	Quantile    []*Quantile
}

func (m *Summary) GetSampleCount() uint64 {
	if m != nil && m.SampleCount != nil {
		return *m.SampleCount
	}
	return 0
}

func (m *Summary) GetSampleSum() float64 {
	if m != nil && m.SampleSum != nil {
		return *m.SampleSum
	}
	return 0
}

func (m *Summary) GetQuantile() []*Quantile {
	if m != nil {
		return m.Quantile
	}
	return nil
}

type Unknown struct {
	Value *float64
}

func (m *Unknown) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Bucket struct {
	CumulativeCount *uint64
	UpperBound      *float64
	Exemplar        *exemplar.Exemplar
}

func (m *Bucket) GetCumulativeCount() uint64 {
	if m != nil && m.CumulativeCount != nil {
		return *m.CumulativeCount
	}
	return 0
}

func (m *Bucket) GetUpperBound() float64 {
	if m != nil && m.UpperBound != nil {
		return *m.UpperBound
	}
	return 0
}

type Histogram struct {
	SampleCount      *uint64
	SampleSum        *float64
	Bucket           []*Bucket
	IsGaugeHistogram bool
}

func (m *Histogram) GetSampleCount() uint64 {
	if m != nil && m.SampleCount != nil {
		return *m.SampleCount
	}
	return 0
}

func (m *Histogram) GetSampleSum() float64 {
	if m != nil && m.SampleSum != nil {
		return *m.SampleSum
	}
	return 0
}

func (m *Histogram) GetBucket() []*Bucket {
	if m != nil {
		return m.Bucket
	}
	return nil
}

type OpenMetric struct {
	Label       []*labels.Label
	Exemplar    *exemplar.Exemplar
	Name        *string
	Gauge       *Gauge
	Counter     *Counter
	Info        *Info
	Stateset    *Stateset
	Summary     *Summary
	Unknown     *Unknown
	Histogram   *Histogram
	TimestampMs *int64
}

func (m *OpenMetric) GetName() *string {
	if m != nil {
		return m.Name
	}
	return nil
}

func (m *OpenMetric) GetLabel() []*labels.Label {
	if m != nil {
		return m.Label
	}
	return nil
}

func (m *OpenMetric) GetGauge() *Gauge {
	if m != nil {
		return m.Gauge
	}
	return nil
}

func (m *OpenMetric) GetCounter() *Counter {
	if m != nil {
		return m.Counter
	}
	return nil
}

func (m *OpenMetric) GetInfo() *Info {
	if m != nil {
		return m.Info
	}
	return nil
}

func (m *OpenMetric) GetStateset() *Stateset {
	if m != nil {
		return m.Stateset
	}
	return nil
}

func (m *OpenMetric) GetSummary() *Summary {
	if m != nil {
		return m.Summary
	}
	return nil
}

func (m *OpenMetric) GetUnknown() *Unknown {
	if m != nil {
		return m.Unknown
	}
	return nil
}

func (m *OpenMetric) GetHistogram() *Histogram {
	if m != nil && m.Histogram != nil && !m.Histogram.IsGaugeHistogram {
		return m.Histogram
	}
	return nil
}

func (m *OpenMetric) GetGaugeHistogram() *Histogram {
	if m != nil && m.Histogram != nil && m.Histogram.IsGaugeHistogram {
		return m.Histogram
	}
	return nil
}

func (m *OpenMetric) GetTimestampMs() int64 {
	if m != nil && m.TimestampMs != nil {
		return *m.TimestampMs
	}
	return 0
}

type OpenMetricFamily struct {
	Name   *string
	Help   *string
	Type   textparse.MetricType
	Unit   *string
	Metric []*OpenMetric
}

func (m *OpenMetricFamily) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}
func (m *OpenMetricFamily) GetUnit() string {
	if m != nil && *m.Unit != "" {
		return *m.Unit
	}
	return ""
}

func (m *OpenMetricFamily) GetMetric() []*OpenMetric {
	if m != nil {
		return m.Metric
	}
	return nil
}

// OpenMetrics helper retrieves openmetrics formatted metrics
// This interface needs to use TextParse
type OpenMetrics interface {
	// GetFamilies requests metric families from openmetrics endpoint and returns them
	GetFamilies() ([]*OpenMetricFamily, error)

	GetProcessedMetrics(mapping *MetricsMapping) ([]common.MapStr, error)

	ProcessMetrics(families []*OpenMetricFamily, mapping *MetricsMapping) ([]common.MapStr, error)

	ReportProcessedMetrics(mapping *MetricsMapping, r mb.ReporterV2) error
}

type openmetrics struct {
	httpfetcher
	logger *logp.Logger
}

type httpfetcher interface {
	FetchResponse() (*http.Response, error)
}

// NewOpenMetricsClient creates new openmetrics helper
func NewOpenMetricsClient(base mb.BaseMetricSet) (OpenMetrics, error) {
	httpclient, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	httpclient.SetHeaderDefault("Accept", acceptHeader)
	httpclient.SetHeaderDefault("Accept-Encoding", "gzip")
	return &openmetrics{httpclient, base.Logger()}, nil
}

// GetFamilies requests metric families from openmetrics endpoint and returns them
func (p *openmetrics) GetFamilies() ([]*OpenMetricFamily, error) {
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

	if resp.StatusCode > 399 {
		bodyBytes, err := ioutil.ReadAll(reader)
		if err == nil {
			p.logger.Debug("error received from openmetrics endpoint: ", string(bodyBytes))
		}
		return nil, fmt.Errorf("unexpected status code %d from server", resp.StatusCode)
	}

	contentType := getContentType(resp.Header)
	if contentType == "" {
		return nil, fmt.Errorf("Invalid format for response of response")
	}

	appendTime := time.Now().Round(0)
	b, err := ioutil.ReadAll(reader)
	families, err := parseMetricFamilies(b, contentType, appendTime)

	return families, nil
}

const (
	suffixInfo   = "_info"
	suffixTotal  = "_total"
	suffixGCount = "_gcount"
	suffixGSum   = "_gsum"
	suffixCount  = "_count"
	suffixSum    = "_sum"
	suffixBucket = "_bucket"
)

func isInfo(name string) bool {
	return len(name) > 5 && name[len(name)-5:] == suffixInfo
}

// Counters have _total suffix
func isTotal(name string) bool {
	return len(name) > 6 && name[len(name)-6:] == suffixTotal
}

func isGCount(name string) bool {
	return len(name) > 7 && name[len(name)-7:] == suffixGCount
}

func isGSum(name string) bool {
	return len(name) > 5 && name[len(name)-5:] == suffixGSum
}

func isCount(name string) bool {
	return len(name) > 6 && name[len(name)-6:] == suffixCount
}

func isSum(name string) bool {
	return len(name) > 4 && name[len(name)-4:] == suffixSum
}

func isBucket(name string) bool {
	return len(name) > 7 && name[len(name)-7:] == suffixBucket
}

func summaryMetricName(name string, s float64, qv string, lbls string, t *int64, summariesByName map[string]map[string]*OpenMetric) (string, *OpenMetric) {
	var summary = &Summary{}
	var quantile = []*Quantile{}
	var quant = &Quantile{}

	switch {
	case isCount(name):
		u := uint64(s)
		summary.SampleCount = &u
		name = name[:len(name)-6]
	case isSum(name):
		summary.SampleSum = &s
		name = name[:len(name)-4]
	default:
		f, err := strconv.ParseFloat(qv, 64)
		if err != nil {
			f = -1
		}
		quant.Quantile = &f
		quant.Value = &s
	}

	_, k := summariesByName[name]
	if !k {
		summariesByName[name] = make(map[string]*OpenMetric)
	}
	metric, ok := summariesByName[name][lbls]
	if !ok {
		metric = &OpenMetric{}
		metric.Name = &name
		metric.Summary = summary
		metric.Summary.Quantile = quantile
		summariesByName[name][lbls] = metric
	}
	if metric.Summary.SampleSum == nil && summary.SampleSum != nil {
		metric.Summary.SampleSum = summary.SampleSum
	} else if metric.Summary.SampleCount == nil && summary.SampleCount != nil {
		metric.Summary.SampleCount = summary.SampleCount
	} else if quant.Quantile != nil {
		metric.Summary.Quantile = append(metric.Summary.Quantile, quant)
	}

	return name, metric
}

func histogramMetricName(name string, s float64, qv string, lbls string, t *int64, isGaugeHistogram bool, e *exemplar.Exemplar, histogramsByName map[string]map[string]*OpenMetric) (string, *OpenMetric) {
	var histogram = &Histogram{}
	var bucket = []*Bucket{}
	var bkt = &Bucket{}

	switch {
	case isCount(name):
		u := uint64(s)
		histogram.SampleCount = &u
		name = name[:len(name)-6]
	case isSum(name):
		histogram.SampleSum = &s
		name = name[:len(name)-4]
	case isGaugeHistogram && isGCount(name):
		u := uint64(s)
		histogram.SampleCount = &u
		name = name[:len(name)-7]
	case isGaugeHistogram && isGSum(name):
		histogram.SampleSum = &s
		name = name[:len(name)-5]
	default:
		if isBucket(name) {
			name = name[:len(name)-7]
		}
		f, err := strconv.ParseFloat(qv, 64)
		if err != nil {
			f = math.MaxUint64
		}
		cnt := uint64(s)
		bkt.UpperBound = &f
		bkt.CumulativeCount = &cnt

		if e != nil {
			if !e.HasTs {
				e.Ts = *t
			}
			bkt.Exemplar = e
		}
	}

	_, k := histogramsByName[name]
	if !k {
		histogramsByName[name] = make(map[string]*OpenMetric)
	}
	metric, ok := histogramsByName[name][lbls]
	if !ok {
		metric = &OpenMetric{}
		metric.Name = &name
		metric.Histogram = histogram
		metric.Histogram.Bucket = bucket
		histogramsByName[name][lbls] = metric
	}
	if metric.Histogram.SampleSum == nil && histogram.SampleSum != nil {
		metric.Histogram.SampleSum = histogram.SampleSum
	} else if metric.Histogram.SampleCount == nil && histogram.SampleCount != nil {
		metric.Histogram.SampleCount = histogram.SampleCount
	} else if bkt.UpperBound != nil {
		metric.Histogram.Bucket = append(metric.Histogram.Bucket, bkt)
	}

	return name, metric
}

func parseMetricFamilies(b []byte, contentType string, ts time.Time) ([]*OpenMetricFamily, error) {
	var (
		parser               = textparse.New(b, contentType)
		defTime              = timestamp.FromTime(ts)
		metricFamiliesByName = map[string]*OpenMetricFamily{}
		summariesByName      = map[string]map[string]*OpenMetric{}
		histogramsByName     = map[string]map[string]*OpenMetric{}
		fam                  *OpenMetricFamily
		mt                   = textparse.MetricTypeUnknown
	)
	var err error

loop:
	for {
		var (
			et textparse.Entry
			ok bool
			e  exemplar.Exemplar
		)
		if et, err = parser.Next(); err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		switch et {
		case textparse.EntryType:
			buf, t := parser.Type()
			s := string(buf)
			fam, ok = metricFamiliesByName[s]
			if !ok {
				fam = &OpenMetricFamily{Name: &s, Type: t}
				metricFamiliesByName[s] = fam
			}
			mt = t
			continue
		case textparse.EntryHelp:
			buf, t := parser.Help()
			s := string(buf)
			h := string(t)
			fam, ok = metricFamiliesByName[s]
			if !ok {
				fam = &OpenMetricFamily{Name: &s, Help: &h, Type: textparse.MetricTypeUnknown}
				metricFamiliesByName[s] = fam
			}
			fam.Help = &h
			continue
		case textparse.EntryUnit:
			buf, t := parser.Unit()
			s := string(buf)
			u := string(t)
			fam, ok = metricFamiliesByName[s]
			if !ok {
				fam = &OpenMetricFamily{Name: &s, Unit: &u, Type: textparse.MetricTypeUnknown}
				metricFamiliesByName[string(buf)] = fam
			}
			fam.Unit = &u
			continue
		case textparse.EntryComment:
			continue
		default:
		}

		t := defTime
		_, tp, v := parser.Series()

		var (
			lset labels.Labels
			mets string
		)

		mets = parser.Metric(&lset)

		if !lset.Has(labels.MetricName) {
			err = errNameLabelMandatory
			break loop
		}

		var lbls strings.Builder
		lbls.Grow(len(mets))
		var labelPairs = []*labels.Label{}
		for _, l := range lset.Copy() {
			if l.Name == labels.MetricName {
				continue
			}

			if l.Name != model.QuantileLabel && l.Name != labels.BucketLabel { // quantile and le are special labels handled below

				lbls.WriteString(l.Name)
				lbls.WriteString(l.Value)
			}
			n := l.Name
			v := l.Value

			labelPairs = append(labelPairs, &labels.Label{
				Name:  n,
				Value: v,
			})
		}

		var metric *OpenMetric

		metricName := lset.Get(labels.MetricName)
		var lookupMetricName string
		var exm *exemplar.Exemplar

		// Suffixes - https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md#suffixes
		switch mt {
		case textparse.MetricTypeCounter:
			var counter = &Counter{Value: &v}
			mn := lset.Get(labels.MetricName)
			metric = &OpenMetric{Name: &mn, Counter: counter, Label: labelPairs}
			if isTotal(metricName) { // Remove suffix _total, get lookup metricname
				lookupMetricName = metricName[:len(metricName)-6]
			}
			break
		case textparse.MetricTypeGauge:
			var gauge = &Gauge{Value: &v}
			metric = &OpenMetric{Name: &metricName, Gauge: gauge, Label: labelPairs}
			lookupMetricName = metricName
			break
		case textparse.MetricTypeInfo:
			value := int64(v)
			var info = &Info{Value: &value}
			metric = &OpenMetric{Name: &metricName, Info: info, Label: labelPairs}
			lookupMetricName = metricName
			break
		case textparse.MetricTypeSummary:
			lookupMetricName, metric = summaryMetricName(metricName, v, lset.Get(model.QuantileLabel), lbls.String(), &t, summariesByName)
			metric.Label = labelPairs
			if !isSum(metricName) {
				continue
			}
			metricName = lookupMetricName
			break
		case textparse.MetricTypeHistogram:
			if hasExemplar := parser.Exemplar(&e); hasExemplar {
				exm = &e
			}
			lookupMetricName, metric = histogramMetricName(metricName, v, lset.Get(labels.BucketLabel), lbls.String(), &t, false, exm, histogramsByName)
			metric.Label = labelPairs
			if !isSum(metricName) {
				continue
			}
			metricName = lookupMetricName
			break
		case textparse.MetricTypeGaugeHistogram:
			if hasExemplar := parser.Exemplar(&e); hasExemplar {
				exm = &e
			}
			lookupMetricName, metric = histogramMetricName(metricName, v, lset.Get(labels.BucketLabel), lbls.String(), &t, true, exm, histogramsByName)
			metric.Label = labelPairs
			metric.Histogram.IsGaugeHistogram = true
			if !isGSum(metricName) {
				continue
			}
			metricName = lookupMetricName
			break
		case textparse.MetricTypeStateset:
			value := int64(v)
			var stateset = &Stateset{Value: &value}
			metric = &OpenMetric{Name: &metricName, Stateset: stateset, Label: labelPairs}
			lookupMetricName = metricName
			break
		case textparse.MetricTypeUnknown:
			var unknown = &Unknown{Value: &v}
			metric = &OpenMetric{Name: &metricName, Unknown: unknown, Label: labelPairs}
			lookupMetricName = metricName
			break
		default:
			lookupMetricName = metricName
		}

		fam, ok = metricFamiliesByName[lookupMetricName]
		if !ok {
			fam = &OpenMetricFamily{Type: mt}
			metricFamiliesByName[lookupMetricName] = fam
		}

		fam.Name = &metricName

		if hasExemplar := parser.Exemplar(&e); hasExemplar && mt != textparse.MetricTypeHistogram && metric != nil {
			if !e.HasTs {
				e.Ts = t
			}
			metric.Exemplar = &e
		}

		if tp != nil && metric != nil {
			t = *tp
			metric.TimestampMs = &t
		}

		fam.Metric = append(fam.Metric, metric)
	}

	families := make([]*OpenMetricFamily, 0, len(metricFamiliesByName))
	for _, v := range metricFamiliesByName {
		if v.Metric != nil {
			families = append(families, v)
		}
	}
	return families, nil
}

// MetricsMapping defines mapping settings for OpenMetrics metrics, to be used with `GetProcessedMetrics`
type MetricsMapping struct {
	// Metrics translates from openmetrics metric name to Metricbeat fields
	Metrics map[string]MetricMap

	// Namespace for metrics managed by this mapping
	Namespace string

	// Labels translate from openmetrics label names to Metricbeat fields
	Labels map[string]LabelMap

	// ExtraFields adds the given fields to all events coming from `GetProcessedMetrics`
	ExtraFields map[string]string
}

func (p *openmetrics) ProcessMetrics(families []*OpenMetricFamily, mapping *MetricsMapping) ([]common.MapStr, error) {

	eventsMap := map[string]common.MapStr{}
	infoMetrics := []*infoMetricData{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			m, ok := mapping.Metrics[family.GetName()]
			if m == nil || !ok {
				// Ignore unknown metrics
				continue
			}

			field := m.GetField()
			value := m.GetValue(metric)

			// Ignore retrieval errors (bad conf)
			if value == nil {
				continue
			}

			storeAllLabels := false
			labelsLocation := ""
			var extraFields common.MapStr
			if m != nil {
				c := m.GetConfiguration()
				storeAllLabels = c.StoreNonMappedLabels
				labelsLocation = c.NonMappedLabelsPlacement
				extraFields = c.ExtraFields
			}

			// Apply extra options
			allLabels := getLabels(metric)
			for _, option := range m.GetOptions() {
				field, value, allLabels = option.Process(field, value, allLabels)
			}

			// Convert labels
			labels := common.MapStr{}
			keyLabels := common.MapStr{}
			for k, v := range allLabels {
				if l, ok := mapping.Labels[k]; ok {
					if l.IsKey() {
						keyLabels.Put(l.GetField(), v)
					} else {
						labels.Put(l.GetField(), v)
					}
				} else if storeAllLabels {
					// if label for this metric is not found at the label mappings but
					// it is configured to store any labels found, make it so
					labels.Put(labelsLocation+"."+k, v)
				}
			}

			// if extra fields have been added through metric configuration
			// add them to labels.
			//
			// not considering these extra fields to be keylabels as that case
			// have not appeared yet
			for k, v := range extraFields {
				labels.Put(k, v)
			}

			// Keep a info document if it's an infoMetric
			if _, ok = m.(*infoMetric); ok {
				labels.DeepUpdate(keyLabels)
				infoMetrics = append(infoMetrics, &infoMetricData{
					Labels: keyLabels,
					Meta:   labels,
				})
				continue
			}

			if field != "" {
				event := getEvent(eventsMap, keyLabels)
				update := common.MapStr{}
				update.Put(field, value)
				// value may be a mapstr (for histograms and summaries), do a deep update to avoid smashing existing fields
				event.DeepUpdate(update)

				event.DeepUpdate(labels)
			}
		}
	}

	// populate events array from values in eventsMap
	events := make([]common.MapStr, 0, len(eventsMap))
	for _, event := range eventsMap {
		// Add extra fields
		for k, v := range mapping.ExtraFields {
			event[k] = v
		}
		events = append(events, event)
	}

	// fill info from infoMetrics
	for _, info := range infoMetrics {
		for _, event := range events {
			found := true
			for k, v := range info.Labels.Flatten() {
				value, err := event.GetValue(k)
				if err != nil || v != value {
					found = false
					break
				}
			}

			// fill info from this metric
			if found {
				event.DeepUpdate(info.Meta)
			}
		}
	}

	return events, nil
}

func (p *openmetrics) GetProcessedMetrics(mapping *MetricsMapping) ([]common.MapStr, error) {
	families, err := p.GetFamilies()
	if err != nil {
		return nil, err
	}
	return p.ProcessMetrics(families, mapping)
}

// infoMetricData keeps data about an infoMetric
type infoMetricData struct {
	Labels common.MapStr
	Meta   common.MapStr
}

func (p *openmetrics) ReportProcessedMetrics(mapping *MetricsMapping, r mb.ReporterV2) error {
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

func getEvent(m map[string]common.MapStr, labels common.MapStr) common.MapStr {
	hash := labels.String()
	res, ok := m[hash]
	if !ok {
		res = labels
		m[hash] = res
	}
	return res
}

func getLabels(metric *OpenMetric) common.MapStr {
	labels := common.MapStr{}
	for _, label := range metric.GetLabel() {
		if label.Name != "" && label.Value != "" {
			labels.Put(label.Name, label.Value)
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

const (
	TextVersion     = "0.0.4"
	OpenMetricsType = `application/openmetrics-text`

	// The Content-Type values for the different wire protocols.
	FmtUnknown string = `<unknown>`
	FmtText    string = `text/plain; version=` + TextVersion + `; charset=utf-8`
)

const (
	hdrContentType = "Content-Type"
)

func getContentType(h http.Header) string {
	ct := h.Get(hdrContentType)

	mediatype, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return FmtUnknown
	}

	const textType = "text/plain"

	switch mediatype {
	case OpenMetricsType:
		if e, ok := params["encoding"]; ok && e != "delimited" {
			return FmtUnknown
		}
		return OpenMetricsType

	case textType:
		if v, ok := params["version"]; ok && v != TextVersion {
			return FmtUnknown
		}
		return FmtText
	}

	return FmtUnknown
}
