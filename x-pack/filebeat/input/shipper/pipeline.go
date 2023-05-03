package shipper

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

func createClientAndSend(pipeline beat.Pipeline, datastreams map[string]mapstr.M, event *messages.Event) error {
	return nil
}
