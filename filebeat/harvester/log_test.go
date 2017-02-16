// +build !integration

package harvester

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestReadLine(t *testing.T) {

	absPath, err := filepath.Abs("../tests/files/logs/")
	// All files starting with tmp are ignored
	logFile := absPath + "/tmp" + strconv.Itoa(rand.Int()) + ".log"

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	if err != nil {
		t.Fatalf("Error creating the absolute path: %s", absPath)
	}

	file, err := os.Create(logFile)
	defer file.Close()
	defer os.Remove(logFile)

	assert.Nil(t, err)
	assert.NotNil(t, file)

	firstLineString := "9Characte\n"
	secondLineString := "This is line 2\n"

	length, err := file.WriteString(firstLineString)
	assert.Nil(t, err)
	assert.NotNil(t, length)

	length, err = file.WriteString(secondLineString)
	assert.Nil(t, err)
	assert.NotNil(t, length)

	file.Sync()

	// Open file for reading
	readFile, err := os.Open(logFile)
	defer readFile.Close()
	assert.Nil(t, err)

	f := source.File{readFile}

	h := Harvester{
		config: harvesterConfig{
			CloseInactive: 500 * time.Millisecond,
			Backoff:       100 * time.Millisecond,
			MaxBackoff:    1 * time.Second,
			BackoffFactor: 2,
			BufferSize:    100,
			MaxBytes:      1000,
		},
		file: f,
	}

	var ok bool
	h.encodingFactory, ok = encoding.FindEncoding(h.config.Encoding)
	assert.True(t, ok)

	h.encoding, err = h.encodingFactory(readFile)
	assert.NoError(t, err)

	r, err := h.newLogFileReader()
	assert.NoError(t, err)

	// Read third line
	_, text, bytesread, _, err := readLine(r)
	fmt.Printf("received line: '%s'\n", text)
	assert.Nil(t, err)
	assert.Equal(t, text, firstLineString[0:len(firstLineString)-1])
	assert.Equal(t, bytesread, len(firstLineString))

	// read second line
	_, text, bytesread, _, err = readLine(r)
	fmt.Printf("received line: '%s'\n", text)
	assert.Equal(t, text, secondLineString[0:len(secondLineString)-1])
	assert.Equal(t, bytesread, len(secondLineString))
	assert.Nil(t, err)

	// Read third line, which doesn't exist
	_, text, bytesread, _, err = readLine(r)
	fmt.Printf("received line: '%s'\n", text)
	assert.Equal(t, "", text)
	assert.Equal(t, bytesread, 0)
	assert.Equal(t, err, ErrInactive)
}

func TestExcludeLine(t *testing.T) {
	regexp, err := InitMatchers("^DBG")
	assert.Nil(t, err)
	assert.True(t, MatchAny(regexp, "DBG: a debug message"))
	assert.False(t, MatchAny(regexp, "ERR: an error message"))
}

func TestIncludeLine(t *testing.T) {
	regexp, err := InitMatchers("^ERR", "^WARN")

	assert.Nil(t, err)
	assert.False(t, MatchAny(regexp, "DBG: a debug message"))
	assert.True(t, MatchAny(regexp, "ERR: an error message"))
	assert.True(t, MatchAny(regexp, "WARNING: a simple warning message"))
}

func TestInitRegexp(t *testing.T) {
	_, err := InitMatchers("(((((")
	assert.NotNil(t, err)
}

// readLine reads a full line into buffer and returns it.
// In case of partial lines, readLine does return an error and an empty string
// This could potentially be improved / replaced by https://github.com/elastic/beats/libbeat/tree/master/common/streambuf
func readLine(reader reader.Reader) (time.Time, string, int, common.MapStr, error) {
	message, err := reader.Next()

	// Full line read to be returned
	if message.Bytes != 0 && err == nil {
		return message.Ts, string(message.Content), message.Bytes, message.Fields, err
	}

	return time.Time{}, "", 0, nil, err
}
