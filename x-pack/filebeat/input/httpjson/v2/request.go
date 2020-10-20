package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms/append"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms/delete"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/v2/internal/transforms/set"
)

const requestNamespace = "request"

func registerRequestTransforms() {
	transforms.RegisterTransform(requestNamespace, set.Name, set.New)
	transforms.RegisterTransform(requestNamespace, append.Name, append.New)
	transforms.RegisterTransform(requestNamespace, delete.Name, delete.New)
}

type requestFactory struct {
	url        url.URL
	method     string
	body       *common.MapStr
	transforms []transforms.Transform
	user       string
	password   string
	log        *logp.Logger
}

func newRequestFactory(config *requestConfig, authConfig *authConfig, log *logp.Logger) *requestFactory {
	// config validation already checked for errors here
	ts, _ := transforms.New(config.Transforms, requestNamespace)
	rf := &requestFactory{
		url:        *config.URL.URL,
		method:     config.Method,
		body:       config.Body,
		transforms: ts.List,
		log:        log,
	}
	if authConfig != nil && authConfig.Basic.isEnabled() {
		rf.user = authConfig.Basic.User
		rf.password = authConfig.Basic.Password
	}
	return rf
}

func (rf *requestFactory) newRequest(ctx context.Context) (*http.Request, error) {
	var err error

	trReq := transforms.NewEmptyTransformable()

	clonedURL, err := url.Parse(rf.url.String())
	if err != nil {
		return nil, err
	}
	trReq.URL = *clonedURL

	if rf.body != nil {
		trReq.Body = rf.body.Clone()
	}

	for _, t := range rf.transforms {
		trReq, err = t.Run(trReq)
		if err != nil {
			return nil, err
		}
	}

	var body []byte
	if len(trReq.Body) > 0 {
		switch rf.method {
		case "POST":
			body, err = json.Marshal(trReq.Body)
			if err != nil {
				return nil, err
			}
		default:
			rf.log.Errorf("A body is set, but method is not POST. The body will be ignored.")
		}
	}

	req, err := http.NewRequest(rf.method, trReq.URL.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	req.Header = trReq.Headers
	req.Header.Set("Accept", "application/json")
	if rf.method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", userAgent)

	if rf.user != "" || rf.password != "" {
		req.SetBasicAuth(rf.user, rf.password)
	}

	return req, nil
}

type requester struct {
	log               *logp.Logger
	client            *http.Client
	requestFactory    *requestFactory
	responseProcessor *responseProcessor
}

func newRequester(client *http.Client, requestFactory *requestFactory, responseProcessor *responseProcessor, log *logp.Logger) *requester {
	return &requester{
		log:               log,
		client:            client,
		requestFactory:    requestFactory,
		responseProcessor: responseProcessor,
	}
}

func (r *requester) processRequest(ctx context.Context, publisher cursor.Publisher) error {
	req, err := r.requestFactory.newRequest(ctx)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute http client.Do: %w", err)
	}

	events, err := r.responseProcessor.getEventsFromResponse(ctx, resp)
	if err != nil {
		return err
	}

	for e := range events {
		if e.failed() {
			r.log.Errorf("failed to create event: %v", e.err)
			continue
		}

		if err := publisher.Publish(e.event, nil); err != nil {
			return err
		}
	}

	return nil
}
