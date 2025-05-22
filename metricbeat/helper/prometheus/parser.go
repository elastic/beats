package prometheus

import (
	"bytes"
	"io"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/pkg/textparse"
)

// -------------------------------------------------

type Parser struct {
}

// -------------------------------------------------

func NewParser() *Parser {
	return &Parser{}
}

// -------------------------------------------------

func (parser *Parser) Parse(body []byte) []*dto.MetricFamily {
	promParser := textparse.NewPromParser(body)
	metricFamilies := make([]*dto.MetricFamily, 0, 100)
	metricFamiliesByName := make(map[string]*dto.MetricFamily)

	for {
		var et textparse.Entry
		var err error
		if et, err = promParser.Next(); err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}

		switch et {
		case textparse.EntrySeries:
			s, _, value := promParser.Series()

			dimStartIndex := bytes.Index(s, []byte("{"))
			var key string
			var dims []byte
			if dimStartIndex > 0 {
				key = string(s[:dimStartIndex])
				dimEndIndex := bytes.LastIndex(s, []byte("}"))
				dims = s[dimStartIndex+1 : dimEndIndex]
			} else {
				key = string(s[:])
			}

			// 创建或获取 MetricFamily
			family, exists := metricFamiliesByName[key]
			if !exists {
				family = &dto.MetricFamily{
					Name: &key,
					Type: dto.MetricType_GAUGE.Enum(), // 默认为 GAUGE 类型
				}
				metricFamiliesByName[key] = family
				metricFamilies = append(metricFamilies, family)
			}

			// 创建 Metric
			metric := &dto.Metric{}

			// 设置标签
			labels := make([]*dto.LabelPair, 0)
			for _, d := range bytes.Split(dims, []byte(",")) {
				sIndex := bytes.Index(d, []byte("="))
				if sIndex < 0 {
					continue
				}
				dimName := string(d[:sIndex])
				dimValue := strings.ReplaceAll(string(d[sIndex+1:]), "\"", "")
				label := &dto.LabelPair{
					Name:  &dimName,
					Value: &dimValue,
				}
				labels = append(labels, label)
			}
			metric.Label = labels

			// 设置值
			gauge := &dto.Gauge{
				Value: &value,
			}
			metric.Gauge = gauge

			// 添加到 MetricFamily
			family.Metric = append(family.Metric, metric)

		default:
			continue
		}
	}

	return metricFamilies
}
