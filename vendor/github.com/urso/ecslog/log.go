package ecslog

import (
	"fmt"
	"strconv"

	"github.com/urso/ecslog/backend"
	"github.com/urso/ecslog/ctxtree"
	"github.com/urso/ecslog/fld"
)

type Logger struct {
	ctx     ctxtree.Ctx
	name    string
	backend backend.Backend
}

type Level = backend.Level

const (
	Trace Level = backend.Trace
	Debug Level = backend.Debug
	Info  Level = backend.Info
	Error Level = backend.Error
)

func New(backend backend.Backend) *Logger {
	return &Logger{
		ctx:     ctxtree.Make(nil, nil),
		name:    "",
		backend: backend,
	}
}

func (l *Logger) IsEnabled(lvl Level) bool {
	return l.backend.IsEnabled(lvl)
}

func (l *Logger) Named(name string) *Logger {
	return &Logger{
		ctx:     ctxtree.Make(&l.ctx, nil),
		backend: l.backend.For(name),
		name:    name,
	}
}

func (l *Logger) With(args ...interface{}) *Logger {
	nl := &Logger{
		ctx:     ctxtree.Make(&l.ctx, nil),
		backend: l.backend,
	}
	nl.ctx.AddAll(args...)
	return nl
}

func (l *Logger) WithFields(fields ...fld.Field) *Logger {
	nl := &Logger{
		ctx:     ctxtree.Make(&l.ctx, nil),
		backend: l.backend,
	}
	nl.ctx.AddFields(fields...)
	return nl
}

func (l *Logger) Trace(args ...interface{})              { l.log(Trace, 1, args) }
func (l *Logger) Tracef(msg string, args ...interface{}) { l.logf(Trace, 1, msg, args) }

func (l *Logger) Debug(args ...interface{})              { l.log(Debug, 1, args) }
func (l *Logger) Debugf(msg string, args ...interface{}) { l.logf(Debug, 1, msg, args) }

func (l *Logger) Info(args ...interface{})              { l.log(Info, 1, args) }
func (l *Logger) Infof(msg string, args ...interface{}) { l.logf(Info, 1, msg, args) }

func (l *Logger) Error(args ...interface{})              { l.log(Error, 1, args) }
func (l *Logger) Errorf(msg string, args ...interface{}) { l.logf(Error, 1, msg, args) }

func (l *Logger) log(lvl Level, skip int, args []interface{}) {
	if !l.IsEnabled(lvl) {
		return
	}

	if l.backend.UseContext() {
		l.logArgsCtx(lvl, skip+1, args)
	} else {
		l.logArgs(lvl, skip+1, args)
	}
}

func (l *Logger) logf(lvl Level, skip int, msg string, args []interface{}) {
	if !l.IsEnabled(lvl) {
		return
	}

	if l.backend.UseContext() {
		l.logfMsgCtx(lvl, skip+1, msg, args)
	} else {
		l.logfMsg(lvl, skip+1, msg, args)
	}
}

func (l *Logger) logArgsCtx(lvl Level, skip int, args []interface{}) {
	msg := argsMessage(args)
	ctx := ctxtree.Make(&l.ctx, nil)

	var causes []error
	for _, arg := range args {
		switch v := arg.(type) {
		case fld.Field:
			ctx.AddField(v)
		case error:
			causes = append(causes, v)
		}
	}

	l.backend.Log(backend.Message{
		Name:    l.name,
		Level:   lvl,
		Caller:  getCaller(skip + 1),
		Message: msg,
		Context: ctx,
		Causes:  causes,
	})
}

func (l *Logger) logArgs(lvl Level, skip int, args []interface{}) {
	msg := argsMessage(args)

	var causes []error
	for _, arg := range args {
		if err, ok := arg.(error); ok {
			causes = append(causes, err)
		}
	}
	l.backend.Log(backend.Message{
		Name:    l.name,
		Level:   lvl,
		Caller:  getCaller(skip + 1),
		Message: msg,
		Context: ctxtree.Make(nil, nil),
		Causes:  causes,
	})
}

func argsMessage(args []interface{}) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		if str, ok := args[0].(string); ok {
			return str
		}
	}
	return fmt.Sprint(args...)
}

func (l *Logger) logfMsgCtx(lvl Level, skip int, msg string, args []interface{}) {
	ctx := ctxtree.Make(&l.ctx, nil)
	var causes []error
	msg, rest := fld.Format(func(key string, idx int, val interface{}) {
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
	}, msg, args...)

	if len(rest) > 0 {
		msg = fmt.Sprintf("%s {EXTRA_FIELDS: %v}", msg, rest)
	}

	l.backend.Log(backend.Message{
		Name:    l.name,
		Level:   lvl,
		Caller:  getCaller(skip + 1),
		Message: msg,
		Context: ctx,
		Causes:  causes,
	})
}

func (l *Logger) logfMsg(lvl Level, skip int, msg string, args []interface{}) {
	var causes []error
	msg, rest := fld.Format(func(key string, idx int, val interface{}) {
		if err, ok := val.(error); ok {
			causes = append(causes, err)
		}
	}, msg, args...)

	if len(rest) > 0 {
		msg = fmt.Sprintf("%s {EXTRA_FIELDS: %v}", msg, rest)
	}

	l.backend.Log(backend.Message{
		Name:    l.name,
		Level:   lvl,
		Caller:  getCaller(skip + 1),
		Message: msg,
		Context: ctxtree.Make(nil, nil),
		Causes:  causes,
	})
}

func ensureKey(key string, idx int) string {
	if key == "" {
		return strconv.FormatInt(int64(idx), 10)
	}
	return key
}

func getCaller(skip int) backend.Caller {
	return backend.GetCaller(skip + 1)
}
