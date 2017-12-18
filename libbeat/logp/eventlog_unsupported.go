// +build !windows

package logp

import (
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

func newEventLog(beatname string, encoder zapcore.Encoder, enab zapcore.LevelEnabler) (zapcore.Core, error) {
	return nil, errors.New("eventlog is only supported on Windows")
}
