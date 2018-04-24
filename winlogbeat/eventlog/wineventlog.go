// +build windows

package eventlog

import (
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/sys"
	win "github.com/elastic/beats/winlogbeat/sys/wineventlog"
)

const (
	// renderBufferSize is the size in bytes of the buffer used to render events.
	renderBufferSize = 1 << 14

	// winEventLogApiName is the name used to identify the Windows Event Log API
	// as both an event type and an API.
	winEventLogAPIName = "wineventlog"
)

var winEventLogConfigKeys = append(commonConfigKeys, "batch_read_size",
	"ignore_older", "include_xml", "event_id", "forwarded", "level", "provider")

type winEventLogConfig struct {
	ConfigCommon  `config:",inline"`
	BatchReadSize int   `config:"batch_read_size"` // Maximum number of events that Read will return.
	IncludeXML    bool  `config:"include_xml"`
	Forwarded     *bool `config:"forwarded"`
	SimpleQuery   query `config:",inline"`
}

// defaultWinEventLogConfig is the default configuration for new wineventlog readers.
var defaultWinEventLogConfig = winEventLogConfig{
	BatchReadSize: 100,
}

// query contains parameters used to customize the event log data that is
// queried from the log.
type query struct {
	IgnoreOlder time.Duration `config:"ignore_older"` // Ignore records older than this period of time.
	EventID     string        `config:"event_id"`     // White-list and black-list of events.
	Level       string        `config:"level"`        // Severity level.
	Provider    []string      `config:"provider"`     // Provider (source name).
}

// Validate validates the winEventLogConfig data and returns an error describing
// any problems or nil.
func (c *winEventLogConfig) Validate() error {
	var errs multierror.Errors
	if c.Name == "" {
		errs = append(errs, fmt.Errorf("event log is missing a 'name'"))
	}

	return errs.Err()
}

// Validate that winEventLog implements the EventLog interface.
var _ EventLog = &winEventLog{}

// winEventLog implements the EventLog interface for reading from the Windows
// Event Log API.
type winEventLog struct {
	config       winEventLogConfig
	query        string
	channelName  string                   // Name of the channel from which to read.
	subscription win.EvtHandle            // Handle to the subscription.
	maxRead      int                      // Maximum number returned in one Read.
	lastRead     checkpoint.EventLogState // Record number of the last read event.

	render    func(event win.EvtHandle, out io.Writer) error // Function for rendering the event to XML.
	renderBuf []byte                                         // Buffer used for rendering event.
	outputBuf *sys.ByteBuffer                                // Buffer for receiving XML
	cache     *messageFilesCache                             // Cached mapping of source name to event message file handles.

	logPrefix string // String to prefix on log messages.
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (l *winEventLog) Name() string {
	return l.channelName
}

func (l *winEventLog) Open(state checkpoint.EventLogState) error {
	var bookmark win.EvtHandle
	var err error
	if len(state.Bookmark) > 0 {
		bookmark, err = win.CreateBookmarkFromXML(state.Bookmark)
	} else {
		bookmark, err = win.CreateBookmarkFromRecordID(l.channelName, state.RecordNumber)
	}
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

	debugf("%s using subscription query=%s", l.logPrefix, l.query)
	subscriptionHandle, err := win.Subscribe(
		0, // Session - nil for localhost
		signalEvent,
		"",       // Channel - empty b/c channel is in the query
		l.query,  // Query - nil means all events
		bookmark, // Bookmark - for resuming from a specific event
		win.EvtSubscribeStartAfterBookmark)
	if err != nil {
		return err
	}

	l.subscription = subscriptionHandle
	return nil
}

func (l *winEventLog) Read() ([]Record, error) {
	handles, _, err := l.eventHandles(l.maxRead)
	if err != nil || len(handles) == 0 {
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
		l.outputBuf.Reset()
		err := l.render(h, l.outputBuf)
		if bufErr, ok := err.(sys.InsufficientBufferError); ok {
			detailf("%s Increasing render buffer size to %d", l.logPrefix,
				bufErr.RequiredSize)
			l.renderBuf = make([]byte, bufErr.RequiredSize)
			l.outputBuf.Reset()
			err = l.render(h, l.outputBuf)
		}
		if err != nil && l.outputBuf.Len() == 0 {
			logp.Err("%s Dropping event with rendering error. %v", l.logPrefix, err)
			incrementMetric(dropReasons, err)
			continue
		}

		r, err := l.buildRecordFromXML(l.outputBuf.Bytes(), err)
		if err != nil {
			logp.Err("%s Dropping event. %v", l.logPrefix, err)
			incrementMetric(dropReasons, err)
			continue
		}

		r.Offset = checkpoint.EventLogState{
			Name:         l.channelName,
			RecordNumber: r.RecordID,
			Timestamp:    r.TimeCreated.SystemTime,
		}
		if r.Offset.Bookmark, err = l.createBookmarkFromEvent(h); err != nil {
			logp.Warn("%s failed creating bookmark: %v", l.logPrefix, err)
		}
		records = append(records, r)
		l.lastRead = r.Offset
	}

	debugf("%s Read() is returning %d records", l.logPrefix, len(records))
	return records, nil
}

func (l *winEventLog) Close() error {
	debugf("%s Closing handle", l.logPrefix)
	return win.Close(l.subscription)
}

func (l *winEventLog) eventHandles(maxRead int) ([]win.EvtHandle, int, error) {
	handles, err := win.EventHandles(l.subscription, maxRead)
	switch err {
	case nil:
		if l.maxRead > maxRead {
			debugf("%s Recovered from RPC_S_INVALID_BOUND error (errno 1734) "+
				"by decreasing batch_read_size to %v", l.logPrefix, maxRead)
		}
		return handles, maxRead, nil
	case win.ERROR_NO_MORE_ITEMS:
		detailf("%s No more events", l.logPrefix)
		return nil, maxRead, nil
	case win.RPC_S_INVALID_BOUND:
		incrementMetric(readErrors, err)
		if err := l.Close(); err != nil {
			return nil, 0, errors.Wrap(err, "failed to recover from RPC_S_INVALID_BOUND")
		}
		if err := l.Open(l.lastRead); err != nil {
			return nil, 0, errors.Wrap(err, "failed to recover from RPC_S_INVALID_BOUND")
		}
		return l.eventHandles(maxRead / 2)
	default:
		incrementMetric(readErrors, err)
		logp.Warn("%s EventHandles returned error %v", l.logPrefix, err)
		return nil, 0, err
	}
}

func (l *winEventLog) buildRecordFromXML(x []byte, recoveredErr error) (Record, error) {
	e, err := sys.UnmarshalEventXML(x)
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

	if e.Level == "" {
		// Fallback on LevelRaw if the Level is not set in the RenderingInfo.
		e.Level = win.EventLevel(e.LevelRaw).String()
	}

	if logp.IsDebug(detailSelector) {
		detailf("%s XML=%s Event=%+v", l.logPrefix, string(x), e)
	}

	r := Record{
		API:   winEventLogAPIName,
		Event: e,
	}

	if l.config.IncludeXML {
		r.XML = string(x)
	}

	return r, nil
}

// newWinEventLog creates and returns a new EventLog for reading event logs
// using the Windows Event Log.
func newWinEventLog(options *common.Config) (EventLog, error) {
	c := defaultWinEventLogConfig
	if err := readConfig(options, &c, winEventLogConfigKeys); err != nil {
		return nil, err
	}

	query, err := win.Query{
		Log:         c.Name,
		IgnoreOlder: c.SimpleQuery.IgnoreOlder,
		Level:       c.SimpleQuery.Level,
		EventID:     c.SimpleQuery.EventID,
		Provider:    c.SimpleQuery.Provider,
	}.Build()
	if err != nil {
		return nil, err
	}

	eventMetadataHandle := func(providerName, sourceName string) sys.MessageFiles {
		mf := sys.MessageFiles{SourceName: sourceName}
		h, err := win.OpenPublisherMetadata(0, sourceName, 0)
		if err != nil {
			mf.Err = err
			return mf
		}

		mf.Handles = []sys.FileHandle{{Handle: uintptr(h)}}
		return mf
	}

	freeHandle := func(handle uintptr) error {
		return win.Close(win.EvtHandle(handle))
	}

	l := &winEventLog{
		config:      c,
		query:       query,
		channelName: c.Name,
		maxRead:     c.BatchReadSize,
		renderBuf:   make([]byte, renderBufferSize),
		outputBuf:   sys.NewByteBuffer(renderBufferSize),
		cache:       newMessageFilesCache(c.Name, eventMetadataHandle, freeHandle),
		logPrefix:   fmt.Sprintf("WinEventLog[%s]", c.Name),
	}

	// Forwarded events should be rendered using RenderEventXML. It is more
	// efficient and does not attempt to use local message files for rendering
	// the event's message.
	switch {
	case c.Forwarded == nil && c.Name == "ForwardedEvents",
		c.Forwarded != nil && *c.Forwarded == true:
		l.render = func(event win.EvtHandle, out io.Writer) error {
			return win.RenderEventXML(event, l.renderBuf, out)
		}
	default:
		l.render = func(event win.EvtHandle, out io.Writer) error {
			return win.RenderEvent(event, 0, l.renderBuf, l.cache.get, out)
		}
	}

	return l, nil
}

func (l *winEventLog) createBookmarkFromEvent(evtHandle win.EvtHandle) (string, error) {
	bmHandle, err := win.CreateBookmarkFromEvent(evtHandle)
	if err != nil {
		return "", err
	}
	l.outputBuf.Reset()
	err = win.RenderBookmarkXML(bmHandle, l.renderBuf, l.outputBuf)
	win.Close(bmHandle)
	return string(l.outputBuf.Bytes()), err
}

func init() {
	// Register wineventlog API if it is available.
	available, _ := win.IsAvailable()
	if available {
		Register(winEventLogAPIName, 0, newWinEventLog, win.Channels)
	}
}
