package kprobes

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type eventID struct {
	ePath string
	eType uint32
}

type eventsVerifier struct {
	sync.Mutex
	basePath         string
	eventsToExpect   map[eventID]int
	eventsToExpectNr int
}

func newEventsVerifier(basePath string) (*eventsVerifier, error) {
	return &eventsVerifier{
		basePath:       basePath,
		eventsToExpect: make(map[eventID]int),
	}, nil
}

func (e *eventsVerifier) Emit(ePath string, _ uint32, eType uint32) error {
	e.Lock()
	defer e.Unlock()

	eID := eventID{
		ePath: ePath,
		eType: eType,
	}
	_, exists := e.eventsToExpect[eID]

	if !exists {
		return ErrVerifyUnexpectedEvent
	}

	e.eventsToExpect[eID]--
	return nil
}

// addEventToExpect adds an event to the eventsVerifier's list of expected events.
func (e *eventsVerifier) addEventToExpect(ePath string, eType uint32) {
	e.Lock()
	defer e.Unlock()

	e.eventsToExpectNr++

	eID := eventID{
		ePath: ePath,
		eType: eType,
	}
	_, exists := e.eventsToExpect[eID]

	if !exists {
		e.eventsToExpect[eID] = 1
		return
	}

	e.eventsToExpect[eID]++
}

func (e *eventsVerifier) GenerateEvents() error {
	targetFilePath := filepath.Join(e.basePath, "validate_file")
	targetMovedFilePath := targetFilePath + "_moved"

	// create file - generates 1 event
	file, err := os.OpenFile(targetFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_CREATE)

	// truncate file - generates 1 event
	if err := file.Truncate(0); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_MODIFY)

	// write to file - generates 1 event
	if _, err := file.WriteString("test"); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_MODIFY)

	// change owner of file - generates 1 event
	if err := file.Chown(os.Getuid(), os.Getgid()); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)

	// change mode of file - generates 1 event
	if err := file.Chmod(0700); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)

	if err := file.Close(); err != nil {
		return err
	}

	// change times of file - generates 1 event
	if err := unix.Utimes(targetFilePath, []unix.Timeval{
		unix.NsecToTimeval(time.Now().UnixNano()),
		unix.NsecToTimeval(time.Now().UnixNano()),
	}); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)

	// add attribute to file - generates 1 event
	// Note that this may fail if the filesystem doesn't support extended attributes
	// This is allVerified we just skip adding the respective event to verify
	attrName := "user.myattr"
	attrValue := []byte("Hello, xattr!")
	if err := unix.Setxattr(targetFilePath, attrName, attrValue, 0); err != nil {
		if !errors.Is(err, unix.EOPNOTSUPP) {
			return err
		}
	} else {
		e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
	}

	// move file - generates 2 events
	if err := os.Rename(targetFilePath, targetMovedFilePath); err != nil {
		return nil
	}
	e.addEventToExpect(targetFilePath, unix.IN_MOVED_FROM)
	e.addEventToExpect(targetMovedFilePath, unix.IN_MOVED_TO)

	// remove file - generates 1 event
	if err := os.Remove(targetMovedFilePath); err != nil {
		return err
	}
	e.addEventToExpect(targetMovedFilePath, unix.IN_DELETE)

	// create a directory - generates 1 event
	if err := os.Mkdir(targetFilePath, 0600); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_CREATE)

	// change mode of directory - generates 1 event
	if err := os.Chmod(targetFilePath, 0644); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)

	// change owner of directory - generates 1 event
	if err := os.Chown(targetFilePath, os.Getuid(), os.Getgid()); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)

	// add attribute to directory - generates 1 event
	// Note that this may fail if the filesystem doesn't support extended attributes
	// This is allVerified we just skip adding the respective event to verify
	if err := unix.Setxattr(targetFilePath, attrName, attrValue, 0); err != nil {
		if !errors.Is(err, unix.EOPNOTSUPP) {
			return err
		}
	} else {
		e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
	}

	// change times of directory - generates 1 event
	if err := unix.Utimes(targetFilePath, []unix.Timeval{
		unix.NsecToTimeval(time.Now().UnixNano()),
		unix.NsecToTimeval(time.Now().UnixNano()),
	}); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)

	// move directory - generates 2 events
	if err := os.Rename(targetFilePath, targetMovedFilePath); err != nil {
		return err
	}
	e.addEventToExpect(targetFilePath, unix.IN_MOVED_FROM)
	e.addEventToExpect(targetMovedFilePath, unix.IN_MOVED_TO)

	// remove the directory - generates 1 event
	if err := os.Remove(targetMovedFilePath); err != nil {
		return err
	}
	e.addEventToExpect(targetMovedFilePath, unix.IN_DELETE)

	return nil
}

// Verified checks that all expected events filled during GenerateEvents() are present without any missing
// or duplicated.
func (e *eventsVerifier) Verified() error {

	if e.eventsToExpectNr == 0 {
		return errors.New("no events to expect")
	}

	for _, status := range e.eventsToExpect {
		switch {
		case status < 0:
			return ErrVerifyOverlappingEvents
		case status > 0:
			return ErrVerifyMissingEvents
		}
	}

	return nil
}
