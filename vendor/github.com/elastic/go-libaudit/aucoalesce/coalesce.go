// Copyright 2017-2018 Elasticsearch Inc.
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
	"fmt"
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
	Type      auparse.AuditMessageType `json:"record_type"      yaml:"record_type"`
	Result    string                   `json:"result,omitempty" yaml:"result,omitempty"`
	Session   string                   `json:"session"          yaml:"session"`
	Tags      []string                 `json:"tags,omitempty"   yaml:"tags,omitempty"`

	Summary Summary  `json:"summary"               yaml:"summary"`
	User    User     `json:"user"                  yaml:"user"`
	Process Process  `json:"process,omitempty"     yaml:"process,omitempty"`
	File    *File    `json:"file,omitempty"        yaml:"file,omitempty"`
	Source  *Address `json:"source,omitempty"      yaml:"source,omitempty"`
	Dest    *Address `json:"destination,omitempty" yaml:"destination,omitempty"`
	Net     *Network `json:"network,omitempty"     yaml:"network,omitempty"`

	Data  map[string]string   `json:"data,omitempty"  yaml:"data,omitempty"`
	Paths []map[string]string `json:"paths,omitempty" yaml:"paths,omitempty"`

	Warnings []error `json:"-" yaml:"-"`
}

type Summary struct {
	Actor  Actor  `json:"actor"             yaml:"actor"`
	Action string `json:"action,omitempty"  yaml:"action,omitempty"`
	Object Object `json:"object,omitempty"  yaml:"object,omitempty"`
	How    string `json:"how,omitempty"     yaml:"how,omitempty"`
}

type Actor struct {
	Primary   string `json:"primary,omitempty"   yaml:"primary,omitempty"`
	Secondary string `json:"secondary,omitempty" yaml:"secondary,omitempty"`
}

type Process struct {
	PID   string   `json:"pid,omitempty"   yaml:"pid,omitempty"`
	PPID  string   `json:"ppid,omitempty"  yaml:"ppid,omitempty"`
	Title string   `json:"title,omitempty" yaml:"title,omitempty"`
	Name  string   `json:"name,omitempty"  yaml:"name,omitempty"` // Comm
	Exe   string   `json:"exe,omitempty"   yaml:"exe,omitempty"`
	CWD   string   `json:"cwd,omitempty"   yaml:"cwd,omitempty"`
	Args  []string `json:"args,omitempty"  yaml:"args,omitempty"`
}

func (p Process) IsEmpty() bool {
	return p.PID == "" && p.PPID == "" && p.Title == "" && p.Name == "" &&
		p.Exe == "" && p.CWD == "" && len(p.Args) == 0
}

type User struct {
	IDs     map[string]string `json:"ids,omitempty"     yaml:"ids,omitempty"`     // Identifying data like auid, uid, euid, suid, fsuid, gid, egid, sgid, fsgid.
	Names   map[string]string `json:"names,omitempty"   yaml:"names,omitempty"`   // Mappings of ID to name (auid -> "root").
	SELinux map[string]string `json:"selinux,omitempty" yaml:"selinux,omitempty"` // SELinux labels.
}

type File struct {
	Path    string            `json:"path,omitempty"    yaml:"path,omitempty"`
	Device  string            `json:"device,omitempty"  yaml:"device,omitempty"`
	Inode   string            `json:"inode,omitempty"   yaml:"inode,omitempty"`
	Mode    string            `json:"mode,omitempty"    yaml:"mode,omitempty"` // Permissions
	UID     string            `json:"uid,omitempty"     yaml:"uid,omitempty"`
	GID     string            `json:"gid,omitempty"     yaml:"gid,omitempty"`
	Owner   string            `json:"owner,omitempty"   yaml:"owner,omitempty"`
	Group   string            `json:"group,omitempty"   yaml:"group,omitempty"`
	SELinux map[string]string `json:"selinux,omitempty" yaml:"selinux,omitempty"` // SELinux labels.
}

type Direction uint8

const (
	IncomingDir Direction = iota + 1
	OutgoingDir
)

func (d Direction) String() string {
	switch d {
	case IncomingDir:
		return "incoming"
	case OutgoingDir:
		return "outgoing"
	}
	return "unknown"
}

func (d Direction) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

type Network struct {
	Direction Direction `json:"direction" yaml:"direction"`
}

type Address struct {
	Hostname string `json:"hostname,omitempty" yaml:"hostname,omitempty"` // Hostname.
	IP       string `json:"ip,omitempty"       yaml:"ip,omitempty"`       // IPv4 or IPv6 address.
	Port     string `json:"port,omitempty"     yaml:"port,omitempty"`     // Port number.
	Path     string `json:"path,omitempty"     yaml:"path,omitempty"`     // Unix socket path.
}

type Object struct {
	Type      string `json:"type,omitempty"      yaml:"type,omitempty"`
	Primary   string `json:"primary,omitempty"   yaml:"primary,omitempty"`
	Secondary string `json:"secondary,omitempty" yaml:"secondary,omitempty"`
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
		addProcess(event)
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
			addSockaddrRecord(msg, event)
		case auparse.AUDIT_EXECVE:
			addExecveRecord(msg, event)
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
		event.Summary.Actor.Primary = auid
	}

	if uid, found := data["uid"]; found {
		event.Summary.Actor.Secondary = uid
	}

	// Ignore error because msg.Data() would have produced the same error.
	event.Tags, _ = msg.Tags()

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
	if event.User.IDs == nil {
		event.User.IDs = map[string]string{}
	}

	event.User.IDs[key] = value
}

func addSubjectSELinuxLabel(key, value string, event *Event) {
	if event.User.SELinux == nil {
		event.User.SELinux = map[string]string{}
	}

	event.User.SELinux[key] = value
}

func addSockaddrRecord(sockaddr *auparse.AuditMessage, event *Event) {
	data, err := sockaddr.Data()
	if err != nil {
		event.Warnings = append(event.Warnings, errors.Wrap(err,
			"failed to parse SOCKADDR message"))
		return
	}

	syscall, found := event.Data["syscall"]
	if !found {
		event.Warnings = append(event.Warnings, errors.New(
			"failed to add SOCKADDR data because syscall is unknown"))
		return
	}

	for k, v := range data {
		event.Data["socket_"+k] = v
	}

	switch syscall {
	case "recvfrom", "recvmsg", "accept", "accept4":
		addAddress(data, &event.Source)
		event.Net = &Network{Direction: IncomingDir}
	case "connect", "sendto", "sendmsg":
		addAddress(data, &event.Dest)
		event.Net = &Network{Direction: OutgoingDir}
	default:
		// These are the other syscalls that contain SOCKADDR, but they
		// have no clear source or destination:
		//   bind, listen, getpeername, getsockname
		return
	}
}

func addAddress(sockaddr map[string]string, addr **Address) {
	var (
		ip   = sockaddr["addr"]
		port = sockaddr["port"]
		path = sockaddr["path"]
	)

	if ip != "" || port != "" || path != "" {
		*addr = &Address{
			IP:   ip,
			Port: port,
			Path: path,
		}
	}
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

func addProcess(event *Event) {
	event.Process.PID = event.Data["pid"]
	delete(event.Data, "pid")
	event.Process.PPID = event.Data["ppid"]
	delete(event.Data, "ppid")
	event.Process.Title = event.Data["proctitle"]
	delete(event.Data, "proctitle")
	event.Process.Name = event.Data["comm"]
	delete(event.Data, "comm")
	event.Process.Exe = event.Data["exe"]
	delete(event.Data, "exe")
	event.Process.CWD = event.Data["cwd"]
	delete(event.Data, "cwd")
}

func addExecveRecord(execve *auparse.AuditMessage, event *Event) {
	data, err := execve.Data()
	if err != nil {
		event.Warnings = append(event.Warnings, errors.Wrap(err,
			"failed to parse EXECVE message"))
		return
	}

	argc, found := data["argc"]
	if !found {
		event.Warnings = append(event.Warnings,
			errors.New("argc key not found in EXECVE message"))
		return
	}
	event.Data["argc"] = argc

	count, err := strconv.ParseUint(argc, 10, 32)
	if err != nil {
		event.Warnings = append(event.Warnings, errors.Wrapf(err,
			"failed to convert argc='%v' to number", argc))
		return
	}

	var args []string
	for i := 0; i < int(count); i++ {
		key := "a" + strconv.Itoa(i)

		arg, found := data[key]
		if !found {
			event.Warnings = append(event.Warnings, errors.Errorf(
				"failed to find arg %v", key))
			return
		}

		delete(data, key)
		args = append(args, arg)
	}

	event.Process.Args = args
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

	event.Summary.Action = norm.Action

	switch norm.Object.What {
	case "file", "filesystem":
		event.Summary.Object.Type = norm.Object.What
		setFileObject(event, norm.Object.PathIndex)
	case "socket":
		event.Summary.Object.Type = norm.Object.What
		setSocketObject(event)
	default:
		event.Summary.Object.Type = norm.Object.What
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

	if event.Source == nil && len(norm.SourceIP.Values) > 0 {
		var err error
		for _, sourceIPKey := range norm.SourceIP.Values {
			if err = setSourceIP(sourceIPKey, event); err == nil {
				break
			}
		}
		if err != nil {
			event.Warnings = append(event.Warnings, errors.Errorf("failed to "+
				"set source IP using keys=%v because they were not found",
				norm.SourceIP.Values))
		}
	}
}

func getValue(key string, event *Event) (string, bool) {
	value, found := event.Data[key]
	if !found {
		// Fallback to user IDs.
		value, found = event.User.IDs[key]
	}
	return value, found
}

func setSourceIP(key string, event *Event) error {
	value, found := event.Data[key]
	if !found {
		return errors.Errorf("failed to set source IP value: key '%v' not found", key)
	}
	delete(event.Data, key)

	event.Source = &Address{
		IP: value,
	}
	event.Net = &Network{
		Direction: IncomingDir,
	}
	return nil
}

func setHow(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set how value: key '%v' not found", key)
	}

	event.Summary.How = value
	return nil
}

func setSubjectPrimary(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set subject primary value: key '%v' not found", key)
	}

	event.Summary.Actor.Primary = value
	return nil
}

func setSubjectSecondary(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set subject secondary value: key '%v' not found", key)
	}

	event.Summary.Actor.Secondary = value
	return nil
}

func setObjectPrimary(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set object primary value: key '%v' not found", key)
	}

	event.Summary.Object.Primary = value
	return nil
}

func setObjectSecondary(key string, event *Event) error {
	value, found := getValue(key, event)
	if !found {
		return errors.Errorf("failed to set object secondary value: key '%v' not found", key)
	}

	event.Summary.Object.Secondary = value
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

	event.File = &File{}

	if value, found := path["name"]; found {
		event.Summary.Object.Primary = value
		event.File.Path = value
	}

	if value, found := path["inode"]; found {
		event.File.Inode = value
	}
	if value, found := path["rdev"]; found {
		event.File.Device = value
	}

	if value, found := path["mode"]; found {
		mode, err := strconv.ParseUint(value, 8, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse file mode")
		}

		m := os.FileMode(mode)
		event.File.Mode = fmt.Sprintf("%#04o", m.Perm())

		switch {
		case m.IsRegular():
			event.Summary.Object.Type = "file"
		case m.IsDir():
			event.Summary.Object.Type = "directory"
		case m&os.ModeCharDevice != 0:
			event.Summary.Object.Type = "character-device"
		case m&modeBlockDevice != 0:
			event.Summary.Object.Type = "block-device"
		case m&os.ModeNamedPipe != 0:
			event.Summary.Object.Type = "named-pipe"
		case m&os.ModeSymlink != 0:
			event.Summary.Object.Type = "symlink"
		case m&os.ModeSocket != 0:
			event.Summary.Object.Type = "socket"
		}
	}

	if value, found := path["ouid"]; found {
		event.File.UID = value
	}
	if value, found := path["ogid"]; found {
		event.File.GID = value
	}

	for k, v := range path {
		if strings.HasPrefix(k, "obj_") {
			addFileSELinuxLabel(k[4:], v, event)
		}
	}

	return nil
}

func addFileSELinuxLabel(key, value string, event *Event) {
	if event.File.SELinux == nil {
		event.File.SELinux = map[string]string{}
	}

	event.File.SELinux[key] = value
}

func setSocketObject(event *Event) error {
	value, found := event.Data["socket_addr"]
	if found {
		event.Summary.Object.Primary = value
	} else {
		value, found = event.Data["socket_path"]
		if found {
			event.Summary.Object.Primary = value
		}
	}

	value, found = event.Data["socket_port"]
	if found {
		event.Summary.Object.Secondary = value
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
	event.Summary.How = exe

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
	event.Summary.How = comm
}
