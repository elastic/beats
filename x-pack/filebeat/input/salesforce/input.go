// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
	"github.com/elastic/go-concert/timed"
)

const (
	inputName         = "salesforce"
	formatRFC3339Like = "2006-01-02T15:04:05.999Z"
)

type salesforceInput struct {
	config
	ctx        context.Context
	publisher  inputcursor.Publisher
	cursor     *state
	sfdcConfig *sfdc.Configuration
	log        *logp.Logger

	clientSession *session.Session
}

// // The Filebeat user-agent is provided to the program as useragent.
// var userAgent = useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String())

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
func (s *salesforceInput) Run(env v2.Context, src inputcursor.Source, cursor inputcursor.Cursor, pub inputcursor.Publisher) error {
	st := &state{}
	if !cursor.IsNew() {
		if err := cursor.Unpack(&st); err != nil {
			return err
		}
	}
	return s.run(env, src.(*source), st, pub)
}

func getTableFromSOQL(query string) (string, error) {
	var (
		lowered = strings.ToLower(query)
		fields  = strings.Fields(lowered)
		index   = slices.Index(fields, "from")
	)
	switch {
	case index == -1, index+1 >= len(fields):
		return "", fmt.Errorf("problem with SOQL query: %s", query)
	default:
		return fields[index+1], nil
	}
}

func (s *salesforceInput) run(env v2.Context, src *source, cursor *state, pub inputcursor.Publisher) (err error) {
	cfg := src.cfg
	log := env.Logger.With("input_url", cfg.URL)

	s.ctx = ctxtool.FromCanceller(env.Cancelation)
	s.publisher = pub
	s.cursor = cursor
	s.log = log
	s.sfdcConfig, err = getSFDCConfig(&cfg)
	if err != nil {
		return err
	}

	cursor.StartTime = time.Now()

	switch {
	case strings.EqualFold(cfg.From, "EventLogFile"):
		return periodically(s.ctx, cfg.Interval, s.RunEventLogFile)
	case strings.EqualFold(cfg.From, "Object"):
		return periodically(s.ctx, cfg.Interval, s.RunObject)
	}

	return fmt.Errorf("bad configuration: value for \"from: %s\" is not correct (supported values are EventLogFile or Object)", cfg.From)
}

func (s *salesforceInput) SetupSFClientConnection() (*soql.Resource, error) {
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

func (s *salesforceInput) FormQueryWithCursor() (*querier, error) {
	qr, err := parseCursor(&s.config, s.cursor, s.log)
	if err != nil {
		return nil, err
	}

	s.log.Infof("salesforce query: %s", qr)

	return &querier{Query: qr}, err
}

func (s *salesforceInput) RunObject() error {
	// Create a new SOQL resource using the session.
	soqlr, err := s.SetupSFClientConnection()
	if err != nil {
		return fmt.Errorf("error setting up connection to Salesforce: %w", err)
	}

	query, err := s.FormQueryWithCursor()
	if err != nil {
		return fmt.Errorf("error forming based on cursor: %w", err)
	}

	res, err := soqlr.Query(query, false)
	if err != nil {
		return err
	}

	totalEvents := 0
	for res.Done() {
		for _, rec := range res.Records() {
			val := rec.Record().Fields()

			jsonStrEvent, err := json.Marshal(val)
			if err != nil {
				return err
			}

			if timstamp, ok := val[s.config.Cursor.Field].(string); ok {
				s.cursor.LogDateTime = timstamp
			} else {
				s.cursor.LogDateTime = time.Now().Format(formatRFC3339Like)
			}

			err = publishEvent(s.publisher, s.cursor, jsonStrEvent)
			if err != nil {
				return err
			}
			totalEvents++
		}

		if res.MoreRecords() {
			res, err = res.Next()
			if err != nil {
				return err
			}
		} else {
			break
		}
	}
	s.log.Debugf("total events: %d", totalEvents)

	return nil
}

func (s *salesforceInput) RunEventLogFile() error {
	soqlr, err := s.SetupSFClientConnection()
	if err != nil {
		return fmt.Errorf("error setting up connection to Salesforce: %w", err)
	}

	query, err := s.FormQueryWithCursor()
	if err != nil {
		return fmt.Errorf("error forming based on cursor: %w", err)
	}

	res, err := soqlr.Query(query, false)
	if err != nil {
		return err
	}

	totalEvents := 0
	for res.Done() {
		for _, rec := range res.Records() {
			req, err := http.NewRequestWithContext(s.ctx, "GET", s.sfdcConfig.Credentials.URL()+rec.Record().Fields()["LogFile"].(string), nil)
			if err != nil {
				return err
			}

			s.clientSession.AuthorizationHeader(req)

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

			if timstamp, ok := rec.Record().Fields()[s.config.Cursor.Field].(string); ok {
				s.cursor.LogDateTime = timstamp
			} else {
				s.cursor.LogDateTime = time.Now().Format(formatRFC3339Like)
			}

			for _, val := range recs {
				jsonStrEvent, err := json.Marshal(val)
				if err != nil {
					return err
				}

				err = publishEvent(s.publisher, s.cursor, jsonStrEvent)
				if err != nil {
					return err
				}
				totalEvents++
			}
		}

		if res.MoreRecords() {
			res, err = res.Next()
			if err != nil {
				return err
			}
		} else {
			break
		}
	}
	s.log.Debugf("total events: %d", totalEvents)

	return nil
}

func getSFDCConfig(cfg *config) (*sfdc.Configuration, error) {
	var (
		creds *credentials.Credentials
		err   error
	)

	switch {
	case cfg.Auth.JWT.isEnabled():
		pemBytes, err := os.ReadFile(cfg.Auth.JWT.ClientKeyPath)
		if err != nil {
			fmt.Errorf("problem with client key path for JWT auth: %w", err)
		}

		signKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
		if err != nil {
			fmt.Errorf("problem with client key for JWT auth: %w", err)
		}

		passCreds := credentials.JwtCredentials{
			URL:            cfg.Auth.JWT.URL,
			ClientId:       cfg.Auth.JWT.ClientId,
			ClientUsername: cfg.Auth.JWT.ClientUsername,
			ClientKey:      signKey,
		}

		creds, err = credentials.NewJWTCredentials(passCreds)
		if err != nil {
			return nil, fmt.Errorf("problem with credentials: %w", err)
		}
	case cfg.Auth.OAuth2.isEnabled():
		passCreds := credentials.PasswordCredentials{
			URL:          cfg.URL,
			Username:     cfg.Auth.OAuth2.User,
			Password:     cfg.Auth.OAuth2.Password,
			ClientID:     cfg.Auth.OAuth2.ClientID,
			ClientSecret: cfg.Auth.OAuth2.ClientSecret,
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

func publishEvent(pub inputcursor.Publisher, cursor *state, jsonStrEvent []byte) error {
	event := beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"event": mapstr.M{
				"message": string(jsonStrEvent),
			},
		},
	}

	return pub.Publish(event, cursor)
}

func periodically(ctx context.Context, each time.Duration, fn func() error) error {
	if err := fn(); err != nil {
		return err
	}
	return timed.Periodic(ctx, each, fn)
}

type textContextError struct {
	error
	body []byte
}

// decodeAsCSV decodes p as a headed CSV document into dst.
func decodeAsCSV(p []byte) ([]map[string]string, error) {
	r := csv.NewReader(bytes.NewReader(p))
	r.ReuseRecord = true // to control sharing of backing array for performance

	// NOTE:
	// Read sets `r.FieldsPerRecord` to the number of fields in the first record,
	// so that future records must have the same field count.

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
