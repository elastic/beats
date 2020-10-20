package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms/append"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms/delete"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms/set"
)

const (
	responseNamespace = "response"
)

func registerResponseTransforms() {
	transforms.RegisterTransform(responseNamespace, set.Name, set.New)
	transforms.RegisterTransform(responseNamespace, append.Name, append.New)
	transforms.RegisterTransform(responseNamespace, delete.Name, delete.New)
}

type responseProcessor struct {
	log        *logp.Logger
	transforms []transforms.Transform
}

func newResponseProcessor(config *responseConfig) *responseProcessor {
	rp := &responseProcessor{}
	if config == nil {
		return rp
	}

	tr, _ := transforms.New(config.Transforms, responseNamespace)
	rp.transforms = tr.List

	return rp
}

type maybeEvent struct {
	event beat.Event
	err   error
}

func (e maybeEvent) failed() bool {
	return e.err != nil
}

func (rp *responseProcessor) getEventsFromResponse(ctx context.Context, resp *http.Response) (<-chan maybeEvent, error) {
	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read http response: %w", err)
	}

	var m common.MapStr
	if err := json.Unmarshal(responseData, &m); err != nil {
		return nil, err
	}

	trResp := transforms.NewEmptyTransformable()
	trResp.Body = m
	trResp.Headers = resp.Header.Clone()
	trResp.URL = *resp.Request.URL

	return rp.run(ctx, trResp), nil
}

func (rp *responseProcessor) run(ctx context.Context, trResp *transforms.Transformable) <-chan maybeEvent {
	ch := make(chan maybeEvent)

	go func() {
		defer close(ch)
		var err error
		for _, tr := range rp.transforms {
			select {
			case <-ctx.Done():
				return
			default:
			}

			trResp, err = tr.Run(trResp)
			if err != nil {
				rp.log.Errorf("error running transform: %v", err)
				continue
			}

			b, err := json.Marshal(trResp.Body)
			if err != nil {
				ch <- maybeEvent{err: err}
				continue
			}

			ch <- maybeEvent{event: makeEvent(string(b))}
		}
	}()

	return ch
}
