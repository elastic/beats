package filter

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type FilterCondition struct {
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

/* FilterList methods */
func New(config []FilterConfig) (*FilterList, error) {

	Filters := &FilterList{}
	Filters.filters = []FilterRule{}

	logp.Debug("filter", "configuration %v", config)
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

// Applies a sequence of filtering rules and returns the filtered event
func (filters *FilterList) Filter(event common.MapStr) common.MapStr {

	// clone the event at first, before starting filtering
	filtered := event.Clone()
	var err error

	for _, filter := range filters.filters {
		filtered, err = filter.Filter(filtered)
		if err != nil {
			logp.Err("fail to apply filtering rule %s: %s", filter, err)
		}
	}

	return filtered
}

func (filters *FilterList) String() string {
	s := []string{}

	for _, filter := range filters.filters {

		s = append(s, filter.String())
	}
	return strings.Join(s, ", ")
}

/* IncludeFields methods */
func NewIncludeFields(fields []string) *IncludeFields {

	/* add read only fields if they are not yet */
	for _, readOnly := range MandatoryExportedFields {
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
			return filtered, fmt.Errorf("Fail to check the key %s: %s", field, err)
		}

		if hasKey {
			errorOnCopy := event.CopyFieldsTo(filtered, field)
			if errorOnCopy != nil {
				return filtered, fmt.Errorf("Fail to copy key %s: %s", field, err)
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
	for _, readOnly := range MandatoryExportedFields {
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
			return event, fmt.Errorf("Fail to delete key %s: %s", field, err)
		}

	}
	return event, nil
}

func (f *DropFields) String() string {

	return "drop_fields=" + strings.Join(f.Fields, ", ")
}
