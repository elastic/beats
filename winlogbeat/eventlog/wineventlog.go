// +build windows

package eventlog

import (
	"fmt"
	"strconv"
	"syscall"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/sys"
	win "github.com/elastic/beats/winlogbeat/sys/wineventlog"
	"golang.org/x/sys/windows"
)

const (
	// defaultMaxNumRead is the maximum number of event Read will return.
	defaultMaxNumRead = 100

	// renderBufferSize is the size in bytes of the buffer used to render events.
	renderBufferSize = 1 << 14

	// winEventLogApiName is the name used to identify the Windows Event Log API
	// as both an event type and an API.
	winEventLogAPIName = "wineventlog"
)

// Validate that winEventLog implements the EventLog interface.
var _ EventLog = &winEventLog{}

// winEventLog implements the EventLog interface for reading from the Windows
// Event Log API.
type winEventLog struct {
	remoteServer string        // Name of the remote server from which to read.
	channelName  string        // Name of the channel from which to read.
	subscription win.EvtHandle // Handle to the subscription.
	maxRead      int           // Maximum number returned in one Read.

	renderBuf []byte             // Buffer used for rendering event.
	cache     *messageFilesCache // Cached mapping of source name to event message file handles.

	logPrefix     string               // String to prefix on log messages.
	eventMetadata common.EventMetadata // Field and tags to add to each event.
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (l *winEventLog) Name() string {
	return l.channelName
}

func (l *winEventLog) Open(recordNumber uint64) error {
	bookmark, err := win.CreateBookmark(l.channelName, recordNumber)
	if err != nil {
		return err
	}
	defer win.Close(bookmark)

	// Using a pull subscription to receive events. See:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385771(v=vs.85).aspx#pull
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return nil
	}

	subscriptionHandle, err := win.Subscribe(
		0, // null session (used for connecting to remote event logs)
		signalEvent,
		l.channelName,
		"",       // Query - nil means all events
		bookmark, // Bookmark - for resuming from a specific event
		win.EvtSubscribeStartAfterBookmark)
	if err != nil {
		return err
	}

	l.subscription = subscriptionHandle
	return nil
}

func (l *winEventLog) Read() ([]Record, error) {
	handles, err := win.EventHandles(l.subscription, l.maxRead)
	if err == win.ERROR_NO_MORE_ITEMS {
		detailf("%s No more events", l.logPrefix)
		return nil, nil
	}
	if err != nil {
		logp.Warn("%s EventHandles returned error %v Errno: %d", l.logPrefix, err)
		return nil, err
	}
	defer func() {
		for _, h := range handles {
			win.Close(h)
		}
	}()
	detailf("%s EventHandles returned %d handles", l.logPrefix, len(handles))

	var records []Record
	for _, h := range handles {
		x, err := win.RenderEvent(h, 0, l.renderBuf, l.cache.get)
		if bufErr, ok := err.(sys.InsufficientBufferError); ok {
			detailf("%s Increasing render buffer size to %d", l.logPrefix,
				bufErr.RequiredSize)
			l.renderBuf = make([]byte, bufErr.RequiredSize)
			x, err = win.RenderEvent(h, 0, l.renderBuf, l.cache.get)
		}
		if err != nil && x == "" {
			logp.Err("%s Dropping event with rendering error. %v", l.logPrefix, err)
			reportDrop(err)
			continue
		}

		r, err := l.buildRecordFromXML(x, err)
		if err != nil {
			logp.Err("%s Dropping event. %v", l.logPrefix, err)
			reportDrop("unmarshal")
			continue
		}
		records = append(records, r)
	}

	debugf("%s Read() is returning %d records", l.logPrefix, len(records))
	return records, nil
}

func (l *winEventLog) Close() error {
	debugf("%s Closing handle", l.logPrefix)
	return win.Close(l.subscription)
}

func (l *winEventLog) buildRecordFromXML(x string, recoveredErr error) (Record, error) {
	e, err := sys.UnmarshalEventXML([]byte(x))
	if err != nil {
		return Record{}, fmt.Errorf("Failed to unmarshal XML='%s'. %v", x, err)
	}

	err = sys.PopulateAccount(&e.User)
	if err != nil {
		debugf("%s SID %s account lookup failed. %v", l.logPrefix,
			e.User.Identifier, err)
	}

	if e.RenderErrorCode != 0 {
		// Convert the render error code to an error message that can be
		// included in the "message_error" field.
		e.RenderErr = syscall.Errno(e.RenderErrorCode).Error()
	} else if recoveredErr != nil {
		e.RenderErr = recoveredErr.Error()
	}

	if logp.IsDebug(detailSelector) {
		detailf("%s XML=%s Event=%+v", l.logPrefix, x, e)
	}

	r := Record{
		API:           winEventLogAPIName,
		EventMetadata: l.eventMetadata,
		Event:         e,
	}

	return r, nil
}

// reportDrop reports a dropped event log record and the reason as an expvar
// metric. The reason should be a windows syscall.Errno or a string. Any other
// types will be reported under the "other" key.
func reportDrop(reason interface{}) {
	switch t := reason.(type) {
	default:
		dropReasons.Add("other", 1)
	case string:
		dropReasons.Add(t, 1)
	case syscall.Errno:
		dropReasons.Add(strconv.Itoa(int(t)), 1)
	}
}

// newWinEventLog creates and returns a new EventLog for reading event logs
// using the Windows Event Log.
func newWinEventLog(c Config) (EventLog, error) {
	eventMetadataHandle := func(providerName, sourceName string) sys.MessageFiles {
		mf := sys.MessageFiles{SourceName: sourceName}
		h, err := win.OpenPublisherMetadata(0, sourceName, 0)
		if err != nil {
			mf.Err = err
			return mf
		}

		mf.Handles = []sys.FileHandle{sys.FileHandle{Handle: uintptr(h)}}
		return mf
	}

	freeHandle := func(handle uintptr) error {
		return win.Close(win.EvtHandle(handle))
	}

	return &winEventLog{
		channelName:   c.Name,
		remoteServer:  c.RemoteAddress,
		maxRead:       defaultMaxNumRead,
		renderBuf:     make([]byte, renderBufferSize),
		cache:         newMessageFilesCache(c.Name, eventMetadataHandle, freeHandle),
		logPrefix:     fmt.Sprintf("WinEventLog[%s]", c.Name),
		eventMetadata: c.EventMetadata,
	}, nil
}

func init() {
	// Register eventlogging API if it is available.
	available, _ := win.IsAvailable()
	if available {
		Register(winEventLogAPIName, 0, newWinEventLog, win.Channels)
	}
}
