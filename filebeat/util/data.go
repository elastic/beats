package util

import (
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
)

type Data struct {
	Event common.MapStr
	state file.State
	Meta  Meta
}

type Meta struct {
	Pipeline string
	Fileset  string
	Module   string
}

func NewData() *Data {
	return &Data{}
}

// SetState sets the state
func (d *Data) SetState(state file.State) {
	d.state = state
}

// GetState returns the current state
func (d *Data) GetState() file.State {
	return d.state
}

// HasState returns true if the data object contains state data
func (d *Data) HasState() bool {
	return d.state != file.State{}
}

// GetEvent returns the event in the data object
// In case meta data contains module and fileset data, the event is enriched with it
func (d *Data) GetEvent() common.MapStr {
	if d.Meta.Fileset != "" && d.Meta.Module != "" {
		d.Event["fileset"] = common.MapStr{
			"name":   d.Meta.Fileset,
			"module": d.Meta.Module,
		}
	}
	return d.Event
}

// GetMetadata creates a common.MapStr containing the metadata to
// be associated with the event.
func (d *Data) GetMetadata() common.MapStr {
	if d.Meta.Pipeline != "" {
		return common.MapStr{
			"pipeline": d.Meta.Pipeline,
		}
	}
	return nil
}

// HasEvent returns true if the data object contains event data
func (d *Data) HasEvent() bool {
	return d.Event != nil
}
