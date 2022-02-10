// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package auditd

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/go-libaudit/v2"
	"github.com/elastic/go-libaudit/v2/aucoalesce"
	"github.com/elastic/go-libaudit/v2/auparse"
	"github.com/elastic/go-libaudit/v2/rule"
)

const (
	namespace = "auditd"

	auditLocked = 2

	unicast   = "unicast"
	multicast = "multicast"
	uidUnset  = "unset"

	lostEventsUpdateInterval        = time.Second * 15
	maxDefaultStreamBufferConsumers = 4

	setPIDMaxRetries = 5
)

type backpressureStrategy uint8

const (
	bsKernel backpressureStrategy = 1 << iota
	bsUserSpace
	bsAuto
)

var (
	auditdMetrics         = monitoring.Default.NewRegistry(moduleName)
	reassemblerGapsMetric = monitoring.NewInt(auditdMetrics, "reassembler_seq_gaps")
	kernelLostMetric      = monitoring.NewInt(auditdMetrics, "kernel_lost")
	userspaceLostMetric   = monitoring.NewInt(auditdMetrics, "userspace_lost")
	receivedMetric        = monitoring.NewInt(auditdMetrics, "received_msgs")
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithHostParser(parse.EmptyHostParser),
		mb.WithNamespace(namespace),
	)
}

// MetricSet listens for audit messages from the Linux kernel using a netlink
// socket. It buffers the messages to ensure ordering and then streams the
// output. MetricSet implements the mb.PushMetricSet interface, and therefore
// does not rely on polling.
type MetricSet struct {
	mb.BaseMetricSet
	config     Config
	client     *libaudit.AuditClient
	log        *logp.Logger
	kernelLost struct {
		enabled bool
		counter uint32
	}
	backpressureStrategy backpressureStrategy
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the auditd config: %w", err)
	}

	log := logp.NewLogger(moduleName)
	_, _, kernel, _ := kernelVersion()
	log.Infof("auditd module is running as euid=%v on kernel=%v", os.Geteuid(), kernel)

	client, err := newAuditClient(&config, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit client: %w", err)
	}

	reassemblerGapsMetric.Set(0)
	kernelLostMetric.Set(0)
	userspaceLostMetric.Set(0)
	receivedMetric.Set(0)

	return &MetricSet{
		BaseMetricSet:        base,
		client:               client,
		config:               config,
		log:                  log,
		backpressureStrategy: getBackpressureStrategy(config.BackpressureStrategy, log),
	}, nil
}

func newAuditClient(c *Config, log *logp.Logger) (*libaudit.AuditClient, error) {
	var err error
	c.SocketType, err = determineSocketType(c, log)
	if err != nil {
		return nil, err
	}
	log.Infof("socket_type=%s will be used.", c.SocketType)

	if c.SocketType == multicast {
		return libaudit.NewMulticastAuditClient(nil)
	}
	return libaudit.NewAuditClient(nil)
}

func closeAuditClient(client *libaudit.AuditClient) error {
	discard := func(bytes []byte) ([]syscall.NetlinkMessage, error) {
		return nil, nil
	}
	// Drain the netlink channel in parallel to Close() to prevent a deadlock.
	// This goroutine will terminate once receive from netlink errors (EBADF,
	// EBADFD, or any other error). This happens because the fd is closed.
	go func() {
		for {
			_, err := client.Netlink.Receive(true, discard)
			switch err {
			case nil, syscall.EINTR:
			case syscall.EAGAIN:
				time.Sleep(50 * time.Millisecond)
			default:
				return
			}
		}
	}()
	return client.Close()
}

// Run initializes the audit client and receives audit messages from the
// kernel until the reporter's done channel is closed.
func (ms *MetricSet) Run(reporter mb.PushReporterV2) {
	defer closeAuditClient(ms.client)

	if err := ms.addRules(reporter); err != nil {
		reporter.Error(err)
		ms.log.Errorw("Failure adding audit rules", "error", err)
		return
	}

	out, err := ms.receiveEvents(reporter.Done())
	if err != nil {
		reporter.Error(err)
		ms.log.Errorw("Failure receiving audit events", "error", err)
		return
	}

	if ms.kernelLost.enabled {
		client, err := libaudit.NewAuditClient(nil)
		if err != nil {
			reporter.Error(err)
			ms.log.Errorw("Failure creating audit monitoring client", "error", err)
		}
		go func() {
			defer func() { // Close the most recently allocated "client" instance.
				if client != nil {
					closeAuditClient(client)
				}
			}()
			timer := time.NewTicker(lostEventsUpdateInterval)
			defer timer.Stop()
			for {
				select {
				case <-reporter.Done():
					return
				case <-timer.C:
					if status, err := client.GetStatus(); err == nil {
						ms.updateKernelLostMetric(status.Lost)
					} else {
						ms.log.Error("get status request failed:", err)
						if err = closeAuditClient(client); err != nil {
							ms.log.Errorw("Error closing audit monitoring client", "error", err)
						}
						client, err = libaudit.NewAuditClient(nil)
						if err != nil {
							ms.log.Errorw("Failure creating audit monitoring client", "error", err)
							reporter.Error(err)
							return
						}
					}
				}
			}
		}()
	}

	// Spawn the stream buffer consumers
	numConsumers := ms.config.StreamBufferConsumers
	// By default (stream_buffer_consumers=0) use as many consumers as local CPUs
	// with a max of `maxDefaultStreamBufferConsumers`
	if numConsumers == 0 {
		if numConsumers = runtime.GOMAXPROCS(-1); numConsumers > maxDefaultStreamBufferConsumers {
			numConsumers = maxDefaultStreamBufferConsumers
		}
	}
	var wg sync.WaitGroup
	wg.Add(numConsumers)

	for i := 0; i < numConsumers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-reporter.Done():
					return
				case msgs := <-out:
					reporter.Event(buildMetricbeatEvent(msgs, ms.config))
				}
			}
		}()
	}
	wg.Wait()
}

func (ms *MetricSet) addRules(reporter mb.PushReporterV2) error {
	rules := ms.config.rules()

	if len(rules) == 0 {
		ms.log.Info("No audit_rules were specified.")
		return nil
	}

	client, err := libaudit.NewAuditClient(nil)
	if err != nil {
		return fmt.Errorf("failed to create audit client for adding rules: %w", err)
	}
	defer closeAuditClient(client)

	// Don't attempt to change configuration if audit rules are locked (enabled == 2).
	// Will result in EPERM.
	status, err := client.GetStatus()
	if err != nil {
		err = fmt.Errorf("failed to get audit status before adding rules: %w", err)
		reporter.Error(err)
		return err
	}
	if status.Enabled == auditLocked {
		return errors.New("Skipping rule configuration: Audit rules are locked")
	}

	// Delete existing rules.
	n, err := client.DeleteRules()
	if err != nil {
		return fmt.Errorf("failed to delete existing rules: %w", err)
	}
	ms.log.Infof("Deleted %v pre-existing audit rules.", n)

	// Add rule to ignore syscalls from this process
	if rule, err := buildPIDIgnoreRule(os.Getpid()); err == nil {
		rules = append([]auditRule{rule}, rules...)
	} else {
		ms.log.Errorf("Failed to build a rule to ignore self: %v", err)
	}
	// Add rules from config.
	var failCount int
	for _, rule := range rules {
		if err = client.AddRule(rule.data); err != nil {
			// Treat rule add errors as warnings and continue.
			err = fmt.Errorf("failed to add audit rule '%v': %w", rule.flags, err)
			reporter.Error(err)
			ms.log.Warnw("Failure adding audit rule", "error", err)
			failCount++
		}
	}
	ms.log.Infof("Successfully added %d of %d audit rules.",
		len(rules)-failCount, len(rules))
	return nil
}

func (ms *MetricSet) initClient() error {
	if ms.config.SocketType == "multicast" {
		// This request will fail with EPERM if this process does not have
		// CAP_AUDIT_CONTROL, but we will ignore the response. The user will be
		// required to ensure that auditing is enabled if the process is only
		// given CAP_AUDIT_READ.
		err := ms.client.SetEnabled(true, libaudit.NoWait)
		return errors.Wrap(err, "failed to enable auditing in the kernel")
	}

	// Unicast client initialization (requires CAP_AUDIT_CONTROL and that the
	// process be in initial PID namespace).
	status, err := ms.client.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get audit status: %w", err)
	}
	ms.kernelLost.enabled = true
	ms.kernelLost.counter = status.Lost

	ms.log.Infow("audit status from kernel at start", "audit_status", status)

	if status.Enabled == auditLocked {
		return errors.New("failed to configure: The audit system is locked")
	}

	if fm, _ := ms.config.failureMode(); status.Failure != fm {
		if err = ms.client.SetFailure(libaudit.FailureMode(fm), libaudit.NoWait); err != nil {
			return fmt.Errorf("failed to set audit failure mode in kernel: %w", err)
		}
	}

	if status.BacklogLimit != ms.config.BacklogLimit {
		if err = ms.client.SetBacklogLimit(ms.config.BacklogLimit, libaudit.NoWait); err != nil {
			return fmt.Errorf("failed to set audit backlog limit in kernel: %w", err)
		}
	}

	if ms.backpressureStrategy&(bsKernel|bsAuto) != 0 {
		// "kernel" backpressure mitigation strategy
		//
		// configure the kernel to drop audit events immediately if the
		// backlog queue is full.
		if status.FeatureBitmap&libaudit.AuditFeatureBitmapBacklogWaitTime != 0 {
			ms.log.Info("Setting kernel backlog wait time to prevent backpressure propagating to the kernel.")
			if err = ms.client.SetBacklogWaitTime(0, libaudit.NoWait); err != nil {
				return fmt.Errorf("failed to set audit backlog wait time in kernel: %w", err)
			}
		} else {
			if ms.backpressureStrategy == bsAuto {
				ms.log.Warn("setting backlog wait time is not supported in this kernel. Enabling workaround.")
				ms.backpressureStrategy |= bsUserSpace
			} else {
				return errors.New("kernel backlog wait time not supported by kernel, but required by backpressure_strategy")
			}
		}
	}

	if ms.backpressureStrategy&(bsKernel|bsUserSpace) == bsUserSpace && ms.config.RateLimit == 0 {
		// force a rate limit if the user-space strategy will be used without
		// corresponding backlog_wait_time setting in the kernel
		ms.config.RateLimit = 5000
	}

	if status.RateLimit != ms.config.RateLimit {
		if err = ms.client.SetRateLimit(ms.config.RateLimit, libaudit.NoWait); err != nil {
			return fmt.Errorf("failed to set audit rate limit in kernel: %w", err)
		}
	}

	if status.Enabled == 0 {
		if err = ms.client.SetEnabled(true, libaudit.NoWait); err != nil {
			return fmt.Errorf("failed to enable auditing in the kernel: %w", err)
		}
	}

	if err := ms.client.WaitForPendingACKs(); err != nil {
		return fmt.Errorf("failed to wait for ACKs: %w", err)
	}

	if err := ms.setPID(setPIDMaxRetries); err != nil {
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.EEXIST && status.PID != 0 {
			return fmt.Errorf("failed to set audit PID. An audit process is already running (PID %d)", status.PID)
		}
		return fmt.Errorf("failed to set audit PID (current audit PID %d): %w", status.PID, err)
	}
	return nil
}

func (ms *MetricSet) setPID(retries int) (err error) {
	if err = ms.client.SetPID(libaudit.WaitForReply); err == nil || !errors.Is(err, syscall.ENOBUFS) || retries == 0 {
		return err
	}
	// At this point the netlink channel is congested (ENOBUFS).
	// Drain and close the client, then retry with a new client.
	closeAuditClient(ms.client)
	if ms.client, err = newAuditClient(&ms.config, ms.log); err != nil {
		return fmt.Errorf("failed to recover from ENOBUFS: %w", err)
	}
	ms.log.Info("Recovering from ENOBUFS ...")
	return ms.setPID(retries - 1)
}

func (ms *MetricSet) updateKernelLostMetric(lost uint32) {
	if !ms.kernelLost.enabled {
		return
	}
	delta := int64(lost - ms.kernelLost.counter)
	if delta >= 0 {
		logFn := ms.log.Debugf
		if delta > 0 {
			logFn = ms.log.Infof
			kernelLostMetric.Add(delta)
		}
		logFn("kernel lost events: %d (total: %d)", delta, lost)
	} else {
		ms.log.Warnf("kernel lost event counter reset from %d to %d", ms.kernelLost, lost)
	}
	ms.kernelLost.counter = lost
}

func (ms *MetricSet) receiveEvents(done <-chan struct{}) (<-chan []*auparse.AuditMessage, error) {
	if err := ms.initClient(); err != nil {
		return nil, err
	}

	out := make(chan []*auparse.AuditMessage, ms.config.StreamBufferQueueSize)

	var st libaudit.Stream = &stream{done, out}
	if ms.backpressureStrategy&bsUserSpace != 0 {
		// "user-space" backpressure mitigation strategy
		//
		// Consume events from our side as fast as possible, by dropping events
		// if the publishing pipeline would block.
		ms.log.Info("Using non-blocking stream to prevent backpressure propagating to the kernel.")
		st = &nonBlockingStream{done, out}
	}
	reassembler, err := libaudit.NewReassembler(int(ms.config.ReassemblerMaxInFlight), ms.config.ReassemblerTimeout, st)
	if err != nil {
		return nil, fmt.Errorf("failed to create Reassembler: %w", err)
	}
	go maintain(done, reassembler)

	go func() {
		defer ms.log.Debug("receiveEvents goroutine exited")
		defer close(out)
		defer reassembler.Close()

		for {
			raw, err := ms.client.Receive(false)
			if err != nil {
				if errors.Is(err, syscall.EBADF) {
					// Client has been closed.
					break
				}
				continue
			}

			if filterRecordType(raw.Type) {
				continue
			}
			receivedMetric.Inc()
			if err := reassembler.Push(raw.Type, raw.Data); err != nil {
				ms.log.Debugw("Dropping audit message",
					"record_type", raw.Type,
					"message", string(raw.Data),
					"error", err)
				continue
			}
		}
	}()

	return out, nil
}

// maintain periodically evicts timed-out events from the Reassembler. This
// function will block until the done channel is closed or the Reassembler is
// closed.
func maintain(done <-chan struct{}, reassembler *libaudit.Reassembler) {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-done:
			return
		case <-tick.C:
			if err := reassembler.Maintain(); err != nil {
				return
			}
		}
	}
}

func filterRecordType(typ auparse.AuditMessageType) bool {
	switch {
	// REPLACE messages are tests to check if Auditbeat is still healthy by
	// seeing if unicast messages can be sent without error from the kernel.
	// Ignore them.
	case typ == auparse.AUDIT_REPLACE:
		return true
	// Messages from 1300-2999 are valid audit message types.
	case (typ < auparse.AUDIT_USER_AUTH || typ > auparse.AUDIT_LAST_USER_MSG2) && typ != auparse.AUDIT_LOGIN:
		return true
	}

	return false
}

func buildMetricbeatEvent(msgs []*auparse.AuditMessage, config Config) mb.Event {
	auditEvent, err := aucoalesce.CoalesceMessages(msgs)
	if err != nil {
		// Add messages on error so that it's possible to debug the problem.
		out := mb.Event{RootFields: common.MapStr{}, Error: err}
		addEventOriginal(msgs, out.RootFields)
		return out
	}

	if config.ResolveIDs {
		aucoalesce.ResolveIDs(auditEvent)
	}

	eventOutcome := auditEvent.Result
	if eventOutcome == "fail" {
		eventOutcome = "failure"
	}
	out := mb.Event{
		Timestamp: auditEvent.Timestamp,
		RootFields: common.MapStr{
			"event": common.MapStr{
				"category": auditEvent.Category.String(),
				"action":   auditEvent.Summary.Action,
				"outcome":  eventOutcome,
			},
		},
		ModuleFields: common.MapStr{
			"message_type": strings.ToLower(auditEvent.Type.String()),
			"sequence":     auditEvent.Sequence,
			"result":       auditEvent.Result,
			"data":         createAuditdData(auditEvent.Data),
		},
	}
	if auditEvent.Session != uidUnset {
		out.ModuleFields.Put("session", auditEvent.Session)
	}

	// Add root level fields.
	addUser(auditEvent.User, out.RootFields)
	addProcess(auditEvent.Process, out.RootFields)
	addFile(auditEvent.File, out.RootFields)
	addAddress(auditEvent.Source, "source", out.RootFields)
	addAddress(auditEvent.Dest, "destination", out.RootFields)
	addNetwork(auditEvent.Net, out.RootFields)
	if len(auditEvent.Tags) > 0 {
		out.RootFields.Put("tags", auditEvent.Tags)
	}
	if config.Warnings && len(auditEvent.Warnings) > 0 {
		warnings := make([]string, 0, len(auditEvent.Warnings))
		for _, err := range auditEvent.Warnings {
			warnings = append(warnings, err.Error())
		}
		out.RootFields.Put("error.message", warnings)
		addEventOriginal(msgs, out.RootFields)
	}
	if config.RawMessage {
		addEventOriginal(msgs, out.RootFields)
	}

	// Add module fields.
	m := out.ModuleFields
	if auditEvent.Summary.Actor.Primary != "" {
		m.Put("summary.actor.primary", auditEvent.Summary.Actor.Primary)
	}
	if auditEvent.Summary.Actor.Secondary != "" {
		m.Put("summary.actor.secondary", auditEvent.Summary.Actor.Secondary)
	}
	if auditEvent.Summary.Object.Primary != "" {
		m.Put("summary.object.primary", auditEvent.Summary.Object.Primary)
	}
	if auditEvent.Summary.Object.Secondary != "" {
		m.Put("summary.object.secondary", auditEvent.Summary.Object.Secondary)
	}
	if auditEvent.Summary.Object.Type != "" {
		m.Put("summary.object.type", auditEvent.Summary.Object.Type)
	}
	if auditEvent.Summary.How != "" {
		m.Put("summary.how", auditEvent.Summary.How)
	}
	if len(auditEvent.Paths) > 0 {
		m.Put("paths", auditEvent.Paths)
	}

	normalizeEventFields(auditEvent, out.RootFields)

	// User set for related.user
	var userSet common.StringSet
	if config.ResolveIDs {
		userSet = make(common.StringSet)
	}

	// Copy user.*/group.* fields from event
	setECSEntity := func(key string, ent aucoalesce.ECSEntityData, root common.MapStr, set common.StringSet) {
		if ent.ID == "" && ent.Name == "" {
			return
		}
		if ent.ID == uidUnset {
			ent.ID = ""
		}
		nameField := key + ".name"
		idField := key + ".id"
		if ent.ID != "" {
			root.Put(idField, ent.ID)
		} else {
			root.Delete(idField)
		}
		if ent.Name != "" {
			root.Put(nameField, ent.Name)
			if set != nil {
				set.Add(ent.Name)
			}
		} else {
			root.Delete(nameField)
		}
	}

	setECSEntity("user", auditEvent.ECS.User.ECSEntityData, out.RootFields, userSet)
	setECSEntity("user.effective", auditEvent.ECS.User.Effective, out.RootFields, userSet)
	setECSEntity("user.target", auditEvent.ECS.User.Target, out.RootFields, userSet)
	setECSEntity("user.changes", auditEvent.ECS.User.Changes, out.RootFields, userSet)
	setECSEntity("group", auditEvent.ECS.Group, out.RootFields, nil)

	if userSet != nil {
		if userSet.Count() != 0 {
			out.RootFields.Put("related.user", userSet.ToSlice())
		}
	}
	getStringField := func(key string, m common.MapStr) (str string) {
		if asIf, _ := m.GetValue(key); asIf != nil {
			str, _ = asIf.(string)
		}
		return str
	}

	// Remove redundant user.effective.* when it's the same as user.*
	removeRedundantEntity := func(target, original string, m common.MapStr) bool {
		for _, suffix := range []string{".id", ".name"} {
			if value := getStringField(original+suffix, m); value != "" && getStringField(target+suffix, m) == value {
				m.Delete(target)
				return true
			}
		}
		return false
	}
	removeRedundantEntity("user.effective", "user", out.RootFields)
	return out
}

func normalizeEventFields(event *aucoalesce.Event, m common.MapStr) {
	m.Put("event.kind", "event")
	if len(event.ECS.Event.Category) > 0 {
		m.Put("event.category", event.ECS.Event.Category)
	}
	if len(event.ECS.Event.Type) > 0 {
		m.Put("event.type", event.ECS.Event.Type)
	}
	if event.ECS.Event.Outcome != "" {
		m.Put("event.outcome", event.ECS.Event.Outcome)
	}
}

func addUser(u aucoalesce.User, m common.MapStr) {
	user := common.MapStr{}
	m.Put("user", user)

	for id, value := range u.IDs {
		if value == uidUnset {
			continue
		}
		switch id {
		case "uid":
			user["id"] = value
		case "gid":
			user.Put("group.id", value)
		case "euid":
			user.Put("effective.id", value)
		case "egid":
			user.Put("effective.group.id", value)
		case "suid":
			user.Put("saved.id", value)
		case "sgid":
			user.Put("saved.group.id", value)
		case "fsuid":
			user.Put("filesystem.id", value)
		case "fsgid":
			user.Put("filesystem.group.id", value)
		case "auid":
			user.Put("audit.id", value)
		default:
			user.Put(id+".id", value)
		}

		if len(u.SELinux) > 0 {
			user["selinux"] = u.SELinux
		}
	}

	for id, value := range u.Names {
		switch id {
		case "uid":
			user["name"] = value
		case "gid":
			user.Put("group.name", value)
		case "euid":
			user.Put("effective.name", value)
		case "egid":
			user.Put("effective.group.name", value)
		case "suid":
			user.Put("saved.name", value)
		case "sgid":
			user.Put("saved.group.name", value)
		case "fsuid":
			user.Put("filesystem.name", value)
		case "fsgid":
			user.Put("filesystem.group.name", value)
		case "auid":
			user.Put("audit.name", value)
		default:
			user.Put(id+".name", value)
		}
	}
}

func addProcess(p aucoalesce.Process, m common.MapStr) {
	if p.IsEmpty() {
		return
	}

	process := common.MapStr{}
	m.Put("process", process)
	if p.PID != "" {
		if pid, err := strconv.Atoi(p.PID); err == nil {
			process["pid"] = pid
		}
	}
	if p.PPID != "" {
		if ppid, err := strconv.Atoi(p.PPID); err == nil {
			process["parent"] = common.MapStr{
				"pid": ppid,
			}
		}
	}
	if p.Title != "" {
		process["title"] = p.Title
	}
	if p.Name != "" {
		process["name"] = p.Name
	}
	if p.Exe != "" {
		process["executable"] = p.Exe
	}
	if p.CWD != "" {
		process["working_directory"] = p.CWD
	}
	if len(p.Args) > 0 {
		process["args"] = p.Args
	}
}

func addFile(f *aucoalesce.File, m common.MapStr) {
	if f == nil {
		return
	}

	file := common.MapStr{}
	m.Put("file", file)
	if f.Path != "" {
		file["path"] = f.Path
	}
	if f.Device != "" {
		file["device"] = f.Device
	}
	if f.Inode != "" {
		file["inode"] = f.Inode
	}
	if f.Mode != "" {
		file["mode"] = f.Mode
	}
	if f.UID != "" {
		file["uid"] = f.UID
	}
	if f.GID != "" {
		file["gid"] = f.GID
	}
	if f.Owner != "" {
		file["owner"] = f.Owner
	}
	if f.Group != "" {
		file["group"] = f.Group
	}
	if len(f.SELinux) > 0 {
		file["selinux"] = f.SELinux
	}
}

func addAddress(addr *aucoalesce.Address, key string, m common.MapStr) {
	if addr == nil {
		return
	}

	address := common.MapStr{}
	m.Put(key, address)
	if addr.Hostname != "" {
		address["domain"] = addr.Hostname
	}
	if addr.IP != "" {
		address["ip"] = addr.IP
	}
	if addr.Port != "" {
		address["port"] = addr.Port
	}
	if addr.Path != "" {
		address["path"] = addr.Path
	}
}

func addNetwork(net *aucoalesce.Network, m common.MapStr) {
	if net == nil {
		return
	}

	network := common.MapStr{
		"direction": net.Direction,
	}
	m.Put("network", network)
}

func addEventOriginal(msgs []*auparse.AuditMessage, m common.MapStr) {
	const key = "event.original"
	if len(msgs) == 0 {
		return
	}
	original, _ := m.GetValue(key)
	if original != nil {
		return
	}
	rawMsgs := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		rawMsgs = append(rawMsgs, "type="+msg.RecordType.String()+" msg="+msg.RawData)
	}
	m.Put(key, rawMsgs)
}

func createAuditdData(data map[string]string) common.MapStr {
	out := make(common.MapStr, len(data))
	for key, v := range data {
		if strings.HasPrefix(key, "socket_") {
			out.Put("socket."+key[7:], v)
			continue
		}

		out.Put(key, v)
	}
	return out
}

// stream type

// stream receives callbacks from the libaudit.Reassembler for completed events
// or lost events that are detected by gaps in sequence numbers.
type stream struct {
	done <-chan struct{}
	out  chan<- []*auparse.AuditMessage
}

func (s *stream) ReassemblyComplete(msgs []*auparse.AuditMessage) {
	select {
	case <-s.done:
		return
	case s.out <- msgs:
	}
}

func (s *stream) EventsLost(count int) {
	reassemblerGapsMetric.Add(int64(count))
}

// nonBlockingStream behaves as stream above, except that it will never block
// on backpressure from the publishing pipeline.
// Instead, events will be discarded.
type nonBlockingStream stream

func (s *nonBlockingStream) ReassemblyComplete(msgs []*auparse.AuditMessage) {
	select {
	case <-s.done:
		return
	case s.out <- msgs:
	default:
		userspaceLostMetric.Add(int64(len(msgs)))
	}
}

func (s *nonBlockingStream) EventsLost(count int) {
	(*stream)(s).EventsLost(count)
}

func hasMulticastSupport() bool {
	// Check the kernel version because 3.16+ should have multicast
	// support.
	major, minor, _, err := kernelVersion()
	if err != nil {
		// Assume not supported.
		return false
	}

	switch {
	case major > 3,
		major == 3 && minor >= 16:
		return true
	}

	return false
}

func kernelVersion() (major, minor int, full string, err error) {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		return 0, 0, "", err
	}

	length := len(uname.Release)
	data := make([]byte, length)
	for i, v := range uname.Release {
		if v == 0 {
			length = i
			break
		}
		data[i] = byte(v)
	}

	release := string(data[:length])
	parts := strings.SplitN(release, ".", 3)
	if len(parts) < 2 {
		return 0, 0, release, fmt.Errorf("failed to parse uname release '%v'", release)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, release, fmt.Errorf("failed to parse major version from '%v': %w", release, err)
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, release, fmt.Errorf("failed to parse minor version from '%v': %w", release, err)
	}

	return major, minor, release, nil
}

func determineSocketType(c *Config, log *logp.Logger) (string, error) {
	client, err := libaudit.NewAuditClient(nil)
	if err != nil {
		if c.SocketType == "" {
			return "", fmt.Errorf("failed to create audit client: %w", err)
		}
		// Ignore errors if a socket type has been specified. It will fail during
		// further setup and its necessary for unit tests to pass
		return c.SocketType, nil
	}
	defer client.Close()
	status, err := client.GetStatus()
	if err != nil {
		if c.SocketType == "" {
			return "", fmt.Errorf("failed to get audit status: %w", err)
		}
		return c.SocketType, nil
	}
	rules := c.rules()

	isLocked := status.Enabled == auditLocked
	hasMulticast := hasMulticastSupport()
	hasRules := len(rules) > 0

	const useAutodetect = "Remove the socket_type option to have auditbeat " +
		"select the most suitable subscription method."
	switch c.SocketType {
	case unicast:
		if isLocked {
			log.Errorf("requested unicast socket_type is not available "+
				"because audit configuration is locked in the kernel "+
				"(enabled=2). %s", useAutodetect)
			return "", errors.New("unicast socket_type not available")
		}
		return c.SocketType, nil

	case multicast:
		if hasMulticast {
			if hasRules {
				log.Warn("The audit rules specified in the configuration " +
					"cannot be applied when using a multicast socket_type.")
			}
			return c.SocketType, nil
		}
		log.Errorf("socket_type is set to multicast but based on the "+
			"kernel version, multicast audit subscriptions are not supported. %s",
			useAutodetect)
		return "", errors.New("multicast socket_type not available")

	default:
		// attempt to determine the optimal socket_type
		if hasMulticast {
			if hasRules {
				if isLocked {
					log.Warn("Audit rules specified in the configuration " +
						"cannot be applied because the audit rules have been locked " +
						"in the kernel (enabled=2). A multicast audit subscription " +
						"will be used instead, which does not support setting rules")
					return multicast, nil
				}
				return unicast, nil
			}
			return multicast, nil
		}
		if isLocked {
			log.Errorf("Cannot continue: audit configuration is locked " +
				"in the kernel (enabled=2) which prevents using unicast " +
				"sockets. Multicast audit subscriptions are not available " +
				"in this kernel. Disable locking the audit configuration " +
				"to use auditbeat.")
			return "", errors.New("no connection to audit available")
		}
		return unicast, nil
	}
}

func getBackpressureStrategy(value string, logger *logp.Logger) backpressureStrategy {
	switch value {
	case "kernel":
		return bsKernel
	case "userspace", "user-space":
		return bsUserSpace
	case "auto":
		return bsAuto
	case "both":
		return bsKernel | bsUserSpace
	case "none":
		return 0
	default:
		logger.Warn("Unknown value for the 'backpressure_strategy' option. Using default.")
		fallthrough
	case "", "default":
		return bsAuto
	}
}

func buildPIDIgnoreRule(pid int) (ruleData auditRule, err error) {
	r := rule.SyscallRule{
		Type:   rule.AppendSyscallRuleType,
		List:   "exit",
		Action: "never",
		Filters: []rule.FilterSpec{
			{
				Type:       rule.ValueFilterType,
				LHS:        "pid",
				Comparator: "=",
				RHS:        strconv.Itoa(pid),
			},
		},
		Syscalls: []string{"all"},
		Keys:     nil,
	}
	ruleData.flags = fmt.Sprintf("-A exit,never -F pid=%d -S all", pid)
	ruleData.data, err = rule.Build(&r)
	return ruleData, err
}
