// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package unifiedlogs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/ctxtool"
)

const (
	inputName        = "unifiedlogs"
	srcArchiveName   = "log-cmd-archive"
	srcPollName      = "log-cmd-poll"
	logDateLayout    = "2006-01-02 15:04:05.999999-0700"
	cursorDateLayout = "2006-01-02 15:04:05-0700"
)

var (
	// override for testing
	timeNow = time.Now
)

func Plugin(log *logp.Logger, store statestore.States) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Manager: &inputcursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       inputName,
			Configure:  cursorConfigure,
		},
	}
}

type logRecord struct {
	Timestamp string `json:"timestamp"`
}

type source struct {
	name string
}

func newSource(config config) source {
	if config.ShowConfig.ArchiveFile != "" || config.ShowConfig.TraceFile != "" {
		return source{name: srcArchiveName}
	}
	return source{name: srcPollName}
}

func (src source) Name() string { return src.name }

type input struct {
	config
	metrics *inputMetrics
}

func cursorConfigure(cfg *conf.C) ([]inputcursor.Source, inputcursor.Input, error) {
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, nil, err
	}
	sources, inp := newCursorInput(conf)
	return sources, inp, nil
}

func newCursorInput(config config) ([]inputcursor.Source, inputcursor.Input) {
	input := &input{config: config}
	return []inputcursor.Source{newSource(config)}, input
}

func (input) Name() string { return inputName }

func (input input) Test(src inputcursor.Source, _ v2.TestContext) error {
	if _, err := exec.LookPath("log"); err != nil {
		return err
	}
	return nil
}

// Run starts the input and blocks until it ends the execution.
func (input *input) Run(ctxt v2.Context, src inputcursor.Source, resumeCursor inputcursor.Cursor, pub inputcursor.Publisher) error {
	reg, unreg := inputmon.NewInputRegistry(input.Name(), ctxt.ID, nil)
	defer unreg()

	stdCtx := ctxtool.FromCanceller(ctxt.Cancelation)
	log := ctxt.Logger.With("source", src.Name())

	startFrom, err := loadCursor(resumeCursor, log)
	if err != nil {
		return err
	}
	if startFrom != "" {
		input.ShowConfig.Start = startFrom
	}

	return input.runWithMetrics(stdCtx, pub, reg, log)
}

func (input *input) runWithMetrics(ctx context.Context, pub inputcursor.Publisher, reg *monitoring.Registry, log *logp.Logger) error {
	input.metrics = newInputMetrics(reg)
	// we create a wrapped publisher for the streaming go routine.
	// It will notify the backfilling goroutine with the end date of the
	// backfilling period and avoid updating the stored date to resume
	// until backfilling is done.
	wrappedPub := newWrappedPublisher(!input.mustBackfill(), pub)

	var g errgroup.Group
	// we start the streaming command in the background
	// it will use the wrapped publisher to set the end date for the
	// backfilling process.
	if input.mustStream() {
		g.Go(func() error {
			logCmd := newLogStreamCmd(ctx, input.CommonConfig)
			return input.runLogCmd(ctx, logCmd, wrappedPub, log)
		})
	}

	if input.mustBackfill() {
		g.Go(func() error {
			if input.mustStream() {
				t := wrappedPub.getFirstProcessedTime()
				// The time resolution of the log tool is microsecond, while it only
				// accepts second resolution as an end parameter.
				// To avoid potentially losing data we move the end forward one second,
				// since it is preferable to have some duplicated events.
				t = t.Add(time.Second)
				input.ShowConfig.End = t.Format(cursorDateLayout)

				// to avoid race conditions updating the cursor, and to be able to
				// resume from the oldest point in time, we only update cursor
				// from the streaming goroutine once backfilling is done.
				defer wrappedPub.startUpdatingCursor()
			}
			logCmd := newLogShowCmd(ctx, input.config)
			err := input.runLogCmd(ctx, logCmd, pub, log)
			if !input.mustStream() {
				log.Debugf("finished processing events, stopping")
			}
			return err
		})
	}

	return g.Wait()
}

// mustStream returns true in case a stream command is needed.
// This is the default case and the only exceptions are when an archive file or an end date are set.
func (input *input) mustStream() bool {
	return input.ShowConfig.ArchiveFile == "" && input.ShowConfig.TraceFile == "" && input.ShowConfig.End == ""
}

// mustBackfill returns true in case a show command is needed.
// This happens when start or end dates are set (for example when resuming filebeat), when an archive file is used,
// or when user forces it via the backfill config.
func (input *input) mustBackfill() bool {
	return input.Backfill || input.ShowConfig.ArchiveFile != "" || input.ShowConfig.TraceFile != "" || input.ShowConfig.Start != "" || input.ShowConfig.End != ""
}

func (input *input) runLogCmd(ctx context.Context, logCmd *exec.Cmd, pub inputcursor.Publisher, log *logp.Logger) error {
	outpipe, err := logCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("get stdout pipe: %w", err)
	}
	errpipe, err := logCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("get stderr pipe: %w", err)
	}

	log.Debugf("exec command start: %v", logCmd)
	defer log.Debugf("exec command end: %v", logCmd)

	if err := logCmd.Start(); err != nil {
		return fmt.Errorf("start log command: %w", err)
	}

	if err := input.processLogs(outpipe, pub, log); err != nil {
		log.Errorf("process logs: %v", err)
	}

	stderrBytes, _ := io.ReadAll(errpipe)
	if err := logCmd.Wait(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("%q exited with an error: %w, %q", logCmd, err, string(stderrBytes))
	}

	return nil
}

func (input *input) processLogs(stdout io.Reader, pub inputcursor.Publisher, log *logp.Logger) error {
	reader := bufio.NewReader(stdout)

	var (
		event         beat.Event
		line          string
		logRecordLine logRecord
		timestamp     time.Time
		err           error
	)

	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			input.metrics.errs.Add(1)
			return err
		}
		if line = strings.Trim(line, " \n\t\r"); line == "" {
			continue
		}
		if err = json.Unmarshal([]byte(line), &logRecordLine); err != nil {
			log.Errorf("invalid json log: %v", err)
			input.metrics.errs.Add(1)
			continue
		}

		if logRecordLine == (logRecord{}) {
			continue
		}

		timestamp, err = time.Parse(logDateLayout, logRecordLine.Timestamp)
		if err != nil {
			input.metrics.errs.Add(1)
			log.Errorf("invalid timestamp: %v", err)
			continue
		}

		event = makeEvent(timestamp, line)
		if err = pub.Publish(event, timestamp); err != nil {
			log.Errorf("publish event: %v", err)
			input.metrics.errs.Add(1)
			continue
		}
	}
}

// wrappedPublisher wraps a publisher and stores the first published event date.
// this is required in order to backfill the events when we start a streaming command.
type wrappedPublisher struct {
	firstTimeOnce      sync.Once
	firstTimeC         chan struct{}
	firstProcessedTime time.Time

	updateCursor *atomic.Bool

	inner inputcursor.Publisher
}

func newWrappedPublisher(updateCursor bool, inner inputcursor.Publisher) *wrappedPublisher {
	var atomicUC atomic.Bool
	atomicUC.Store(updateCursor)
	return &wrappedPublisher{
		firstTimeC:   make(chan struct{}),
		updateCursor: &atomicUC,
		inner:        inner,
	}
}

func (pub *wrappedPublisher) Publish(event beat.Event, cursor interface{}) error {
	pub.firstTimeOnce.Do(func() {
		pub.firstProcessedTime, _ = cursor.(time.Time)
		close(pub.firstTimeC)
	})
	if !pub.updateCursor.Load() {
		cursor = nil
	}
	return pub.inner.Publish(event, cursor)
}

// getFirstProcessedTime will block until there is a value set for firstProcessedTime.
func (pub *wrappedPublisher) getFirstProcessedTime() time.Time {
	<-pub.firstTimeC
	return pub.firstProcessedTime
}

func (pub *wrappedPublisher) startUpdatingCursor() {
	pub.updateCursor.Store(true)
}

func loadCursor(c inputcursor.Cursor, log *logp.Logger) (string, error) {
	if c.IsNew() {
		return "", nil
	}
	var (
		startFrom string
		cursor    time.Time
	)
	if err := c.Unpack(&cursor); err != nil {
		return "", fmt.Errorf("unpack cursor: %w", err)
	}
	log.Infof("cursor loaded, resuming from: %v", startFrom)
	return cursor.Format(cursorDateLayout), nil
}

func newLogShowCmd(ctx context.Context, cfg config) *exec.Cmd {
	return exec.CommandContext(ctx, "log", newLogCmdArgs("show", cfg)...) // #nosec G204
}

func newLogStreamCmd(ctx context.Context, cfg commonConfig) *exec.Cmd {
	return exec.CommandContext(ctx, "log", newLogCmdArgs("stream", config{CommonConfig: cfg})...) // #nosec G204
}

func newLogCmdArgs(subcmd string, config config) []string {
	args := []string{subcmd, "--style", "ndjson"}
	if config.ShowConfig.ArchiveFile != "" {
		args = append(args, "--archive", config.ShowConfig.ArchiveFile)
	}
	if config.ShowConfig.TraceFile != "" {
		args = append(args, "--file", config.ShowConfig.TraceFile)
	}
	if len(config.CommonConfig.Predicate) > 0 {
		for _, p := range config.CommonConfig.Predicate {
			args = append(args, "--predicate", p)
		}
	}
	if len(config.CommonConfig.Process) > 0 {
		for _, p := range config.CommonConfig.Process {
			args = append(args, "--process", p)
		}
	}
	if config.CommonConfig.Source {
		args = append(args, "--source")
	}
	if config.CommonConfig.Info {
		args = append(args, "--info")
	}
	if config.CommonConfig.Debug {
		args = append(args, "--debug")
	}
	if config.CommonConfig.Backtrace {
		args = append(args, "--backtrace")
	}
	if config.CommonConfig.Signpost {
		args = append(args, "--signpost")
	}
	if config.CommonConfig.Unreliable {
		args = append(args, "--unreliable")
	}
	if config.CommonConfig.MachContinuousTime {
		args = append(args, "--mach-continuous-time")
	}
	if config.ShowConfig.Start != "" {
		args = append(args, "--start", config.ShowConfig.Start)
	}
	if config.ShowConfig.End != "" {
		args = append(args, "--end", config.ShowConfig.End)
	}
	return args
}

func makeEvent(timestamp time.Time, message string) beat.Event {
	now := timeNow()
	fields := mapstr.M{
		"event": mapstr.M{
			"created": now,
		},
		"message": message,
	}

	return beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	}
}
