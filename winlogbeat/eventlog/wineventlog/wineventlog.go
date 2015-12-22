package wineventlog

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/eventlog"
	sys "github.com/elastic/beats/winlogbeat/sys/wineventlog"
)

// Debug logging functions for this package.
var (
	debugf  = logp.MakeDebug("wineventlog")
	detailf = logp.MakeDebug("wineventlog_detail")
)

var _ eventlog.EventLoggingAPI = &eventLog{}

type eventLog struct {
	remoteServer string
	channelName  string
	subscription sys.EvtHandle
	maxRead      int

	renderBuf []byte

	logPrefix string
}

// New creates and returns a new EventLoggingAPI for reading a Windows Event Log.
func New(channel string, maxRead int) eventlog.EventLoggingAPI {
	return &eventLog{
		channelName: channel,
		maxRead:     maxRead,
		renderBuf:   make([]byte, 1<<14),
		logPrefix:   fmt.Sprintf("WinEventLog[%s]", channel),
	}
}

func (l *eventLog) Name() string {
	return l.channelName
}
