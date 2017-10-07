// +build !integration

package smtp

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestSmtpParser_startResponse(t *testing.T) {
	var m *message
	var err error

	p := parser{
		message: new(message),
	}

	p.buf.Write([]byte("220 mx.google.com ESMTP l5si801447pli.743 - gsmtp\r\n"))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, stateCommand, p.state)
	assert.Equal(t, 220, m.statusCode)
	// Not a request
	assert.Exactly(t, common.NetString(nil), m.command)
}

func TestSmtpParser_startIncompleteResponse(t *testing.T) {
	var m *message
	var err error

	p := parser{
		message: new(message),
	}

	p.buf.Write([]byte("220 mx.google.com ESMTP l5si801447pli.743 - gsmtp"))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.Nil(t, m)

	p.buf.Write([]byte("\r\n"))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, stateCommand, p.state)
	assert.Equal(t, 220, m.statusCode)
	assert.Exactly(t, common.NetString(nil), m.command)
}

func TestSmtpParser_MultilineResponse(t *testing.T) {
	var m *message
	var err error

	p := parser{
		message: new(message),
	}

	data := "220-localhost ESMTP Exim 4.89 Sun, 15 Aug 2012 08:41:55 -0700\r\n" +
		"220 Go ahead\r\n"

	p.buf.Write([]byte(data))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, stateCommand, p.state)
	assert.Equal(t, 220, m.statusCode)
	assert.Len(t, m.statusPhrases, 2)
	assert.Contains(t, string(m.statusPhrases[0]), "Exim")
	assert.Exactly(t, common.NetString("Go ahead"), m.statusPhrases[1])
	assert.Exactly(t, common.NetString(nil), m.command)
}

func TestSmtpParser_MultilineIncompleteResponse(t *testing.T) {
	var m *message
	var err error

	p := parser{
		message: new(message),
	}

	data := "250-mx.google.com at your service, [8.8.8.8]\r\n" +
		"250-SIZE 157286400\r\n" +
		"250-8BITMIME\r\n" +
		"250-STARTTLS\r\n" +
		"250-ENHANCEDSTATUSCODES\r\n" +
		"250-PIPELINING\r\n" +
		"250-CHUNKING\r\n" +
		"250 SMTPUTF8"

	p.buf.Write([]byte(data))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.Nil(t, m)

	p.buf.Write([]byte("\r\n"))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, stateCommand, p.state)
	assert.Equal(t, 250, m.statusCode)
	assert.Len(t, m.statusPhrases, 8)
	assert.Exactly(t, common.NetString("CHUNKING"), m.statusPhrases[6])
	assert.Exactly(t, common.NetString(nil), m.command)
}

func TestSmtpParser_CommandRequest(t *testing.T) {
	var m *message
	var err error

	p := parser{
		message: new(message),
	}

	p.buf.Write([]byte("EHLO localhost\r\n"))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, stateCommand, p.state)
	assert.Equal(t, common.NetString("EHLO"), m.command)
	assert.Equal(t, common.NetString("localhost"), m.param)
	// Not a response
	assert.Exactly(t, 0, m.statusCode)
}

func TestSmtpParser_IncompleteCommandRequest(t *testing.T) {
	var m *message
	var err error

	p := parser{
		message: new(message),
	}

	p.buf.Write([]byte("EHLO localhost"))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.Nil(t, m)

	p.buf.Write([]byte("\r\n"))
	m, err = p.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, stateCommand, p.state)
	assert.Equal(t, common.NetString("EHLO"), m.command)
	assert.Equal(t, common.NetString("localhost"), m.param)
	assert.Exactly(t, 0, m.statusCode)
}

func TestSmtpParser_DataTransaction(t *testing.T) {
	var m *message
	var err error

	// Client
	pc := parser{
		message: new(message),
	}

	pc.buf.Write([]byte("DATA\r\n"))
	m, err = pc.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Exactly(t, 0, m.statusCode)

	// Server
	ps := parser{
		message: new(message),
	}

	ps.buf.Write([]byte("354 Enter message, ending with \".\" on a line by itself\r\n"))
	m, err = ps.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Exactly(t, common.NetString(nil), m.command)
	assert.Exactly(t, 354, m.statusCode)

	data := "From: bar@example.org\r\n" +
		"To: foo@example.com\r\n" +
		"Cc: baz@example.com\r\n" +
		"Subject: Test\r\n" +
		"Date: Thu, 20 Dec 2012 12:00:00 +0000\r\n" +
		"\r\n" +
		"Hello world\r\n" +
		".\r\n"

	pc.message = new(message)
	pc.message.headers = make(map[string]common.NetString)
	pc.buf.Write([]byte(data))
	m, err = pc.parse()
	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Exactly(t, 0, m.statusCode)
	assert.Len(t, m.headers, 5)
	assert.Exactly(t, common.NetString("Test"), m.headers["Subject"])
	assert.Exactly(t, common.NetString("Hello world\r\n"), m.body)
}
