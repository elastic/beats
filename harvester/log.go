package harvester

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/elastic/filebeat/config"
	"github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/logp"
)

// Log harvester reads files line by line and sends events to logstash
// Multiline log support is required

func (h *Harvester) Harvest() {
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

	reader := bufio.NewReaderSize(h.file, config.CmdlineOptions.HarvesterBufferSize) // 16kb buffer by default
	buffer := new(bytes.Buffer)

	var readTimeout = 10 * time.Second
	lastReadTime := time.Now()
	for {
		text, bytesread, err := h.readline(reader, buffer, readTimeout)

		if err != nil {
			err = h.handleReadlineError(lastReadTime, err)

			if err != nil {
				return
			} else {
				continue
			}
		}

		lastReadTime = time.Now()

		line++
		event := &input.FileEvent{
			Source:   &h.Path,
			Offset:   h.Offset,
			Line:     line,
			Text:     text,
			Fields:   &h.FileConfig.Fields,
			Fileinfo: &info,
		}
		h.Offset += int64(bytesread)

		h.SpoolerChan <- event // ship the new event downstream
	}
}

// initOffset finds the current offset of the file and sets it in the harvester as position
func (h *Harvester) initOffset() {
	// get current offset in file
	offset, _ := h.file.Seek(0, os.SEEK_CUR)

	if h.Offset > 0 {
		logp.Debug("harvester", "harvest: %q position:%d (offset snapshot:%d)", h.Path, h.Offset, offset)
	} else if config.CmdlineOptions.TailOnRotate {
		logp.Debug("harvester", "harvest: (tailing) %q (offset snapshot:%d)", h.Path, offset)
	} else {
		logp.Debug("harvester", "harvest: %q (offset snapshot:%d)", h.Path, offset)
	}

	h.Offset = offset
}

// Sets the offset of the file to the right place. Takes configuration options into account
func (h *Harvester) setFileOffset() {
	if h.Offset > 0 {
		h.file.Seek(h.Offset, os.SEEK_SET)
	} else if config.CmdlineOptions.TailOnRotate {
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
			// TODO: This is currently end endless retry, should be set to a max?
			// retry on failure.
			logp.Err("Failed opening %s: %s", h.Path, err)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	file := &input.File{
		File: h.file,
	}

	// Check we are not following a rabbit hole (symlinks, etc.)
	if !file.IsRegularFile() {
		// TODO: This should be replaced by a normal error
		panic(fmt.Errorf("Harvester file error"))
	}

	h.setFileOffset()

	return h.file
}

func (h *Harvester) readline(reader *bufio.Reader, buffer *bytes.Buffer, eof_timeout time.Duration) (*string, int, error) {
	// TODO: Read line should be improved in a way so it can also read multi lines or even full files when required. See "type" in config file
	var isPartial bool = true
	var newline_length int = 1
	start_time := time.Now()

	for {
		segment, err := reader.ReadBytes('\n')

		if segment != nil && len(segment) > 0 {
			if segment[len(segment)-1] == '\n' {
				// Found a complete line
				isPartial = false

				// Check if also a CR present
				if len(segment) > 1 && segment[len(segment)-2] == '\r' {
					newline_length++
				}
			}

			// TODO(sissel): if buffer exceeds a certain length, maybe report an error condition? chop it?
			buffer.Write(segment)
		}

		if err != nil {
			if err == io.EOF && isPartial {
				time.Sleep(1 * time.Second) // TODO(sissel): Implement backoff

				// Give up waiting for data after a certain amount of time.
				// If we time out, return the error (eof)
				if time.Since(start_time) > eof_timeout {
					return nil, 0, err
				}
				continue
			} else {
				logp.Err("Harvester.readLine: %s", err.Error())
				return nil, 0, err // TODO(sissel): don't do this?
			}
		}

		// If we got a full line, return the whole line without the EOL chars (CRLF or LF)
		if !isPartial {
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

// Handles eror during reading file. If EOF and nothing special, exit without errors
func (h *Harvester) handleReadlineError(lastTimeRead time.Time, err error) error {
	if err == io.EOF {
		// timed out waiting for data, got eof.
		// Check to see if the file was truncated
		info, _ := h.file.Stat()

		if h.FileConfig.IgnoreOlder != "" {
			logp.Debug("harvester", "Ignore Unmodified After: %s", h.FileConfig.IgnoreOlder)
		}

		if info.Size() < h.Offset {
			logp.Debug("harvester", "File truncated, seeking to beginning: %s", h.Path)
			h.file.Seek(0, os.SEEK_SET)
			h.Offset = 0
		} else if age := time.Since(lastTimeRead); age > h.FileConfig.IgnoreOlderDuration {
			// if lastTimeRead was more than ignore older and ignore older is set, this file is probably dead. Stop watching it.
			logp.Debug("harvester", "Stopping harvest of ", h.Path, "last change was: ", age)
			return err
		}
	} else {
		logp.Err("Unexpected state reading from %s; error: %s", h.Path, err)
		return err
	}
	return nil
}
