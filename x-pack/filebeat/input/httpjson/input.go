// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	inputName = "httpjson"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

type httpjsonInput struct {
	config

	log      *logp.Logger
	outlet   channel.Outleter // Output of received messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on worker goroutine.

}

type requestInfo struct {
	URL        string
	ContentMap common.MapStr
}

// NewInput creates a new misp input that consumes events from MISP with a configurable interval
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	// Extract and validate the input's configuration.
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	// Build outlet for events.
	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
	})
	if err != nil {
		return nil, err
	}

	// Wrap input.Context's Done channel with a context.Context. This goroutine
	// stops with the parent closes the Done channel.
	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Done:
		case <-inputCtx.Done():
		}
	}()

	// If the input ever needs to be made restartable, then context would need
	// to be recreated with each restart.
	workerCtx, workerCancel := context.WithCancel(inputCtx)

	in := &httpjsonInput{
		config: conf,
		log: logp.NewLogger("httpjson").With(
			"url", conf.URL),
		outlet:       out,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}

	in.log.Info("Initialized httpjson input.")
	return in, nil
}

// Run starts the misp input worker then returns. Only the first invocation
// will ever start the misp worker.
func (in *httpjsonInput) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.log.Info("httpjson input worker has started.")
			defer in.log.Info("httpjson input worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()
			if err := in.run(); err != nil {
				in.log.Error(err)
				return
			}
		}()
	})
}

// Create HTTP request for the input
func (in *httpjsonInput) createHTTPRequest(ctx context.Context, ri *requestInfo) (*http.Request, error) {
	b, _ := json.Marshal(ri.ContentMap)
	body := strings.NewReader(string(b))
	req, err := http.NewRequest(in.HTTPMethod, ri.URL, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "filebeat-input-httpjson")
	if in.APIKey != "" {
		req.Header.Set("Authorization", in.APIKey)
	}
	return req, nil
}

// Process HTTP request, recursively handle pagination if enable
func (in *httpjsonInput) processHTTPRequest(ctx context.Context, client *http.Client, req *http.Request, ri *requestInfo) error {
	msg, err := client.Do(req)
	if err != nil {
		in.log.Error(err)
		return errors.New("Failed to do http request. Stopping input worker - ")
	}
	if msg.StatusCode != http.StatusOK {
		e := fmt.Sprintf("HTTP return status is %s - ", msg.Status)
		in.log.Error(e)
		return errors.New(e)
	}
	responseData, err := ioutil.ReadAll(msg.Body)
	defer msg.Body.Close()
	if err != nil {
		in.log.Error(err)
		return err
	}
	var m, v interface{}
	err = json.Unmarshal(responseData, &m)
	if err != nil {
		in.log.Error(err)
		return err
	}
	switch m.(type) {
	case map[string]interface{}:
		break
	default:
		return errors.New("HTTP Response is not valid JSON - ")
	}
	if in.JSONObjects == "" {
		ok := in.outlet.OnEvent(makeEvent(string(responseData)))
		if !ok {
			return errors.New("OnEvent returned false - ")
		}
	} else {
		v, err = common.MapStr(m.(map[string]interface{})).GetValue(in.JSONObjects)
		if err != nil {
			in.log.Error(err)
			return err
		}
		switch v.(type) {
		case []interface{}:
			ts := v.([]interface{})
			for _, t := range ts {
				switch t.(type) {
				case map[string]interface{}:
					d, err := json.Marshal(t.(map[string]interface{}))
					if err != nil {
						in.log.Error(err)
						return errors.New("Failed to process http response data - ")
					}
					ok := in.outlet.OnEvent(makeEvent(string(d)))
					if !ok {
						in.log.Error(ok)
						return errors.New("OnEvent returned false - ")
					}
				default:
					e := "Invalid json_objects_array configuration"
					in.log.Error(e)
					return errors.New(e)
				}
			}
		default:
			e := "Invalid json_objects_array configuration"
			in.log.Error(e)
			return errors.New(e)
		}
	}
	if in.PaginationEnable {
		v, err = common.MapStr(m.(map[string]interface{})).GetValue(in.PaginationIdField)
		if err != nil {
			in.log.Info("Successfully processed HTTP request. Pagination finished.")
			return nil
		}
		if in.PaginationRequestField != "" {
			ri.ContentMap.Put(in.PaginationRequestField, v)
			if in.PaginationURL != "" {
				ri.URL = in.PaginationURL
			}
		} else {
			switch v.(type) {
			case string:
				ri.URL = v.(string)
			default:
				e := "Pagination ID is not string, which is required for URL - "
				in.log.Error(e)
				return errors.New(e)
			}
		}
		if in.PaginationExtraBodyContent != nil {
			switch in.PaginationExtraBodyContent.(type) {
			case map[string]interface{}:
				ri.ContentMap.Update(common.MapStr(in.PaginationExtraBodyContent.(map[string]interface{})))
			default:
			}
		}
		req, err = in.createHTTPRequest(ctx, ri)
		if err != nil {
			in.log.Error(err)
			return err
		}
		in.processHTTPRequest(ctx, client, req, ri)
	}
	return nil
}

func (in *httpjsonInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()

	// Make http client.
	var client *http.Client
	if in.ServerName == "" {
		in.log.Info("ServerName is empty, hence TLS will not be used.")
		client = &http.Client{
			Timeout:   time.Second * time.Duration(in.HTTPClientTimeout),
			Transport: &http.Transport{DisableKeepAlives: true},
		}
	} else {
		in.log.Info("ServerName is " + in.ServerName + ", and TLS  will be used.")
		client = &http.Client{
			Timeout: time.Second * time.Duration(in.HTTPClientTimeout),
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					ServerName: in.ServerName,
					// InsecureSkipVerify: true,
				},
				DisableKeepAlives: true,
			},
		}
	}
	ri := &requestInfo{
		URL:        in.URL,
		ContentMap: common.MapStr(make(map[string]interface{})),
	}
	if in.HTTPMethod == "POST" && in.HTTPRequestBody != nil {
		switch in.HTTPRequestBody.(type) {
		case map[string]interface{}:
			ri.ContentMap.Update(common.MapStr(in.HTTPRequestBody.(map[string]interface{})))
		default:
			in.log.Error("HTTPRequestBody configuration is wrong, hence ignored!")
		}
	}
	req, err := in.createHTTPRequest(ctx, ri)
	if err != nil {
		in.log.Error(err)
		return err
	}
	err = in.processHTTPRequest(ctx, client, req, ri)
	if err == nil && in.Interval > 0 {
		ticker := time.NewTicker(time.Duration(in.Interval) * time.Second)
		for {
			select {
			case <-ctx.Done():
				in.log.Info("Context done.")
				return nil
			case <-ticker.C:
				err = in.processHTTPRequest(ctx, client, req, ri)
			}
		}
	}
	return err
}

// Stop stops the misp input and waits for it to fully stop.
func (in *httpjsonInput) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *httpjsonInput) Wait() {
	in.Stop()
}

func makeEvent(body string) beat.Event {
	id := uuid.New().String()

	fields := common.MapStr{
		"event": common.MapStr{
			"id":      id,
			"created": time.Now().UTC(),
		},
		"message": body,
	}

	return beat.Event{
		Timestamp: time.Now().UTC(),
		Meta: common.MapStr{
			"id": id,
		},
		Fields: fields,
	}
}
