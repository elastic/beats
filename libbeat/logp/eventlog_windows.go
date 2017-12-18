package logp

import (
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	// eventID is arbitrary but must be between [1-1000].
	eventID  = 100
	supports = eventlog.Error | eventlog.Warning | eventlog.Info
)

const alreadyExistsMsg = "registry key already exists"

type eventLogCore struct {
	zapcore.LevelEnabler
	encoder zapcore.Encoder
	fields  []zapcore.Field
	log     *eventlog.Log
}

func newEventLog(appName string, encoder zapcore.Encoder, enab zapcore.LevelEnabler) (zapcore.Core, error) {
	if appName == "" {
		return nil, errors.New("appName cannot be empty")
	}
	appName = strings.Title(strings.ToLower(appName))

	if err := eventlog.InstallAsEventCreate(appName, supports); err != nil {
		if !strings.Contains(err.Error(), alreadyExistsMsg) {
			return nil, errors.Wrap(err, "failed to setup eventlog")
		}
	}

	log, err := eventlog.Open(appName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open eventlog")
	}

	return &eventLogCore{
		LevelEnabler: enab,
		encoder:      encoder,
		log:          log,
	}, nil
}

func (c *eventLogCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.Clone()
	clone.fields = append(clone.fields, fields...)
	return clone
}

func (c *eventLogCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *eventLogCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	buffer, err := c.encoder.EncodeEntry(entry, fields)
	if err != nil {
		return errors.Wrap(err, "failed to encode entry")
	}

	msg := buffer.String()
	switch entry.Level {
	case zapcore.DebugLevel, zapcore.InfoLevel:
		return c.log.Info(eventID, msg)
	case zapcore.WarnLevel:
		return c.log.Warning(eventID, msg)
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return c.log.Error(eventID, msg)
	default:
		return errors.Errorf("unhandled log level: %v", entry.Level)
	}
}

func (c *eventLogCore) Sync() error {
	return nil
}

func (c *eventLogCore) Clone() *eventLogCore {
	clone := *c
	clone.encoder = c.encoder.Clone()
	clone.fields = make([]zapcore.Field, len(c.fields), len(c.fields)+10)
	copy(clone.fields, c.fields)
	return &clone
}
