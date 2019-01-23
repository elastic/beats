package timeseries

import (
	"strings"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/libbeat/asset"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type timeseriesProcessor struct {
	dimensions map[string]interface{}
	prefixes   map[string]interface{}
}

// NewTimeSeriesProcessor returns a processor to add timeseries info to events
// Events are processed to extract all their dimensions (keyword fields that
// hold a dimension of the metrics) and compute a hash of all their values into
// `timeseries.instance` field.
func NewTimeSeriesProcessor(beatName string) (processors.Processor, error) {
	fieldsYAML, err := asset.GetFields(beatName)
	if err != nil {
		return nil, err
	}

	fields, err := common.NewFieldsFromYAML(fieldsYAML)
	if err != nil {
		return nil, err
	}

	dimensions := map[string]interface{}{}
	prefixes := map[string]interface{}{}
	populateDimensions("", dimensions, prefixes, fields)

	return &timeseriesProcessor{dimensions: dimensions, prefixes: prefixes}, nil
}

func (t *timeseriesProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if event.TimeSeries {
		instanceFields := common.MapStr{}

		// map all dimensions & values
		for k, v := range event.Fields.Flatten() {
			if t.isDimension(k) {
				instanceFields[k] = v
			}
		}

		h, err := hashstructure.Hash(instanceFields, nil)
		if err != nil {
			// this should not happen, keep the event in any case
			return event, err
		}
		event.Fields["timeseries"] = common.MapStr{
			"instance": h,
		}
	}

	return event, nil
}

func (t *timeseriesProcessor) isDimension(field string) bool {
	if _, ok := t.dimensions[field]; ok {
		return true
	}

	// field matches any of the prefixes
	for prefix := range t.prefixes {
		if strings.HasPrefix(field, prefix) {
			return true
		}
	}

	return false
}

// put all dimension fields in the given map for quick access
func populateDimensions(prefix string, dimensions map[string]interface{}, prefixes map[string]interface{}, fields common.Fields) {
	for _, f := range fields {
		name := f.Name
		if prefix != "" {
			name = prefix + "." + name
		}

		if len(f.Fields) > 0 {
			populateDimensions(name, dimensions, prefixes, f.Fields)
			continue
		}

		if isDimension(f) {
			if f.Type == "object" {
				// everything with this prefix is a dimension
				prefixes[prefix] = nil
			} else {
				dimensions[name] = nil
			}
		}
	}
}

func isDimension(f common.Field) bool {
	// keywords are dimensions by default (disabled with dimension: false in fields.yml)
	if f.Dimension == nil {
		return f.Type == "keyword" || (f.Type == "object" && f.ObjectType == "keyword")
	}

	// user defined dimension (dimension: true in fields.yml)
	return *f.Dimension
}

func (t *timeseriesProcessor) String() string {
	return "timeseries"
}
