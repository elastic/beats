// +build !windows

package eventlog

type Handle struct {
	name string
}

func queryEventMessageFiles(eventLogName, sourceName string) ([]Handle, error) {
	return nil, nil
}

func freeLibrary(handle Handle) error {
	return nil
}

func (el *eventLog) Open(recordNumber uint32) error {
	return nil
}

func (el *eventLog) Read() ([]LogRecord, error) {
	return nil, nil
}

func (el *eventLog) Close() error {
	return nil
}
