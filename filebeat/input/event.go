package input

import (
	"time"

	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/jsontransform"
)

// Event is sent to the output and must contain all relevant information
type Event struct {
	common.EventMetadata
	ReadTime     time.Time
	InputType    string
	DocumentType string
	Bytes        int
	Text         *string
	JSONConfig   *reader.JSONConfig
	State        file.State
	Data         common.MapStr // Use in readers to add data to the event
	Pipeline     string
	Fileset      string
	Module       string
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

	if e.Fileset != "" && e.Module != "" {
		event["fileset"] = common.MapStr{
			"name":   e.Fileset,
			"module": e.Module,
		}
	}

	// Add data fields which are added by the readers
	for key, value := range e.Data {
		event[key] = value
	}

	// Check if json fields exist
	var jsonFields common.MapStr
	if fields, ok := event["json"]; ok {
		jsonFields = fields.(common.MapStr)
	}

	if e.JSONConfig != nil && len(jsonFields) > 0 {
		mergeJSONFields(e, event, jsonFields)
	} else if e.Text != nil {
		event["message"] = *e.Text
	}

	return event
}

// Metadata creates a common.MapStr containing the metadata to
// be associated with the event.
func (e *Event) Metadata() common.MapStr {
	if e.Pipeline != "" {
		return common.MapStr{
			"pipeline": e.Pipeline,
		}
	}
	return nil
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
func mergeJSONFields(e *Event, event common.MapStr, jsonFields common.MapStr) {

	// The message key might have been modified by multiline
	if len(e.JSONConfig.MessageKey) > 0 && e.Text != nil {
		jsonFields[e.JSONConfig.MessageKey] = *e.Text
	}

	if e.JSONConfig.KeysUnderRoot {
		// Delete existing json key
		delete(event, "json")

		jsontransform.WriteJSONKeys(event, jsonFields, e.JSONConfig.OverwriteKeys, reader.JsonErrorKey)
	}
}
