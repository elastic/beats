package synthexec

import (
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"net/url"
	"time"
)

type SynthEvent struct {
	Type           string                 `json:"type"`
	PackageVersion string                 `json:"package_version"`
	Index          int                    `json:"index""`
	Step           *Step                   `json:"step"`
	Journey        *Journey                `json:"journey"`
	Timestamp      time.Time              `json:"@timestamp"`
	Payload        map[string]interface{} `json:"payload"`
	Blob           *string                 `json:"blob"`
	Error          *SynthError                 `json:"error"`
	URL            *string                 `json:"url"`
}

type SynthError struct {
	Name string `json:"name"`
	Message string `json:"message"`
	Stack string `json:"stack"`
}

func (se *SynthError) String() string {
	return fmt.Sprintf("%s: %s\n%s", se.Name, se.Message, se.Stack)
}

func (se SynthEvent) ToMap() common.MapStr {
	// We don't add @timestamp to the map string since that's specially handled in beat.Event
	e := common.MapStr{
		"type": se.Type,
		"package_version": se.PackageVersion,
		"index": se.Index,
		"payload": se.Payload,
		"blob": se.Blob,
	}
	if se.Step != nil {
		e.Put("step", se.Step.ToMap())
	}
	if se.Journey != nil {
		e.Put("journey", se.Journey.ToMap())
	}
	m := common.MapStr{"synthetics": e}
	if se.Error != nil {
		m["error"] = common.MapStr{
			"type": "synthetics",
			"message": se.Error.String(),
		}
	}
	if se.URL != nil {
		u, e := url.Parse(*se.URL)
		if e != nil {
			logp.Warn("Could not parse synthetics URL '%s': %s", *se.URL, e.Error())
		} else {
			m["url"] = wrappers.URLFields(u)
		}
	}

	return m
}

type Step struct {
	Name string `json:"name"`
	Index int `json:"index"`
}

func (s *Step) ToMap() common.MapStr {
	return common.MapStr{
		"name": s.Name,
		"index": s.Index,
	}
}

type Journey struct {
	Name string `json:"name"`
	Id string `json:"id"`
}

func (j Journey) ToMap() common.MapStr {
	return common.MapStr{
		"name": j.Name,
		"id": j.Id,
	}
}
