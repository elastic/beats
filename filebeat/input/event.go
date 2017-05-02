package input

import (
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
)

// Event is sent to the output and must contain all relevant information
type Event struct {
	EventMeta
	Data common.MapStr // Use in readers to add data to the event
}

type EventMeta struct {
	Pipeline  string
	Fileset   string
	Module    string
	InputType string
	Bytes     int
	State     file.State
}

type Data struct {
	Event    common.MapStr
	Metadata EventMeta
}

func NewEvent(state file.State) *Event {
	return &Event{
		EventMeta: EventMeta{
			State: state,
		},
	}

}

func (e *Event) ToMapStr() common.MapStr {

	event := e.Data
	if event == nil {
		event = common.MapStr{}
	}

	if e.Fileset != "" && e.Module != "" {
		event["fileset"] = common.MapStr{
			"name":   e.Fileset,
			"module": e.Module,
		}
	}

	return event
}

func (e *Event) GetData() Data {
	return Data{
		Event: e.ToMapStr(),
		Metadata: EventMeta{
			Pipeline: e.Pipeline,
			Bytes:    e.Bytes,
			State:    e.State,
			Fileset:  e.Fileset,
			Module:   e.Module,
		},
	}
}

// Metadata creates a common.MapStr containing the metadata to
// be associated with the event.
func (eh *Data) GetMetadata() common.MapStr {
	if eh.Metadata.Pipeline != "" {
		return common.MapStr{
			"pipeline": eh.Metadata.Pipeline,
		}
	}
	return nil
}

// HasData returns true if the event itself contains data
// Events without data are only state updates
func (eh *Data) HasData() bool {
	return eh.Metadata.Bytes > 0
}
