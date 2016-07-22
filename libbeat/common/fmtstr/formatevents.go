package fmtstr

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// EventFormatString implements format string support on events
// of type common.MapStr.
//
// The concrete event expansion requires the field name enclosed by brackets.
// For example: '%{[field.name]}'. Field names can be separated by points or
// multiple braces. This format `%{[field.name]}` is equivalent to `%{[field][name]}`.
//
// Default values are given defined by the colon operator. For example:
// `%{[field.name]:default value}`.
type EventFormatString struct {
	formatter StringFormatter
	ctx       *eventEvalContext
	fields    []fieldInfo
}

type eventFieldEvaler struct {
	ctx   *eventEvalContext
	index int
}

type defaultEventFieldEvaler struct {
	ctx          *eventEvalContext
	index        int
	defaultValue string
}

type eventFieldCompiler struct {
	ctx   *eventEvalContext
	keys  map[string]keyInfo
	index int
}

type fieldInfo struct {
	path     string
	required bool
}

type keyInfo struct {
	index    int
	required bool
}

type eventEvalContext struct {
	keys []string
}

var (
	errMissingKeys   = errors.New("missing keys")
	errConvertString = errors.New("can not convert to string")
)

// MustCompileEvent copmiles an event format string into an runnable
// EventFormatString. Generates a panic if compilation fails.
func MustCompileEvent(in string) *EventFormatString {
	fs, err := CompileEvent(in)
	if err != nil {
		panic(err)
	}
	return fs
}

// CompileEvent compiles an event format string into an runnable
// EventFormatString. Returns error if parsing or compilation fails.
func CompileEvent(in string) (*EventFormatString, error) {
	ctx := &eventEvalContext{}
	efComp := &eventFieldCompiler{
		ctx:   ctx,
		keys:  map[string]keyInfo{},
		index: 0,
	}

	sf, err := Compile(in, efComp.compileEventField)
	if err != nil {
		return nil, err
	}

	keys := make([]fieldInfo, len(efComp.keys))
	for path, info := range efComp.keys {
		keys[info.index] = fieldInfo{
			path:     path,
			required: info.required,
		}
	}

	ctx.keys = make([]string, len(keys))
	efs := &EventFormatString{
		formatter: sf,
		ctx:       ctx,
		fields:    keys,
	}
	return efs, nil
}

// Unpack tries to initialize the EventFormatString from provided value
// (which must be a string). Unpack method satisfies go-ucfg.Unpacker interface
// required by common.Config, in order to use EventFormatString with
// `common.(*Config).Unpack()`.
func (fs *EventFormatString) Unpack(v interface{}) error {
	s, err := tryConvString(v)
	if err != nil {
		return err
	}

	tmp, err := CompileEvent(s)
	if err != nil {
		return err
	}

	// init fs from tmp
	*fs = *tmp
	return nil
}

// NumFields returns number of unique event fields used by the format string.
func (fs *EventFormatString) NumFields() int {
	return len(fs.fields)
}

// Fields returns list of unique event fields required by the format string.
func (fs *EventFormatString) Fields() []string {
	var fields []string

	for _, fi := range fs.fields {
		if fi.required {
			fields = append(fields, fi.path)
		}
	}
	return fields
}

// Run executes the format string returning a new expanded string or an error
// if execution or event field expansion fails.
func (fs *EventFormatString) Run(event common.MapStr) (string, error) {
	if err := fs.collectFields(event); err != nil {
		return "", err
	}
	return fs.formatter.Run()
}

// Eval executes the format string, writing the resulting string into the provided output buffer. Returns error if execution or event field expansion fails.
func (fs *EventFormatString) Eval(out *bytes.Buffer, event common.MapStr) error {
	if err := fs.collectFields(event); err != nil {
		return err
	}
	return fs.formatter.Eval(out)
}

// collectFields tries to extract and convert all required fields into an array
// of strings.
func (fs *EventFormatString) collectFields(event common.MapStr) error {
	for i, fi := range fs.fields {
		s, err := fieldString(event, fi.path)
		if err != nil {
			if fi.required {
				return err
			}

			s = ""
		}
		fs.ctx.keys[i] = s
	}

	return nil
}

func (e *eventFieldCompiler) compileEventField(
	field string,
	ops []VariableOp,
) (FormatEvaler, error) {
	if len(ops) > 1 {
		return nil, errors.New("Too many format modifiers given")
	}

	defaultValue := ""
	if len(ops) == 1 {
		op := ops[0]
		if op.op != ":" {
			return nil, fmt.Errorf("unsupported format operator: %v", op.op)
		}
		defaultValue = op.param
	}

	path, err := parseEventPath(field)
	if err != nil {
		return nil, err
	}

	info, found := e.keys[path]
	if !found {
		info = keyInfo{
			required: len(ops) == 0,
			index:    e.index,
		}
		e.index++
		e.keys[path] = info
	} else if !info.required && len(ops) == 0 {
		info.required = true
		e.keys[path] = info
	}

	idx := info.index

	if len(ops) == 0 {
		return &eventFieldEvaler{e.ctx, idx}, nil
	}

	return &defaultEventFieldEvaler{e.ctx, idx, defaultValue}, nil
}

func (e *eventFieldEvaler) Eval(out *bytes.Buffer) error {
	type stringer interface {
		String() string
	}

	s := e.ctx.keys[e.index]
	_, err := out.WriteString(s)
	return err
}

func (e *defaultEventFieldEvaler) Eval(out *bytes.Buffer) error {
	type stringer interface {
		String() string
	}

	s := e.ctx.keys[e.index]
	if s == "" {
		s = e.defaultValue
	}
	_, err := out.WriteString(s)
	return err
}

func parseEventPath(field string) (string, error) {
	field = strings.Trim(field, " \n\r\t")
	var fields []string

	for len(field) > 0 {
		if field[0] != '[' {
			return "", errors.New("expected field extractor start with '['")
		}

		idx := strings.IndexByte(field, ']')
		if idx < 0 {
			return "", errors.New("missing closing ']'")
		}

		path := field[1:idx]
		if path == "" {
			return "", errors.New("empty fields selector '[]'")
		}

		fields = append(fields, path)
		field = field[idx+1:]
	}

	path := strings.Join(fields, ".")
	return path, nil
}

// TODO: move to libbeat/common?
func fieldString(event common.MapStr, field string) (string, error) {
	v, err := event.GetValue(field)
	if err != nil {
		return "", err
	}

	s, err := tryConvString(v)
	if err != nil {
		logp.Warn("Can not convert key '%v' value to string", v)
	}
	return s, err
}

func tryConvString(v interface{}) (string, error) {
	type stringer interface {
		String() string
	}

	switch s := v.(type) {
	case string:
		return s, nil
	case []byte:
		return string(s), nil
	case stringer:
		return s.String(), nil
	case bool:
		if s {
			return "true", nil
		}
		return "false", nil
	case int8, int16, int32, int64, int:
		i := reflect.ValueOf(s).Int()
		return strconv.FormatInt(i, 10), nil
	case uint8, uint16, uint32, uint64, uint:
		u := reflect.ValueOf(s).Uint()
		return strconv.FormatUint(u, 10), nil
	case float32:
		return strconv.FormatFloat(float64(s), 'g', -1, 32), nil
	case float64:
		return strconv.FormatFloat(s, 'g', -1, 64), nil
	default:
		return "", errConvertString
	}
}
