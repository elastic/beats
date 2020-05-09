// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

func init() {
	outputs.RegisterType("http", makeHTTP)
}

type httpOutput struct {
	log       *logp.Logger
	beat      beat.Info
	observer  outputs.Observer
	codec     codec.Codec
	client    *http.Client
	serialize func(event *publisher.Event) ([]byte, error)
	reqPool   sync.Pool
	conf      config
}

// makeHTTP instantiates a new http output instance.
func makeHTTP(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	ho := &httpOutput{
		log:      logp.NewLogger("http"),
		beat:     beat,
		observer: observer,
		conf:     config,
	}

	// disable bulk support in publisher pipeline
	if err := cfg.SetInt("bulk_max_size", -1, -1); err != nil {
		ho.log.Error("Disable bulk error: ", err)
	}

	//select serializer
	ho.serialize = ho.serializeAll

	if config.OnlyFields {
		ho.serialize = ho.serializeOnlyFields
	}

	// init output
	if err := ho.init(beat, config); err != nil {
		return outputs.Fail(err)
	}

	return outputs.Success(-1, config.MaxRetries, ho)
}

func (out *httpOutput) init(beat beat.Info, c config) error {
	var err error

	out.codec, err = codec.CreateEncoder(beat, c.Codec)
	if err != nil {
		return err
	}

	tr := &http.Transport{
		MaxIdleConns:          out.conf.MaxIdleConns,
		ResponseHeaderTimeout: time.Duration(out.conf.ResponseHeaderTimeout) * time.Millisecond,
		IdleConnTimeout:       time.Duration(out.conf.IdleConnTimeout) * time.Second,
		DisableCompression:    !out.conf.Compression,
		DisableKeepAlives:     !out.conf.KeepAlive,
	}

	out.client = &http.Client{
		Transport: tr,
	}

	out.reqPool = sync.Pool{
		New: func() interface{} {
			req, err := http.NewRequest("POST", out.conf.URL, nil)
			if err != nil {
				return err
			}
			return req
		},
	}

	out.log.Infof("Initialized http output:\n"+
		"url=%v\n"+
		"codec=%v\n"+
		"only_fields=%v\n"+
		"max_retries=%v\n"+
		"compression=%v\n"+
		"keep_alive=%v\n"+
		"max_idle_conns=%v\n"+
		"idle_conn_timeout=%vs\n"+
		"response_header_timeout=%vms\n"+
		"username=%v\n"+
		"password=%v\n",
		c.URL, c.Codec, c.OnlyFields, c.MaxRetries, c.Compression,
		c.KeepAlive, c.MaxIdleConns, c.IdleConnTimeout, c.ResponseHeaderTimeout,
		c.Username, maskPass(c.Password))
	return nil
}

func maskPass(password string) string {
	result := ""
	if len(password) <= 8 {
		for i := 0; i < len(password); i++ {
			result += "*"
		}
		return result
	}

	for i, char := range password {
		if i > 1 && i < len(password)-2 {
			result += "*"
		} else {
			result += string(char)
		}
	}

	return result
}

// Implement Client
func (out *httpOutput) Close() error {
	out.client.CloseIdleConnections()
	return nil
}

func (out *httpOutput) serializeOnlyFields(event *publisher.Event) ([]byte, error) {
	serializedEvent, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(&event.Content.Fields)
	if err != nil {
		out.log.Error("Serialization error: ", err)
		return make([]byte, 0), err
	}
	return serializedEvent, nil
}

func (out *httpOutput) serializeAll(event *publisher.Event) ([]byte, error) {
	serializedEvent, err := out.codec.Encode(out.beat.Beat, &event.Content)
	if err != nil {
		out.log.Error("Serialization error: ", err)
		return make([]byte, 0), err
	}
	return serializedEvent, nil
}

func (out *httpOutput) Publish(_ context.Context, batch publisher.Batch) error {
	st := out.observer
	events := batch.Events()
	st.NewBatch(len(events))

	if len(events) == 0 {
		batch.ACK()
		return nil
	}

	for i := range events {
		event := events[i]

		serializedEvent, err := out.serialize(&event)

		if err != nil {
			if event.Guaranteed() {
				out.log.Errorf("Failed to serialize the event: %+v", err)
			} else {
				out.log.Warnf("Failed to serialize the event: %+v", err)
			}
			out.log.Debugf("Failed event: %v", event)

			batch.RetryEvents(events)
			st.Failed(len(events))
			return nil
		}

		if err = out.send(serializedEvent); err != nil {
			if event.Guaranteed() {
				out.log.Errorf("Writing event to http failed with: %+v", err)
			} else {
				out.log.Warnf("Writing event to http failed with: %+v", err)
			}

			batch.RetryEvents(events)
			st.Failed(len(events))
			return nil
		}
	}

	batch.ACK()
	st.Acked(len(events))
	return nil
}

func (out *httpOutput) String() string {
	return "http(" + out.conf.URL + ")"
}

func (out *httpOutput) send(data []byte) error {

	req, err := out.getReq(data)
	if err != nil {
		return err
	}
	defer out.putReq(req)

	resp, err := out.client.Do(req)
	if err != nil {
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		out.log.Warn("Close response body error:", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response code: %d", resp.StatusCode)
	}

	return nil
}

func (out *httpOutput) getReq(data []byte) (*http.Request, error) {
	tmp := out.reqPool.Get()

	req, ok := tmp.(*http.Request)
	if ok {
		buf := bytes.NewBuffer(data)
		req.Body = ioutil.NopCloser(buf)
		req.Header.Set("User-Agent", "beat "+out.beat.Version)
		if out.conf.Username != "" {
			req.SetBasicAuth(out.conf.Username, out.conf.Password)
		}
		return req, nil
	}

	err, ok := tmp.(error)
	if ok {
		return nil, err
	}

	return nil, errors.New("pool assertion error")
}

func (out *httpOutput) putReq(req *http.Request) {
	out.reqPool.Put(req)
}
