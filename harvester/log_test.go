package harvester

import (
	"bufio"
	"bytes"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"

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

	h := Harvester{}
	assert.NotNil(t, h)

	// Read only 10 bytes which is not the end of the file
	reader := bufio.NewReaderSize(readFile, 100)
	buffer := new(bytes.Buffer)

	// Read third line
	text, bytesread, err := h.readLine(reader, buffer, 0)

	assert.Equal(t, *text, firstLineString[0:len(firstLineString)-1])
	assert.Equal(t, bytesread, len(firstLineString))
	assert.Nil(t, err)

	// read second line
	text, bytesread, err = h.readLine(reader, buffer, 0)

	assert.Equal(t, *text, secondLineString[0:len(secondLineString)-1])
	assert.Equal(t, bytesread, len(secondLineString))
	assert.Nil(t, err)

	// Read third line, which doesn't exist
	text, bytesread, err = h.readLine(reader, buffer, 0)
	assert.Nil(t, text)
	assert.Equal(t, bytesread, 0)
	assert.Equal(t, err, io.EOF)
}

func TestIsLine(t *testing.T) {
	notLine := []byte("This is not a line")
	assert.False(t, isLine(notLine))

	notLine = []byte("This is not a line\n\r")
	assert.False(t, isLine(notLine))

	notLine = []byte("This is \n not a line")
	assert.False(t, isLine(notLine))

	line := []byte("This is a line \n")
	assert.True(t, isLine(line))

	line = []byte("This is a line\r\n")
	assert.True(t, isLine(line))
}

func TestLineEndingChars(t *testing.T) {

	line := []byte("Not ending line")
	assert.Equal(t, 0, lineEndingChars(line))

	line = []byte("N ending \n")
	assert.Equal(t, 1, lineEndingChars(line))

	line = []byte("RN ending \r\n")
	assert.Equal(t, 2, lineEndingChars(line))

	// This is an invalid option
	line = []byte("NR ending \n\r")
	assert.Equal(t, 0, lineEndingChars(line))
}
