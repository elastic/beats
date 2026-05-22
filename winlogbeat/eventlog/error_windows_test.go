//go:build windows

package eventlog

import (
	"testing"

	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
	"github.com/stretchr/testify/assert"
)

func TestIsRecoverable(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		isFile bool
		want   bool
	}{
		{"RPC_S_UNKNOWN_IF is recoverable", win.RPC_S_UNKNOWN_IF, false, true},
		{"RPC_S_SERVER_UNAVAILABLE recoverable", win.RPC_S_SERVER_UNAVAILABLE, false, true},
		{"RPC_S_CALL_CANCELLED recoverable", win.RPC_S_CALL_CANCELLED, false, true},
		{"ERROR_INVALID_HANDLE recoverable", win.ERROR_INVALID_HANDLE, false, true},
		{"nil is not recoverable", nil, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsRecoverable(tc.err, tc.isFile))
		})
	}
}
