// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpinput

import (
	"context"
	"encoding/json"
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

	workerCtx    context.Context     // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc  // Used to signal that the worker should stop.
	workerOnce   sync.Once           // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup      // Waits on worker goroutine.
	httpServer   *http.Server        // The currently running HTTP instance
	httpMux      *http.ServeMux      // Current HTTP Handler
	httpRequest  http.Request        // Current Request
	httpResponse http.ResponseWriter // Current ResponseWriter
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

// Stop stops the misp input and waits for it to fully stop.
func (in *HttpInput) Stop() {
	in.httpServer.Shutdown(in.workerCtx)
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *HttpInput) Wait() {
	in.Stop()
}

// Create a response to the request
func (in *HttpInput) apiResponse(w http.ResponseWriter, r *http.Request) {

	// Storing for validation
	in.httpRequest = *r
	in.httpResponse = w

	// Validates request, writes response directly on error.
	objmap := in.validateRequest()

	if objmap == nil || len(objmap) == 0 {
		in.log.Error("Request could not be processed")
		return
	}
	ok := in.outlet.OnEvent(beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"message": "testing",
			in.config.Prefix:    objmap,
		},
	})

	if !ok {
		return
	}

	// On success, returns the configured response parameters
	w.Write([]byte(in.config.ResponseBody))
	w.WriteHeader(in.config.ResponseCode)
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

func (in *HttpInput) validateRequest() map[string]interface{} {
	// Check auth settings and credentials
	if in.config.BasicAuth == true {
		if in.config.Username == "" || in.config.Password == "" {
			in.log.Fatal("Username and password required when basicauth is enabled")
			return nil
		}

		username, password, _ := in.httpRequest.BasicAuth()
		if in.config.Username != username || in.config.Password != password {
			in.httpResponse.WriteHeader(http.StatusUnauthorized)
			in.httpResponse.Write([]byte(`{"message": "Incorrect username or password"}`))
			return nil
		}
	}

	// Only allow POST requests
	if in.httpRequest.Method != http.MethodPost {
		in.httpResponse.WriteHeader(http.StatusMethodNotAllowed)
		in.httpResponse.Write([]byte(`{"message": "only post request supported"}`))
		return nil
	}

	// Only allow JSON
	if in.httpRequest.Header.Get("Content-Type") != "application/json" {
		in.httpResponse.WriteHeader(http.StatusUnsupportedMediaType)
		in.httpResponse.Write([]byte(`{"message": "wrong content-type header"}`))
		return nil
	}

	// Only accept JSON in return
	if in.httpRequest.Header.Get("Accept") != "application/json" {
		in.httpResponse.WriteHeader(http.StatusNotAcceptable)
		in.httpResponse.Write([]byte(`{"message": "wrong accept header"}`))
		return nil
	}

	if in.httpRequest.Body == http.NoBody {
		in.httpResponse.WriteHeader(http.StatusNotAcceptable)
		in.httpResponse.Write([]byte(`{"message": "empty body"}`))
		return nil
	}

	// Write full []byte to string
	body, err := ioutil.ReadAll(in.httpRequest.Body)

	// If body cannot be read
	if err != nil {
		in.httpResponse.WriteHeader(http.StatusInternalServerError)
		in.httpResponse.Write([]byte(`{"message": "failure"}`))
		return nil
	}

	// Declare interface for request body
	objmap := make(map[string]interface{})

	err = json.Unmarshal(body, &objmap)

	// If body can be read, but not converted to JSON
	if err != nil {
		in.httpResponse.WriteHeader(http.StatusBadRequest)
		in.httpResponse.Write([]byte(`{"message": "malformed JSON body"}`))
		return nil
	}
	return objmap
}
