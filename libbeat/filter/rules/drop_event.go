package rules

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/filter"
)

type DropEvent struct {
	Cond *filter.Condition
}

type DropEventConfig struct {
	filter.ConditionConfig `config:",inline"`
}

func init() {
	if err := filter.RegisterPlugin("drop_event", newDropEvent); err != nil {
		panic(err)
	}
}

func newDropEvent(c common.Config) (filter.FilterRule, error) {

	f := DropEvent{}

	if err := f.CheckConfig(c); err != nil {
		return nil, err
	}

	config := DropEventConfig{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the drop_event configuration: %s", err)
	}

	cond, err := filter.NewCondition(config.ConditionConfig)
	if err != nil {
		return nil, err
	}
	f.Cond = cond

	return &f, nil
}

func (f *DropEvent) CheckConfig(c common.Config) error {

	for _, field := range c.GetFields() {
		if !filter.AvailableCondition(field) {
			return fmt.Errorf("unexpected %s option in the drop_event configuration", field)
		}
	}
	return nil
}

func (f *DropEvent) Filter(event common.MapStr) (common.MapStr, error) {

	if f.Cond != nil && !f.Cond.Check(event) {
		return event, nil
	}

	// return event=nil to delete the entire event
	return nil, nil
}

func (f DropEvent) String() string {
	if f.Cond != nil {
		return "drop_event, condition=" + f.Cond.String()
	}
	return "drop_event"
}
