package reader

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/xbeats/journalbeat/config"
)

const (
	CURSOR_FILE = ".journalbeat_position"
)

type Reader struct {
	j       *sdjournal.Journal
	changes chan int

	backoff       time.Duration
	backoffMax    time.Duration
	backoffFactor int
}

func New(config config.Config) (*Reader, error) {
	var j *sdjournal.Journal
	var err error

	if len(config.Paths) == 0 {
		j, err = sdjournal.NewJournal()
		if err != nil {
			logp.Err("Failed to open local journal: %v", err)
			return nil, err
		}
	} else if len(config.Paths) == 1 {
		var f os.FileInfo
		f, err = os.Stat(config.Paths[0])
		if err != nil {
			logp.Err("Failed to open file: %v", err)
			return nil, err
		}
		if f.IsDir() {
			j, err = sdjournal.NewJournalFromDir(config.Paths[0])
			if err != nil {
				logp.Err("Failed to open journal directory: %v", err)
				return nil, err
			}
		} else {
			j, err = sdjournal.NewJournalFromFiles(config.Paths...)
			if err != nil {
				logp.Err("Failed to open journal file: %v", err)
				return nil, err
			}
		}

	} else {
		j, err = sdjournal.NewJournalFromFiles(config.Paths...)
		if err != nil {
			logp.Err("Failed to open journal files: %v", err)
			return nil, err
		}
	}

	seekToSavedPosition(j)

	r := &Reader{
		j:             j,
		changes:       make(chan int),
		backoff:       config.Backoff,
		backoffMax:    config.MaxBackoff,
		backoffFactor: config.BackoffFactor,
	}
	return r, nil
}

func seekToSavedPosition(j *sdjournal.Journal) {
	if _, err := os.Stat(CURSOR_FILE); os.IsNotExist(err) {
		return
	}

	pos, err := ioutil.ReadFile(CURSOR_FILE)
	if err != nil {
		logp.Info("Cannot open cursor file, starting to tail journal")
		j.SeekTail()
		return
	}
	cursor := string(pos[:])

	j.SeekCursor(cursor)
}

func (r *Reader) Follow(stop <-chan struct{}) <-chan *beat.Event {
	out := make(chan *beat.Event)
	go r.readEntriesFromJournal(stop, out)

	return out
}

func (r *Reader) readEntriesFromJournal(stop <-chan struct{}, entries chan<- *beat.Event) {
	defer close(entries)

process:
	for {
		select {
		case <-stop:
			return
		default:
			err := r.readUntilNotNull(entries)
			if err != nil {
				logp.Err("Unexpected error while reading from journal: %v", err)
			}
			logp.Debug("reader", "End of journal reached; Backoff now.")
		}

		for {
			go r.stopOrWait(stop)

			select {
			case <-stop:
				return
			case e := <-r.changes:
				switch e {
				case sdjournal.SD_JOURNAL_NOP:
					r.wait(stop)
				case sdjournal.SD_JOURNAL_APPEND, sdjournal.SD_JOURNAL_INVALIDATE:
					continue process
				default:
					logp.Err("Unexpected change: %d", e)
				}
			}
		}
	}
}

func (r *Reader) readUntilNotNull(entries chan<- *beat.Event) error {
	n, err := r.j.Next()
	if err != nil && err != io.EOF {
		return err
	}

	for n != 0 {
		entry, err := r.j.GetEntry()
		if err != nil {
			return err
		}
		event := getEvent(entry)
		entries <- event

		n, err = r.j.Next()
		if err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}

func getEvent(entry *sdjournal.JournalEntry) *beat.Event {
	fields := common.MapStr{}
	for k, v := range entry.Fields {
		key := strings.TrimLeft(strings.ToLower(k), "_")
		fields[key] = v
	}
	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    fields,
	}
	fmt.Println("%s", event.Fields["message"])
	return &event
}

func (r *Reader) stopOrWait(stop <-chan struct{}) {
	select {
	case <-stop:
	case r.changes <- r.j.Wait(100 * time.Millisecond):
	}
}

func (r *Reader) wait(stop <-chan struct{}) {
	select {
	case <-stop:
		return
	case <-time.After(r.backoff):
	}

	if r.backoff < r.backoffMax {
		r.backoff = r.backoff * time.Duration(r.backoffFactor)
		if r.backoff > r.backoffMax {
			r.backoff = r.backoffMax
		}
		logp.Debug("reader", "Increasing backoff time to: %v factor: %v", r.backoff, r.backoffFactor)
	}
}

func (r *Reader) Close() {
	r.savePosition()
	r.j.Close()
}

func (r *Reader) savePosition() {
	c, err := r.j.GetCursor()
	if err != nil {
		logp.Err("Unable to get cursor from journal: %v", err)
	}

	err = ioutil.WriteFile(CURSOR_FILE, []byte(c), 600)
	if err != nil {
		logp.Err("Unable to write cursor to file: %v", err)
	}

}
