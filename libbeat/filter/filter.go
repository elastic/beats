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
	Cond Condition
}

/* extend FilterRule */
type DropFields struct {
	Fields []string
	// condition
	Cond Condition
}

type FilterList struct {
	filters []FilterRule
}

/* FilterList methods */
func New(config []FilterConfig) (*FilterList, error) {

	Filters := &FilterList{}
	Filters.filters = []FilterRule{}

	for _, filterConfig := range config {
		if filterConfig.DropFields != nil {
			rule, err := NewDropFields(*filterConfig.DropFields)
			if err != nil {
				return nil, err
			}
			Filters.Register(rule)
		}

		if filterConfig.IncludeFields != nil {
			rule, err := NewIncludeFields(*filterConfig.IncludeFields)
			if err != nil {
				return nil, err
			}
			Filters.Register(rule)
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

	// Check if filters are set, just return event if not
	if len(filters.filters) == 0 {
		return event
	}

	// clone the event at first, before starting filtering
	filtered := event.Clone()
	var err error

	for _, filter := range filters.filters {
		filtered, err = filter.Filter(filtered)
		if err != nil {
			logp.Debug("filter", "fail to apply filtering rule %s: %s", filter, err)
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
func NewIncludeFields(config IncludeFieldsConfig) (*IncludeFields, error) {

	/* add read only fields if they are not yet */
	for _, readOnly := range MandatoryExportedFields {
		found := false
		for _, field := range config.Fields {
			if readOnly == field {
				found = true
			}
		}
		if !found {
			config.Fields = append(config.Fields, readOnly)
		}
	}

	cond, err := NewCondition(config.ConditionConfig)
	if err != nil {
		return nil, err
	}
	return &IncludeFields{Fields: config.Fields, Cond: *cond}, nil
}

func (f *IncludeFields) Filter(event common.MapStr) (common.MapStr, error) {

	if !f.Cond.Check(event) {
		return event, nil
	}

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
	return "include_fields=" + strings.Join(f.Fields, ", ") + ", condition=" + f.Cond.String()
}

/* DropFields methods */
func NewDropFields(config DropFieldsConfig) (*DropFields, error) {

	/* remove read only fields */
	for _, readOnly := range MandatoryExportedFields {
		for i, field := range config.Fields {
			if readOnly == field {
				config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)
			}
		}
	}

	cond, err := NewCondition(config.ConditionConfig)
	if err != nil {
		return nil, err
	}
	return &DropFields{Fields: config.Fields, Cond: *cond}, nil
}

func (f *DropFields) Filter(event common.MapStr) (common.MapStr, error) {

	if !f.Cond.Check(event) {
		return event, nil
	}

	for _, field := range f.Fields {
		err := event.Delete(field)
		if err != nil {
			return event, fmt.Errorf("Fail to delete key %s: %s", field, err)
		}

	}
	return event, nil
}

func (f *DropFields) String() string {

	return "drop_fields=" + strings.Join(f.Fields, ", ") + ", condition=" + f.Cond.String()
}
