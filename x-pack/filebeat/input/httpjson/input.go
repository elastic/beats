// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/common/useragent"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs/transport"
)

const (
	inputName = "httpjson"
)

var userAgent = useragent.UserAgent("Filebeat")

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
	Headers    common.MapStr
}

// NewInput creates a new httpjson input
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

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
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

// createHTTPRequest creates an HTTP/HTTPs request for the input
func (in *httpjsonInput) createHTTPRequest(ctx context.Context, ri *requestInfo) (*http.Request, error) {
	b, _ := json.Marshal(ri.ContentMap)
	body := bytes.NewReader(b)
	req, err := http.NewRequest(in.config.HTTPMethod, ri.URL, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if in.config.APIKey != "" {
		req.Header.Set("Authorization", in.config.APIKey)
	}
	for k, v := range ri.Headers {
		switch vv := v.(type) {
		case string:
			req.Header.Set(k, vv)
		default:
		}
	}
	return req, nil
}

// processHTTPRequest processes HTTP request, and handles pagination if enabled
func (in *httpjsonInput) processHTTPRequest(ctx context.Context, client *http.Client, ri *requestInfo) error {
	for {
		req, err := in.createHTTPRequest(ctx, ri)
		if err != nil {
			return errors.Wrapf(err, "failed to create http request")
		}
		msg, err := client.Do(req)
		if err != nil {
			return errors.Wrapf(err, "failed to execute http client.Do")
		}
		responseData, err := ioutil.ReadAll(msg.Body)
		msg.Body.Close()
		if err != nil {
			return errors.Wrapf(err, "failed to read http.response.body")
		}
		if msg.StatusCode != http.StatusOK {
			in.log.Debugw("HTTP request failed", "http.response.status_code", msg.StatusCode, "http.response.body", string(responseData))
			return errors.Errorf("http request was unsuccessful with a status code %d", msg.StatusCode)
		}
		var m, v interface{}
		err = json.Unmarshal(responseData, &m)
		if err != nil {
			return errors.Wrapf(err, "failed to unmarshal http.response.body")
		}
		switch mmap := m.(type) {
		case map[string]interface{}:
			if in.config.JSONObjects == "" {
				ok := in.outlet.OnEvent(makeEvent(string(responseData)))
				if !ok {
					return errors.New("function OnEvent returned false")
				}
			} else {
				v, err = common.MapStr(mmap).GetValue(in.config.JSONObjects)
				if err != nil {
					return err
				}
				switch ts := v.(type) {
				case []interface{}:
					for _, t := range ts {
						switch tv := t.(type) {
						case map[string]interface{}:
							d, err := json.Marshal(tv)
							if err != nil {
								return errors.Wrapf(err, "failed to marshal json_objects_array")
							}
							ok := in.outlet.OnEvent(makeEvent(string(d)))
							if !ok {
								return errors.New("function OnEvent returned false")
							}
						default:
							return errors.New("invalid json_objects_array configuration")
						}
					}
				default:
					return errors.New("invalid json_objects_array configuration")
				}
			}
			if in.config.Pagination != nil && in.config.Pagination.IsEnabled {
				v, err = common.MapStr(mmap).GetValue(in.config.Pagination.IDField)
				if err != nil {
					in.log.Info("Successfully processed HTTP request. Pagination finished.")
					return nil
				}
				if in.config.Pagination.RequestField != "" {
					ri.ContentMap.Put(in.config.Pagination.RequestField, v)
					if in.config.Pagination.URL != "" {
						ri.URL = in.config.Pagination.URL
					}
				} else {
					switch v.(type) {
					case string:
						ri.URL = v.(string)
					default:
						return errors.New("pagination ID is not of string type")
					}
				}
				if in.config.Pagination.ExtraBodyContent != nil {
					ri.ContentMap.Update(common.MapStr(in.config.Pagination.ExtraBodyContent))
				}
				continue
			}
			return nil
		default:
			in.log.Debugw("http.response.body is not valid JSON", string(responseData))
			return errors.New("http.response.body is not valid JSON")
		}
	}
}

func (in *httpjsonInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()

	tlsConfig, err := tlscommon.LoadTLSConfig(in.config.TLS)
	if err != nil {
		return err
	}

	var dialer, tlsDialer transport.Dialer

	dialer = transport.NetDialer(in.config.HTTPClientTimeout)
	tlsDialer, err = transport.TLSDialer(dialer, tlsConfig, in.config.HTTPClientTimeout)
	if err != nil {
		return err
	}

	// Make transport client
	var client *http.Client
	client = &http.Client{
		Transport: &http.Transport{
			Dial:              dialer.Dial,
			DialTLS:           tlsDialer.Dial,
			TLSClientConfig:   tlsConfig.ToConfig(),
			DisableKeepAlives: true,
		},
		Timeout: in.config.HTTPClientTimeout,
	}

	ri := &requestInfo{
		URL:        in.URL,
		ContentMap: common.MapStr{},
		Headers:    in.HTTPHeaders,
	}
	if in.config.HTTPMethod == "POST" && in.config.HTTPRequestBody != nil {
		ri.ContentMap.Update(common.MapStr(in.config.HTTPRequestBody))
	}
	err = in.processHTTPRequest(ctx, client, ri)
	if err == nil && in.Interval > 0 {
		ticker := time.NewTicker(in.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				in.log.Info("Context done.")
				return nil
			case <-ticker.C:
				in.log.Info("Process another repeated request.")
				err = in.processHTTPRequest(ctx, client, ri)
				if err != nil {
					return err
				}
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
	fields := common.MapStr{
		"event": common.MapStr{
			"created": time.Now().UTC(),
		},
		"message": body,
	}

	return beat.Event{
		Timestamp: time.Now().UTC(),
		Fields:    fields,
	}
}
