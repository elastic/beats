// +build windows nacl plan9

package logp

import (
	"go.uber.org/zap/zapcore"
)

func newSyslog(_ zapcore.Encoder, _ zapcore.LevelEnabler) (zapcore.Core, error) {
	return zapcore.NewNopCore(), nil
}
