package crawler

import (
	"bufio"
	"bytes"
	"fmt"
	cfg "github.com/elastic/filebeat/config"
	. "github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
	"io"
	"os"
	"time"
)

type Harvester struct {
	Path       string /* the file path to harvest */
	FileConfig cfg.FileConfig
	Offset     int64
	FinishChan chan int64

	file *os.File /* the file being watched */
}

func (h *Harvester) Harvest(output chan *FileEvent) {
	h.open()
	info, e := h.file.Stat()
	if e != nil {
		panic(fmt.Sprintf("Harvest: unexpected error: %s", e.Error()))
	}

	defer h.file.Close()

	// On completion, push offset so we can continue where we left off if we relaunch on the same file
	defer func() {
		h.FinishChan <- h.Offset
	}()

	var line uint64 = 0 // Ask registrar about the line number

	h.initOffset()

	reader := bufio.NewReaderSize(h.file, cfg.CmdlineOptions.HarvesterBufferSize) // 16kb buffer by default
	buffer := new(bytes.Buffer)

	var read_timeout = 10 * time.Second
	last_read_time := time.Now()
	for {
		text, bytesread, err := h.readline(reader, buffer, read_timeout)

		if err != nil {
			if err == io.EOF {
				// timed out waiting for data, got eof.
				// Check to see if the file was truncated
				info, _ := h.file.Stat()
				if info.Size() < h.Offset {
					logp.Info("harvester", "File truncated, seeking to beginning: %s", h.Path)
					h.file.Seek(0, os.SEEK_SET)
					h.Offset = 0
				} else if age := time.Since(last_read_time); age > h.FileConfig.DeadtimeSpan {
					// if last_read_time was more than dead time, this file is probably
					// dead. Stop watching it.
					logp.Info("harvester", "Stopping harvest of ", h.Path, "last change was: ", age)
					return
				}
				continue
			} else {
				logp.Info("harvester", "Unexpected state reading from %s; error: %s", h.Path, err)
				return
			}
		}
		last_read_time = time.Now()

		line++
		event := &FileEvent{
			Source:   &h.Path,
			Offset:   h.Offset,
			Line:     line,
			Text:     text,
			Fields:   &h.FileConfig.Fields,
			Fileinfo: &info,
		}
		h.Offset += int64(bytesread)

		output <- event // ship the new event downstream
	} /* forever */
}

// initOffset finds the current offset of the file and sets it in the harvester as position
func (h *Harvester) initOffset() {
	// get current offset in file
	offset, _ := h.file.Seek(0, os.SEEK_CUR)

	if h.Offset > 0 {
		logp.Info("harvester", "harvest: %q position:%d (offset snapshot:%d)", h.Path, h.Offset, offset)
	} else if cfg.CmdlineOptions.TailOnRotate {
		logp.Info("harvester", "harvest: (tailing) %q (offset snapshot:%d)", h.Path, offset)
	} else {
		logp.Info("harvester", "harvest: %q (offset snapshot:%d)", h.Path, offset)
	}

	h.Offset = offset
}

// Sets the offset of the file to the right place. Takes configuration options into account
func (h *Harvester) setFileOffset() {
	if h.Offset > 0 {
		h.file.Seek(h.Offset, os.SEEK_SET)
	} else if cfg.CmdlineOptions.TailOnRotate {
		h.file.Seek(0, os.SEEK_END)
	} else {
		h.file.Seek(0, os.SEEK_SET)
	}

}

func (h *Harvester) open() *os.File {
	// Special handling that "-" means to read from standard input
	if h.Path == "-" {
		h.file = os.Stdin
		return h.file
	}

	for {
		var err error
		h.file, err = os.Open(h.Path)

		if err != nil {
			// retry on failure.
			logp.Info("harvester", "Failed opening %s: %s", h.Path, err)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	file := &File{
		File: h.file,
	}

	// Check we are not following a rabbit hole (symlinks, etc.)
	if !file.IsRegularFile() {
		panic(fmt.Errorf("Harvester file error"))
	}

	h.setFileOffset()

	return h.file
}

func (h *Harvester) readline(reader *bufio.Reader, buffer *bytes.Buffer, eof_timeout time.Duration) (*string, int, error) {
	var is_partial bool = true
	var newline_length int = 1
	start_time := time.Now()

	for {
		segment, err := reader.ReadBytes('\n')

		if segment != nil && len(segment) > 0 {
			if segment[len(segment)-1] == '\n' {
				// Found a complete line
				is_partial = false

				// Check if also a CR present
				if len(segment) > 1 && segment[len(segment)-2] == '\r' {
					newline_length++
				}
			}

			// TODO(sissel): if buffer exceeds a certain length, maybe report an error condition? chop it?
			buffer.Write(segment)
		}

		if err != nil {
			if err == io.EOF && is_partial {
				time.Sleep(1 * time.Second) // TODO(sissel): Implement backoff

				// Give up waiting for data after a certain amount of time.
				// If we time out, return the error (eof)
				if time.Since(start_time) > eof_timeout {
					return nil, 0, err
				}
				continue
			} else {
				logp.Info("harvester", "error: Harvester.readLine: %s", err.Error())
				return nil, 0, err // TODO(sissel): don't do this?
			}
		}

		// If we got a full line, return the whole line without the EOL chars (CRLF or LF)
		if !is_partial {
			// Get the str length with the EOL chars (LF or CRLF)
			bufferSize := buffer.Len()
			str := new(string)
			*str = buffer.String()[:bufferSize-newline_length]
			// Reset the buffer for the next line
			buffer.Reset()
			return str, bufferSize, nil
		}
	}
}
