package salesforce

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/channel"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
	"github.com/g8rswimmer/go-sfdc"
	"github.com/g8rswimmer/go-sfdc/credentials"
	"github.com/g8rswimmer/go-sfdc/session"
	"github.com/g8rswimmer/go-sfdc/soql"
)

const inputName = "salesforce"

type salesforceInput struct {
	config

	time func() time.Time

	log      *logp.Logger
	outlet   channel.Outleter // Output of received messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on worker goroutine.
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

// now is time.Now with a modifiable time source.
func (s *salesforceInput) now() time.Time {
	if s.time == nil {
		return time.Now()
	}
	return s.time()
}

func (s *salesforceInput) Name() string { return inputName }

func (s *salesforceInput) Test(src inputcursor.Source, _ v2.TestContext) error {
	return nil
}

// Run starts the input and blocks until it ends completes. It will return on
// context cancellation or type invalidity errors, any other error will be retried.
func (s *salesforceInput) Run(env v2.Context, src inputcursor.Source, cursor inputcursor.Cursor, pub inputcursor.Publisher) error {
	st := &state{}
	if !cursor.IsNew() {
		now := time.Now()
		st.setCheckpoint(now.Format(time.RFC3339))
	}
	return s.run(env, src.(*source), st, pub)
}

func (s *salesforceInput) run(env v2.Context, src *source, cursor *state, pub inputcursor.Publisher) error {
	cfg := src.cfg
	// log := env.Logger.With("input_url", cfg.Url)

	ctx := ctxtool.FromCanceller(env.Cancelation)

	passCreds := credentials.PasswordCredentials{
		URL:          cfg.Url,
		Username:     cfg.Auth.OAuth2.User,
		Password:     cfg.Auth.OAuth2.Password,
		ClientID:     cfg.Auth.OAuth2.ClientID,
		ClientSecret: cfg.Auth.OAuth2.ClientSecret,
	}

	creds, err := credentials.NewPasswordCredentials(passCreds)
	if err != nil {
		return err
	}

	config := sfdc.Configuration{
		Credentials: creds,
		Client:      http.DefaultClient,
		Version:     cfg.Version,
	}

	query := querier{Query: cfg.Query.SOQL}

	err = periodically(ctx, cfg.Interval, func() error {
		cursor.StartTime = time.Now()

		// Open creates a session using the configuration.
		session, err := session.Open(config)
		if err != nil {
			return fmt.Errorf("error setting up salesforce session: %w", err)
		}

		// Create a new SOQL resource using the session.
		soqlr, err := soql.NewResource(session)
		if err != nil {
			return fmt.Errorf("error setting up salesforce SOQL resource: %w", err)
		}

		res, err := soqlr.Query(query, false)
		if err != nil {
			return err
		}

		for res.Done() {
			for _, rec := range res.Records() {
				req, err := http.NewRequestWithContext(ctx, "GET", creds.URL()+rec.Record().Fields()["LogFile"].(string), nil)
				if err != nil {
					return err
				}

				// temp := make(mapstr.M)

				// temp.Put("cursor", cursor.StartTime.Format(time.RFC3339))

				// // tp := template.New("SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE Interval = 'Hourly' AND EventType = 'Login' AND LogDate > [[.cursor]] ORDER BY CreatedDate ASC NULLS FIRST")

				// buf := new(bytes.Buffer)

				// err = cfg.Query.Value.Execute(buf, temp)
				// if err != nil {
				// 	return err
				// }

				// log.Infof("\n\n\ngenerated query from template: %s\n\n\n\n\n", buf.String())

				session.AuthorizationHeader(req)
				req.Header.Add("X-PrettyPrint", "1")

				resp, err := config.Client.Do(req)
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

				for _, v := range recs {
					val := v.(map[string]interface{})

					event := beat.Event{
						Timestamp: time.Now(),
						Fields:    mapstr.M{"salesforce": mapstr.M{val["EVENT_TYPE"].(string): v}},
					}

					cursor.setCheckpoint(val["TIMESTAMP_DERIVED"].(string))
					err = pub.Publish(event, cursor)
					if err != nil {
						return err
					}
				}
			}

			if res.MoreRecords() {
				res, err = res.Next()
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return err
}

func periodically(ctx context.Context, each time.Duration, fn func() error) error {
	err := fn()
	if err != nil {
		return err
	}
	return timed.Periodic(ctx, each, fn)
}

type textContextError struct {
	error
	body []byte
}

// decodeAsCSV decodes p as a headed CSV document into dst.
func decodeAsCSV(p []byte) ([]interface{}, error) {
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

	var results []interface{}

	event, err := r.Read()
	for ; err == nil; event, err = r.Read() {
		if err != nil {
			continue
		}
		o := make(map[string]interface{}, len(header))
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
