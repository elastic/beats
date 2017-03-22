package elasticsearch

import (
	"errors"
	"io"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/monitoring/report"
	"github.com/elastic/beats/libbeat/outputs"
	esout "github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modeutil"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

type reporter struct {
	done *stopper

	// metrics snaphot channel (buffer). windowsSize is maximum amount
	// of events being batched up.
	ch         chan outputs.Data
	windowSize int

	// client/connection objects for publishing events and checking availablity
	// of monitoring endpoint
	clients    []mode.ProtocolClient
	conn       mode.ConnectionMode
	checkRetry time.Duration

	// metrics report interval
	period time.Duration

	// event metadata
	beatMeta common.MapStr
	tags     []string
}

var debugf = logp.MakeDebug("monitoring")

var errNoMonitoring = errors.New("xpack monitoring not available")

// default monitoring api parameters
var defaultParams = map[string]string{
	"system_id":          "beats",
	"system_api_version": "2",
}

func init() {
	report.RegisterReporterFactory("elasticsearch", New)
}

func New(beat common.BeatInfo, cfg *common.Config) (report.Reporter, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	clientFactory, err := makeClientFactory(&config)
	if err != nil {
		return nil, err
	}

	clients, err := modeutil.MakeClients(cfg, clientFactory)
	if err != nil {
		return nil, err
	}

	// backoff parameters
	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second

	// TODO: make Settings configurable
	conn, err := modeutil.NewConnectionMode(clients, modeutil.Settings{
		Failover:     true,
		MaxAttempts:  1, // try to send data at most once, no retry
		WaitRetry:    backoff,
		MaxWaitRetry: maxBackoff,
		Timeout:      60 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	windowSize := config.BulkMaxSize - 1
	if windowSize <= 0 {
		windowSize = 1
	}
	// check endpoint availablity on startup only every 30 seconds
	checkRetry := 30 * time.Second

	r := &reporter{
		done:       newStopper(),
		ch:         make(chan outputs.Data, config.BufferSize),
		windowSize: windowSize,
		clients:    clients,
		conn:       conn,
		checkRetry: checkRetry,
		period:     config.Period,
		beatMeta:   makeMeta(beat),
		tags:       config.Tags,
	}
	go r.initLoop()

	return r, nil
}

func (r *reporter) Stop() {
	r.done.Stop()
}

func (r *reporter) initLoop() {
	logp.Info("Start monitoring endpoint init loop.")
	defer logp.Info("Stop monitoring endpoint init loop.")

	for {
		// Select one configured endpoint by random and check if xpack is available
		client := r.clients[rand.Intn(len(r.clients))]
		err := client.Connect(60 * time.Second)
		if err == nil {
			closing(client)
			break
		}

		select {
		case <-r.done.C():
			return
		case <-time.After(r.checkRetry):
		}
	}

	// Start collector and send loop if monitoring endpoint has been found.
	go r.snapshotLoop()
	go r.sendLoop()
}

func (r *reporter) snapshotLoop() {
	ticker := time.NewTicker(r.period)
	defer ticker.Stop()

	logp.Info("Start monitoring metrics snapshot loop.")
	defer logp.Info("Stop monitoring metrics snapshot loop.")

	for {
		var ts time.Time

		select {
		case <-r.done.C():
			return
		case ts = <-ticker.C:
		}

		snapshot := makeSnapshot(monitoring.Default)
		if snapshot == nil {
			debugf("Empty snapshot.")
			continue
		}

		event := common.MapStr{
			"timestamp": common.Time(ts),
			"beat":      r.beatMeta,
			"metrics":   snapshot,
		}
		if len(r.tags) > 0 {
			event["tags"] = r.tags
		}

		select {
		case <-r.done.C():
			return
		case r.ch <- outputs.Data{Event: event}:
		}
	}
}

// sendLoop publishes monitoring snapshots to elasticsearch from
// local buffer `r.ch`. If multiple snapshots are buffered, e.g. due to network
// outage, buffered snapshots will be combined into bulk requests.
// If shutdown signal is received, any snapshots buffered
// will be dropped and shoutdown proceeds.
func (r *reporter) sendLoop() {

	logp.Info("Start monitoring metrics send loop.")
	defer logp.Info("Stop monitoring metrics send loop.")

	// Ensure blocked connection is closed if shutdown is signaled.
	go r.done.DoWait(func() { closing(r.conn) })

	for {
		var event outputs.Data

		// check done has been closed before trying to receive an event
		select {
		case <-r.done.C():
			return
		default:
		}

		// wait for next
		select {
		case <-r.done.C():
			return
		case event = <-r.ch:
		}

		L := len(r.ch)
		if w := r.windowSize; L > w {
			L = w - 1
		}
		debugf("Collect %v waiting events in pipeline.", L+1)

		if L == 0 {
			debugf("Publish monitoring event")
			err := r.conn.PublishEvent(nil, outputs.Options{}, event)
			if err != nil {
				logp.Err("Failed to publish monitoring metrics: %v", err)
			}
			continue
		}

		// in case we did block, collect some more events from pipeline for
		// reporting all events in a batch
		batch := make([]outputs.Data, 0, L)
		batch = append(batch, event)
		for ; L >= 0; L-- {
			batch = append(batch, <-r.ch)
		}
		err := r.conn.PublishEvents(nil, outputs.Options{}, batch)
		if err != nil {
			logp.Err("Failed to publish monitoring metrics: %v", err)
		}
	}
}

func makeClientFactory(config *config) (modeutil.ClientFactory, error) {
	proxyURL, err := parseProxyURL(config.ProxyURL)
	if err != nil {
		return nil, err
	}
	if proxyURL != nil {
		logp.Info("Using proxy URL: %s", proxyURL)
	}

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	params := map[string]string{}
	for k, v := range config.Params {
		params[k] = v
	}
	for k, v := range defaultParams {
		params[k] = v
	}
	params["interval"] = config.Period.String()

	return func(host string) (mode.ProtocolClient, error) {
		url, err := esout.MakeURL(config.Protocol, "", host)
		if err != nil {
			return nil, err
		}

		esClient, err := esout.NewClient(esout.ClientSettings{
			URL:              url,
			Proxy:            proxyURL,
			TLS:              tlsConfig,
			Username:         config.Username,
			Password:         config.Password,
			Parameters:       params,
			Headers:          config.Headers,
			Index:            outil.MakeSelector(outil.ConstSelectorExpr("_xpack")),
			Pipeline:         nil,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
		}, nil)
		if err != nil {
			return nil, err
		}

		return newPublishClient(esClient, params, config.BulkMaxSize), nil
	}, nil
}

// TODO: make this reusable. Same definition in elasticsearch monitoring module
func parseProxyURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}

	url, err := url.Parse(raw)
	if err == nil && strings.HasPrefix(url.Scheme, "http") {
		return url, err
	}

	// Proxy was bogus. Try prepending "http://" to it and
	// see if that parses correctly.
	return url.Parse("http://" + raw)
}

func makeMeta(beat common.BeatInfo) common.MapStr {
	return common.MapStr{
		"type":    beat.Beat,
		"version": beat.Version,
		"name":    beat.Name,
		"host":    beat.Hostname,
		"uuid":    beat.UUID,
	}
}

func closing(c io.Closer) {
	if err := c.Close(); err != nil {
		logp.Warn("Closed failed with: %v", err)
	}
}
