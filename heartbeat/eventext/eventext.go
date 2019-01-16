package eventext

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// MergeEventFields merges the given common.MapStr into the given Event's Fields.
func MergeEventFields(e *beat.Event, merge common.MapStr) {
	if e.Fields != nil {
		e.Fields.DeepUpdate(merge)
	} else {
		e.Fields = merge
	}
}
