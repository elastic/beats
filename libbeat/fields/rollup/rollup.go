package rollup

import (
	"github.com/elastic/beats/libbeat/common"
)

// Processor struct to process fields to rollup
type Processor struct {
	metrics []common.MapStr
	terms   []string
}

// NewProcessor creates a new rollup fields processor. It presets the required terms for Metricbeat.
func NewProcessor() *Processor {
	return &Processor{
		terms: []string{"beat.name", "metricset.name", "metricset.module"},
	}
}

// Process recursively processes the given fields and checks for rollup configs
func (p *Processor) Process(fields common.Fields, path string) error {
	for _, field := range fields {

		field.Path = path

		switch field.Type {
		case "group":
			var newPath string
			if path == "" {
				newPath = field.Name
			} else {
				newPath = path + "." + field.Name
			}

			if err := p.Process(field.Fields, newPath); err != nil {
				return err
			}
		default:
			if len(field.RollupMetrics) > 0 {
				p.addMetric(field)
			}
			if field.RollupTerm {
				p.addTerm(field)
			}
		}
	}
	return nil
}

func (p *Processor) addMetric(f common.Field) {
	metric := common.MapStr{
		"field":   f.Path + "." + f.Name,
		"metrics": f.RollupMetrics,
	}
	p.metrics = append(p.metrics, metric)
}

func (p *Processor) addTerm(f common.Field) {
	p.terms = append(p.terms, f.Path)
}

// Generate creates a common.MapStr object for the rollup job with the terms and fields.
func (p *Processor) Generate() common.MapStr {
	return common.MapStr{
		"index_pattern": "metricbeat-*",
		"rollup_index":  "metricbeat_rollup",
		"cron":          "*/30 * * * * ?",
		"page_size":     "10",
		"groups": common.MapStr{
			"date_histogram": common.MapStr{
				"field":    "@timestamp",
				"interval": "1h",
				"delay":    "7d",
			},
			"terms": common.MapStr{
				"fields": p.terms,
			},
		},
		"metrics": p.metrics,
	}
}
