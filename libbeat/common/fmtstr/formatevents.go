package fmtstr

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/dtfmt"
	"github.com/elastic/beats/libbeat/logp"
)

// EventFormatString implements format string support on events
// of type beat.Event.
//
// The concrete event expansion requires the field name enclosed by brackets.
// For example: '%{[field.name]}'. Field names can be separated by points or
// multiple braces. This format `%{[field.name]}` is equivalent to `%{[field][name]}`.
//
// Default values are given defined by the colon operator. For example:
// `%{[field.name]:default value}`.
type EventFormatString struct {
	formatter StringFormatter
	fields    []fieldInfo
	timestamp bool
}

type eventFieldEvaler struct {
	index int
}

type defaultEventFieldEvaler struct {
	index        int
	defaultValue string
}

type eventTimestampEvaler struct {
	formatter *dtfmt.Formatter
}

type eventFieldCompiler struct {
	keys      map[string]keyInfo
	timestamp bool
	index     int
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
	ts   time.Time
	buf  *bytes.Buffer
}

var (
	errMissingKeys   = errors.New("missing keys")
	errConvertString = errors.New("can not convert to string")
)

var eventCtxPool = &sync.Pool{
	New: func() interface{} { return &eventEvalContext{} },
}

func newEventCtx(sz int) *eventEvalContext {
	ctx := eventCtxPool.Get().(*eventEvalContext)
	if ctx.keys == nil || cap(ctx.keys) < sz {
		ctx.keys = make([]string, 0, sz)
	} else {
		ctx.keys = ctx.keys[:0]
	}

	return ctx
}

func releaseCtx(c *eventEvalContext) {
	eventCtxPool.Put(c)
}

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
		keys:      map[string]keyInfo{},
		index:     0,
		timestamp: false,
	}

	sf, err := Compile(in, efComp.compileExpression)
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
		fields:    keys,
		timestamp: efComp.timestamp,
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
func (fs *EventFormatString) Run(event *beat.Event) (string, error) {
	ctx := newEventCtx(len(fs.fields))
	defer releaseCtx(ctx)

	if ctx.buf == nil {
		ctx.buf = bytes.NewBuffer(nil)
	} else {
		ctx.buf.Reset()
	}

	if err := fs.collectFields(ctx, event); err != nil {
		return "", err
	}
	err := fs.formatter.Eval(ctx, ctx.buf)
	if err != nil {
		return "", err
	}
	return ctx.buf.String(), nil
}

// RunBytes executes the format string returning a new expanded string of type
// `[]byte` or an error if execution or event field expansion fails.
func (fs *EventFormatString) RunBytes(event *beat.Event) ([]byte, error) {
	ctx := newEventCtx(len(fs.fields))
	defer releaseCtx(ctx)

	buf := bytes.NewBuffer(nil)
	if err := fs.collectFields(ctx, event); err != nil {
		return nil, err
	}
	err := fs.formatter.Eval(ctx, buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Eval executes the format string, writing the resulting string into the provided output buffer. Returns error if execution or event field expansion fails.
func (fs *EventFormatString) Eval(out *bytes.Buffer, event *beat.Event) error {
	ctx := newEventCtx(len(fs.fields))
	defer releaseCtx(ctx)

	if err := fs.collectFields(ctx, event); err != nil {
		return err
	}
	return fs.formatter.Eval(ctx, out)
}

// IsConst checks the format string always returning the same constant string
func (fs *EventFormatString) IsConst() bool {
	return fs.formatter.IsConst()
}

// collectFields tries to extract and convert all required fields into an array
// of strings.
func (fs *EventFormatString) collectFields(
	ctx *eventEvalContext,
	event *beat.Event,
) error {
	for _, fi := range fs.fields {
		s, err := fieldString(event, fi.path)
		if err != nil {
			if fi.required {
				return err
			}

			s = ""
		}
		ctx.keys = append(ctx.keys, s)
	}

	if fs.timestamp {
		ctx.ts = event.Timestamp
	}

	return nil
}

func (e *eventFieldCompiler) compileExpression(
	s string,
	opts []VariableOp,
) (FormatEvaler, error) {
	if len(s) == 0 {
		return nil, errors.New("empty expression")
	}

	switch s[0] {
	case '[':
		return e.compileEventField(s, opts)
	case '+':
		return e.compileTimestamp(s, opts)
	default:
		return nil, fmt.Errorf(`unsupported format expression "%v"`, s)
	}
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
		return &eventFieldEvaler{idx}, nil
	}

	return &defaultEventFieldEvaler{idx, defaultValue}, nil
}

func (e *eventFieldCompiler) compileTimestamp(
	expression string,
	ops []VariableOp,
) (FormatEvaler, error) {
	if expression[0] != '+' {
		return nil, errors.New("No timestamp expression")
	}

	formatter, err := dtfmt.NewFormatter(expression[1:])
	if err != nil {
		return nil, fmt.Errorf("%v in timestamp expression", err)
	}

	e.timestamp = true
	return &eventTimestampEvaler{formatter}, nil
}

func (e *eventFieldEvaler) Eval(c interface{}, out *bytes.Buffer) error {
	type stringer interface {
		String() string
	}

	ctx := c.(*eventEvalContext)
	s := ctx.keys[e.index]
	_, err := out.WriteString(s)
	return err
}

func (e *defaultEventFieldEvaler) Eval(c interface{}, out *bytes.Buffer) error {
	type stringer interface {
		String() string
	}

	ctx := c.(*eventEvalContext)
	s := ctx.keys[e.index]
	if s == "" {
		s = e.defaultValue
	}
	_, err := out.WriteString(s)
	return err
}

func (e *eventTimestampEvaler) Eval(c interface{}, out *bytes.Buffer) error {
	ctx := c.(*eventEvalContext)
	_, err := e.formatter.Write(out, ctx.ts)
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
func fieldString(event *beat.Event, field string) (string, error) {
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
	case common.Time:
		return s.String(), nil
	case time.Time:
		return common.Time(s).String(), nil
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
