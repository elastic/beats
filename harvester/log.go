package harvester

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

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

	reader := bufio.NewReaderSize(h.file, h.BufferSize)
	buffer := new(bytes.Buffer)

	var readTimeout = 10 * time.Second
	lastReadTime := time.Now()
	for {
		text, bytesread, err := h.readLine(reader, buffer, readTimeout)

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
			Fields:   &h.ProspectorConfig.Fields,
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
	} else if h.TailOnRotate {
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
	} else if h.TailOnRotate {
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

// TODO: It seems like this function does not depend at all on harvester
// To could potentialy be improved / replaced by https://github.com/elastic/libbeat/tree/master/common/streambuf
func (h *Harvester) readLine(reader *bufio.Reader, buffer *bytes.Buffer, eofTimeout time.Duration) (*string, int, error) {
	// TODO: Read line should be improved in a way so it can also read multi lines or even full files when required. See "type" in config file

	isPartial := true
	startTime := time.Now()

	for {
		segment, err := reader.ReadBytes('\n')

		if segment != nil && len(segment) > 0 {
			if isLine(segment) {
				isPartial = false
			}

			// TODO(sissel): if buffer exceeds a certain length, maybe report an error condition? chop it?
			buffer.Write(segment)
		}

		if err != nil {
			if err == io.EOF && isPartial {
				time.Sleep(1 * time.Second) // TODO(sissel): Implement backoff

				// Give up waiting for data after a certain amount of time.
				// If we time out, return the error (eof)
				if time.Since(startTime) >= eofTimeout {
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
			*str = buffer.String()[:bufferSize-lineEndingChars(segment)]
			// Reset the buffer for the next line
			buffer.Reset()
			return str, bufferSize, nil
		}
	}
}

// isLine checks if the given byte array is a line, means has a line ending \n
func isLine(line []byte) bool {
	if line == nil || len(line) == 0 {
		return false
	}

	if line[len(line)-1] != '\n' {
		return false
	}
	return true
}

// lineEndingChars returns the number of line ending chars the given by array has
// In case of Unix/Linux files, it is -1, in case of Windows mostly -2
func lineEndingChars(line []byte) int {
	if !isLine(line) {
		return 0
	}

	if line[len(line)-1] == '\n' {
		if len(line) > 1 && line[len(line)-2] == '\r' {
			return 2
		}

		return 1
	}
	return 0
}

// Handles error during reading file. If EOF and nothing special, exit without errors
func (h *Harvester) handleReadlineError(lastTimeRead time.Time, err error) error {
	if err == io.EOF {
		// timed out waiting for data, got eof.
		// Check to see if the file was truncated
		info, statErr := h.file.Stat()

		// This could happen if the file was removed / rotate after reading and before calling the stat function
		if statErr != nil {
			logp.Err("Unexpected error reading from %s; error: %s", h.Path, statErr)
			return statErr
		}

		if h.ProspectorConfig.IgnoreOlder != "" {
			logp.Debug("harvester", "Ignore Unmodified After: %s", h.ProspectorConfig.IgnoreOlder)
		}

		if info.Size() < h.Offset {
			logp.Debug("harvester", "File truncated, seeking to beginning: %s", h.Path)
			h.file.Seek(0, os.SEEK_SET)
			h.Offset = 0
		} else if age := time.Since(lastTimeRead); age > h.ProspectorConfig.IgnoreOlderDuration {
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
