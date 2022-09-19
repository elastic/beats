package move_fields

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

func init() {
	processors.RegisterPlugin("move_fields",
		checks.ConfigChecked(NewMoveFields, checks.AllowedFields("from", "to")))
	jsprocessor.RegisterPlugin("MoveFields", NewMoveFields)
}

type moveFieldsConfig struct {
	ParentPath string   `config:"parent_path"`
	From       []string `config:"from"`
	To         string   `config:"to"`
	Exclude    []string `config:"exclude"`

	excludeMap map[string]bool
}

type moveFields struct {
	config moveFieldsConfig
}

func (u moveFields) Run(event *beat.Event) (*beat.Event, error) {
	root := event.Fields.Clone()
	parent := root
	if p := u.config.ParentPath; p != "" {
		parentValue, err := root.GetValue(p)
		if err != nil {
			return nil, fmt.Errorf("move field read parent path field failed: %w", err)
		}
		var ok bool
		parent, ok = parentValue.(common.MapStr)
		if !ok {
			return nil, fmt.Errorf("move field parent is not message map")
		}
	}

	keys := u.config.From
	if len(keys) == 0 {
		keys = make([]string, 0, len(parent))
		for k := range parent {
			keys = append(keys, k)
		}
	}

	for _, k := range keys {
		if _, ok := u.config.excludeMap[k]; ok {
			continue
		}
		v, err := parent.GetValue(k)
		if err != nil {
			return nil, fmt.Errorf("move field read field from parent, sub key: %s, failed: %w", k, err)
		}
		if err = parent.Delete(k); err != nil {
			return nil, fmt.Errorf("move field delete field from parent sub key: %s, failed: %w", k, err)
		}
		newKey := fmt.Sprintf("%s%s", u.config.To, k)
		if _, err = root.Put(newKey, v); err != nil {
			return nil, fmt.Errorf("move field write field to sub key: %s, new key: %s, failed: %w", k, newKey, err)
		}
	}

	event.Fields = root
	return event, nil
}

func (u moveFields) String() string {
	return "move_fields"
}

func NewMoveFields(c *common.Config) (processors.Processor, error) {
	fc := moveFieldsConfig{}
	err := c.Unpack(&fc)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack move fields config: %w", err)
	}

	fc.excludeMap = make(map[string]bool)
	for _, k := range fc.Exclude {
		fc.excludeMap[k] = true
	}

	return &moveFields{
		config: fc,
	}, nil
}
