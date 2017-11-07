// Copyright 2017 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package aucoalesce provides functions to coalesce compound audit messages
// into a single event and normalize all message types with some common fields.
package aucoalesce

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-libaudit/auparse"
	"github.com/pkg/errors"
)

// modeBlockDevice is the file mode bit representing block devices. This OS
// package does not have a constant defined for this.
const modeBlockDevice = 060000

type Event struct {
	Timestamp time.Time                `json:"@timestamp"       yaml:"timestamp"`
	Sequence  uint32                   `json:"sequence"         yaml:"sequence"`
	Category  AuditEventType           `json:"category"         yaml:"category"`
	Type      auparse.AuditMessageType `json:"record_type"      yaml:"type"`
	Result    string                   `json:"result,omitempty" yaml:"result,omitempty"`
	Session   string                   `json:"session"          yaml:"session"`
	Subject   Subject                  `json:"actor"            yaml:"actor"`
	Action    string                   `json:"action,omitempty" yaml:"action,omitempty"`
	Object    Object                   `json:"thing,omitempty"  yaml:"thing,omitempty"`
	How       string                   `json:"how,omitempty"    yaml:"how,omitempty"`
	Key       string                   `json:"key,omitempty"    yaml:"key,omitempty"`

	Data   map[string]string   `json:"data,omitempty"   yaml:"data,omitempty"`
	Paths  []map[string]string `json:"paths,omitempty"  yaml:"paths,omitempty"`
	Socket map[string]string   `json:"socket,omitempty" yaml:"socket,omitempty"`

	Warnings []error `json:"-" yaml:"-"`
}

type Subject struct {
	Primary    string            `json:"primary,omitempty"   yaml:"primary,omitempty"`
	Secondary  string            `json:"secondary,omitempty" yaml:"secondary,omitempty"`
	Attributes map[string]string `json:"attrs,omitempty"     yaml:"attrs,omitempty"`   // Other identifying data like euid, suid, fsuid, gid, egid, sgid, fsgid.
	SELinux    map[string]string `json:"selinux,omitempty"   yaml:"selinux,omitempty"` // SELinux labels.
}

type Object struct {
	Primary   string            `json:"primary,omitempty"   yaml:"primary,omitempty"`
	Secondary string            `json:"secondary,omitempty" yaml:"secondary,omitempty"`
	What      string            `json:"what,omitempty"      yaml:"what,omitempty"`
	SELinux   map[string]string `json:"selinux,omitempty"   yaml:"selinux,omitempty"`
}

// CoalesceMessages combines the given messages into a single event. It assumes
// that all the messages in the slice have the same timestamp and sequence
// number. An error is returned is msgs is empty or nil or only contains and EOE
// (end-of-event) message.
func CoalesceMessages(msgs []*auparse.AuditMessage) (*Event, error) {
	msgs = filterEOE(msgs)

	var event *Event
	var err error
	switch len(msgs) {
	case 0:
		return nil, errors.New("messages is empty")
	case 1:
		event, err = normalizeSimple(msgs[0])
	default:
		event, err = normalizeCompound(msgs)
	}

	if event != nil {
		applyNormalization(event)
	}
	return event, err
}

// filterEOE returns a slice (backed by the given msgs slice) that does not
// contain EOE (end-of-event) messages. EOE messages are sentinel messages used
// to signal the completion of an event, but they carry no data.
func filterEOE(msgs []*auparse.AuditMessage) []*auparse.AuditMessage {
	if len(msgs) > 0 && msgs[len(msgs)-1].RecordType == auparse.AUDIT_EOE {
		return msgs[:len(msgs)-1]
	}
	return msgs
}

func normalizeSimple(msg *auparse.AuditMessage) (*Event, error) {
	return newEvent(msg, nil), nil
}

func normalizeCompound(msgs []*auparse.AuditMessage) (*Event, error) {
	var special, syscall *auparse.AuditMessage
	for i, msg := range msgs {
		if i == 0 && msg.RecordType != auparse.AUDIT_SYSCALL {
			special = msg
			continue
		}
		if msg.RecordType == auparse.AUDIT_SYSCALL {
			syscall = msg
			break
		}
	}
	if syscall == nil {
		// All compound records have syscall messages.
		return nil, errors.New("missing syscall message in compound event")
	}

	event := newEvent(special, syscall)

	for _, msg := range msgs {
		switch msg.RecordType {
		case auparse.AUDIT_SYSCALL:
			delete(event.Data, "items")
		case auparse.AUDIT_PATH:
			addPathRecord(msg, event)
		case auparse.AUDIT_SOCKADDR:
			data, _ := msg.Data()
			event.Socket = data
		default:
			addFieldsToEventData(msg, event)
		}
	}

	return event, nil
}

func newEvent(msg *auparse.AuditMessage, syscall *auparse.AuditMessage) *Event {
	if msg == nil {
		msg = syscall
	}
	event := &Event{
		Timestamp: msg.Timestamp,
		Sequence:  msg.Sequence,
		Category:  GetAuditEventType(msg.RecordType),
		Type:      msg.RecordType,
		Data:      make(map[string]string, 10),
	}

	if syscall != nil {
		msg = syscall
	}

	data, err := msg.Data()
	if err != nil {
		event.Warnings = append(event.Warnings, err)
		return event
	}

	if result, found := data["result"]; found {
		event.Result = result
		delete(data, "result")
	} else {
		event.Result = "unknown"
	}

	if ses, found := data["ses"]; found {
		event.Session = ses
		delete(data, "ses")
	}

	if auid, found := data["auid"]; found {
		event.Subject.Primary = auid
	}

	if uid, found := data["uid"]; found {
		event.Subject.Secondary = uid
	}

	if key, found := data["key"]; found {
		event.Key = key
		delete(data, "key")
	}

	for k, v := range data {
		if strings.HasSuffix(k, "uid") || strings.HasSuffix(k, "gid") {
			addSubjectAttribute(k, v, event)
		} else if strings.HasPrefix(k, "subj_") {
			addSubjectSELinuxLabel(k[5:], v, event)
		} else {
			event.Data[k] = v
		}
	}

	return event
}

func addSubjectAttribute(key, value string, event *Event) {
	if event.Subject.Attributes == nil {
		event.Subject.Attributes = map[string]string{}
	}

	event.Subject.Attributes[key] = value
}

func addSubjectSELinuxLabel(key, value string, event *Event) {
	if event.Subject.SELinux == nil {
		event.Subject.SELinux = map[string]string{}
	}

	event.Subject.SELinux[key] = value
}

func addObjectSELinuxLabel(key, value string, event *Event) {
	if event.Object.SELinux == nil {
		event.Object.SELinux = map[string]string{}
	}

	event.Object.SELinux[key] = value
}

func addPathRecord(path *auparse.AuditMessage, event *Event) {
	data, err := path.Data()
	if err != nil {
		event.Warnings = append(event.Warnings, errors.Wrap(err,
			"failed to parse PATH message"))
		return
	}

	event.Paths = append(event.Paths, data)
}

func addFieldsToEventData(msg *auparse.AuditMessage, event *Event) {
	data, err := msg.Data()
	if err != nil {
		event.Warnings = append(event.Warnings,
			errors.Wrap(err, "failed to parse message"))
		return
	}

	for k, v := range data {
		if _, found := event.Data[k]; found {
			event.Warnings = append(event.Warnings, errors.Errorf(
				"duplicate key (%v) from %v message", k, msg.RecordType))
			continue
		}
		event.Data[k] = v
	}
}

func applyNormalization(event *Event) {
	setHowDefaults(event)

	var norm *Normalization
	if event.Type == auparse.AUDIT_SYSCALL {
		syscall := event.Data["syscall"]
		norm = syscallNorms[syscall]
	} else {
		norm = recordTypeNorms[event.Type.String()]
	}
	if norm == nil {
		event.Warnings = append(event.Warnings, errors.New("no normalization found for event"))
		return
	}

	event.Action = norm.Action

	switch norm.Object.What {
	case "file", "filesystem":
		event.Object.What = norm.Object.What
		setFileObject(event, norm.Object.PathIndex)
	case "socket":
		event.Object.What = norm.Object.What
		setSocketObject(event)
	default:
		event.Object.What = norm.Object.What
	}

	if len(norm.Subject.PrimaryFieldName.Values) > 0 {
		var err error
		for _, subjKey := range norm.Subject.PrimaryFieldName.Values {
			if err = setSubjectPrimary(subjKey, event); err == nil {
				break
			}
		}
		if err != nil {
			event.Warnings = append(event.Warnings, errors.Errorf("failed to set subject primary using keys=%v because they were not found", norm.Subject.PrimaryFieldName.Values))
		}
	}

	if len(norm.Subject.SecondaryFieldName.Values) > 0 {
		var err error
		for _, subjKey := range norm.Subject.SecondaryFieldName.Values {
			if err = setSubjectSecondary(subjKey, event); err == nil {
				break
			}
		}
		if err != nil {
			event.Warnings = append(event.Warnings, errors.Errorf("failed to set subject secondary using keys=%v because they were not found", norm.Subject.SecondaryFieldName.Values))
		}
	}

	if len(norm.Object.PrimaryFieldName.Values) > 0 {
		var err error
		for _, objKey := range norm.Object.PrimaryFieldName.Values {
			if err = setObjectPrimary(objKey, event); err == nil {
				break
			}
		}
		if err != nil {
			event.Warnings = append(event.Warnings, errors.Errorf("failed to set object primary using keys=%v because they were not found", norm.Object.PrimaryFieldName.Values))
		}
	}

	if len(norm.Object.SecondaryFieldName.Values) > 0 {
		var err error
		for _, objKey := range norm.Object.SecondaryFieldName.Values {
			if err = setObjectSecondary(objKey, event); err == nil {
				break
			}
		}
		if err != nil {
			event.Warnings = append(event.Warnings, errors.Errorf("failed to set object secondary using keys=%v because they were not found", norm.Object.SecondaryFieldName.Values))
		}
	}

	if len(norm.How.Values) > 0 {
		var err error
		for _, howKey := range norm.How.Values {
			if err = setHow(howKey, event); err == nil {
				break
			}
		}
		if err != nil {
			event.Warnings = append(event.Warnings, errors.Errorf("failed to set how using keys=%v because they were not found", norm.How.Values))
		}
	}
}

func getValue(key string, event *Event) (string, bool) {
	value, found := event.Data[key]
	if !found {
		value, found = event.Subject.Attributes[key]
	}
	return value, found
}

func setHow(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set how value: key '%v' not found", key)
	}

	event.How = value
	return nil
}

func setSubjectPrimary(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set subject primary value: key '%v' not found", key)
	}

	event.Subject.Primary = value
	return nil
}

func setSubjectSecondary(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set subject secondary value: key '%v' not found", key)
	}

	event.Subject.Secondary = value
	return nil
}

func setObjectPrimary(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set object primary value: key '%v' not found", key)
	}

	event.Object.Primary = value
	return nil
}

func setObjectSecondary(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set object secondary value: key '%v' not found", key)
	}

	event.Object.Secondary = value
	return nil
}

func setFileObject(event *Event, pathIndexHint int) error {
	if len(event.Paths) == 0 {
		return errors.New("path message not found")
	}

	var pathIndex int
	if len(event.Paths) > pathIndexHint {
		pathIndex = pathIndexHint
	}

	path := event.Paths[pathIndex]
	for _, p := range event.Paths[pathIndex:] {
		// Skip over PARENT and UNKNOWN types in case the path index was wrong.
		if nametype := p["nametype"]; nametype != "PARENT" && nametype != "UNKNOWN" {
			path = p
			break
		}
	}

	value, found := path["name"]
	if found {
		event.Object.Primary = value
	}

	value, found = path["inode"]
	if found {
		event.Object.Secondary = value
	}

	value, found = path["mode"]
	if found {
		mode, err := strconv.ParseUint(value, 8, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse file mode")
		}

		m := os.FileMode(mode)
		switch {
		case m.IsRegular():
			event.Object.What = "file"
		case m.IsDir():
			event.Object.What = "directory"
		case m&os.ModeCharDevice != 0:
			event.Object.What = "character-device"
		case m&modeBlockDevice != 0:
			event.Object.What = "block-device"
		case m&os.ModeNamedPipe != 0:
			event.Object.What = "named-pipe"
		case m&os.ModeSymlink != 0:
			event.Object.What = "symlink"
		case m&os.ModeSocket != 0:
			event.Object.What = "socket"
		}
	}

	for k, v := range path {
		if strings.HasPrefix(k, "obj_") {
			addObjectSELinuxLabel(k[4:], v, event)
		}
	}

	return nil
}

func setSocketObject(event *Event) error {
	value, found := event.Socket["addr"]
	if found {
		event.Object.Primary = value
	} else {
		value, found = event.Socket["path"]
		if found {
			event.Object.Primary = value
		}
	}

	value, found = event.Socket["port"]
	if found {
		event.Object.Secondary = value
	}
	return nil
}

func setHowDefaults(event *Event) {
	exe, found := event.Data["exe"]
	if !found {
		// Fallback to comm.
		exe, found = event.Data["comm"]
		if !found {
			return
		}
	}
	event.How = exe

	switch {
	case strings.HasPrefix(exe, "/usr/bin/python"),
		strings.HasPrefix(exe, "/usr/bin/sh"),
		strings.HasPrefix(exe, "/usr/bin/bash"),
		strings.HasPrefix(exe, "/usr/bin/perl"):
	default:
		return
	}

	// It's probably some kind of interpreted script so use "comm".
	comm, found := event.Data["comm"]
	if !found {
		return
	}
	event.How = comm
}
