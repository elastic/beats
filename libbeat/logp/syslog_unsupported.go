// +build windows nacl plan9

package logp

import (
	"log/syslog"

	"go.uber.org/zap/zapcore"
)

func NewSyslog(_ zapcore.Encoder, _ *syslog.Writer, _ zapcore.LevelEnabler) (zapcore.Core, nil) {
	return zapcore.NewNopCore(), nil
}
