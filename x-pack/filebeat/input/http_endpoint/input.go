// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	inputName = "http_endpoint"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

type HttpEndpoint struct {
	config
	log      *logp.Logger
	outlet   channel.Outleter // Output of received messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context         // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc      // Used to signal that the worker should stop.
	workerOnce   sync.Once               // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup          // Waits on worker goroutine.
	server       *HttpServer             // Server instance
	eventObject  *map[string]interface{} // Current event object
	finalHandler http.HandlerFunc
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
	out, err := connector.Connect(cfg)
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

	in := &HttpEndpoint{
		config:       conf,
		log:          logp.NewLogger(inputName),
		outlet:       out,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}

	// Create an instance of the HTTP server with the beat context
	in.server, err = createServer(in)
	if err != nil {
		return nil, err
	}

	in.log.Infof("Initialized %v input on %v:%v", inputName, in.config.ListenAddress, in.config.ListenPort)

	return in, nil
}

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (in *HttpEndpoint) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go in.run()
	})
}

func (in *HttpEndpoint) run() {
	defer in.workerWg.Done()
	defer in.log.Infof("%v worker has stopped.", inputName)
	in.server.Start()
}

// Stops HTTP input and waits for it to finish
func (in *HttpEndpoint) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *HttpEndpoint) Wait() {
	in.Stop()
}

// If middleware validation successed, event is sent
func (in *HttpEndpoint) sendEvent(w http.ResponseWriter, r *http.Request) {
	event := in.outlet.OnEvent(beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			in.config.Prefix: in.eventObject,
		},
	})
	if !event {
		in.sendResponse(w, http.StatusInternalServerError, in.createErrorMessage("Unable to send event"))
	}
}

// Triggers if middleware validation returns successful
func (in *HttpEndpoint) apiResponse(w http.ResponseWriter, r *http.Request) {
	in.sendEvent(w, r)
	w.Header().Add("Content-Type", "application/json")
	in.sendResponse(w, uint(in.config.ResponseCode), in.config.ResponseBody)
}

func (in *HttpEndpoint) sendResponse(w http.ResponseWriter, h uint, b string) {
	w.WriteHeader(int(h))
	w.Write([]byte(b))
}

// Runs all validations for each request
func (in *HttpEndpoint) validateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if in.config.BasicAuth {
			status, err := in.validateAuth(w, r)
			if err != "" && status != 0 {
				in.sendResponse(w, status, err)
				return
			}
		}

		status, err := in.validateMethod(w, r)
		if err != "" && status != 0 {
			in.sendResponse(w, status, err)
			return
		}

		status, err = in.validateHeader(w, r)
		if err != "" && status != 0 {
			in.sendResponse(w, status, err)
			return
		}

		status, err = in.validateBody(w, r)
		if err != "" && status != 0 {
			in.sendResponse(w, status, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Validate that only supported Accept and Content type headers are used
func (in *HttpEndpoint) validateHeader(w http.ResponseWriter, r *http.Request) (uint, string) {
	if r.Header.Get("Content-Type") != "application/json" {
		return http.StatusUnsupportedMediaType, in.createErrorMessage("Wrong Content-Type header, expecting application/json")
	}

	return 0, ""
}

// Validate if headers are current and authentication is successful
func (in *HttpEndpoint) validateAuth(w http.ResponseWriter, r *http.Request) (uint, string) {
	if in.config.Username == "" || in.config.Password == "" {
		return http.StatusUnauthorized, in.createErrorMessage("Username and password required when basicauth is enabled")
	}

	username, password, _ := r.BasicAuth()
	if in.config.Username != username || in.config.Password != password {
		return http.StatusUnauthorized, in.createErrorMessage("Incorrect username or password")
	}

	return 0, ""
}

// Validates that body is not empty, not a list of objects and valid JSON
func (in *HttpEndpoint) validateBody(w http.ResponseWriter, r *http.Request) (uint, string) {
	if r.Body == http.NoBody {
		return http.StatusNotAcceptable, in.createErrorMessage("Body cannot be empty")
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError, in.createErrorMessage("Unable to read body")
	}

	isObject := in.isObjectOrList(body)
	if isObject == "list" {
		return http.StatusBadRequest, in.createErrorMessage("List of JSON objects is not supported")
	}

	objmap := make(map[string]interface{})
	err = json.Unmarshal(body, &objmap)
	if err != nil {
		return http.StatusBadRequest, in.createErrorMessage("Malformed JSON body")
	}

	in.eventObject = &objmap

	return 0, ""
}

// Ensure only valid HTTP Methods used
func (in *HttpEndpoint) validateMethod(w http.ResponseWriter, r *http.Request) (uint, string) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, in.createErrorMessage("Only POST requests supported")
	}

	return 0, ""
}

func (in *HttpEndpoint) createErrorMessage(r string) string {
	return fmt.Sprintf(`{"message": "%v"}`, r)
}

func (in *HttpEndpoint) isObjectOrList(b []byte) string {
	obj := bytes.TrimLeft(b, " \t\r\n")
	if len(obj) > 0 && obj[0] == '{' {
		return "object"
	}

	if len(obj) > 0 && obj[0] == '[' {
		return "list"
	}

	return ""
}
