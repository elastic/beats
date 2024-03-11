// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/g8rswimmer/go-sfdc"
	"github.com/g8rswimmer/go-sfdc/credentials"
	"github.com/g8rswimmer/go-sfdc/session"
	"github.com/g8rswimmer/go-sfdc/soql"
	"github.com/golang-jwt/jwt"
	"golang.org/x/exp/slices"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-concert/ctxtool"
)

const (
	inputName         = "salesforce"
	formatRFC3339Like = "2006-01-02T15:04:05.999Z"
)

type salesforceInput struct {
	ctx           context.Context
	publisher     inputcursor.Publisher
	cancel        context.CancelCauseFunc
	cursor        *state
	srcConfig     *config
	sfdcConfig    *sfdc.Configuration
	log           *logp.Logger
	clientSession *session.Session
	soqlr         *soql.Resource
	config
}

// // The Filebeat user-agent is provided to the program as useragent.
// var userAgent = useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String())

// Plugin returns the input plugin.
func Plugin(log *logp.Logger, store inputcursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:      inputName,
		Stability: feature.Stable,
		Manager:   NewInputManager(log, store),
	}
}

func (s *salesforceInput) Name() string { return inputName }

func (s *salesforceInput) Test(_ inputcursor.Source, _ v2.TestContext) error {
	return nil
}

// Run starts the input and blocks until it ends completes. It will return on
// context cancellation or type invalidity errors, any other error will be retried.
func (s *salesforceInput) Run(env v2.Context, src inputcursor.Source, cursor inputcursor.Cursor, pub inputcursor.Publisher) (err error) {
	st := &state{}
	if !cursor.IsNew() {
		if err = cursor.Unpack(&st); err != nil {
			return err
		}
	}

	if err = s.Setup(env, src, st, pub); err != nil {
		return err
	}

	return s.run()
}

// Setup sets up the input. It will create a new SOQL resource and all other
// necessary configurations.
func (s *salesforceInput) Setup(env v2.Context, src inputcursor.Source, cursor *state, pub inputcursor.Publisher) (err error) {
	cfg := src.(*source).cfg

	ctx := ctxtool.FromCanceller(env.Cancelation)
	childCtx, cancel := context.WithCancelCause(ctx)

	s.srcConfig = &cfg
	s.ctx = childCtx
	s.cancel = cancel
	s.publisher = pub
	s.cursor = cursor
	s.log = env.Logger.With("input_url", cfg.URL)
	s.sfdcConfig, err = getSFDCConfig(&cfg)
	if err != nil {
		return fmt.Errorf("error with configuration: %w", err)
	}

	s.soqlr, err = s.SetupSFClientConnection() // create a new SOQL resource
	if err != nil {
		return fmt.Errorf("error setting up connection to Salesforce: %w", err)
	}

	return nil
}

// run is the main loop of the input. It will run until the context is cancelled
// and based on the configuration, it will run the different methods -- EventLogFile
// or Object to collect events at defined intervals.
func (s *salesforceInput) run() error {
	if s.srcConfig.EventMonitoringMethod.EventLogFile.isEnabled() {
		err := s.RunEventLogFile()
		if err != nil {
			s.log.Errorf("Problem running EventLogFile collection: %s", err)
		}
	}

	if s.srcConfig.EventMonitoringMethod.Object.isEnabled() {
		err := s.RunObject()
		if err != nil {
			s.log.Errorf("Problem running Object collection: %s", err)
		}
	}

	eventLogFileTicker, objectMethodTicker := &time.Ticker{}, &time.Ticker{}
	eventLogFileTicker.C, objectMethodTicker.C = nil, nil

	if s.srcConfig.EventMonitoringMethod.EventLogFile.isEnabled() {
		eventLogFileTicker = time.NewTicker(s.srcConfig.EventMonitoringMethod.EventLogFile.Interval)
		defer eventLogFileTicker.Stop()
	}

	if s.srcConfig.EventMonitoringMethod.Object.isEnabled() {
		objectMethodTicker = time.NewTicker(s.srcConfig.EventMonitoringMethod.Object.Interval)
		defer objectMethodTicker.Stop()
	}

	for {
		// Always check for cancel first, to not accidentally trigger another
		// run if the context is already cancelled, but we have already received
		// another ticker making the channel ready.
		select {
		case <-s.ctx.Done():
			return s.isError(s.ctx.Err())
		default:
		}

		select {
		case <-s.ctx.Done():
			return s.isError(s.ctx.Err())
		case <-eventLogFileTicker.C:
			if err := s.RunEventLogFile(); err != nil {
				s.log.Errorf("Problem running EventLogFile collection: %s", err)
			}
		case <-objectMethodTicker.C:
			if err := s.RunObject(); err != nil {
				s.log.Errorf("Problem running Object collection: %s", err)
			}
		}
	}
}

func (s *salesforceInput) isError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		s.log.Infof("input stopped because context was cancelled with: %v", err)
		return nil
	}

	return err
}

func (s *salesforceInput) SetupSFClientConnection() (*soql.Resource, error) {
	if s.sfdcConfig == nil {
		return nil, errors.New("internal error: salesforce configuration is not set properly")
	}

	// Open creates a session using the configuration.
	session, err := session.Open(*s.sfdcConfig)
	if err != nil {
		return nil, err
	}

	// set clientSession for re-use (EventLogFile)
	s.clientSession = session

	// Create a new SOQL resource using the session.
	soqlr, err := soql.NewResource(session)
	if err != nil {
		return nil, fmt.Errorf("error setting up salesforce SOQL resource: %w", err)
	}
	return soqlr, nil
}

// FormQueryWithCursor takes a queryConfig and a cursor and returns a querier.
func (s *salesforceInput) FormQueryWithCursor(queryConfig *QueryConfig, cursor mapstr.M) (*querier, error) {
	qr, err := parseCursor(queryConfig, cursor, s.log)
	if err != nil {
		return nil, err
	}

	s.log.Infof("Salesforce query: %s", qr)

	return &querier{Query: qr}, err
}

// RunObject runs the Object method of the Event Monitoring API to collect events.
func (s *salesforceInput) RunObject() error {
	s.log.Debugf("Scrape Objects every %s", s.srcConfig.EventMonitoringMethod.Object.Interval)

	var cursor mapstr.M
	if !(s.cursor.Object.FirstEventTime == "" && s.cursor.Object.LastEventTime == "") {
		object := make(mapstr.M)
		if s.cursor.Object.FirstEventTime != "" {
			object.Put("first_event_time", s.cursor.Object.FirstEventTime)
		}
		if s.cursor.Object.LastEventTime != "" {
			object.Put("last_event_time", s.cursor.Object.LastEventTime)
		}
		cursor = mapstr.M{"object": object}
	}
	query, err := s.FormQueryWithCursor(s.config.EventMonitoringMethod.Object.Query, cursor)
	if err != nil {
		return fmt.Errorf("error forming query based on cursor: %w", err)
	}

	res, err := s.soqlr.Query(query, false)
	if err != nil {
		return err
	}

	totalEvents := 0
	firstEvent := true

	for res.TotalSize() > 0 {
		for _, rec := range res.Records() {
			val := rec.Record().Fields()

			jsonStrEvent, err := json.Marshal(val)
			if err != nil {
				return err
			}

			if timestamp, ok := val[s.config.EventMonitoringMethod.Object.Cursor.Field].(string); ok {
				if firstEvent {
					s.cursor.Object.FirstEventTime = timestamp
				}
				s.cursor.Object.LastEventTime = timestamp
			}

			err = publishEvent(s.publisher, s.cursor, jsonStrEvent, "Object")
			if err != nil {
				return err
			}
			firstEvent = false
			totalEvents++
		}

		if !res.MoreRecords() { // returns true if there are more records.
			break
		}

		res, err = res.Next()
		if err != nil {
			return err
		}
	}
	s.log.Debugf("Total events: %d", totalEvents)

	return nil
}

// RunEventLogFile runs the EventLogFile method of the Event Monitoring API to
// collect events.
func (s *salesforceInput) RunEventLogFile() error {
	s.log.Debugf("Scrape EventLogFiles every %s", s.srcConfig.EventMonitoringMethod.EventLogFile.Interval)

	var cursor mapstr.M
	if !(s.cursor.EventLogFile.FirstEventTime == "" && s.cursor.EventLogFile.LastEventTime == "") {
		eventLogFile := make(mapstr.M)
		if s.cursor.EventLogFile.FirstEventTime != "" {
			eventLogFile.Put("first_event_time", s.cursor.EventLogFile.FirstEventTime)
		}
		if s.cursor.EventLogFile.LastEventTime != "" {
			eventLogFile.Put("last_event_time", s.cursor.EventLogFile.LastEventTime)
		}
		cursor = mapstr.M{"event_log_file": eventLogFile}
	}

	query, err := s.FormQueryWithCursor(s.config.EventMonitoringMethod.EventLogFile.Query, cursor)
	if err != nil {
		return fmt.Errorf("error forming query based on cursor: %w", err)
	}

	res, err := s.soqlr.Query(query, false)
	if err != nil {
		return err
	}

	if s.sfdcConfig.Client == nil {
		return errors.New("internal error: salesforce configuration is not set properly")
	}

	totalEvents, firstEvent := 0, true
	for res.TotalSize() > 0 {
		for _, rec := range res.Records() {
			req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, s.config.URL+rec.Record().Fields()["LogFile"].(string), nil)
			if err != nil {
				return err
			}

			s.clientSession.AuthorizationHeader(req)

			// NOTE: X-PrettyPrint:1 is for formatted response and ideally we do
			// not need it. But see:
			// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_event_log_file_download.htm?q=X-PrettyPrint%3A1
			req.Header.Add("X-PrettyPrint", "1")

			resp, err := s.sfdcConfig.Client.Do(req)
			if err != nil {
				return err
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				return err
			}
			resp.Body.Close()

			recs, err := decodeAsCSV(body)
			if err != nil {
				return err
			}

			if timestamp, ok := rec.Record().Fields()[s.config.EventMonitoringMethod.EventLogFile.Cursor.Field].(string); ok {
				if firstEvent {
					s.cursor.EventLogFile.FirstEventTime = timestamp
				}
				s.cursor.EventLogFile.LastEventTime = timestamp
			}

			for _, val := range recs {
				jsonStrEvent, err := json.Marshal(val)
				if err != nil {
					return err
				}

				err = publishEvent(s.publisher, s.cursor, jsonStrEvent, "EventLogFile")
				if err != nil {
					return err
				}
				totalEvents++
			}
			firstEvent = false
		}

		if !res.MoreRecords() {
			break
		}

		res, err = res.Next()
		if err != nil {
			return err
		}
	}
	s.log.Debugf("Total events: %d", totalEvents)

	return nil
}

// getSFDCConfig returns a new Salesforce configuration based on the configuration.
func getSFDCConfig(cfg *config) (*sfdc.Configuration, error) {
	var (
		creds *credentials.Credentials
		err   error
	)

	if cfg.Auth == nil {
		return nil, errors.New("no auth provider enabled")
	}

	switch {
	case cfg.Auth.OAuth2.JWTBearerFlow != nil && cfg.Auth.OAuth2.JWTBearerFlow.isEnabled():
		pemBytes, err := os.ReadFile(cfg.Auth.OAuth2.JWTBearerFlow.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("problem with client key path for JWT auth: %w", err)
		}

		signKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("problem with client key for JWT auth: %w", err)
		}

		passCreds := credentials.JwtCredentials{
			URL:            cfg.Auth.OAuth2.JWTBearerFlow.URL,
			ClientId:       cfg.Auth.OAuth2.JWTBearerFlow.ClientID,
			ClientUsername: cfg.Auth.OAuth2.JWTBearerFlow.ClientUsername,
			ClientKey:      signKey,
		}

		creds, err = credentials.NewJWTCredentials(passCreds)
		if err != nil {
			return nil, fmt.Errorf("problem with credentials: %w", err)
		}
	case cfg.Auth.OAuth2.UserPasswordFlow != nil && cfg.Auth.OAuth2.UserPasswordFlow.isEnabled():
		passCreds := credentials.PasswordCredentials{
			URL:          cfg.Auth.OAuth2.UserPasswordFlow.TokenURL,
			Username:     cfg.Auth.OAuth2.UserPasswordFlow.Username,
			Password:     cfg.Auth.OAuth2.UserPasswordFlow.Password,
			ClientID:     cfg.Auth.OAuth2.UserPasswordFlow.ClientID,
			ClientSecret: cfg.Auth.OAuth2.UserPasswordFlow.ClientSecret,
		}

		creds, err = credentials.NewPasswordCredentials(passCreds)
		if err != nil {
			return nil, fmt.Errorf("problem with credentials: %w", err)
		}

	}

	return &sfdc.Configuration{
		Credentials: creds,
		Client:      http.DefaultClient,
		Version:     cfg.Version,
	}, nil
}

// publishEvent publishes an event using the configured publisher pub.
func publishEvent(pub inputcursor.Publisher, cursor *state, jsonStrEvent []byte, dataCollectionMethod string) error {
	event := beat.Event{
		Timestamp: timeNow(),
		Fields: mapstr.M{
			"message": string(jsonStrEvent),
			"event": mapstr.M{
				"provider": dataCollectionMethod,
			},
		},
	}

	return pub.Publish(event, cursor)
}

type textContextError struct {
	error
	body []byte
}

// decodeAsCSV decodes p as a headed CSV document into dst.
func decodeAsCSV(p []byte) ([]map[string]string, error) {
	r := csv.NewReader(bytes.NewReader(p))

	// To share the backing array for performance.
	r.ReuseRecord = true

	// Header row is always expected, otherwise we can't map values to keys in
	// the event.
	header, err := r.Read()
	if err != nil {
		if err == io.EOF { //nolint:errorlint // csv.Reader never wraps io.EOF.
			return nil, nil
		}
		return nil, err
	}

	// As buffer reuse is enabled, copying header is important.
	header = slices.Clone(header)

	var results []map[string]string //nolint:prealloc // not sure about the size to prealloc with

	// NOTE:
	//
	// Read sets `r.FieldsPerRecord` to the number of fields in the first record,
	// so that future records must have the same field count.
	// So, if len(header) != len(event), the Read will return an error and hence
	// we need not put an explicit check.
	event, err := r.Read()
	for ; err == nil; event, err = r.Read() {
		if err != nil {
			continue
		}
		o := make(map[string]string, len(header))
		for i, h := range header {
			o[h] = event[i]
		}
		results = append(results, o)
	}

	if err != nil {
		if err != io.EOF { //nolint:errorlint // csv.Reader never wraps io.EOF.
			return nil, textContextError{error: err, body: p}
		}
	}

	return results, nil
}
