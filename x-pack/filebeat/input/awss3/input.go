// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bufio"
	"context"
	"fmt"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/file"
	ipfix_reader "github.com/elastic/beats/v7/x-pack/libbeat/reader/ipfix"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

const inputName = "aws-s3"

func Plugin(store beater.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from s3",
		Manager:    &s3InputManager{store: store},
	}
}

// XXX: this is a hack
type s3IpfixPollerConfig struct {
	ip config

	Paths                     []string	`config:"paths"`
}

// XXX: this is a hack
type s3IpfixPollerInput struct {
	logger           *logp.Logger
	pipeline         beat.Pipeline
	config           s3IpfixPollerConfig
	store            beater.StateStore
	provider         string
	Metrics          *netflowMetrics
	states           *states
	filterProvider   *filterProvider
	client           beat.Client
	customFields     []fields.FieldDict
	internalNetworks []string
	wg               sync.WaitGroup
	ctx              context.Context
	cancelFunc       context.CancelFunc
	w                chan bool

	// Workers send on workRequestChan to indicate they're ready for the next
	// message, and the reader loop replies on workResponseChan.
	workRequestChan  chan struct{}
	workResponseChan chan string

}

// XXX: this is a hack
func newS3IpfixPollerInput(
	config config,
	store beater.StateStore,
) *s3IpfixPollerInput {
	cfg := s3IpfixPollerConfig{ip: config, Paths: config.IpfixPaths}
	return &s3IpfixPollerInput{
		config:         cfg,
		store:          store,
		filterProvider: newFilterProvider(&config),

		workRequestChan: make(chan struct{}, 1),
		workResponseChan: make(chan string),
	}
}

// XXX: this is a hack
func (in *s3IpfixPollerInput) Name() string { return "aws-s3-ipfix" }

// XXX: this is a hack
func (in *s3IpfixPollerInput) Test(ctx v2.TestContext) error {
	return nil
}

// XXX: this is a hack
type FilePattern struct {
	Patterns       []*regexp.Regexp
	Dir            string
	Processed      map[string]interface{}
}

// XXX: this is a hack
func newFilePattern(logger *logp.Logger, paths []string) *FilePattern {
	var rexes []*regexp.Regexp
	var dir string
	for _,path_ := range paths {
		reg := filepath.Base(path_)
		d := filepath.Dir(path_)
		if d != "." {
			if dir != "" && dir != d {
				logger.Errorf("There can only be one dir we are watching! We have two: %q, %q", d, dir)
				logger.Warnf("Ignoring %q", path_)
				continue
			}
			dir = d
		}
		r,err := regexp.Compile(reg)
		if err != nil {
			logger.Warnf("Ignoring path [%s], due to [%v]", reg, err)
		} else {
			rexes = append(rexes, r)
		}
	}

	// XXX: this is a hack
	if dir == "" {
		dir = "/var/run/srv/netflow"
	}

	processed := make(map[string]interface{})
	result := FilePattern{
		Dir: dir,
		Patterns: rexes,
		Processed: processed,
	}

	return &result
}

// XXX: this is a hack
func (fp *FilePattern) getNextFile() (string, error) {
	files, err := os.ReadDir(fp.Dir)
	if err != nil {
		return "", err
	}

	for _,f := range files {
		// did we already process this file?
		fname := f.Name()
		fpath := filepath.Join(fp.Dir, fname)
		if _, exists := fp.Processed[fpath]; exists {
			// ignore this one
		} else {
			start := time.Now().In(time.UTC)
			fp.Processed[fpath] = start
			// does this file match our regexes?
			for _,r := range fp.Patterns {
				if r.MatchString(fname) {
					return fpath, nil
				}
			}
		}
	}

	return "", nil
}

func (in *s3IpfixPollerInput) setup(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {

	in.logger = inputContext.Logger.Named("s3-ipfix")
	in.pipeline = pipeline

	in.Metrics = newMetrics(inputContext.ID)

	return nil
}

// XXX: this is a hack
func (n *s3IpfixPollerInput) run(ctx context.Context) {
	// start building the watcher

	// start workers (only one)
	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		worker, err := n.newWorker(ctx)
		if err != nil {
			n.logger.Error(err)
			return
		}
		go worker.run(ctx)
	}()

	// reader loop
	fp := newFilePattern(n.logger, n.config.Paths)

	// keep regular expressions
	// read the channel
	// track files we have looked at
	stop := false
	for {
		if stop {
			break
		}

		// wait on getting a new request
		select {
		case <-ctx.Done():
			return
		case <-n.workRequestChan:
		}

		// loop until we get another file
		ticker := time.NewTicker(1 * time.Second)
		innerStop := false
		fpath := ""
		for {
			if innerStop {
				break
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tmp, _ := fp.getNextFile()
				if tmp != "" {
					fpath = tmp
					innerStop = true
				}
			}
		}

		select {
		case <-ctx.Done():
			return
		case n.workResponseChan<-fpath:
		}
	}

	n.wg.Wait()
}

// XXX: this is a hack
func (n *s3IpfixPollerInput) Run(
	env v2.Context,
	pipeline beat.Pipeline,
) error {

	n.setup(env, pipeline)
	ctx := v2.GoContextFromCanceler(env.Cancelation)

	n.run(ctx)

	return nil
}

func (p *s3IpfixWorker) addGzipDecoderIfNeeded(body io.Reader) (io.Reader, error) {
	bufReader := bufio.NewReader(body)

	gzipped, err := isStreamGzipped(bufReader)
	if err != nil {
		return nil, err
	}
	if !gzipped {
		return bufReader, nil
	}

	return gzip.NewReader(bufReader)
}

func (n *s3IpfixWorker) processFile(fpath string) {
	if fpath == "" {
		return
	}
	n.logger.Infof("processing file [%v] now", fpath)
	start := time.Now().In(time.UTC)
	n.input.Metrics.Log(fpath, 1)
	defer n.input.Metrics.Log(fpath, 0)

	// this will actually be the file to read, not the packet
	fi, err := os.Stat(fpath)
	if err != nil {
		// log something
		n.logger.Warnf("Error stat on file [%s]: %v", fpath, err)
		return
	}

	// check for pipe?
	if fi.Mode()&os.ModeNamedPipe != 0 {
		n.logger.Warnf("Error on file %s: Named Pipes are not supported", fpath)
		return
	}
	// check for regular file?

	f, err := file.ReadOpen(fpath)
	if err != nil {
		n.logger.Warnf("Error ReadOpen on file %s: %v", fpath, err)
		return
	}

	defer f.Close()
	defer os.Remove(fpath)

	reader, err := n.addGzipDecoderIfNeeded(f)
	if err != nil {
		n.logger.Warnf("Failed to add gzip decoder: [%v]", err)
	}

	ip := ipfix_reader.Config{}
	decoder, err := ipfix_reader.NewBufferedReader(reader, &ip)
	for {
		if !decoder.Next() {
			break
		}
		events, err := decoder.Record()
		if err != nil {
			n.logger.Warnf("Error parsing NetFlow Record")
			if decodeErrors := n.input.Metrics.DecodeErrors(); decodeErrors != nil {
				decodeErrors.Inc()
			}
			continue
		}

		n.client.PublishAll(events)
	}

	n.input.Metrics.ProcessingTime.Update(time.Since(start).Nanoseconds())
}

// XXX: this is a hack
type netflowMetrics struct {
	unregister func()
	FilesOpened *monitoring.Uint
	FilesClosed *monitoring.Uint
	MessagesRead *monitoring.Uint
	discardedEvents *monitoring.Uint
	decodeErrors    *monitoring.Uint
	flows           *monitoring.Uint
	activeSessions  *monitoring.Uint
	ProcessingTime  metrics.Sample
}

// XXX: this is a hack
func newMetrics(id string) *netflowMetrics {
	reg, unreg := inputmon.NewInputRegistry("s3-ipfix", id, nil)

	n := netflowMetrics{
		unregister: unreg,
		FilesOpened: monitoring.NewUint(reg, "files_opened_total"),
		FilesClosed: monitoring.NewUint(reg, "files_closed_total"),
		MessagesRead: monitoring.NewUint(reg, "messages_read_total"),
		discardedEvents: monitoring.NewUint(reg, "discarded_events_total"),
		flows:           monitoring.NewUint(reg, "flows_total"),
		decodeErrors:    monitoring.NewUint(reg, "decode_errors_total"),
		activeSessions:  monitoring.NewUint(reg, "open_files"),
		ProcessingTime: metrics.NewUniformSample(1024),
	}

	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(n.ProcessingTime))

	return &n
}

// XXX: this is a hack
func (n *netflowMetrics) DiscardedEvents() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.discardedEvents
}

// XXX: this is a hack
func (n *netflowMetrics) DecodeErrors() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.decodeErrors
}

// XXX: this is a hack
func (n *netflowMetrics) Flows() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.flows
}

// XXX: this is a hack
func (n *netflowMetrics) Log(path string, what int) {
	if n == nil {
		return
	}

	if what == 0 {
		n.FilesClosed.Inc()
	} else {
		n.FilesOpened.Inc()
	}

}

// XXX: this is a hack
func (n *netflowMetrics) ActiveSessions() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.activeSessions
}

type s3IpfixWorker struct {
	logger *logp.Logger
	input *s3IpfixPollerInput
	client beat.Client
}

func (iw *s3IpfixWorker) run(ctx context.Context) {
	defer iw.client.Close()

	for ctx.Err() == nil {
		// Send a work request
		select {
		case <-ctx.Done():
			// Shutting down
			return
		case iw.input.workRequestChan <- struct{}{}:
		}
		// The request is sent, wait for a response
		select {
		case <-ctx.Done():
			return
		case msg := <-iw.input.workResponseChan:
			iw.processFile(msg)
		}
	}
}

func (in *s3IpfixPollerInput) newWorker(ctx context.Context) (*s3IpfixWorker, error) {
	// Create a pipeline client scoped to this worker.
	client, err := in.pipeline.ConnectWith(beat.ClientConfig{
		EventListener: nil,
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: boolPtr(false),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to pipeline: %w", err)
	}

	worker:= s3IpfixWorker{
		logger:     in.logger.Named("worker"),
		input:      in,
		client:     client,
	}

	go worker.run(ctx)
	return &worker, nil
}

type s3InputManager struct {
	store beater.StateStore
}

func (im *s3InputManager) Init(grp unison.Group) error {
	return nil
}

func (im *s3InputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	if len(config.IpfixPaths) > 0 {
		return newS3IpfixPollerInput(config, im.store), nil
	}

	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)
	if err != nil {
		return nil, fmt.Errorf("initializing AWS config: %w", err)
	}

	if config.RegionName != "" {
		// The awsConfig now contains the region from the credential profile or default region
		// if the region is explicitly set in the config, then it wins
		awsConfig.Region = config.RegionName
	}

	if config.QueueURL != "" {
		return newSQSReaderInput(config, awsConfig), nil
	}

	if config.BucketARN != "" || config.AccessPointARN != "" || config.NonAWSBucketName != "" {
		return newS3PollerInput(config, awsConfig, im.store)
	}

	return nil, fmt.Errorf("configuration has no SQS queue URL and no S3 bucket ARN")
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
