package errx

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urso/ecslog/ctxtree"
	"github.com/urso/ecslog/fld"
)

type Builder struct {
	ctx *ctxtree.Ctx
}

type errValue struct {
	at  loc
	msg string
	ctx *ctxtree.Ctx
}

type kvpair struct {
	key   string
	value interface{}
}

type wrappedErrValue struct {
	errValue
	cause error
}

type multiErrValue struct {
	errValue
	causes []error
}

type loc struct {
	file string
	line int
}

var emptyBuilder = &Builder{}

func Errf(msg string, vs ...interface{}) error {
	return emptyBuilder.Errf(msg, vs...)
}

func Wrap(cause error, msg string, vs ...interface{}) error {
	return emptyBuilder.Wrap(cause, msg, vs...)
}

func WrapAll(causes []error, msg string, vs ...interface{}) error {
	return emptyBuilder.WrapAll(causes, msg, vs...)
}

func With(fields ...interface{}) *Builder {
	return emptyBuilder.With(fields...)
}

func (b *Builder) With(fields ...interface{}) *Builder {
	ctx := ctxtree.New(b.ctx, nil)
	ctx.AddAll(fields...)
	return &Builder{ctx}
}

func (b *Builder) Errf(msg string, vs ...interface{}) error {
	val, causes := makeErrValue(2, b.ctx, msg, vs)
	switch len(causes) {
	case 0:
		return &val
	case 1:
		return &wrappedErrValue{errValue: val, cause: causes[0]}
	default:
		return &multiErrValue{errValue: val, causes: causes}
	}
}

func (b *Builder) Wrap(cause error, msg string, vs ...interface{}) error {
	val, extra := makeErrValue(2, b.ctx, msg, vs)
	if len(extra) > 0 {
		if cause != nil {
			extra = append(extra, cause)
		}

		if len(extra) == 1 {
			return &wrappedErrValue{errValue: val, cause: extra[0]}
		}
		return &multiErrValue{errValue: val, causes: extra}
	}

	if cause == nil {
		return &val
	}

	return &wrappedErrValue{errValue: val, cause: cause}
}

func (b *Builder) WrapAll(causes []error, msg string, vs ...interface{}) error {
	if len(causes) == 0 {
		return nil
	}

	val, extra := makeErrValue(2, b.ctx, msg, vs)
	if len(extra) > 0 {
		causes = append(extra, causes...)
	}

	return &multiErrValue{errValue: val, causes: causes}
}

func makeErrValue(skip int, parent *ctxtree.Ctx, msg string, vs []interface{}) (errValue, []error) {
	var ctx *ctxtree.Ctx
	var causes []error

	m, _ := fld.Format(func(key string, idx int, val interface{}) {
		if ctx == nil {
			ctx = ctxtree.New(parent, nil)
		}

		if field, ok := (val).(fld.Field); ok {
			if key != "" {
				ctx.Add(fmt.Sprintf("%v.%v", key, field.Key), field.Value)
			} else {
				ctx.AddField(field)
			}
			return
		}

		switch v := val.(type) {
		case fld.Value:
			ctx.Add(ensureKey(key, idx), v)
		case error:
			causes = append(causes, v)
			if key != "" {
				ctx.AddField(fld.String(key, v.Error()))
			}
		default:
			ctx.AddField(fld.Any(ensureKey(key, idx), val))
		}
	}, msg, vs...)

	if ctx == nil {
		ctx = parent
	}
	return errValue{at: getCaller(skip + 1), msg: m, ctx: ctx}, causes
}

func ensureKey(key string, idx int) string {
	if key == "" {
		return fmt.Sprintf("%v", idx)
	}
	return key
}

func (e *errValue) At() (string, int) {
	return e.at.file, e.at.line
}

func (e *errValue) Error() string {
	return e.report(false)
}

func (e *errValue) Format(st fmt.State, c rune) {
	switch c {
	case 'v':
		if st.Flag('+') {
			io.WriteString(st, e.report(true))
			return
		}
		fallthrough
	case 's':
		io.WriteString(st, e.report(false))
	case 'q':
		io.WriteString(st, fmt.Sprintf("%q", e.report(false)))
	default:
		panic("unsupported format directive")
	}
}

type ctxValBuf strings.Builder

func (b *ctxValBuf) OnObjStart(key string) error {
	_, err := fmt.Fprintf((*strings.Builder)(b), "%v={", key)
	return err
}

func (b *ctxValBuf) OnObjEnd() error {
	_, err := fmt.Fprint((*strings.Builder)(b), "}")
	return err
}

func (b *ctxValBuf) OnValue(key string, v fld.Value) (err error) {
	v.Reporter.Ifc(&v, func(val interface{}) {
		_, err = fmt.Fprintf((*strings.Builder)(b), "%v=%v", key, val)
	})
	return err
}

func (e *errValue) report(verbose bool) string {
	buf := &strings.Builder{}

	if !verbose && e.msg != "" {
		return e.msg
	}

	if verbose && e.msg != "" {
		fmt.Fprintf(buf, "%v:%v", filepath.Base(e.at.file), e.at.line)
	}

	putStr(buf, e.msg)

	if verbose && e.ctx.Len() > 0 {
		pad(buf, " ")
		buf.WriteRune('(')
		e.ctx.VisitKeyValues((*ctxValBuf)(buf))
		buf.WriteRune(')')
	}

	return buf.String()
}

func (e *errValue) Context() *ctxtree.Ctx {
	return e.ctx
}

func (e *wrappedErrValue) Error() string {
	return e.report(false)
}

func (e *wrappedErrValue) Format(st fmt.State, c rune) {
	switch c {
	case 'v':
		if st.Flag('+') {
			io.WriteString(st, e.report(true))
			return
		}
		fallthrough
	case 's':
		io.WriteString(st, e.report(false))
	case 'q':
		io.WriteString(st, fmt.Sprintf("%q", e.report(false)))
	default:
		panic("unsupported format directive")
	}
}

func (e *wrappedErrValue) report(verbose bool) string {
	buf := &strings.Builder{}
	buf.WriteString(e.errValue.report(verbose))
	sep := ": "
	if verbose {
		sep = "\n\t"
	}
	putSubErr(buf, sep, e.cause, verbose)
	return buf.String()
}

func (e *wrappedErrValue) NumCauses() int {
	if e.cause == nil {
		return 0
	}
	return 1
}

func (e *wrappedErrValue) Cause(i int) error {
	if i > 0 {
		return nil
	}
	return e.cause
}

func (e *multiErrValue) NumCauses() int {
	return len(e.causes)
}

func (e *multiErrValue) Cause(i int) error {
	if i < len(e.causes) {
		return e.causes[i]
	}
	return nil
}

func getCaller(skip int) loc {
	var pcs [1]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	if n == 0 {
		return loc{}
	}

	pc := pcs[0]
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return loc{}
	}

	file, line := fn.FileLine(pc)
	return loc{
		file: file,
		line: line,
	}
}
