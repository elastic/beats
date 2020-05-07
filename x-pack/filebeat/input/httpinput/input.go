// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpinput

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
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
	inputName = "httpinput"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

type HttpInput struct {
	config
	log      *logp.Logger
	outlet   channel.Outleter // Output of received messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context         // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc      // Used to signal that the worker should stop.
	workerOnce   sync.Once               // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup          // Waits on worker goroutine.
	httpServer   *http.Server            // The currently running HTTP instance
	httpMux      *http.ServeMux          // Current HTTP Handler
	httpRequest  http.Request            // Current Request
	httpResponse http.ResponseWriter     // Current ResponseWriter
	eventObject  *map[string]interface{} // Current event object
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

	in := &HttpInput{
		config:       conf,
		log:          logp.NewLogger("httpinput"),
		outlet:       out,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}

	in.log.Info("Initialized httpinput input.")
	return in, nil
}

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (in *HttpInput) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.log.Info("httpinput worker has started.")
			defer in.log.Info("httpinput worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()
			if err := in.run(); err != nil {
				in.log.Error(err)
				return
			}
		}()
	})
}

func (in *HttpInput) run() error {
	var err error
	// Create worker context
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()

	// Initialize the HTTP server
	err = in.createServer()

	if err != nil && err != http.ErrServerClosed {
		in.log.Fatalf("HTTP Server could not start, error: %v", err)
	}

	// Infinite Loop waiting for agent to stop
	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}
	return err
}

// Stops HTTP input and waits for it to finish
func (in *HttpInput) Stop() {
	in.httpServer.Shutdown(in.workerCtx)
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *HttpInput) Wait() {
	in.Stop()
}

func (in *HttpInput) createServer() error {
	// Merge listening address and port
	var address strings.Builder
	address.WriteString(in.config.ListenAddress + ":" + in.config.ListenPort)

	in.httpMux = http.NewServeMux()
	in.httpMux.HandleFunc(in.config.URL, in.apiResponse)

	if in.config.UseSSL == true {
		in.httpServer = &http.Server{Addr: address.String(), Handler: in.httpMux}
		return in.httpServer.ListenAndServeTLS(in.config.SSLCertificate, in.config.SSLKey)
	}
	if in.config.UseSSL == false {
		in.httpServer = &http.Server{Addr: address.String(), Handler: in.httpMux}
		return in.httpServer.ListenAndServe()
	}
	return errors.New("SSL settings missing")
}

// Create a response to the request
func (in *HttpInput) apiResponse(w http.ResponseWriter, r *http.Request) {
	var err string
	var status uint

	// Storing for validation
	in.httpRequest = *r
	in.httpResponse = w

	// Validates request, writes response directly on error.
	status, err = in.createEvent()

	if err != "" || status != 0 {
		in.sendResponse(status, err)
		return
	}

	// On success, returns the configured response parameters
	in.sendResponse(http.StatusOK, in.config.ResponseBody)
}

func (in *HttpInput) createEvent() (uint, string) {
	var err string
	var status uint

	status, err = in.validateRequest()

	// Check if any of the validations failed, and if so, return them
	if err != "" || status != 0 {
		return status, err
	}

	// Create the event
	ok := in.outlet.OnEvent(beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"message":        "testing",
			in.config.Prefix: in.eventObject,
		},
	})

	// If event cannot be sent
	if !ok {
		return http.StatusInternalServerError, in.createErrorMessage("unable to send event")
	}

	return 0, ""
}

func (in *HttpInput) validateRequest() (uint, string) {
	// Only allow POST requests
	var err string
	var status uint

	// Check auth settings and credentials
	if in.config.BasicAuth == true {
		status, err = in.validateAuth()
	}

	if err != "" && status != 0 {
		return status, err
	}

	// Validate headers
	status, err = in.validateHeader()

	if err != "" && status != 0 {
		return status, err
	}

	// Validate body
	status, err = in.validateBody()

	if err != "" && status != 0 {
		return status, err
	}

	return 0, ""

}

func (in *HttpInput) validateHeader() (uint, string) {
	// Only allow JSON
	if in.httpRequest.Header.Get("Content-Type") != "application/json" {
		return http.StatusUnsupportedMediaType, in.createErrorMessage("wrong content-type header, expecting application/json")
	}

	// Only accept JSON in return
	if in.httpRequest.Header.Get("Accept") != "application/json" {
		return http.StatusNotAcceptable, in.createErrorMessage("wrong accept header, expecting application/json")
	}
	return 0, ""
}

func (in *HttpInput) validateAuth() (uint, string) {
	// Check if username or password is missing
	if in.config.Username == "" || in.config.Password == "" {
		return http.StatusUnauthorized, in.createErrorMessage("Username and password required when basicauth is enabled")
	}

	// Check if username and password combination is correct
	username, password, _ := in.httpRequest.BasicAuth()
	if in.config.Username != username || in.config.Password != password {
		return http.StatusUnauthorized, in.createErrorMessage("Incorrect username or password")
	}

	return 0, ""
}

func (in *HttpInput) validateBody() (uint, string) {
	// Checks if body is empty
	if in.httpRequest.Body == http.NoBody {
		return http.StatusNotAcceptable, in.createErrorMessage("body can not be empty")
	}

	// Write full []byte to string
	body, err := ioutil.ReadAll(in.httpRequest.Body)

	// If body cannot be read
	if err != nil {
		return http.StatusInternalServerError, in.createErrorMessage("unable to read body")
	}

	// Declare interface for request body
	objmap := make(map[string]interface{})

	err = json.Unmarshal(body, &objmap)

	// If body can be read, but not converted to JSON
	if err != nil {
		return http.StatusBadRequest, in.createErrorMessage("malformed JSON body")
	}
	// Assign the current Unmarshaled object when no errors
	in.eventObject = &objmap

	return 0, ""
}

func (in *HttpInput) validateMethod() (uint, string) {
	// Ensure HTTP method is POST
	if in.httpRequest.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, in.createErrorMessage("only POST requests supported")
	}

	return 0, ""
}

func (in *HttpInput) createErrorMessage(r string) string {
	return fmt.Sprintf(`{"message": "%v"}`, r)
}

func (in *HttpInput) sendResponse(h uint, b string) {
	in.httpResponse.WriteHeader(int(h))
	in.httpResponse.Write([]byte(b))
}
