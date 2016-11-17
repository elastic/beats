package input

import (
	"fmt"
	"time"

	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Event is sent to the output and must contain all relevant information
type Event struct {
	common.EventMetadata
	ReadTime     time.Time
	InputType    string
	DocumentType string
	Bytes        int
	Text         *string
	JSONFields   common.MapStr
	JSONConfig   *reader.JSONConfig
	State        file.State
}

func NewEvent(state file.State) *Event {
	return &Event{
		State: state,
	}
}

func (e *Event) ToMapStr() common.MapStr {
	event := common.MapStr{
		common.EventMetadataKey: e.EventMetadata,
		"@timestamp":            common.Time(e.ReadTime),
		"source":                e.State.Source,
		"offset":                e.State.Offset, // Offset here is the offset before the starting char.
		"type":                  e.DocumentType,
		"input_type":            e.InputType,
	}

	if e.JSONConfig != nil && len(e.JSONFields) > 0 {
		mergeJSONFields(e, event)
	} else if e.Text != nil {
		event["message"] = *e.Text
	}

	return event
}

// HasData returns true if the event itself contains data
// Events without data are only state updates
func (e *Event) HasData() bool {
	return e.Bytes > 0
}

// mergeJSONFields writes the JSON fields in the event map,
// respecting the KeysUnderRoot and OverwriteKeys configuration options.
// If MessageKey is defined, the Text value from the event always
// takes precedence.
func mergeJSONFields(e *Event, event common.MapStr) {

	// The message key might have been modified by multiline
	if len(e.JSONConfig.MessageKey) > 0 && e.Text != nil {
		e.JSONFields[e.JSONConfig.MessageKey] = *e.Text
	}

	if e.JSONConfig.KeysUnderRoot {
		for k, v := range e.JSONFields {
			if e.JSONConfig.OverwriteKeys {
				if k == "@timestamp" {
					vstr, ok := v.(string)
					if !ok {
						logp.Err("JSON: Won't overwrite @timestamp because value is not string")
						event[reader.JsonErrorKey] = "@timestamp not overwritten (not string)"
						continue
					}

					// @timestamp must be of format RFC3339
					ts, err := time.Parse(time.RFC3339, vstr)
					if err != nil {
						logp.Err("JSON: Won't overwrite @timestamp because of parsing error: %v", err)
						event[reader.JsonErrorKey] = fmt.Sprintf("@timestamp not overwritten (parse error on %s)", vstr)
						continue
					}
					event[k] = common.Time(ts)
				} else if k == "type" {
					vstr, ok := v.(string)
					if !ok {
						logp.Err("JSON: Won't overwrite type because value is not string")
						event[reader.JsonErrorKey] = "type not overwritten (not string)"
						continue
					}
					if len(vstr) == 0 || vstr[0] == '_' {
						logp.Err("JSON: Won't overwrite type because value is empty or starts with an underscore")
						event[reader.JsonErrorKey] = fmt.Sprintf("type not overwritten (invalid value [%s])", vstr)
						continue
					}
					event[k] = vstr
				} else {
					event[k] = v
				}
			} else if _, exists := event[k]; !exists {
				event[k] = v
			}
		}
	} else {
		event["json"] = e.JSONFields
	}
}
