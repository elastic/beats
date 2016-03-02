// +build windows

package eventlog

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/sys/eventlogging"
	sys "github.com/elastic/beats/winlogbeat/sys/wineventlog"
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
	subscription sys.EvtHandle // Handle to the subscription.
	maxRead      int           // Maximum number returned in one Read.

	renderBuf []byte             // Buffer used for rendering event.
	systemCtx sys.EvtHandle      // System render context.
	cache     *messageFilesCache // Cached mapping of source name to event message file handles.

	logPrefix     string               // String to prefix on log messages.
	eventMetadata common.EventMetadata // Field and tags to add to each event.
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (l *winEventLog) Name() string {
	return l.channelName
}

func (l *winEventLog) Open(recordNumber uint64) error {
	bookmark, err := sys.CreateBookmark(l.channelName, recordNumber)
	if err != nil {
		return err
	}
	defer sys.Close(bookmark)

	// Using a pull subscription to receive events. See:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385771(v=vs.85).aspx#pull
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return nil
	}

	subscriptionHandle, err := sys.Subscribe(
		0, // null session (used for connecting to remote event logs)
		signalEvent,
		l.channelName,
		"",       // Query - nil means all events
		bookmark, // Bookmark - for resuming from a specific event
		sys.EvtSubscribeStartAfterBookmark)
	if err != nil {
		return err
	}

	l.subscription = subscriptionHandle
	return nil
}

func (l *winEventLog) Read() ([]Record, error) {
	handles, err := sys.EventHandles(l.subscription, l.maxRead)
	if err == sys.ERROR_NO_MORE_ITEMS {
		detailf("%s No more events", l.logPrefix)
		return nil, nil
	}
	if err != nil {
		logp.Warn("%s EventHandles returned error %v Errno: %d", l.logPrefix, err)
		return nil, err
	}
	defer func() {
		for _, h := range handles {
			sys.Close(h)
		}
	}()
	detailf("%s EventHandles returned %d handles", l.logPrefix, len(handles))

	var records []Record
	for _, h := range handles {
		e, err := sys.RenderEvent(h, l.systemCtx, 0, l.renderBuf, l.cache.get)
		if err != nil {
			logp.Err("%s Dropping event with rendering error. %v", l.logPrefix, err)
			continue
		}

		r := Record{
			API:           winEventLogAPIName,
			EventLogName:  e.Channel,
			SourceName:    e.ProviderName,
			ComputerName:  e.Computer,
			RecordNumber:  e.RecordID,
			EventID:       uint32(e.EventID),
			Level:         e.Level,
			Category:      e.Task,
			Message:       e.Message,
			MessageErr:    e.MessageErr,
			EventMetadata: l.eventMetadata,
		}

		if e.TimeCreated != nil {
			r.TimeGenerated = *e.TimeCreated
		}

		if e.UserSID != nil {
			r.User = &User{
				Identifier: e.UserSID.Identifier,
				Name:       e.UserSID.Name,
				Domain:     e.UserSID.Domain,
				Type:       e.UserSID.Type.String(),
			}
		}

		records = append(records, r)
	}

	debugf("%s Read() is returning %d records", l.logPrefix, len(records))
	return records, nil
}

func (l *winEventLog) Close() error {
	debugf("%s Closing handle", l.logPrefix)
	return sys.Close(l.subscription)
}

// newWinEventLog creates and returns a new EventLog for reading event logs
// using the Windows Event Log.
func newWinEventLog(c Config) (EventLog, error) {
	eventMetadataHandle := func(providerName, sourceName string) eventlogging.MessageFiles {
		mf := eventlogging.MessageFiles{SourceName: sourceName}
		h, err := sys.OpenPublisherMetadata(0, sourceName, 0)
		if err != nil {
			mf.Err = err
			return mf
		}

		mf.Handles = []eventlogging.FileHandle{eventlogging.FileHandle{Handle: uintptr(h)}}
		return mf
	}

	freeHandle := func(handle uintptr) error {
		return sys.Close(sys.EvtHandle(handle))
	}

	ctx, err := sys.CreateRenderContext(nil, sys.EvtRenderContextSystem)
	if err != nil {
		return nil, err
	}

	return &winEventLog{
		channelName:   c.Name,
		remoteServer:  c.RemoteAddress,
		maxRead:       defaultMaxNumRead,
		renderBuf:     make([]byte, renderBufferSize),
		systemCtx:     ctx,
		cache:         newMessageFilesCache(c.Name, eventMetadataHandle, freeHandle),
		logPrefix:     fmt.Sprintf("WinEventLog[%s]", c.Name),
		eventMetadata: c.EventMetadata,
	}, nil
}

func init() {
	// Register eventlogging API if it is available.
	available, _ := sys.IsAvailable()
	if available {
		Register(winEventLogAPIName, 0, newWinEventLog, sys.Channels)
	}
}
