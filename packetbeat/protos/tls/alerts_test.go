// +build !integration

package tls

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
)

func getParser() *parser {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"tls", "tlsdetailed"})
	}
	return &parser{}
}

func mkBuf(t *testing.T, s string, length int) *bufferView {
	bytes, err := hex.DecodeString(s)
	assert.Nil(t, err)
	return newBufferView(streambuf.New(bytes), 0, length)
}

func TestParse(t *testing.T) {
	parser := getParser()
	err := parser.parseAlert(mkBuf(t, "0102", 2))
	assert.Nil(t, err)
	assert.Len(t, parser.alerts, 1)
	assert.Equal(t, alertSeverity(1), parser.alerts[0].severity)
	assert.Equal(t, alertCode(2), parser.alerts[0].code)
}

func TestShortBuffer(t *testing.T) {
	parser := getParser()
	err := parser.parseAlert(mkBuf(t, "", 2))
	assert.NotNil(t, err)
	assert.Empty(t, parser.alerts)

	err = parser.parseAlert(mkBuf(t, "01", 2))
	assert.NotNil(t, err)
	assert.Empty(t, parser.alerts)
}

func TestEncrypted(t *testing.T) {
	parser := getParser()
	err := parser.parseAlert(mkBuf(t, "010200000000", 6))
	assert.Nil(t, err)
	assert.Empty(t, parser.alerts)
}
