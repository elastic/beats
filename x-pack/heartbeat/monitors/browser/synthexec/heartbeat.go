// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package synthexec

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/beat"
)

const (
	heartbeatEventFD    = 3
	heartbeatControlFD  = 4
	heartbeatProtocolID = "heartbeat/complete"
)

var errHeartbeatRunnerClosed = errors.New("heartbeat synthetics runner is not running")

// HeartbeatRunner keeps one @elastic/synthetics process alive and multiplexes
// API journey runs through its internal worker-pool protocol.
type HeartbeatRunner struct {
	newCmd func() *SynthCmd

	startOnce sync.Once
	startErr  error

	mu      sync.Mutex
	writeMu sync.Mutex
	cmd     *SynthCmd
	stdin   io.WriteCloser
	active  map[string]*heartbeatPending
	closed  chan struct{}
	ready   chan error
	closeMu sync.Once
}

type heartbeatPending struct {
	request  heartbeatRunRequest
	mpx      *ExecMultiplexer
	finished chan struct{}
	once     sync.Once
	timeout  time.Duration
}

type heartbeatTimeoutError struct {
	timeout time.Duration
	cmd     string
}

func (e heartbeatTimeoutError) Error() string {
	return fmt.Sprintf("command %q timed out after %s", e.cmd, e.timeout)
}

type heartbeatRunRequest struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type"`
	Source  heartbeatSource         `json:"source"`
	Context heartbeatMonitorContext `json:"context"`
	Options heartbeatRunOptions     `json:"options"`
}

type heartbeatSource struct {
	Type   string `json:"type"`
	Script string `json:"script,omitempty"`
	Path   string `json:"path,omitempty"`
}

type heartbeatMonitorContext struct {
	TraceID     string `json:"traceID,omitempty"`
	MonitorID   string `json:"monitorID,omitempty"`
	MonitorType string `json:"monitorType,omitempty"`
	LocationID  string `json:"locationID,omitempty"`
}

type heartbeatRunOptions struct {
	Params            map[string]interface{} `json:"params,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	Match             string                 `json:"match,omitempty"`
	PlaywrightOptions map[string]interface{} `json:"playwrightOptions,omitempty"`
	IgnoreHTTPSErrors bool                   `json:"ignoreHTTPSErrors,omitempty"`
}

type heartbeatControlMessage struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type heartbeatEventEnvelope struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
}

// NewHeartbeatRunner returns a lazy persistent Synthetics runner. It starts
// only when the monitor's first job executes.
func NewHeartbeatRunner(newCmd func() *SynthCmd) *HeartbeatRunner {
	return &HeartbeatRunner{
		newCmd: newCmd,
		active: make(map[string]*heartbeatPending),
		closed: make(chan struct{}),
		ready:  make(chan error, 1),
	}
}

// NewHeartbeatInlineRunner creates a runner for inline API journeys.
func NewHeartbeatInlineRunner() *HeartbeatRunner {
	return NewHeartbeatRunner(func() *SynthCmd {
		return &SynthCmd{Cmd: exec.Command("elastic-synthetics", "heartbeat")} //nolint:gosec,noctx // executable and arguments are fixed
	})
}

// NewHeartbeatProjectRunner creates a runner that loads API journeys from a
// project source directory.
func NewHeartbeatProjectRunner(projectPath string) (*HeartbeatRunner, error) {
	npmRoot, err := getNpmRoot(projectPath)
	if err != nil {
		return nil, err
	}

	bin := filepath.Join(npmRoot, "node_modules/.bin/elastic-synthetics")
	return NewHeartbeatRunner(func() *SynthCmd {
		cmd := exec.Command(bin, "heartbeat") //nolint:gosec,noctx // binary is resolved from the monitor project
		cmd.Dir = npmRoot
		return &SynthCmd{Cmd: cmd}
	}), nil
}

// HeartbeatRunnerLease holds a reference to a runner shared by API monitor
// source jobs. Closing the lease stops the child only after its final user has
// stopped.
type HeartbeatRunnerLease struct {
	Runner  *HeartbeatRunner
	release func()
	once    sync.Once
}

// Close releases the runner reference held by this lease.
func (l *HeartbeatRunnerLease) Close() {
	if l == nil {
		return
	}
	l.once.Do(l.release)
}

type sharedHeartbeatRunner struct {
	runner *HeartbeatRunner
	refs   int
}

var heartbeatRunnerPool = struct {
	sync.Mutex
	runners map[string]*sharedHeartbeatRunner
}{runners: make(map[string]*sharedHeartbeatRunner)}

func acquireHeartbeatRunner(key string, newRunner func() *HeartbeatRunner) *HeartbeatRunnerLease {
	heartbeatRunnerPool.Lock()
	shared := heartbeatRunnerPool.runners[key]
	if shared == nil || shared.runner.Closed() {
		shared = &sharedHeartbeatRunner{runner: newRunner()}
		heartbeatRunnerPool.runners[key] = shared
	}
	shared.refs++
	heartbeatRunnerPool.Unlock()

	return &HeartbeatRunnerLease{
		Runner: shared.runner,
		release: func() {
			heartbeatRunnerPool.Lock()
			current := heartbeatRunnerPool.runners[key]
			if current != shared {
				heartbeatRunnerPool.Unlock()
				return
			}
			shared.refs--
			if shared.refs > 0 {
				heartbeatRunnerPool.Unlock()
				return
			}
			delete(heartbeatRunnerPool.runners, key)
			heartbeatRunnerPool.Unlock()
			_ = shared.runner.Close()
		},
	}
}

// AcquireHeartbeatInlineRunner returns the shared inline API worker pool.
func AcquireHeartbeatInlineRunner() *HeartbeatRunnerLease {
	return acquireHeartbeatRunner("inline", NewHeartbeatInlineRunner)
}

// AcquireHeartbeatProjectRunner returns the shared API worker pool for a
// Synthetics installation. Different projects at the same npm root can share
// workers because the project path is supplied in each protocol request.
func AcquireHeartbeatProjectRunner(projectPath string) (*HeartbeatRunnerLease, error) {
	npmRoot, err := getNpmRoot(projectPath)
	if err != nil {
		return nil, err
	}

	bin := filepath.Join(npmRoot, "node_modules/.bin/elastic-synthetics")
	return acquireHeartbeatRunner("project:"+npmRoot, func() *HeartbeatRunner {
		return NewHeartbeatRunner(func() *SynthCmd {
			cmd := exec.Command(bin, "heartbeat") //nolint:gosec,noctx // binary is resolved from the monitor project
			cmd.Dir = npmRoot
			return &SynthCmd{Cmd: cmd}
		})
	}), nil
}

// Closed reports whether the child process has exited or was closed.
func (r *HeartbeatRunner) Closed() bool {
	select {
	case <-r.closed:
		return true
	default:
		return false
	}
}

// Close stops the persistent Synthetics process. A subsequent monitor run must
// create a fresh runner.
func (r *HeartbeatRunner) Close() error {
	var closeErr error
	r.closeMu.Do(func() {
		close(r.closed)
		r.mu.Lock()
		cmd := r.cmd
		stdin := r.stdin
		active := r.active
		r.active = make(map[string]*heartbeatPending)
		r.mu.Unlock()
		if stdin != nil {
			_ = stdin.Close()
		}
		if cmd != nil && cmd.Process != nil {
			closeErr = cmd.Process.Kill()
		}
		for _, pending := range active {
			r.finish(pending, errHeartbeatRunnerClosed)
		}
	})
	return closeErr
}

func (r *HeartbeatRunner) start() error {
	r.startOnce.Do(func() {
		r.startErr = r.startProcess()
	})
	if r.startErr != nil {
		_ = r.Close()
	}
	return r.startErr
}

func (r *HeartbeatRunner) startProcess() error {
	cmd := r.newCmd()
	platformCmdMutate(cmd)

	eventReader, eventWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create heartbeat event pipe: %w", err)
	}
	defer eventWriter.Close()

	controlReader, controlWriter, err := os.Pipe()
	if err != nil {
		_ = eventReader.Close()
		return fmt.Errorf("create heartbeat control pipe: %w", err)
	}
	defer controlWriter.Close()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = eventReader.Close()
		_ = controlReader.Close()
		return fmt.Errorf("open heartbeat stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = eventReader.Close()
		_ = controlReader.Close()
		return fmt.Errorf("open heartbeat stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = eventReader.Close()
		_ = controlReader.Close()
		return fmt.Errorf("open heartbeat stderr: %w", err)
	}

	cmd.ExtraFiles = []*os.File{eventWriter, controlWriter}
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "NODE_ENV=production", fmt.Sprintf("ELASTIC_SYNTHETICS_CONTROL_FD=%d", heartbeatControlFD))

	cmdStarted := make(chan error, 1)
	cmdDone := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		startErr := cmd.Start()
		cmdStarted <- startErr
		if startErr == nil {
			cmdDone <- cmd.Wait()
		}
	}()

	if err := <-cmdStarted; err != nil {
		_ = eventReader.Close()
		_ = controlReader.Close()
		return fmt.Errorf("start heartbeat runner: %w", err)
	}

	r.mu.Lock()
	r.cmd = cmd
	r.stdin = stdin
	r.mu.Unlock()

	go r.readHeartbeatEvents(eventReader)
	go r.readHeartbeatControl(controlReader)
	go r.readHeartbeatOutput(stdout, Stdout)
	go r.readHeartbeatOutput(stderr, Stderr)
	go func() {
		err := <-cmdDone
		if err != nil {
			logp.L().Warnf("Heartbeat Synthetics runner exited: %v", err)
		}
		_ = r.Close()
		r.signalReady(errHeartbeatRunnerClosed)
	}()

	select {
	case err := <-r.ready:
		if err != nil {
			return err
		}
		return nil
	case <-time.After(10 * time.Second):
		_ = r.Close()
		return fmt.Errorf("waiting for heartbeat runner readiness: timed out")
	}
}

func (r *HeartbeatRunner) signalReady(err error) {
	select {
	case r.ready <- err:
	default:
	}
}

func (r *HeartbeatRunner) readHeartbeatEvents(reader *os.File) {
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if !errors.Is(err, io.EOF) {
				logp.L().Warnf("Could not decode Heartbeat Synthetics event: %v", err)
			}
			return
		}

		var envelope heartbeatEventEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			logp.L().Warnf("Could not decode Heartbeat Synthetics event envelope: %v", err)
			continue
		}
		if envelope.Type == heartbeatProtocolID {
			r.finishActiveForID(envelope.ID, nil)
			continue
		}
		if envelope.Type == "heartbeat/event" {
			var event SynthEvent
			if err := json.Unmarshal(envelope.Event, &event); err != nil {
				logp.L().Warnf("Could not decode Heartbeat Synthetics event: %v", err)
				continue
			}
			r.writeActiveEventForID(envelope.ID, &event)
			continue
		}

		var event SynthEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			logp.L().Warnf("Could not decode Heartbeat Synthetics event: %v", err)
			continue
		}
		r.writeAllActiveEvents(&event)
	}
}

func (r *HeartbeatRunner) readHeartbeatControl(reader *os.File) {
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	for {
		var message heartbeatControlMessage
		if err := decoder.Decode(&message); err != nil {
			if !errors.Is(err, io.EOF) {
				logp.L().Warnf("Could not decode Heartbeat Synthetics control message: %v", err)
			}
			return
		}

		switch message.Type {
		case "ready":
			r.signalReady(nil)
		case "error":
			r.finishActiveForID(message.ID, errors.New(message.Error.Message))
		}
	}
}

func (r *HeartbeatRunner) readHeartbeatOutput(reader io.Reader, typ string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		r.writeAllActiveEvents(&SynthEvent{
			Type:                 typ,
			TimestampEpochMicros: float64(time.Now().UnixMicro()),
			Payload:              mapstr.M{"message": scanner.Text()},
		})
	}
	if err := scanner.Err(); err != nil {
		logp.L().Warnf("Could not read Heartbeat Synthetics %s: %v", typ, err)
	}
}

func (r *HeartbeatRunner) writeActiveEventForID(id string, event *SynthEvent) {
	r.mu.Lock()
	pending := r.active[id]
	r.mu.Unlock()
	if pending != nil {
		pending.mpx.writeSynthEvent(event)
	}
}

func (r *HeartbeatRunner) writeAllActiveEvents(event *SynthEvent) {
	r.mu.Lock()
	active := make([]*heartbeatPending, 0, len(r.active))
	for _, pending := range r.active {
		active = append(active, pending)
	}
	r.mu.Unlock()
	for _, pending := range active {
		pending.mpx.writeSynthEvent(event)
	}
}

func (r *HeartbeatRunner) finishActiveForID(id string, err error) {
	r.mu.Lock()
	pending := r.active[id]
	if pending == nil {
		r.mu.Unlock()
		return
	}
	delete(r.active, id)
	r.mu.Unlock()
	r.finish(pending, err)
}

func (r *HeartbeatRunner) finish(pending *heartbeatPending, err error) {
	if pending == nil {
		return
	}
	pending.once.Do(func() {
		var synthErr *SynthError
		if err != nil {
			var timeoutErr heartbeatTimeoutError
			if errors.As(err, &timeoutErr) {
				synthErr = ECSErrToSynthError(ecserr.NewCmdTimeoutStatusErr(timeoutErr.timeout, timeoutErr.cmd))
			} else {
				synthErr = ECSErrToSynthError(ecserr.NewBadCmdStatusErr(1, err.Error()))
			}
		}
		pending.mpx.writeSynthEvent(&SynthEvent{
			Type:                 CmdStatus,
			Error:                synthErr,
			TimestampEpochMicros: float64(time.Now().UnixMicro()),
		})
		pending.mpx.Close()
		close(pending.finished)
	})
}

func (r *HeartbeatRunner) run(ctx context.Context, request heartbeatRunRequest) (*ExecMultiplexer, error) {
	if err := r.start(); err != nil {
		return nil, err
	}
	if r.Closed() {
		return nil, errHeartbeatRunnerClosed
	}

	pending := &heartbeatPending{
		request:  request,
		mpx:      NewExecMultiplexer(),
		finished: make(chan struct{}),
	}
	pending.timeout, _ = ctx.Value(SynthexecTimeoutKey).(time.Duration)
	r.mu.Lock()
	if r.Closed() || r.stdin == nil {
		r.mu.Unlock()
		return nil, errHeartbeatRunnerClosed
	}
	stdin := r.stdin
	r.active[request.ID] = pending
	r.mu.Unlock()

	r.writeMu.Lock()
	err := json.NewEncoder(stdin).Encode(request)
	r.writeMu.Unlock()
	if err != nil {
		r.finishActiveForID(request.ID, err)
		return nil, err
	}
	if pending.timeout > 0 {
		go r.watchTimeout(pending)
	}
	return pending.mpx, nil
}

func (r *HeartbeatRunner) watchTimeout(pending *heartbeatPending) {
	timer := time.NewTimer(pending.timeout)
	defer timer.Stop()
	select {
	case <-pending.finished:
	case <-r.closed:
	case <-timer.C:
		r.mu.Lock()
		cmd := r.cmd
		r.mu.Unlock()
		cmdString := "elastic-synthetics heartbeat"
		if cmd != nil {
			cmdString = cmd.String()
		}
		r.finishActiveForID(pending.request.ID, heartbeatTimeoutError{timeout: pending.timeout, cmd: cmdString})
	}
}

// InlineJourneyJob returns a job that sends an inline API journey to the
// persistent Heartbeat runner.
func (r *HeartbeatRunner) InlineJourneyJob(
	ctx context.Context,
	script string,
	params func() map[string]interface{},
	filterJourneys FilterJourneyConfig,
	fields stdfields.StdMonitorFields,
	playwrightOptions map[string]interface{},
	ignoreHTTPSErrors bool,
) jobs.Job {
	return r.job(ctx, heartbeatSource{Type: "inline", Script: script}, params, filterJourneys, fields, playwrightOptions, ignoreHTTPSErrors)
}

// ProjectJourneyJob returns a job that sends a project API journey to the
// persistent Heartbeat runner.
func (r *HeartbeatRunner) ProjectJourneyJob(
	ctx context.Context,
	projectPath string,
	params func() map[string]interface{},
	filterJourneys FilterJourneyConfig,
	fields stdfields.StdMonitorFields,
	playwrightOptions map[string]interface{},
	ignoreHTTPSErrors bool,
) jobs.Job {
	return r.job(ctx, heartbeatSource{Type: "project", Path: projectPath}, params, filterJourneys, fields, playwrightOptions, ignoreHTTPSErrors)
}

func (r *HeartbeatRunner) job(
	ctx context.Context,
	source heartbeatSource,
	params func() map[string]interface{},
	filterJourneys FilterJourneyConfig,
	fields stdfields.StdMonitorFields,
	playwrightOptions map[string]interface{},
	ignoreHTTPSErrors bool,
) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		senr := newStreamEnricher(fields)
		traceID := checkGroupFromEvent(event)
		if traceID == "" {
			traceID = senr.checkGroup
		}

		request := heartbeatRunRequest{
			ID:     fmt.Sprintf("%d", heartbeatRequestID.Add(1)),
			Type:   "run",
			Source: source,
			Context: heartbeatMonitorContext{
				TraceID:     traceID,
				MonitorID:   fields.ID,
				MonitorType: fields.Type,
			},
			Options: heartbeatRunOptions{
				Tags:              filterJourneys.Tags,
				Match:             filterJourneys.Match,
				PlaywrightOptions: playwrightOptions,
				IgnoreHTTPSErrors: ignoreHTTPSErrors,
			},
		}
		if fields.RunFrom != nil {
			request.Context.LocationID = fields.RunFrom.ID
		}
		if params != nil {
			request.Options.Params = params()
		}

		mpx, err := r.run(ctx, request)
		if err != nil {
			err = senr.enrich(event, &SynthEvent{
				Type:  "cmd/could_not_start",
				Error: ECSErrToSynthError(ecserr.NewSyntheticsCmdCouldNotStartErr(err)),
			})
			return nil, err
		}
		return readResultsJob(ctx, mpx.SynthEvents(), senr.enrich)(event)
	}
}

var heartbeatRequestID atomic.Uint64
