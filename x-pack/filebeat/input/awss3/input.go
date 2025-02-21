// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
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
	"github.com/elastic/beats/v7/libbeat/management/status"
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
	metrics          *netflowMetrics
	states           *states
	filterProvider   *filterProvider
	client           beat.Client
	customFields     []fields.FieldDict
	internalNetworks []string
	wg               sync.WaitGroup
	ctx              context.Context
	cancelFunc       context.CancelFunc
	w                chan bool
}

// XXX: this is a hack
func newS3IpfixPollerInput(
	config config,
	store beater.StateStore,
) (v2.Input, error) {
	cfg := s3IpfixPollerConfig{ip: config, Paths: config.IpfixPaths}
	return &s3IpfixPollerInput{
		config:         cfg,
		store:          store,
		filterProvider: newFilterProvider(&config),
	}, nil
}

// XXX: this is a hack
func (in *s3IpfixPollerInput) Name() string { return "aws-s3-ipfix" }

// XXX: this is a hack
func (in *s3IpfixPollerInput) Test(ctx v2.TestContext) error {
	return nil
}

// XXX: this is a hack
func (n *s3IpfixPollerInput) stop(){
	n.logger.Info("stopping")
	<- n.ctx.Done()
	n.w <- true
}

// XXX: this is a hack
type FilePattern struct {
	Patterns       []*regexp.Regexp
	Dir            string
	Processed      map[string]interface{}
}

// XXX: this is a hack
func newFilePattern(dir string, rexes []*regexp.Regexp) *FilePattern {
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

// XXX: this is a hack
func (n *s3IpfixPollerInput) Run(
	env v2.Context,
	pipeline beat.Pipeline,
) error {
	n.logger = env.Logger.Named("s3")
	n.pipeline = pipeline

	n.ctx = v2.GoContextFromCanceler(env.Cancelation)

	n.metrics = newMetrics(env.ID)

	// start building the watcher

	n.logger.Info("Starting netflow input")

	n.logger.Info("Connecting to beat event publishing")

	const pollInterval = time.Minute

	n.metrics = newMetrics("nfipfix")
	var err error

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		PublishMode: beat.DefaultGuarantees,
		Processing: beat.ProcessingConfig{
			EventNormalization: boolPtr(false),
		},
		EventListener: nil,
	})
	if err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed connecting to beat event publishing: %v", err))
		n.logger.Errorw("Failed connecting to beat event publishing", "error", err)
		n.stop()
		return err
	}

	n.client = client
	n.wg.Add(1)

	var rexes []*regexp.Regexp
	var dir string
	for _,path_ := range n.config.Paths {
		reg := filepath.Base(path_)
		d := filepath.Dir(path_)
		if d != "." {
			if dir != "" && dir != d {
				n.logger.Errorf("There can only be one dir we are watching! We have two: %q, %q", d, dir)
				n.logger.Warnf("Ignoring %q", path_)
				continue
			}
			dir = d
		}
		r,err := regexp.Compile(reg)
		if err != nil {
			n.logger.Warnf("Ignoring path [%s], due to [%v]", reg, err)
		} else {
			rexes = append(rexes, r)
		}
	}

	// XXX: this is a hack
	if dir == "" {
		dir = "/var/run/srv/netflow"
	}
	fp := newFilePattern(dir, rexes)

	// start a timer so we check the directory on a cadence
	ticker := time.NewTicker(100 * time.Millisecond)
	// start watching the directory

	// keep regular expressions
	// read the channel
	// track files we have looked at
	for {
		select {
		case <-n.ctx.Done():
			break
		case <-ticker.C:

			fpath, err := fp.getNextFile()
			if err != nil {
				break
			}
			if fpath == "" {
				continue
			}

			start := time.Now().In(time.UTC)
			err = n.processFile(fpath)
			n.metrics.ProcessingTime.Update(time.Since(start).Nanoseconds())
			if err != nil {
				break
			}
		case <- n.w:
			break
		}
	}

	env.UpdateStatus(status.Running, "continuing to monitor")
	<-n.ctx.Done()
	n.stop()

	env.UpdateStatus(status.Running, "stopped monitoring")
	return nil
}

func (n *s3IpfixPollerInput) processFile(fpath string) error {
	// this will actually be the file to read, not the packet
	fi, err := os.Stat(fpath)
	if err != nil {
		// log something
		n.logger.Warnf("Error stat on file %s: %v", fpath, err)
		return err
	}

	// check for pipe?
	if fi.Mode()&os.ModeNamedPipe != 0 {
		n.logger.Warnf("Error on file %s: Named Pipes are not supported", fpath)
		return fmt.Errorf("File is a named pipe: %s", fpath)
	}
	// check for regular file?

	f, err := file.ReadOpen(fpath)
	if err != nil {
		n.logger.Warnf("Error ReadOpen on file %s: %v", fpath, err)
		return err
	}

	defer f.Close()

	// log that we opened a file here
	n.metrics.Log(fpath, 1)
	ip := ipfix_reader.Config{}
	decoder, err := ipfix_reader.NewBufferedReader(f, &ip)
	for {
		if !decoder.Next() {
			break
		}
		events, err := decoder.Record()
		if err != nil {
			n.logger.Warnf("Error parsing NetFlow Record")
			if decodeErrors := n.metrics.DecodeErrors(); decodeErrors != nil {
				decodeErrors.Inc()
			}
			continue
		}

		n.client.PublishAll(events)
	}
	n.metrics.Log(fpath, 0)

	os.Remove(fpath)

	return nil
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
		return newS3IpfixPollerInput(config, im.store)
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
