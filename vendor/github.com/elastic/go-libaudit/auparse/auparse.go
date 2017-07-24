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

package auparse

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

//go:generate sh -c "go run mk_audit_msg_types.go && gofmt -s -w zaudit_msg_types.go"
//go:generate sh -c "perl mk_audit_syscalls.pl > zaudit_syscalls.go && gofmt -s -w zaudit_syscalls.go"
//go:generate perl mk_audit_arches.pl
//go:generate go run mk_audit_exit_codes.go

const (
	typeToken = "type="
	msgToken  = "msg="
)

var (
	// errInvalidAuditHeader means some part of the audit header was invalid.
	errInvalidAuditHeader = errors.New("invalid audit message header")
	// errParseFailure indicates a generic failure to parse.
	errParseFailure = errors.New("failed to parse audit message")
)

// AuditMessage represents a single audit message.
type AuditMessage struct {
	RecordType AuditMessageType // Record type from netlink header.
	Timestamp  time.Time        // Timestamp parsed from payload in netlink message.
	Sequence   uint32           // Sequence parsed from payload.
	RawData    string           // Raw message as a string.

	offset int               // offset is the index into RawData where the header ends and message begins.
	data   map[string]string // The key value pairs parsed from the message.
	error  error             // Error that occurred while parsing.
}

// Data returns the key-value pairs that are contained in the audit message.
// This information is parsed from the raw message text the first time this
// method is called, all future invocations return the stored result. A nil
// map may be returned error is non-nil. A non-nil error is returned if there
// was a failure parsing or enriching the data.
func (m *AuditMessage) Data() (map[string]string, error) {
	if m.data != nil || m.error != nil {
		return m.data, m.error
	}

	if m.offset < 0 {
		m.error = errors.New("message has no data content")
		return nil, m.error
	}

	message, err := normalizeAuditMessage(m.RecordType, m.RawData[m.offset:])
	if err != nil {
		m.error = err
		return nil, m.error
	}

	m.data = map[string]string{}
	extractKeyValuePairs(message, m.data)

	if err = enrichData(m); err != nil {
		m.error = err
	}

	return m.data, m.error
}

// ToMapStr returns a new map containing the parsed key value pairs, the
// record_type, @timestamp, and sequence. The parsed key value pairs have
// a lower precedence than the well-known keys and will not override them.
// If an error occurred while parsing the message then an error key will be
// present.
func (m *AuditMessage) ToMapStr() map[string]string {
	// Ensure event has been parsed.
	data, err := m.Data()

	out := make(map[string]string, len(data)+4)
	for k, v := range data {
		out[k] = v
	}

	out["record_type"] = m.RecordType.String()
	out["@timestamp"] = m.Timestamp.UTC().String()
	out["sequence"] = strconv.FormatUint(uint64(m.Sequence), 10)
	out["raw_msg"] = m.RawData
	if err != nil {
		out["error"] = err.Error()
	}
	return out
}

// ParseLogLine parses an audit message as logged by the Linux audit daemon.
// It expects logs line that begin with the message type. For example,
// "type=SYSCALL msg=audit(1488862769.030:19469538)". A non-nil error is
// returned if it fails to parse the message header (type, timestamp, sequence).
func ParseLogLine(line string) (*AuditMessage, error) {
	msgIndex := strings.Index(line, msgToken)
	if msgIndex == -1 {
		return nil, errInvalidAuditHeader
	}

	// Verify type=XXX is before msg=
	if msgIndex < len(typeToken)+1 {
		return nil, errInvalidAuditHeader
	}

	// Convert the type to a number (i.e. type=SYSCALL -> 1300).
	typName := line[len(typeToken) : msgIndex-1]
	typ, err := GetAuditMessageType(typName)
	if err != nil {
		return nil, err
	}

	msg := line[msgIndex+len(msgToken):]
	return Parse(typ, msg)
}

// Parse parses an audit message in the format it was received from the kernel.
// It expects a message type, which is the message type value from the netlink
// header, and a message, which is raw data from the netlink message. The
// message should begin the the audit header that contains the timestamp and
// sequence number -- "audit(1488862769.030:19469538)".
//
// A non-nil error is returned if it fails to parse the message header
// (timestamp, sequence).
func Parse(typ AuditMessageType, message string) (*AuditMessage, error) {
	message = strings.TrimSpace(message)

	timestamp, seq, end, err := parseAuditHeader([]byte(message))
	if err != nil {
		return nil, err
	}

	msg := &AuditMessage{
		RecordType: typ,
		Timestamp:  timestamp,
		Sequence:   seq,
		offset:     indexOfMessage(message[end:]),
		RawData:    message,
	}
	return msg, nil
}

// parseAuditHeader parses the timestamp and sequence number from the audit
// message header that has the form of "audit(1490137971.011:50406):".
func parseAuditHeader(line []byte) (time.Time, uint32, int, error) {
	// Find tokens.
	start := bytes.IndexRune(line, '(')
	if start == -1 {
		return time.Time{}, 0, 0, errInvalidAuditHeader
	}
	dot := bytes.IndexRune(line[start:], '.')
	if dot == -1 {
		return time.Time{}, 0, 0, errInvalidAuditHeader
	}
	dot += start
	sep := bytes.IndexRune(line[dot:], ':')
	if sep == -1 {
		return time.Time{}, 0, 0, errInvalidAuditHeader
	}
	sep += dot
	end := bytes.IndexRune(line[sep:], ')')
	if end == -1 {
		return time.Time{}, 0, 0, errInvalidAuditHeader
	}
	end += sep

	// Parse timestamp.
	sec, err := strconv.ParseInt(string(line[start+1:dot]), 10, 64)
	if err != nil {
		return time.Time{}, 0, 0, errInvalidAuditHeader
	}
	msec, err := strconv.ParseInt(string(line[dot+1:sep]), 10, 64)
	if err != nil {
		return time.Time{}, 0, 0, errInvalidAuditHeader
	}
	tm := time.Unix(sec, msec*int64(time.Millisecond)).UTC()

	// Parse sequence.
	sequence, err := strconv.ParseUint(string(line[sep+1:end]), 10, 32)
	if err != nil {
		return time.Time{}, 0, 0, errInvalidAuditHeader
	}

	return tm, uint32(sequence), end, nil
}

func indexOfMessage(msg string) int {
	return strings.IndexFunc(msg, func(r rune) bool {
		switch r {
		case ':', ' ':
			return true
		default:
			return false
		}
	})
}

// Key/Value Parsing Helpers

var (
	// kvRegex is the regular expression used to match quoted and unquoted key
	// value pairs.
	kvRegex = regexp.MustCompile(`([a-z0-9_-]+)=((?:[^"'\s]+)|'(?:\\'|[^'])*'|"(?:\\"|[^"])*")`)

	// avcMessageRegex matches the beginning of AVC messages to parse the
	// seresult and seperms parameters. Example: "avc:  denied  { read } for  "
	avcMessageRegex = regexp.MustCompile(`avc:\s+(\w+)\s+\{\s*(.*)\s*\}\s+for\s+`)
)

// normalizeAuditMessage fixes some of the peculiarities of certain audit
// messages in order to make them parsable as key-value pairs.
func normalizeAuditMessage(typ AuditMessageType, msg string) (string, error) {
	switch typ {
	case AUDIT_AVC:
		i := avcMessageRegex.FindStringSubmatchIndex(msg)
		if len(i) != 3*2 {
			return "", errParseFailure
		}
		perms := strings.Fields(msg[i[4]:i[5]])
		msg = fmt.Sprintf("seresult=%v seperms=%v %v", msg[i[2]:i[3]], strings.Join(perms, ","), msg[i[1]:])
	case AUDIT_LOGIN:
		msg = strings.Replace(msg, "old ", "old_", 2)
		msg = strings.Replace(msg, "new ", "new_", 2)
	case AUDIT_CRED_DISP, AUDIT_USER_START, AUDIT_USER_END:
		msg = strings.Replace(msg, " (hostname=", " hostname=", 2)
		msg = strings.TrimRight(msg, ")'")
	}

	return msg, nil
}

func extractKeyValuePairs(msg string, data map[string]string) {
	matches := kvRegex.FindAllStringSubmatch(msg, -1)
	for _, m := range matches {
		key := m[1]
		value := trimQuotesAndSpace(m[2])

		// Drop fields with useless values.
		switch value {
		case "", "?", "?,", "(null)":
			continue
		}

		if key == "msg" {
			extractKeyValuePairs(value, data)
		} else {
			data[key] = value
		}
	}
}

func trimQuotesAndSpace(v string) string {
	return strings.Trim(v, `'" `)
}

// Enrichment after KV parsing

func enrichData(msg *AuditMessage) error {
	normalizeUnsetID("auid", msg.data)
	normalizeUnsetID("ses", msg.data)

	// Many different message types can have subj field so check them all.
	parseSELinuxContext("subj", msg.data)

	// Normalize success/res to result.
	result(msg.data)

	// Convert exit codes to named POSIX exit codes.
	exit(msg.data)

	// Normalize keys that are of the form key="key=user_command".
	auditRuleKey(msg.data)

	hexDecode("cwd", msg.data)

	switch msg.RecordType {
	case AUDIT_SYSCALL:
		if err := arch(msg.data); err != nil {
			return err
		}
		if err := syscall(msg.data); err != nil {
			return err
		}
		if err := hexDecode("exe", msg.data); err != nil {
			return err
		}
	case AUDIT_SOCKADDR:
		if err := saddr(msg.data); err != nil {
			return err
		}
	case AUDIT_PROCTITLE:
		if err := hexDecode("proctitle", msg.data); err != nil {
			return err
		}
	case AUDIT_USER_CMD:
		if err := hexDecode("cmd", msg.data); err != nil {
			return err
		}
	case AUDIT_TTY, AUDIT_USER_TTY:
		if err := hexDecode("data", msg.data); err != nil {
			return err
		}
	case AUDIT_EXECVE:
		if err := execveCmdline(msg.data); err != nil {
			return err
		}
	case AUDIT_PATH:
		parseSELinuxContext("obj", msg.data)
	case AUDIT_USER_LOGIN:
		// acct only exists in failed logins.
		hexDecode("acct", msg.data)
	}

	return nil
}

func arch(data map[string]string) error {
	hex, found := data["arch"]
	if !found {
		return errors.New("arch key not found")
	}

	arch, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse arch")
	}

	data["arch"] = AuditArch(arch).String()
	return nil
}

func syscall(data map[string]string) error {
	num, found := data["syscall"]
	if !found {
		return errors.New("syscall key not found")
	}

	syscall, err := strconv.Atoi(num)
	if err != nil {
		return errors.Wrap(err, "failed to parse syscall")
	}

	arch := data["arch"]
	data["syscall"] = AuditSyscalls[arch][syscall]
	return nil
}

func saddr(data map[string]string) error {
	saddr, found := data["saddr"]
	if !found {
		return errors.New("saddr key not found")
	}

	saddrData, err := parseSockaddr(saddr)
	if err != nil {
		return errors.Wrap(err, "failed to parse saddr")
	}

	delete(data, "saddr")
	for k, v := range saddrData {
		data[k] = v
	}
	return nil
}

func normalizeUnsetID(key string, data map[string]string) {
	id, found := data[key]
	if !found {
		return
	}

	switch id {
	case "4294967295", "-1":
		data[key] = "unset"
	}
}

func hexDecode(key string, data map[string]string) error {
	hexValue, found := data[key]
	if !found {
		return errors.Errorf("%v key not found", key)
	}

	ascii, err := hexToASCII(hexValue)
	if err != nil {
		// Field is not in hex. Ignore.
		return nil
	}

	data[key] = ascii
	return nil
}

func execveCmdline(data map[string]string) error {
	argc, found := data["argc"]
	if !found {
		return errors.New("argc key not found")
	}

	count, err := strconv.ParseUint(argc, 10, 32)
	if err != nil {
		return errors.Wrapf(err, "failed to convert argc='%v' to number", argc)
	}

	var args []string
	for i := 0; i < int(count); i++ {
		key := "a" + strconv.Itoa(i)

		arg, found := data[key]
		if !found {
			return errors.Errorf("failed to find arg %v", key)
		}

		if ascii, err := hexToASCII(arg); err == nil {
			arg = ascii
		}

		args = append(args, arg)
	}

	// Delete aN keys after successfully extracting all values.
	for i := 0; i < int(count); i++ {
		key := "a" + strconv.Itoa(i)
		delete(data, key)
	}

	data["cmdline"] = joinQuoted(args, " ")
	return nil
}

// joinQuoted concatenates the elements of a to create a single string. The
// separator string sep is placed between elements in the resulting string. Each
// element of a is double quoted (any inner quotes are escaped).
func joinQuoted(a []string, sep string) string {
	switch len(a) {
	case 0:
		return ""
	case 1:
		return strconv.Quote(a[0])
	}

	n := len(sep) * (len(a) - 1)
	for i := 0; i < len(a); i++ {
		n += len(a[i]) + 2
	}

	b := make([]byte, 0, n)
	b = strconv.AppendQuote(b, a[0])
	for _, s := range a[1:] {
		b = append(b, []byte(sep)...)
		b = strconv.AppendQuote(b, s)
	}
	return string(b)
}

// parseSELinuxContext parses a SELinux security context of the form
// 'user:role:domain:level:category'.
func parseSELinuxContext(key string, data map[string]string) error {
	context, found := data[key]
	if !found {
		return errors.Errorf("%v key not found", key)
	}

	keys := []string{"_user", "_role", "_domain", "_level", "_category"}
	contextParts := strings.SplitN(context, ":", len(keys))
	if len(contextParts) == 0 {
		return errors.Errorf("failed to split SELinux context field %v", key)
	}
	delete(data, key)

	for i, part := range contextParts {
		data[key+keys[i]] = part
	}
	return nil
}

func result(data map[string]string) error {
	// Syscall messages use "success". Other messages use "res".
	result, found := data["success"]
	if !found {
		result, found = data["res"]
		if !found {
			return errors.New("success and res key not found")
		}
		delete(data, "res")
	} else {
		delete(data, "success")
	}

	result = strings.ToLower(result)
	switch {
	case result == "yes", result == "1", strings.HasPrefix(result, "suc"):
		result = "success"
	default:
		result = "fail"
	}

	data["result"] = result
	return nil
}

func auditRuleKey(data map[string]string) {
	value, found := data["key"]
	if !found {
		return
	}

	// TODO: test multiple keys
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return
	}

	data["key"] = parts[1]
}

func exit(data map[string]string) error {
	value, found := data["exit"]
	if !found {
		return errors.New("exit key not found")
	}

	exitCode, err := strconv.Atoi(value)
	if err != nil {
		return errors.Wrap(err, "failed to parse exit")
	}

	if exitCode >= 0 {
		return nil
	}

	name, found := AuditErrnoToName[-1*exitCode]
	if !found {
		return nil
	}

	data["exit"] = name
	return nil
}
