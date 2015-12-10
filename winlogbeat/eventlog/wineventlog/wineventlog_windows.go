// +build windows

package wineventlog

import (
	"syscall"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/eventlog"
	sys "github.com/elastic/beats/winlogbeat/sys/wineventlog"
	"golang.org/x/sys/windows"
)

// IsAvailable returns true if the Windows Event Log API is available on this
// system.
func IsAvailable() (bool, error) {
	avail, err := sys.IsAvailable()
	return avail, err
}

// Channels returns a list of the available event log channels.
func Channels() ([]string, error) {
	return sys.Channels()
}

func (l *eventLog) Open(recordNumber uint64) error {
	bookmark, err := sys.CreateBookmark(l.channelName, recordNumber)
	if err != nil {
		return err
	}

	// Using a pull subscription to receive events. See:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385771(v=vs.85).aspx#pull
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return nil
	}

	subscriptionHandle, err := sys.Subscribe(
		sys.NullEvtHandle,
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

func (l *eventLog) Read() ([]eventlog.LogRecord, error) {
	var events []eventlog.LogRecord
	for {
		handles, err := sys.EventHandles(l.subscription, l.maxRead)
		if err == sys.ERROR_NO_MORE_ITEMS {
			detailf("%s No more events", l.logPrefix)
			break
		}
		if err != nil {
			errno, _ := err.(syscall.Errno)
			logp.Warn("%s Failed to read event handles. %v %v", l.logPrefix,
				err, uint32(errno))
			return nil, err
		}
		detailf("%s EventHandles returned %d events", l.logPrefix, len(handles))

		for _, h := range handles {
			evt, _, err := sys.RenderEvent(h, sys.NullEvtHandle, 0, l.renderBuf, nil)
			if err != nil {
				logp.Err("%s Error rendering event. %v", l.logPrefix, err)
				continue
			}

			events = append(events, eventlog.LogRecord{
				EventLogName:  evt.Channel,
				SourceName:    evt.ProviderName,
				ComputerName:  evt.Computer,
				RecordNumber:  evt.RecordID,
				EventID:       uint32(evt.EventID),
				EventType:     evt.Level, // convert
				EventCategory: evt.Task,
				TimeGenerated: *evt.TimeCreated,
				UserSID:       evt.UserSID,
				Message:       evt.Message,
			})
		}
	}

	return events, nil
}

func (l *eventLog) Close() error {
	return sys.Close(l.subscription)
}
