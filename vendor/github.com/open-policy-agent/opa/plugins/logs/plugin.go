// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package logs implements decision log buffering and uploading.
package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/internal/ref"
	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/metrics"
	"github.com/open-policy-agent/opa/plugins"
	"github.com/open-policy-agent/opa/plugins/rest"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/server"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/util"
)

// Logger defines the interface for decision logging plugins.
type Logger interface {
	plugins.Plugin

	Log(context.Context, EventV1) error
}

// EventV1 represents a decision log event.
// WARNING: The AST() function for EventV1 must be kept in sync with
// the struct. Any changes here MUST be reflected in the AST()
// implementation below.
type EventV1 struct {
	Labels      map[string]string       `json:"labels"`
	DecisionID  string                  `json:"decision_id"`
	Revision    string                  `json:"revision,omitempty"` // Deprecated: Use Bundles instead
	Bundles     map[string]BundleInfoV1 `json:"bundles,omitempty"`
	Path        string                  `json:"path,omitempty"`
	Query       string                  `json:"query,omitempty"`
	Input       *interface{}            `json:"input,omitempty"`
	Result      *interface{}            `json:"result,omitempty"`
	Erased      []string                `json:"erased,omitempty"`
	Masked      []string                `json:"masked,omitempty"`
	Error       error                   `json:"error,omitempty"`
	RequestedBy string                  `json:"requested_by"`
	Timestamp   time.Time               `json:"timestamp"`
	Metrics     map[string]interface{}  `json:"metrics,omitempty"`

	inputAST ast.Value
}

// BundleInfoV1 describes a bundle associated with a decision log event.
type BundleInfoV1 struct {
	Revision string `json:"revision,omitempty"`
}

// AST returns the BundleInfoV1 as an AST value
func (b *BundleInfoV1) AST() ast.Value {
	result := ast.NewObject()
	if len(b.Revision) > 0 {
		result.Insert(ast.StringTerm("revision"), ast.StringTerm(b.Revision))
	}
	return result
}

// Key ast.Term values for the Rego AST representation of the EventV1
var labelsKey = ast.StringTerm("labels")
var decisionIDKey = ast.StringTerm("decision_id")
var revisionKey = ast.StringTerm("revision")
var bundlesKey = ast.StringTerm("bundles")
var pathKey = ast.StringTerm("path")
var queryKey = ast.StringTerm("query")
var inputKey = ast.StringTerm("input")
var resultKey = ast.StringTerm("result")
var erasedKey = ast.StringTerm("erased")
var maskedKey = ast.StringTerm("masked")
var errorKey = ast.StringTerm("error")
var requestedByKey = ast.StringTerm("requested_by")
var timestampKey = ast.StringTerm("timestamp")
var metricsKey = ast.StringTerm("metrics")

// AST returns the Rego AST representation for a given EventV1 object.
// This avoids having to round trip through JSON while applying a decision log
// mask policy to the event.
func (e *EventV1) AST() (ast.Value, error) {
	var err error
	event := ast.NewObject()

	if e.Labels != nil {
		labelsObj := ast.NewObject()
		for k, v := range e.Labels {
			labelsObj.Insert(ast.StringTerm(k), ast.StringTerm(v))
		}
		event.Insert(labelsKey, ast.NewTerm(labelsObj))
	} else {
		event.Insert(labelsKey, ast.NullTerm())
	}

	event.Insert(decisionIDKey, ast.StringTerm(e.DecisionID))

	if len(e.Revision) > 0 {
		event.Insert(revisionKey, ast.StringTerm(e.Revision))
	}

	if len(e.Bundles) > 0 {
		bundlesObj := ast.NewObject()
		for k, v := range e.Bundles {
			bundlesObj.Insert(ast.StringTerm(k), ast.NewTerm(v.AST()))
		}
		event.Insert(bundlesKey, ast.NewTerm(bundlesObj))
	}

	if len(e.Path) > 0 {
		event.Insert(pathKey, ast.StringTerm(e.Path))
	}

	if len(e.Query) > 0 {
		event.Insert(queryKey, ast.StringTerm(e.Query))
	}

	if e.Input != nil {
		if e.inputAST == nil {
			e.inputAST, err = roundtripJSONToAST(e.Input)
			if err != nil {
				return nil, err
			}
		}
		event.Insert(inputKey, ast.NewTerm(e.inputAST))
	}

	if e.Result != nil {
		results, err := roundtripJSONToAST(e.Result)
		if err != nil {
			return nil, err
		}
		event.Insert(resultKey, ast.NewTerm(results))
	}

	if len(e.Erased) > 0 {
		erased := make([]*ast.Term, len(e.Erased))
		for i, v := range e.Erased {
			erased[i] = ast.StringTerm(v)
		}
		event.Insert(erasedKey, ast.NewTerm(ast.NewArray(erased...)))
	}

	if len(e.Masked) > 0 {
		masked := make([]*ast.Term, len(e.Masked))
		for i, v := range e.Masked {
			masked[i] = ast.StringTerm(v)
		}
		event.Insert(maskedKey, ast.NewTerm(ast.NewArray(masked...)))
	}

	if e.Error != nil {
		evalErr, err := roundtripJSONToAST(e.Error)
		if err != nil {
			return nil, err
		}
		event.Insert(errorKey, ast.NewTerm(evalErr))
	}

	event.Insert(requestedByKey, ast.StringTerm(e.RequestedBy))

	// Use the timestamp JSON marshaller to ensure the format is the same as
	// round tripping through JSON.
	timeBytes, err := e.Timestamp.MarshalJSON()
	if err != nil {
		return nil, err
	}
	event.Insert(timestampKey, ast.StringTerm(strings.Trim(string(timeBytes), "\"")))

	if e.Metrics != nil {
		m, err := ast.InterfaceToValue(e.Metrics)
		if err != nil {
			return nil, err
		}
		event.Insert(metricsKey, ast.NewTerm(m))
	}

	return event, nil
}

func roundtripJSONToAST(x interface{}) (ast.Value, error) {
	rawPtr := util.Reference(x)
	// roundtrip through json: this turns slices (e.g. []string, []bool) into
	// []interface{}, the only array type ast.InterfaceToValue can work with
	if err := util.RoundTrip(rawPtr); err != nil {
		return nil, err
	}

	return ast.InterfaceToValue(*rawPtr)
}

const (
	// min amount of time to wait following a failure
	minRetryDelay               = time.Millisecond * 100
	defaultMinDelaySeconds      = int64(300)
	defaultMaxDelaySeconds      = int64(600)
	defaultUploadSizeLimitBytes = int64(32768) // 32KB limit
	defaultBufferSizeLimitBytes = int64(0)     // unlimited
	defaultMaskDecisionPath     = "/system/log/mask"
	logDropCounterName          = "decision_logs_dropped"
	defaultResourcePath         = "/logs"
)

// ReportingConfig represents configuration for the plugin's reporting behaviour.
type ReportingConfig struct {
	BufferSizeLimitBytes  *int64               `json:"buffer_size_limit_bytes,omitempty"`  // max size of in-memory buffer
	UploadSizeLimitBytes  *int64               `json:"upload_size_limit_bytes,omitempty"`  // max size of upload payload
	MinDelaySeconds       *int64               `json:"min_delay_seconds,omitempty"`        // min amount of time to wait between successful poll attempts
	MaxDelaySeconds       *int64               `json:"max_delay_seconds,omitempty"`        // max amount of time to wait between poll attempts
	MaxDecisionsPerSecond *float64             `json:"max_decisions_per_second,omitempty"` // max number of decision logs to buffer per second
	Trigger               *plugins.TriggerMode `json:"trigger,omitempty"`                  // trigger mode
}

// Config represents the plugin configuration.
type Config struct {
	Plugin          *string         `json:"plugin"`
	Service         string          `json:"service"`
	PartitionName   string          `json:"partition_name,omitempty"`
	Reporting       ReportingConfig `json:"reporting"`
	MaskDecision    *string         `json:"mask_decision"`
	ConsoleLogs     bool            `json:"console"`
	Resource        *string         `json:"resource"`
	maskDecisionRef ast.Ref
}

func (c *Config) validateAndInjectDefaults(services []string, pluginsList []string, trigger *plugins.TriggerMode) error {

	if c.Plugin != nil {
		var found bool
		for _, other := range pluginsList {
			if other == *c.Plugin {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid plugin name %q in decision_logs", *c.Plugin)
		}
	} else if c.Service == "" && len(services) != 0 && !c.ConsoleLogs {
		// For backwards compatibility allow defaulting to the first
		// service listed, but only if console logging is disabled. If enabled
		// we can't tell if the deployer wanted to use only console logs or
		// both console logs and the default service option.
		c.Service = services[0]
	} else if c.Service != "" {
		found := false

		for _, svc := range services {
			if svc == c.Service {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("invalid service name %q in decision_logs", c.Service)
		}
	}

	if c.Plugin == nil && c.Service == "" && !c.ConsoleLogs {
		return fmt.Errorf("invalid decision_log config, must have a `service`, `plugin`, or `console` logging enabled")
	}

	t, err := plugins.ValidateAndInjectDefaultsForTriggerMode(trigger, c.Reporting.Trigger)
	if err != nil {
		return errors.Wrap(err, "invalid decision_log config")
	}
	c.Reporting.Trigger = t

	min := defaultMinDelaySeconds
	max := defaultMaxDelaySeconds

	// reject bad min/max values
	if c.Reporting.MaxDelaySeconds != nil && c.Reporting.MinDelaySeconds != nil {
		if *c.Reporting.MaxDelaySeconds < *c.Reporting.MinDelaySeconds {
			return fmt.Errorf("max reporting delay must be >= min reporting delay in decision_logs")
		}
		min = *c.Reporting.MinDelaySeconds
		max = *c.Reporting.MaxDelaySeconds
	} else if c.Reporting.MaxDelaySeconds == nil && c.Reporting.MinDelaySeconds != nil {
		return fmt.Errorf("reporting configuration missing 'max_delay_seconds' in decision_logs")
	} else if c.Reporting.MinDelaySeconds == nil && c.Reporting.MaxDelaySeconds != nil {
		return fmt.Errorf("reporting configuration missing 'min_delay_seconds' in decision_logs")
	}

	// scale to seconds
	minSeconds := int64(time.Duration(min) * time.Second)
	c.Reporting.MinDelaySeconds = &minSeconds

	maxSeconds := int64(time.Duration(max) * time.Second)
	c.Reporting.MaxDelaySeconds = &maxSeconds

	// default the upload size limit
	uploadLimit := defaultUploadSizeLimitBytes
	if c.Reporting.UploadSizeLimitBytes != nil {
		uploadLimit = *c.Reporting.UploadSizeLimitBytes
	}

	c.Reporting.UploadSizeLimitBytes = &uploadLimit

	if c.Reporting.BufferSizeLimitBytes != nil && c.Reporting.MaxDecisionsPerSecond != nil {
		return fmt.Errorf("invalid decision_log config, specify either 'buffer_size_limit_bytes' or 'max_decisions_per_second'")
	}

	// default the buffer size limit
	bufferLimit := defaultBufferSizeLimitBytes
	if c.Reporting.BufferSizeLimitBytes != nil {
		bufferLimit = *c.Reporting.BufferSizeLimitBytes
	}

	c.Reporting.BufferSizeLimitBytes = &bufferLimit

	if c.MaskDecision == nil {
		maskDecision := defaultMaskDecisionPath
		c.MaskDecision = &maskDecision
	}

	c.maskDecisionRef, err = ref.ParseDataPath(*c.MaskDecision)
	if err != nil {
		return errors.Wrap(err, "invalid mask_decision in decision_logs")
	}

	if c.PartitionName != "" {
		resourcePath := fmt.Sprintf("/logs/%v", c.PartitionName)
		c.Resource = &resourcePath
	} else if c.Resource == nil {
		resourcePath := defaultResourcePath
		c.Resource = &resourcePath
	} else {
		if _, err := url.Parse(*c.Resource); err != nil {
			return fmt.Errorf("invalid resource path %q: %w", *c.Resource, err)
		}
	}

	return nil
}

// Plugin implements decision log buffering and uploading.
type Plugin struct {
	manager   *plugins.Manager
	config    Config
	buffer    *logBuffer
	enc       *chunkEncoder
	mtx       sync.Mutex
	stop      chan chan struct{}
	reconfig  chan reconfigure
	mask      *rego.PreparedEvalQuery
	maskMutex sync.Mutex
	limiter   *rate.Limiter
	metrics   metrics.Metrics
	logger    logging.Logger
}

type reconfigure struct {
	config interface{}
	done   chan struct{}
}

// ParseConfig validates the config and injects default values.
func ParseConfig(config []byte, services []string, pluginList []string) (*Config, error) {
	t := plugins.DefaultTriggerMode
	return NewConfigBuilder().WithBytes(config).WithServices(services).WithPlugins(pluginList).WithTriggerMode(&t).Parse()
}

// ConfigBuilder assists in the construction of the plugin configuration.
type ConfigBuilder struct {
	raw      []byte
	services []string
	plugins  []string
	trigger  *plugins.TriggerMode
}

// NewConfigBuilder returns a new ConfigBuilder to build and parse the plugin config.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{}
}

// WithBytes sets the raw plugin config.
func (b *ConfigBuilder) WithBytes(config []byte) *ConfigBuilder {
	b.raw = config
	return b
}

// WithServices sets the services that implement control plane APIs.
func (b *ConfigBuilder) WithServices(services []string) *ConfigBuilder {
	b.services = services
	return b
}

// WithPlugins sets the list of named plugins for decision logging.
func (b *ConfigBuilder) WithPlugins(plugins []string) *ConfigBuilder {
	b.plugins = plugins
	return b
}

// WithTriggerMode sets the plugin trigger mode.
func (b *ConfigBuilder) WithTriggerMode(trigger *plugins.TriggerMode) *ConfigBuilder {
	b.trigger = trigger
	return b
}

// Parse validates the config and injects default values.
func (b *ConfigBuilder) Parse() (*Config, error) {
	if b.raw == nil {
		return nil, nil
	}

	var parsedConfig Config

	if err := util.Unmarshal(b.raw, &parsedConfig); err != nil {
		return nil, err
	}

	if err := parsedConfig.validateAndInjectDefaults(b.services, b.plugins, b.trigger); err != nil {
		return nil, err
	}

	return &parsedConfig, nil
}

// New returns a new Plugin with the given config.
func New(parsedConfig *Config, manager *plugins.Manager) *Plugin {

	plugin := &Plugin{
		manager:  manager,
		config:   *parsedConfig,
		stop:     make(chan chan struct{}),
		buffer:   newLogBuffer(*parsedConfig.Reporting.BufferSizeLimitBytes),
		enc:      newChunkEncoder(*parsedConfig.Reporting.UploadSizeLimitBytes),
		reconfig: make(chan reconfigure),
		logger:   manager.Logger().WithFields(map[string]interface{}{"plugin": Name}),
	}

	if parsedConfig.Reporting.MaxDecisionsPerSecond != nil {
		limit := *parsedConfig.Reporting.MaxDecisionsPerSecond
		plugin.limiter = rate.NewLimiter(rate.Limit(limit), int(math.Max(1, limit)))
	}

	manager.RegisterCompilerTrigger(plugin.compilerUpdated)

	manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateNotReady})

	return plugin
}

// WithMetrics sets the global metrics provider to be used by the plugin.
func (p *Plugin) WithMetrics(m metrics.Metrics) *Plugin {
	p.metrics = m
	return p
}

// Name identifies the plugin on manager.
const Name = "decision_logs"

// Lookup returns the decision logs plugin registered with the manager.
func Lookup(manager *plugins.Manager) *Plugin {
	if p := manager.Plugin(Name); p != nil {
		return p.(*Plugin)
	}
	return nil
}

// Start starts the plugin.
func (p *Plugin) Start(ctx context.Context) error {
	p.logger.Info("Starting decision logger.")
	go p.loop()
	p.manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateOK})
	return nil
}

// Stop stops the plugin.
func (p *Plugin) Stop(ctx context.Context) {
	p.logger.Info("Stopping decision logger.")

	if *p.config.Reporting.Trigger == plugins.TriggerPeriodic {
		if _, ok := ctx.Deadline(); ok && p.config.Service != "" {
			p.flushDecisions(ctx)
		}
	}

	done := make(chan struct{})
	p.stop <- done
	<-done
	p.manager.UpdatePluginStatus(Name, &plugins.Status{State: plugins.StateNotReady})
}

func (p *Plugin) flushDecisions(ctx context.Context) {
	p.logger.Info("Flushing decision logs.")

	done := make(chan bool)

	go func(ctx context.Context, done chan bool) {
		for ctx.Err() == nil {
			if _, err := p.oneShot(ctx); err != nil {
				p.logger.Error("Error flushing decisions: %s", err)
				// Wait some before retrying, but skip incrementing interval since we are shutting down
				time.Sleep(1 * time.Second)
			} else {
				done <- true
				return
			}
		}
	}(ctx, done)

	select {
	case <-done:
		p.logger.Info("All decisions in buffer uploaded.")
	case <-ctx.Done():
		switch ctx.Err() {
		case context.DeadlineExceeded, context.Canceled:
			p.logger.Error("Plugin stopped with decisions possibly still in buffer.")
		}
	}
}

// Log appends a decision log event to the buffer for uploading.
func (p *Plugin) Log(ctx context.Context, decision *server.Info) error {

	bundles := map[string]BundleInfoV1{}
	for name, info := range decision.Bundles {
		bundles[name] = BundleInfoV1{Revision: info.Revision}
	}

	event := EventV1{
		Labels:      p.manager.Labels(),
		DecisionID:  decision.DecisionID,
		Revision:    decision.Revision,
		Bundles:     bundles,
		Path:        decision.Path,
		Query:       decision.Query,
		Input:       decision.Input,
		Result:      decision.Results,
		RequestedBy: decision.RemoteAddr,
		Timestamp:   decision.Timestamp,
		inputAST:    decision.InputAST,
	}

	if decision.Metrics != nil {
		event.Metrics = decision.Metrics.All()
	}

	if decision.Error != nil {
		event.Error = decision.Error
	}

	err := p.maskEvent(ctx, decision.Txn, &event)
	if err != nil {
		// TODO(tsandall): see note below about error handling.
		p.logger.Error("Log event masking failed: %v.", err)
		return nil
	}

	if p.config.ConsoleLogs {
		err := p.logEvent(event)
		if err != nil {
			p.logger.Error("Failed to log to console: %v.", err)
		}
	}

	if p.config.Plugin != nil {
		proxy, ok := p.manager.Plugin(*p.config.Plugin).(Logger)
		if !ok {
			return fmt.Errorf("plugin does not implement Logger interface")
		}
		return proxy.Log(ctx, event)
	}

	if p.config.Service != "" {
		p.mtx.Lock()
		defer p.mtx.Unlock()
		p.encodeAndBufferEvent(event)
	}

	return nil
}

// Reconfigure notifies the plugin with a new configuration.
func (p *Plugin) Reconfigure(_ context.Context, config interface{}) {

	done := make(chan struct{})
	p.reconfig <- reconfigure{config: config, done: done}

	p.maskMutex.Lock()
	defer p.maskMutex.Unlock()
	p.mask = nil

	<-done
}

// Trigger can be used to control when the plugin attempts to upload
// a new decision log in manual triggering mode.
func (p *Plugin) Trigger(ctx context.Context) error {
	done := make(chan error)

	go func() {
		if p.config.Service != "" {
			err := p.doOneShot(ctx)
			if err != nil {
				if ctx.Err() == nil {
					done <- err
				}
			}
		}
		close(done)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// compilerUpdated is called when a compiler trigger on the plugin manager
// fires. This indicates a new compiler instance is available. The decision
// logger needs to prepare a new masking query.
func (p *Plugin) compilerUpdated(txn storage.Transaction) {
	p.maskMutex.Lock()
	defer p.maskMutex.Unlock()
	p.mask = nil
}

func (p *Plugin) loop() {

	ctx, cancel := context.WithCancel(context.Background())

	for {

		var retry int
		var waitC chan struct{}

		if *p.config.Reporting.Trigger == plugins.TriggerPeriodic && p.config.Service != "" {
			err := p.doOneShot(ctx)

			var delay time.Duration

			if err == nil {
				min := float64(*p.config.Reporting.MinDelaySeconds)
				max := float64(*p.config.Reporting.MaxDelaySeconds)
				delay = time.Duration(((max - min) * rand.Float64()) + min)
			} else {
				delay = util.DefaultBackoff(float64(minRetryDelay), float64(*p.config.Reporting.MaxDelaySeconds), retry)
			}

			p.logger.Debug("Waiting %v before next upload/retry.", delay)

			waitC = make(chan struct{})
			go func() {
				select {
				case <-time.After(delay):
					if err != nil {
						retry++
					} else {
						retry = 0
					}
					close(waitC)
				case <-ctx.Done():
				}
			}()
		}

		select {
		case <-waitC:
		case update := <-p.reconfig:
			p.reconfigure(update.config)
			update.done <- struct{}{}
		case done := <-p.stop:
			cancel()
			done <- struct{}{}
			return
		}
	}
}

func (p *Plugin) doOneShot(ctx context.Context) error {
	uploaded, err := p.oneShot(ctx)

	if err != nil {
		p.logger.Error("%v.", err)
	} else if uploaded {
		p.logger.Info("Logs uploaded successfully.")
	} else {
		p.logger.Debug("Log upload queue was empty.")
	}
	return err
}

func (p *Plugin) oneShot(ctx context.Context) (ok bool, err error) {
	// Make a local copy of the plugins's encoder and buffer and create
	// a new encoder and buffer. This is needed as locking the buffer for
	// the upload duration will block policy evaluation and result in
	// increased latency for OPA clients
	p.mtx.Lock()
	oldChunkEnc := p.enc
	oldBuffer := p.buffer
	p.buffer = newLogBuffer(*p.config.Reporting.BufferSizeLimitBytes)
	p.enc = newChunkEncoder(*p.config.Reporting.UploadSizeLimitBytes)
	p.mtx.Unlock()

	// Along with uploading the compressed events in the buffer
	// to the remote server, flush any pending compressed data to the
	// underlying writer and add to the buffer.
	chunk, err := oldChunkEnc.Flush()
	if err != nil {
		return false, err
	} else if chunk != nil {
		p.bufferChunk(oldBuffer, chunk)
	}

	if oldBuffer.Len() == 0 {
		return false, nil
	}

	for bs := oldBuffer.Pop(); bs != nil; bs = oldBuffer.Pop() {
		if err == nil {
			err = uploadChunk(ctx, p.manager.Client(p.config.Service), *p.config.Resource, bs)
		}
		if err != nil {
			if p.limiter != nil {
				events, decErr := newChunkDecoder(bs).decode()
				if decErr != nil {
					continue
				}

				p.mtx.Lock()
				for _, event := range events {
					p.encodeAndBufferEvent(event)
				}
				p.mtx.Unlock()

			} else {
				// requeue the chunk
				p.mtx.Lock()
				p.bufferChunk(p.buffer, bs)
				p.mtx.Unlock()
			}
		}
	}

	return err == nil, err
}

func (p *Plugin) reconfigure(config interface{}) {

	newConfig := config.(*Config)

	if reflect.DeepEqual(p.config, *newConfig) {
		p.logger.Debug("Decision log uploader configuration unchanged.")
		return
	}

	p.logger.Info("Decision log uploader configuration changed.")
	p.config = *newConfig
}

func (p *Plugin) encodeAndBufferEvent(event EventV1) {
	if p.limiter != nil {
		if !p.limiter.Allow() {
			if p.metrics != nil {
				p.metrics.Counter(logDropCounterName).Incr()
			}

			p.logger.Error("Decision log dropped as rate limit exceeded. Reduce reporting interval or increase rate limit.")
			return
		}
	}

	result, err := p.enc.Write(event)
	if err != nil {
		// TODO(tsandall): revisit this now that we have an API that
		// can return an error. Should the default behaviour be to
		// fail-closed as we do for plugins?
		p.logger.Error("Log encoding failed: %v.", err)
		return
	}

	if result != nil {
		p.bufferChunk(p.buffer, result)
	}
}

func (p *Plugin) bufferChunk(buffer *logBuffer, bs []byte) {
	dropped := buffer.Push(bs)
	if dropped > 0 {
		p.logger.Error("Dropped %v chunks from buffer. Reduce reporting interval or increase buffer size.", dropped)
	}
}

func (p *Plugin) maskEvent(ctx context.Context, txn storage.Transaction, event *EventV1) error {

	mask, err := func() (rego.PreparedEvalQuery, error) {

		p.maskMutex.Lock()
		defer p.maskMutex.Unlock()

		if p.mask == nil {

			query := ast.NewBody(ast.NewExpr(ast.NewTerm(p.config.maskDecisionRef)))

			r := rego.New(
				rego.ParsedQuery(query),
				rego.Compiler(p.manager.GetCompiler()),
				rego.Store(p.manager.Store),
				rego.Transaction(txn),
				rego.Runtime(p.manager.Info),
			)

			pq, err := r.PrepareForEval(context.Background())
			if err != nil {
				return rego.PreparedEvalQuery{}, err
			}

			p.mask = &pq
		}

		return *p.mask, nil
	}()

	if err != nil {
		return err
	}

	input, err := event.AST()
	if err != nil {
		return err
	}

	rs, err := mask.Eval(
		ctx,
		rego.EvalParsedInput(input),
		rego.EvalTransaction(txn),
	)

	if err != nil {
		return err
	} else if len(rs) == 0 {
		return nil
	}

	mRuleSet, err := newMaskRuleSet(
		rs[0].Expressions[0].Value,
		func(mRule *maskRule, err error) {
			p.logger.Error("mask rule skipped: %s: %s", mRule.String(), err.Error())
		},
	)
	if err != nil {
		return err
	}

	mRuleSet.Mask(event)

	return nil
}

func uploadChunk(ctx context.Context, client rest.Client, uploadPath string, data []byte) error {

	resp, err := client.
		WithHeader("Content-Type", "application/json").
		WithHeader("Content-Encoding", "gzip").
		WithBytes(data).
		Do(ctx, "POST", uploadPath)

	if err != nil {
		return errors.Wrap(err, "Log upload failed")
	}

	defer util.Close(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("Log upload failed, server replied with not found")
	case http.StatusUnauthorized:
		return fmt.Errorf("Log upload failed, server replied with not authorized")
	default:
		return fmt.Errorf("Log upload failed, server replied with HTTP %v", resp.StatusCode)
	}
}

func (p *Plugin) logEvent(event EventV1) error {
	eventBuf, err := json.Marshal(&event)
	if err != nil {
		return err
	}
	fields := map[string]interface{}{}
	err = util.UnmarshalJSON(eventBuf, &fields)
	if err != nil {
		return err
	}
	p.manager.ConsoleLogger().WithFields(fields).WithFields(map[string]interface{}{
		"type": "openpolicyagent.org/decision_logs",
	}).Info("Decision Log")
	return nil
}
