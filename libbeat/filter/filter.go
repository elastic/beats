package filter

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type FilterCondition struct {
}

type DropFieldsConfig struct {
	Fields []string `yaml:"fields"`
}

type IncludeFieldsConfig struct {
	Fields []string `yaml:"fields"`
}

type FilterConfig struct {
	DropFields    *DropFieldsConfig    `yaml:"drop_fields"`
	IncludeFields *IncludeFieldsConfig `yaml:"include_fields"`
}

type FilterRule interface {
	Filter(event common.MapStr) (common.MapStr, error)
	String() string
}

/* extends FilterRule */
type IncludeFields struct {
	Fields []string
	// condition
}

/* extend FilterRule */
type DropFields struct {
	Fields []string
	// condition
}

type FilterList struct {
	filters []FilterRule
}

// fields that should be always exported
var ReadOnlyFields = []string{"@timestamp", "beat", "type", "count"}

/* FilterList methods */
func New(config []FilterConfig) (*FilterList, error) {

	Filters := &FilterList{}
	Filters.filters = []FilterRule{}

	for _, filterConfig := range config {
		if filterConfig.DropFields != nil {
			Filters.Register(NewDropFields(filterConfig.DropFields.Fields))
		}

		if filterConfig.IncludeFields != nil {
			Filters.Register(NewIncludeFields(filterConfig.IncludeFields.Fields))
		}
	}

	logp.Debug("filter", "filters: %v", Filters)
	return Filters, nil
}

func (filters *FilterList) Register(filter FilterRule) {
	filters.filters = append(filters.filters, filter)
	logp.Debug("filter", "Register filter: %v", filter)
}

func (filters *FilterList) Get(index int) FilterRule {
	return filters.filters[index]
}

// Applies a sequence of filtering rules and returns the filtered event and if
// the event should be dropped.
func (filters *FilterList) Filter(event common.MapStr) (common.MapStr, bool) {

	// clone the event at first, before starting filtering
	filtered := event.Clone()
	var err error

	for _, filter := range filters.filters {
		filtered, err = filter.Filter(filtered)
		if err != nil {
			logp.Err("fail to apply filtering rule %s: %s", filter, err)
		}
	}

	if NoDataEvent(filtered) {
		// if the event contains no extra fields except the "mandatory" fields
		return filtered, true /* drop the event */
	}
	return filtered, false
}

func (filters *FilterList) String() string {
	s := []string{}

	for _, filter := range filters.filters {

		s = append(s, filter.String())
	}
	return strings.Join(s, ", ")
}

func NoDataEvent(event common.MapStr) bool {

	return len(event) <= len(ReadOnlyFields)
}

/* IncludeFields methods */
func NewIncludeFields(fields []string) *IncludeFields {

	/* add read only fields if they are not yet */
	for _, readOnly := range ReadOnlyFields {
		found := false
		for _, field := range fields {
			if readOnly == field {
				found = true
			}
		}
		if !found {
			fields = append(fields, readOnly)
		}
	}

	return &IncludeFields{Fields: fields}
}

func (f *IncludeFields) Filter(event common.MapStr) (common.MapStr, error) {

	filtered := common.MapStr{}

	for _, field := range f.Fields {
		hasKey, err := event.HasKey(field)
		if err != nil {
			return filtered, fmt.Errorf("Fail to check the key %s", field)
		}

		if hasKey {
			errorOnCopy := event.CopyTo(filtered, field)
			if errorOnCopy != nil {
				return filtered, fmt.Errorf("Fail to copy key %s", field)
			}
		}
	}

	return filtered, nil
}

func (f *IncludeFields) String() string {
	return "include_fields=" + strings.Join(f.Fields, ", ")
}

/* DropFields methods */
func NewDropFields(fields []string) *DropFields {

	/* remove read only fields */
	for _, readOnly := range ReadOnlyFields {
		for i, field := range fields {
			if readOnly == field {
				fields = append(fields[:i], fields[i+1:]...)
			}
		}
	}
	return &DropFields{Fields: fields}
}

func (f *DropFields) Filter(event common.MapStr) (common.MapStr, error) {

	for _, field := range f.Fields {
		err := event.Delete(field)
		if err != nil {
			return event, fmt.Errorf("Fail to delete key %s", field)
		}

	}
	return event, nil
}

func (f *DropFields) String() string {

	return "drop_fields=" + strings.Join(f.Fields, ", ")
}
