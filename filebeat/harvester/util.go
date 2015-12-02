package harvester

import (
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/libbeat/logp"
	"time"
)

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

// readLine reads a full line into buffer and returns it.
// In case of partial lines, readLine does return and error and en empty string
// This could potentialy be improved / replaced by https://github.com/elastic/beats/libbeat/tree/master/common/streambuf
func readLine(
	reader *encoding.LineReader,
	lastReadTime *time.Time,
) (string, int, error) {
	for {
		line, size, err := reader.Next()

		// Full line read to be returned
		if size != 0 && err == nil {
			logp.Debug("harvester", "full line read")
			return readlineString(line, size)
		} else {
			return "", 0, err
		}
	}
}

// readlineString removes line ending characters from given by array.
func readlineString(bytes []byte, size int) (string, int, error) {
	s := string(bytes)[:len(bytes)-lineEndingChars(bytes)]
	return s, size, nil
}
