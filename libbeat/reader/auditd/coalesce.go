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

//go:build linux

package auditd

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	libaudit "github.com/elastic/go-libaudit/v2"
	"github.com/elastic/go-libaudit/v2/aucoalesce"
	"github.com/elastic/go-libaudit/v2/auparse"
)

// coalescingParser groups audit records by sequence number using go-libaudit's
// Reassembler and produces compound events via aucoalesce.CoalesceMessages.
// It implements reader.Reader and libaudit.Stream.
type coalescingParser struct {
	cfg    Config
	inner  reader.Reader
	reasm  *libaudit.Reassembler
	logger *logp.Logger

	pending     []pendingEvent
	deferredErr error
	node        string // most recently seen node= value
	bytesRead   int    // bytes consumed from inner since last distribution
}

type pendingEvent struct {
	evt   *aucoalesce.Event
	bytes int
}

var (
	_ reader.Reader   = (*coalescingParser)(nil)
	_ libaudit.Stream = (*coalescingParser)(nil)
)

func newCoalescingParser(r reader.Reader, cfg Config, logger *logp.Logger) *coalescingParser {
	p := &coalescingParser{
		cfg:    cfg,
		inner:  r,
		logger: logger.Named("reader_auditd_coalesce"),
	}
	reasm, err := libaudit.NewReassembler(reassemblerMaxInFlight, reassemblerTimeout, p)
	if err != nil {
		// This should never happen. The only error return
		// from libaudit.NewReassembler is for the case that
		// the libaudit.Stream parameter, p, is nil. This is
		// never the case.
		panic("auditd: failed to create reassembler: " + err.Error())
	}
	p.reasm = reasm
	return p
}

const (
	reassemblerMaxInFlight = 50
	reassemblerTimeout     = 2 * time.Second
)

// ReassemblyComplete implements libaudit.Stream. It is called by the
// Reassembler when a group of records sharing the same sequence number
// is complete.
func (p *coalescingParser) ReassemblyComplete(msgs []*auparse.AuditMessage) {
	evt, err := aucoalesce.CoalesceMessages(msgs)
	if err != nil {
		if p.cfg.LogErrors {
			p.logger.Debugw("coalesce error", "error", err)
		}
		return
	}
	p.pending = append(p.pending, pendingEvent{evt: evt})
}

// EventsLost implements libaudit.Stream. It is called when sequence gaps are
// detected during reassembly.
func (p *coalescingParser) EventsLost(count int) {
	p.logger.Warnw("audit events lost (sequence gap)", "count", count)
}

// Next returns the next coalesced audit event. It reads lines from the inner
// reader, pushes them into the Reassembler, and returns compound events as
// they complete. Timeout-based flushing relies on Maintain() being called at
// the top of each iteration; the TimeoutReader in the filestream chain ensures
// Next() is re-entered periodically via ErrReadDeadline.
func (p *coalescingParser) Next() (reader.Message, error) {
	for {
		// Deliver deferred error from a previous Close flush.
		if p.deferredErr != nil && len(p.pending) == 0 {
			err := p.deferredErr
			p.deferredErr = nil
			return reader.Message{}, err
		}

		// Drain pending coalesced events. When entering this block with
		// undistributed bytes, spread them across all pending events so
		// that none end up with Bytes=0 (which would make them appear
		// empty to the filestream session and get dropped).
		if len(p.pending) != 0 {
			if p.bytesRead > 0 {
				p.distributeBytesRead()
			}
			pe := p.pending[0]
			p.pending[0] = pendingEvent{} // allow GC
			p.pending = p.pending[1:]
			return p.eventToMessage(pe.evt, pe.bytes), nil
		}

		// Flush timed-out groups.
		if err := p.reasm.Maintain(); err != nil {
			return reader.Message{}, err
		}
		if len(p.pending) != 0 {
			continue
		}

		// Read next line from inner reader.
		msg, err := p.inner.Next()
		if err != nil {
			if errors.Is(err, reader.ErrReadDeadline) {
				continue
			}
			// Terminal error (EOF, context cancelled): flush all pending groups.
			p.reasm.Close()
			if len(p.pending) != 0 {
				p.deferredErr = err
				continue
			}
			return reader.Message{}, err
		}
		p.bytesRead += msg.Bytes

		// Parse and push into Reassembler.
		line, nodeVal := stripNodePrefix(string(msg.Content))
		if nodeVal != "" {
			p.node = nodeVal
		}

		auditMsg, parseErr := auparse.ParseLogLine(line)
		if parseErr != nil {
			if p.cfg.LogErrors {
				p.logger.Debugw("skipping unparseable line", "error", parseErr)
			}
			continue
		}
		p.reasm.PushMessage(auditMsg)
	}
}

// distributeBytesRead spreads accumulated bytesRead evenly across all pending
// events so that each has a non-zero Bytes value. The remainder goes to the
// first event (the one about to be returned).
func (p *coalescingParser) distributeBytesRead() {
	// The distribution does not need to be precise per-event. The only
	// consumers of Bytes are offset tracking (cumulative sum), metrics
	// (cumulative sum), and the IsEmpty gate (non-zero check). None
	// observe individual event values, so even distribution is sufficient
	// as long as the sum is conserved and no event gets zero.
	n := len(p.pending)
	perEvent := p.bytesRead / n
	remainder := p.bytesRead % n
	for i := range p.pending {
		p.pending[i].bytes += perEvent
	}
	p.pending[0].bytes += remainder
	p.bytesRead = 0
}

// eventToMessage converts a coalesced aucoalesce.Event into a reader.Message
// with fields matching the auditd_manager integration's output namespace
// (auditd.data.*, auditd.summary.*, etc.) plus ECS root fields.
func (p *coalescingParser) eventToMessage(evt *aucoalesce.Event, bytes int) reader.Message {
	msg := reader.Message{
		Ts:    evt.Timestamp,
		Bytes: bytes,
	}

	auditdFields := mapstr.M{
		"message_type": strings.ToLower(evt.Type.String()),
		"sequence":     evt.Sequence,
		"result":       evt.Result,
	}
	if evt.Session != "" && evt.Session != uidUnset {
		auditdFields["session"] = evt.Session
	}
	if len(evt.Data) != 0 {
		auditdFields["data"] = createDataFields(evt.Data)
	}
	if p.node != "" {
		data, _ := auditdFields["data"].(mapstr.M)
		if data == nil {
			auditdFields["data"] = mapstr.M{"node": p.node}
		} else {
			data["node"] = p.node
		}
	}
	if len(evt.Paths) != 0 {
		auditdFields["paths"] = evt.Paths
	}
	if len(evt.Warnings) != 0 {
		warnings := make([]string, 0, len(evt.Warnings))
		for _, w := range evt.Warnings {
			warnings = append(warnings, w.Error())
		}
		auditdFields["warnings"] = warnings
	}

	addSummaryFields(auditdFields, evt)
	addUserFields(auditdFields, evt.User)
	addFileFields(auditdFields, evt.File)

	rootFields := mapstr.M{
		"auditd": auditdFields,
	}

	addECSEvent(rootFields, evt)
	addECSProcess(rootFields, evt.Process)
	addECSFile(rootFields, evt.File)
	addECSUser(rootFields, evt)
	addECSSource(rootFields, evt.Source)
	addECSDest(rootFields, evt.Dest)
	addECSNetwork(rootFields, evt.Net)
	if len(evt.Tags) != 0 {
		rootFields["tags"] = evt.Tags
	}

	msg.Fields = rootFields
	return msg
}

// SetReadDeadline delegates to the wrapped reader (see reader.DeadlineSetter).
func (p *coalescingParser) SetReadDeadline(t time.Time) bool {
	return reader.SetReadDeadline(p.inner, t)
}

func (p *coalescingParser) Close() error {
	if p.reasm != nil {
		p.reasm.Close()
	}
	return p.inner.Close()
}

func createDataFields(data map[string]string) mapstr.M {
	out := make(mapstr.M, len(data))
	for key, v := range data {
		if strings.HasPrefix(key, "socket_") {
			_, _ = out.Put("socket."+key[len("socket_"):], v)
			continue
		}
		_, _ = out.Put(key, v)
	}
	return out
}

func addSummaryFields(dst mapstr.M, evt *aucoalesce.Event) {
	s := evt.Summary
	if s.Actor.Primary != "" {
		_, _ = dst.Put("summary.actor.primary", s.Actor.Primary)
	}
	if s.Actor.Secondary != "" {
		_, _ = dst.Put("summary.actor.secondary", s.Actor.Secondary)
	}
	if s.Action != "" {
		_, _ = dst.Put("summary.action", s.Action)
	}
	if s.Object.Type != "" {
		_, _ = dst.Put("summary.object.type", s.Object.Type)
	}
	if s.Object.Primary != "" {
		_, _ = dst.Put("summary.object.primary", s.Object.Primary)
	}
	if s.Object.Secondary != "" {
		_, _ = dst.Put("summary.object.secondary", s.Object.Secondary)
	}
	if s.How != "" {
		_, _ = dst.Put("summary.how", s.How)
	}
}

func addUserFields(dst mapstr.M, u aucoalesce.User) {
	if len(u.IDs) != 0 {
		ids := make(mapstr.M, len(u.IDs))
		for k, v := range u.IDs {
			if v != uidUnset {
				ids[k] = v
			}
		}
		if len(ids) != 0 {
			_, _ = dst.Put("user.ids", ids)
		}
	}
	if len(u.Names) != 0 {
		_, _ = dst.Put("user.names", u.Names)
	}
	if len(u.SELinux) != 0 {
		_, _ = dst.Put("user.selinux", u.SELinux)
	}
}

func addFileFields(dst mapstr.M, f *aucoalesce.File) {
	if f == nil {
		return
	}
	if len(f.SELinux) != 0 {
		_, _ = dst.Put("file.selinux", f.SELinux)
	}
}

func addECSEvent(dst mapstr.M, evt *aucoalesce.Event) {
	event := mapstr.M{
		"kind":     "event",
		"category": evt.Category.String(),
		"action":   evt.Summary.Action,
	}
	outcome := evt.Result
	if outcome == "fail" {
		outcome = "failure"
	}
	if outcome != "" {
		event["outcome"] = outcome
	}
	if len(evt.ECS.Event.Category) != 0 {
		event["category"] = evt.ECS.Event.Category
	}
	if len(evt.ECS.Event.Type) != 0 {
		event["type"] = evt.ECS.Event.Type
	}
	if evt.ECS.Event.Outcome != "" {
		event["outcome"] = evt.ECS.Event.Outcome
	}
	dst["event"] = event
}

func addECSProcess(dst mapstr.M, proc aucoalesce.Process) {
	if proc.IsEmpty() {
		return
	}
	process := mapstr.M{}
	if proc.PID != "" {
		if pid, err := strconv.Atoi(proc.PID); err == nil {
			process["pid"] = pid
		}
	}
	if proc.PPID != "" {
		if ppid, err := strconv.Atoi(proc.PPID); err == nil {
			_, _ = process.Put("parent.pid", ppid)
		}
	}
	if proc.Title != "" {
		process["title"] = proc.Title
	}
	if proc.Name != "" {
		process["name"] = proc.Name
	}
	if proc.Exe != "" {
		process["executable"] = proc.Exe
	}
	if proc.CWD != "" {
		process["working_directory"] = proc.CWD
	}
	if len(proc.Args) != 0 {
		process["args"] = proc.Args
	}
	dst["process"] = process
}

func addECSFile(dst mapstr.M, f *aucoalesce.File) {
	if f == nil {
		return
	}
	file := mapstr.M{}
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
	if len(file) != 0 {
		dst["file"] = file
	}
}

func addECSUser(dst mapstr.M, evt *aucoalesce.Event) {
	u := evt.User
	user := mapstr.M{}

	for id, value := range u.IDs {
		if value == uidUnset {
			continue
		}
		switch id {
		case "uid":
			user["id"] = value
		case "gid":
			_, _ = user.Put("group.id", value)
		case "euid":
			_, _ = user.Put("effective.id", value)
		case "egid":
			_, _ = user.Put("effective.group.id", value)
		case "auid":
			_, _ = user.Put("audit.id", value)
		}
	}
	for id, value := range u.Names {
		switch id {
		case "uid":
			user["name"] = value
		case "gid":
			_, _ = user.Put("group.name", value)
		case "euid":
			_, _ = user.Put("effective.name", value)
		case "egid":
			_, _ = user.Put("effective.group.name", value)
		case "auid":
			_, _ = user.Put("audit.name", value)
		}
	}
	if len(u.SELinux) != 0 {
		user["selinux"] = u.SELinux
	}

	ecs := evt.ECS.User
	if ecs.ID != "" && ecs.ID != uidUnset {
		user["id"] = ecs.ID
	}
	if ecs.Name != "" {
		user["name"] = ecs.Name
	}

	if len(user) != 0 {
		dst["user"] = user
	}
}

const uidUnset = "unset"

func addECSSource(dst mapstr.M, addr *aucoalesce.Address) {
	if addr == nil {
		return
	}
	saddr := mapstr.M{}
	if addr.IP != "" {
		saddr["ip"] = addr.IP
	}
	if addr.Port != "" {
		saddr["port"] = addr.Port
	}
	if addr.Hostname != "" {
		saddr["domain"] = addr.Hostname
	}
	if len(saddr) != 0 {
		dst["source"] = saddr
	}
}

func addECSDest(dst mapstr.M, addr *aucoalesce.Address) {
	if addr == nil {
		return
	}
	daddr := mapstr.M{}
	if addr.IP != "" {
		daddr["ip"] = addr.IP
	}
	if addr.Port != "" {
		daddr["port"] = addr.Port
	}
	if addr.Hostname != "" {
		daddr["domain"] = addr.Hostname
	}
	if len(daddr) != 0 {
		dst["destination"] = daddr
	}
}

func addECSNetwork(dst mapstr.M, net *aucoalesce.Network) {
	if net == nil {
		return
	}
	dst["network"] = mapstr.M{
		"direction": net.Direction.String(),
	}
}
