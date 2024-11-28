// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package unifiedlogs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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

func Plugin(log *logp.Logger, store inputcursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Beta,
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
	if config.ArchiveFile != "" || config.TraceFile != "" {
		return source{name: srcArchiveName}
	}
	return source{name: srcPollName}
}

func (src source) Name() string { return src.name }

type input struct {
	config
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
	if src.Name() == srcArchiveName {
		if _, err := os.Stat(input.ArchiveFile); input.ArchiveFile != "" && os.IsNotExist(err) {
			return err
		}
		if _, err := os.Stat(input.TraceFile); input.TraceFile != "" && os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Run starts the input and blocks until it ends the execution.
func (input *input) Run(ctxt v2.Context, src inputcursor.Source, resumeCursor inputcursor.Cursor, pub inputcursor.Publisher) error {
	reg, unreg := inputmon.NewInputRegistry(input.Name(), ctxt.ID, nil)
	defer unreg()

	stdCtx := ctxtool.FromCanceller(ctxt.Cancelation)
	metrics := newInputMetrics(reg)
	log := ctxt.Logger.With("source", src.Name())

	return input.runWithMetrics(stdCtx, resumeCursor, pub, metrics, log)
}

func (input *input) runWithMetrics(ctx context.Context, resumeCursor inputcursor.Cursor, pub inputcursor.Publisher, metrics *inputMetrics, log *logp.Logger) error {
	var startFrom string
	if !resumeCursor.IsNew() {
		var cursor time.Time
		if err := resumeCursor.Unpack(&cursor); err != nil {
			return fmt.Errorf("unpack cursor: %w", err)
		}
		startFrom = cursor.Format(cursorDateLayout)
		log.Infof("cursor loaded, resuming from: %v", startFrom)
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		metrics.intervals.Add(1)

		select {
		case <-ctx.Done():
			log.Infof("input stopped because context was cancelled with: %v", ctx.Err())
			return nil
		case <-tick.C:
		}

		logCmd, err := newLogCmd(ctx, input.config, startFrom)
		if err != nil {
			return fmt.Errorf("new log command: %w", err)
		}

		outpipe, err := logCmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("get stdout pipe: %w", err)
		}
		errpipe, err := logCmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("get stderr pipe: %w", err)
		}

		log.Debugf("exec command start: %v", logCmd)
		if err := logCmd.Start(); err != nil {
			return fmt.Errorf("start log command: %w", err)
		}

		lastProcessedDate, err := input.processLogs(outpipe, pub, metrics, log)
		if err != nil {
			log.Errorf("process logs: %v", err)
		} else {
			startFrom = lastProcessedDate
		}

		stderrBytes, _ := io.ReadAll(errpipe)
		if err := logCmd.Wait(); err != nil {
			return fmt.Errorf("log command exited with an error: %w, %q", err, string(stderrBytes))
		}

		if input.isArchive() {
			log.Info("finished processing the archived logs, stopping")
			return nil
		}
	}
}

func (input *input) isArchive() bool { return input.ArchiveFile != "" || input.TraceFile != "" }

func (input *input) processLogs(stdout io.Reader, pub inputcursor.Publisher, metrics *inputMetrics, log *logp.Logger) (string, error) {
	scanner := bufio.NewScanner(stdout)

	var (
		event             beat.Event
		line              string
		logRecord         logRecord
		lastProcessedDate string
		timestamp         time.Time
		err               error
		count             int64
	)

	defer func() { metrics.intervalEvents.Update(count) }()
	for scanner.Scan() {
		line = scanner.Text()
		if err = json.Unmarshal([]byte(line), &logRecord); err != nil {
			log.Errorf("invalid json log: %v", err)
			metrics.errs.Add(1)
			continue
		}

		timestamp, err = time.Parse(logDateLayout, logRecord.Timestamp)
		if err != nil {
			metrics.errs.Add(1)
			log.Errorf("invalid timestamp: %v", err)
			continue
		}

		event = makeEvent(timestamp, line)
		if err = pub.Publish(event, timestamp); err != nil {
			log.Errorf("publish event: %v", err)
			metrics.errs.Add(1)
			continue
		}
		lastProcessedDate = timestamp.Format(cursorDateLayout)
		count++
	}
	if err = scanner.Err(); err != nil {
		metrics.errs.Add(1)
		return "", fmt.Errorf("scanning stdout: %w", err)
	}

	return lastProcessedDate, nil
}

func newLogCmd(ctx context.Context, config config, startFrom string) (*exec.Cmd, error) {
	args := []string{"show", "--style", "ndjson"}
	if config.ArchiveFile != "" {
		args = append(args, "--archive", config.ArchiveFile)
	}
	if config.TraceFile != "" {
		args = append(args, "--file", config.TraceFile)
	}
	if len(config.Predicate) > 0 {
		for _, p := range config.Predicate {
			args = append(args, "--predicate", p)
		}
	}
	if len(config.Process) > 0 {
		for _, p := range config.Process {
			args = append(args, "--process", p)
		}
	}
	if config.Source {
		args = append(args, "--source")
	}
	if config.Info {
		args = append(args, "--info")
	}
	if config.Debug {
		args = append(args, "--debug")
	}
	if config.Signposts {
		args = append(args, "--signposts")
	}
	if config.Timezone != "" {
		args = append(args, "--timezone", config.Timezone)
	}
	start := config.Start
	if startFrom != "" {
		start = startFrom
	}
	if start != "" {
		args = append(args, "--start", start)
	}
	return exec.CommandContext(ctx, "log", args...), nil
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
