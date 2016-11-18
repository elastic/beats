// +build !integration

package http

import (
	"bytes"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/publish"
	"github.com/stretchr/testify/assert"
)

type testParser struct {
	payloads []string
	http     *httpPlugin
	stream   *stream
}

var testParserConfig = parserConfig{}

func newTestParser(http *httpPlugin, payloads ...string) *testParser {
	if http == nil {
		http = httpModForTests()
	}
	tp := &testParser{
		http:     http,
		payloads: payloads,
		stream:   &stream{data: []byte{}, message: new(message)},
	}
	return tp
}

func (tp *testParser) parse() (*message, bool, bool) {
	st := tp.stream
	if len(tp.payloads) > 0 {
		st.data = append(st.data, tp.payloads[0]...)
		tp.payloads = tp.payloads[1:]
	}

	parser := newParser(&tp.http.parserConfig)
	ok, complete := parser.parse(st, 0)
	return st.message, ok, complete
}

func httpModForTests() *httpPlugin {
	results := &publish.ChanTransactions{Channel: make(chan common.MapStr, 10)}
	http, err := New(false, results, common.NewConfig())
	if err != nil {
		panic(err)
	}
	return http.(*httpPlugin)
}

func testParse(http *httpPlugin, data string) (*message, bool, bool) {
	tp := newTestParser(http, data)
	return tp.parse()
}

func testParseStream(http *httpPlugin, st *stream, extraLen int) (bool, bool) {
	parser := newParser(&http.parserConfig)
	return parser.parse(st, extraLen)
}

func TestHttpParser_simpleResponse(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"
	message, ok, complete := testParse(nil, data)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, message.isRequest)
	assert.Equal(t, 200, int(message.statusCode))
	assert.Equal(t, "OK", string(message.statusPhrase))
	assert.True(t, isVersion(message.version, 1, 1))
	assert.Equal(t, 262, int(message.size))
	assert.Equal(t, 0, message.contentLength)
}

func TestHttpParser_simpleResponseCaseInsensitive(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"EXPIRES: -1\r\n" +
		"cACHE-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"SERVER: gws\r\n" +
		"content-LeNgTh: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"
	message, ok, complete := testParse(nil, data)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, message.isRequest)
	assert.Equal(t, 200, int(message.statusCode))
	assert.Equal(t, "OK", string(message.statusPhrase))
	assert.True(t, isVersion(message.version, 1, 1))
	assert.Equal(t, 262, int(message.size))
	assert.Equal(t, 0, message.contentLength)
}

func TestHttpParser_simpleRequest(t *testing.T) {
	http := httpModForTests()
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true

	data := "GET / HTTP/1.1\r\n" +
		"Host: www.google.ro\r\n" +
		"Connection: keep-alive\r\n" +
		"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_4) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/21.0.1180.75 Safari/537.1\r\n" +
		"Accept: */*\r\n" +
		"X-Chrome-Variations: CLa1yQEIj7bJAQiftskBCKS2yQEIp7bJAQiptskBCLSDygE=\r\n" +
		"Referer: http://www.google.ro/\r\n" +
		"Accept-Encoding: gzip,deflate,sdch\r\n" +
		"Accept-Language: en-US,en;q=0.8\r\n" +
		"Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.3\r\n" +
		"Cookie: PREF=ID=6b67d166417efec4:U=69097d4080ae0e15:FF=0:TM=1340891937:LM=1340891938:S=8t97UBiUwKbESvVX; NID=61=sf10OV-t02wu5PXrc09AhGagFrhSAB2C_98ZaI53-uH4jGiVG_yz9WmE3vjEBcmJyWUogB1ZF5puyDIIiB-UIdLd4OEgPR3x1LHNyuGmEDaNbQ_XaxWQqqQ59mX1qgLQ\r\n" +
		"\r\n" +
		"garbage"

	message, ok, complete := testParse(http, data)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.True(t, message.isRequest)
	assert.True(t, isVersion(message.version, 1, 1))
	assert.Equal(t, 669, int(message.size))
	assert.Equal(t, "GET", string(message.method))
	assert.Equal(t, "/", string(message.requestURI))
	assert.Equal(t, "www.google.ro", string(message.headers["host"]))
}

func TestHttpParser_Request_ContentLength_0(t *testing.T) {
	http := httpModForTests()
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true

	data := "POST / HTTP/1.1\r\n" +
		"user-agent: curl/7.35.0\r\n" + "host: localhost:9000\r\n" +
		"accept: */*\r\n" +
		"authorization: Company 1\r\n" +
		"content-length: 0\r\n" +
		"connection: close\r\n" +
		"\r\n"

	_, ok, complete := testParse(http, data)
	assert.True(t, ok)
	assert.True(t, complete)
}

func TestHttpParser_eatBody(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := httpModForTests()
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true

	data := []byte("POST / HTTP/1.1\r\n" +
		"user-agent: curl/7.35.0\r\n" +
		"host: localhost:9000\r\n" +
		"accept: */*\r\n" +
		"authorization: Company 1\r\n" +
		"content-length: 20\r\n" +
		"connection: close\r\n" +
		"\r\n" +
		"0123456789")

	st := &stream{data: data, message: new(message)}
	ok, complete := testParseStream(http, st, 0)
	assert.True(t, ok)
	assert.False(t, complete)
	assert.Equal(t, st.bodyReceived, 10)

	ok, complete = testParseStream(http, st, 5)
	assert.True(t, ok)
	assert.False(t, complete)
	assert.Equal(t, st.bodyReceived, 15)

	ok, complete = testParseStream(http, st, 5)
	assert.True(t, ok)
	assert.True(t, complete)
	assert.Equal(t, st.bodyReceived, 20)
	assert.Equal(t, st.message.end, len(data))
}

func TestHttpParser_eatBody_connclose(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := httpModForTests()
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true

	data := []byte("HTTP/1.1 200 ok\r\n" +
		"user-agent: curl/7.35.0\r\n" +
		"host: localhost:9000\r\n" +
		"accept: */*\r\n" +
		"authorization: Company 1\r\n" +
		"connection: close\r\n" +
		"\r\n" +
		"0123456789")

	st := &stream{data: data, message: new(message)}
	ok, complete := testParseStream(http, st, 0)
	assert.True(t, ok)
	assert.False(t, complete)
	assert.Equal(t, st.bodyReceived, 10)

	ok, complete = testParseStream(http, st, 5)
	assert.True(t, ok)
	assert.False(t, complete)
	assert.Equal(t, st.bodyReceived, 15)

	ok, complete = testParseStream(http, st, 5)
	assert.True(t, ok)
	assert.False(t, complete)
	assert.Equal(t, st.bodyReceived, 20)
}

func TestHttpParser_splitResponse(t *testing.T) {
	data1 := "HTTP/1.1 200 ok\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n"
	data2 := "Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"
	tp := newTestParser(nil, data1, data2)

	_, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.False(t, complete)

	_, ok, complete = tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
}

func TestHttpParser_splitResponse_midHeaderName(t *testing.T) {
	http := httpModForTests()
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true

	data1 := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-En"
	data2 := "coding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"
	tp := newTestParser(http, data1, data2)

	_, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.False(t, complete)

	message, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, message.isRequest)
	assert.Equal(t, 200, int(message.statusCode))
	assert.Equal(t, "OK", string(message.statusPhrase))
	assert.True(t, isVersion(message.version, 1, 1))
	assert.Equal(t, 262, int(message.size))
	assert.Equal(t, 0, message.contentLength)
}

func TestHttpParser_splitResponse_midHeaderValue(t *testing.T) {
	data1 := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: g"
	data2 := "zip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"
	tp := newTestParser(nil, data1, data2)

	_, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.False(t, complete)

	_, ok, complete = tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
}

func TestHttpParser_splitResponse_midNewLine(t *testing.T) {
	data1 := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r"
	data2 := "\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"
	tp := newTestParser(nil, data1, data2)

	_, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.False(t, complete)

	_, ok, complete = tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
}

func TestHttpParser_ResponseWithBody(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 30\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n" +
		"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" +
		"garbage"
	tp := newTestParser(nil, data)

	message, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
	assert.Equal(t, 30, message.contentLength)
	assert.Equal(t, "garbage", string(tp.stream.data[tp.stream.parseOffset:]))
}

func TestHttpParser_Response_HTTP_10_without_content_length(t *testing.T) {
	data := "HTTP/1.0 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		"test"

	message, ok, complete := testParse(nil, data)
	assert.True(t, ok)
	assert.False(t, complete)
	assert.Equal(t, 4, message.contentLength)
}

func TestHttpParser_splitResponse_midBody(t *testing.T) {
	data1 := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 3"
	data2 := "0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n" +
		"xxxxxxxxxx"
	data3 := "xxxxxxxxxxxxxxxxxxxx"
	tp := newTestParser(nil, data1, data2, data3)

	_, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.False(t, complete)

	_, ok, complete = tp.parse()
	assert.True(t, ok)
	assert.False(t, complete)

	message, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
	assert.Equal(t, 30, message.contentLength)
	assert.Equal(t, []byte(""), tp.stream.data[tp.stream.parseOffset:])
}

func TestHttpParser_RequestResponse(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	data := "GET / HTTP/1.1\r\n" +
		"Host: www.google.ro\r\n" +
		"Connection: keep-alive\r\n" +
		"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_4) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/21.0.1180.75 Safari/537.1\r\n" +
		"Accept: */*\r\n" +
		"X-Chrome-Variations: CLa1yQEIj7bJAQiftskBCKS2yQEIp7bJAQiptskBCLSDygE=\r\n" +
		"Referer: http://www.google.ro/\r\n" +
		"Accept-Encoding: gzip,deflate,sdch\r\n" +
		"Accept-Language: en-US,en;q=0.8\r\n" +
		"Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.3\r\n" +
		"Cookie: PREF=ID=6b67d166417efec4:U=69097d4080ae0e15:FF=0:TM=1340891937:LM=1340891938:S=8t97UBiUwKbESvVX; NID=61=sf10OV-t02wu5PXrc09AhGagFrhSAB2C_98ZaI53-uH4jGiVG_yz9WmE3vjEBcmJyWUogB1ZF5puyDIIiB-UIdLd4OEgPR3x1LHNyuGmEDaNbQ_XaxWQqqQ59mX1qgLQ\r\n" +
		"\r\n" +
		"HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"

	tp := newTestParser(nil, data)
	_, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)

	tp.stream.PrepareForNewMessage()
	tp.stream.message = &message{ts: time.Now()}
	_, ok, complete = tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
}

func TestHttpParser_RequestResponseBody(t *testing.T) {
	data1 := "GET / HTTP/1.1\r\n" +
		"Host: www.google.ro\r\n" +
		"Connection: keep-alive\r\n" +
		"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_4) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/21.0.1180.75 Safari/537.1\r\n" +
		"Accept: */*\r\n" +
		"X-Chrome-Variations: CLa1yQEIj7bJAQiftskBCKS2yQEIp7bJAQiptskBCLSDygE=\r\n" +
		"Referer: http://www.google.ro/\r\n" +
		"Accept-Encoding: gzip,deflate,sdch\r\n" +
		"Accept-Language: en-US,en;q=0.8\r\n" +
		"Content-Length: 2\r\n" +
		"Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.3\r\n" +
		"Cookie: PREF=ID=6b67d166417efec4:U=69097d4080ae0e15:FF=0:TM=1340891937:LM=1340891938:S=8t97UBiUwKbESvVX; NID=61=sf10OV-t02wu5PXrc09AhGagFrhSAB2C_98ZaI53-uH4jGiVG_yz9WmE3vjEBcmJyWUogB1ZF5puyDIIiB-UIdLd4OEgPR3x1LHNyuGmEDaNbQ_XaxWQqqQ59mX1qgLQ\r\n" +
		"\r\n" +
		"xx"
	data2 := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"
	data := data1 + data2
	tp := newTestParser(nil, data)

	msg, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
	assert.Equal(t, 2, msg.contentLength)
	assert.Equal(t, []byte(data1), tp.stream.data[tp.stream.message.start:tp.stream.message.end])

	tp.stream.PrepareForNewMessage()
	tp.stream.message = &message{ts: time.Now()}
	msg, ok, complete = tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
}

func TestHttpParser_301_response(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http"})
	}

	data := "HTTP/1.1 301 Moved Permanently\r\n" +
		"Date: Sun, 29 Sep 2013 16:53:59 GMT\r\n" +
		"Server: Apache\r\n" +
		"Location: http://www.hotnews.ro/\r\n" +
		"Vary: Accept-Encoding\r\n" +
		"Content-Length: 290\r\n" +
		"Connection: close\r\n" +
		"Content-Type: text/html; charset=iso-8859-1\r\n" +
		"\r\n" +
		"<!DOCTYPE HTML PUBLIC \"-//IETF//DTD HTML 2.0//EN\">\r\n" +
		"<html><head>\r\n" +
		"<title>301 Moved Permanently</title>\r\n" +
		"</head><body>\r\n" +
		"<h1>Moved Permanently</h1>\r\n" +
		"<p>The document has moved <a href=\"http://www.hotnews.ro/\">here</a>.</p>\r\n" +
		"<hr>\r\n" +
		"<address>Apache Server at hotnews.ro Port 80</address>\r\n" +
		"</body></html>"

	msg, ok, complete := testParse(nil, data)
	assert.True(t, ok)
	assert.True(t, complete)
	assert.Equal(t, 290, msg.contentLength)
}

func TestEatBodyChunked(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	msgs := [][]byte{
		[]byte("03\r"),
		[]byte("\n123\r\n03\r\n123\r"),
		[]byte("\n0\r\n\r\n"),
	}
	st := &stream{
		data:         msgs[0],
		parseOffset:  0,
		bodyReceived: 0,
		parseState:   stateBodyChunkedStart,
	}
	msg := &message{
		chunkedLength: 5,
		contentLength: 0,
	}
	parser := newParser(&testParserConfig)

	cont, ok, complete := parser.parseBodyChunkedStart(st, msg)
	if cont != false || ok != true || complete != false {
		t.Errorf("Wrong return values")
	}
	assert.Equal(t, 0, st.parseOffset)

	st.data = append(st.data, msgs[1]...)
	cont, ok, complete = parser.parseBodyChunkedStart(st, msg)
	assert.True(t, cont)
	assert.Equal(t, 3, msg.chunkedLength)
	assert.Equal(t, 4, st.parseOffset)
	assert.Equal(t, stateBodyChunked, st.parseState)

	cont, ok, complete = parser.parseBodyChunked(st, msg)
	assert.True(t, cont)
	assert.Equal(t, stateBodyChunkedStart, st.parseState)
	assert.Equal(t, 9, st.parseOffset)

	cont, ok, complete = parser.parseBodyChunkedStart(st, msg)
	assert.True(t, cont)
	assert.Equal(t, 3, msg.chunkedLength)
	assert.Equal(t, 13, st.parseOffset)
	assert.Equal(t, stateBodyChunked, st.parseState)

	cont, ok, complete = parser.parseBodyChunked(st, msg)
	assert.False(t, cont)
	assert.True(t, ok)
	assert.False(t, complete)
	assert.Equal(t, 13, st.parseOffset)
	assert.Equal(t, 0, st.bodyReceived)
	assert.Equal(t, stateBodyChunked, st.parseState)

	st.data = append(st.data, msgs[2]...)
	cont, ok, complete = parser.parseBodyChunked(st, msg)
	assert.True(t, cont)
	assert.Equal(t, 18, st.parseOffset)
	assert.Equal(t, stateBodyChunkedStart, st.parseState)

	cont, ok, complete = parser.parseBodyChunkedStart(st, msg)
	assert.False(t, cont)
	assert.True(t, ok)
	assert.True(t, complete)
}

func TestEatBodyChunkedWaitCRLF(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	msgs := [][]byte{
		[]byte("03\r\n123\r\n0\r\n\r"),
		[]byte("\n"),
	}
	st := &stream{
		data:         msgs[0],
		parseOffset:  0,
		bodyReceived: 0,
		parseState:   stateBodyChunkedStart,
	}
	msg := &message{
		chunkedLength: 5,
		contentLength: 0,
	}
	parser := newParser(&testParserConfig)

	cont, ok, complete := parser.parseBodyChunkedStart(st, msg)
	if cont != true || ok != true || complete != false {
		t.Error("Wrong return values", cont, ok, complete)
	}
	if st.parseState != stateBodyChunked {
		t.Error("Unexpected state", st.parseState)
	}

	cont, ok, complete = parser.parseBodyChunked(st, msg)
	if cont != true || ok != true || complete != false {
		t.Error("Wrong return values", cont, ok, complete)
	}
	if st.parseState != stateBodyChunkedStart {
		t.Error("Unexpected state", st.parseState)
	}

	cont, ok, complete = parser.parseBodyChunkedStart(st, msg)
	if cont != false || ok != true || complete != false {
		t.Error("Wrong return values", cont, ok, complete)
	}
	if st.parseState != stateBodyChunkedWaitFinalCRLF {
		t.Error("Unexpected state", st.parseState)
	}

	logp.Debug("http", "parseOffset", st.parseOffset)

	ok, complete = parser.parseBodyChunkedWaitFinalCRLF(st, msg)
	if ok != true || complete != false {
		t.Error("Wrong return values", ok, complete)

	}
	st.data = append(st.data, msgs[1]...)

	ok, complete = parser.parseBodyChunkedWaitFinalCRLF(st, msg)
	if ok != true || complete != true {
		t.Error("Wrong return values", ok, complete)
	}
	if msg.end != 14 {
		t.Error("Wrong message end", msg.end)
	}
}

func TestHttpParser_censorPasswordURL(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := httpModForTests()
	http.hideKeywords = []string{"password", "pass"}
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true

	data1 := "GET http://localhost:8080/test?password=secret HTTP/1.1\r\n" +
		"Host: www.google.com\r\n" +
		"Connection: keep-alive\r\n" +
		"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_4) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/21.0.1180.75 Safari/537.1\r\n" +
		"Accept: */*\r\n" +
		"X-Chrome-Variations: CLa1yQEIj7bJAQiftskBCKS2yQEIp7bJAQiptskBCLSDygE=\r\n" +
		"Referer: http://www.google.com/\r\n" +
		"Accept-Encoding: gzip,deflate,sdch\r\n" +
		"Accept-Language: en-US,en;q=0.8\r\n" +
		"Content-Type: application/x-www-form-urlencoded\r\n" +
		"Content-Length: 23\r\n" +
		"Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.3\r\n" +
		"Cookie: PREF=ID=6b67d166417efec4:U=69097d4080ae0e15:FF=0:TM=1340891937:LM=1340891938:S=8t97UBiUwKbESvVX; NID=61=sf10OV-t02wu5PXrc09AhGagFrhSAB2C_98ZaI53-uH4jGiVG_yz9WmE3vjEBcmJyWUogB1ZF5puyDIIiB-UIdLd4OEgPR3x1LHNyuGmEDaNbQ_XaxWQqqQ59mX1qgLQ\r\n" +
		"\r\n" +
		"username=ME&pass=secret"
	tp := newTestParser(http, data1)

	msg, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)
	rawMsg := tp.stream.data[tp.stream.message.start:tp.stream.message.end]
	path, params, err := http.extractParameters(msg, rawMsg)
	assert.Nil(t, err)
	assert.Equal(t, "/test", path)
	assert.False(t, strings.Contains(params, "secret"))
}

func TestHttpParser_censorPasswordPOST(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := httpModForTests()
	http.hideKeywords = []string{"password"}
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true

	data1 :=
		"POST /users/login HTTP/1.1\r\n" +
			"HOST: www.example.com\r\n" +
			"Content-Type: application/x-www-form-urlencoded\r\n" +
			"Content-Length: 28\r\n" +
			"\r\n" +
			"username=ME&password=secret\r\n"
	tp := newTestParser(http, data1)

	msg, ok, complete := tp.parse()
	assert.True(t, ok)
	assert.True(t, complete)

	rawMsg := tp.stream.data[tp.stream.message.start:tp.stream.message.end]
	path, params, err := http.extractParameters(msg, rawMsg)
	assert.Nil(t, err)
	assert.Equal(t, "/users/login", path)
	assert.False(t, strings.Contains(params, "secret"))
}

func TestHttpParser_censorPasswordGET(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := httpModForTests()
	http.hideKeywords = []string{"password"}
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true
	http.sendRequest = false
	http.sendResponse = false

	data1 := []byte(
		"GET /users/login HTTP/1.1\r\n" +
			"HOST: www.example.com\r\n" +
			"Content-Type: application/x-www-form-urlencoded\r\n" +
			"Content-Length: 53\r\n" +
			"\r\n" +
			"password=my_secret_pass&Password=my_secret_password_2\r\n")

	st := &stream{data: data1, message: new(message)}

	ok, complete := testParseStream(http, st, 0)
	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}

	msg := st.data[st.message.start:st.message.end]
	path, params, err := http.extractParameters(st.message, msg)
	if err != nil {
		t.Errorf("Faile to parse parameters")
	}
	logp.Debug("httpdetailed", "parameters %s", params)

	if path != "/users/login" {
		t.Errorf("Wrong path: %s", path)
	}

	if strings.Contains(params, "secret") {
		t.Errorf("Failed to censor the password: %s", msg)
	}
}

func TestHttpParser_RedactAuthorization(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := httpModForTests()
	http.redactAuthorization = true
	http.parserConfig.sendHeaders = true
	http.parserConfig.sendAllHeaders = true

	data := []byte("POST /services/ObjectControl?ID=client0 HTTP/1.1\r\n" +
		"User-Agent: Mozilla/4.0 (compatible; MSIE 6.0; MS Web Services Client Protocol 2.0.50727.5472)\r\n" +
		"Content-Type: text/xml; charset=utf-8\r\n" +
		"SOAPAction: \"\"\r\n" +
		"Authorization: Basic ZHVtbXk6NmQlc1AwOC1XemZ3Cg\r\n" +
		"Proxy-Authorization: Basic cHJveHk6MWM3MGRjM2JhZDIwCg==\r\n" +
		"Host: production.example.com\r\n" +
		"Content-Length: 0\r\n" +
		"Expect: 100-continue\r\n" +
		"Accept-Encoding: gzip\r\n" +
		"X-Forwarded-For: 10.216.89.132\r\n" +
		"\r\n")

	st := &stream{data: data, message: new(message)}

	ok, _ := testParseStream(http, st, 0)

	st.message.raw = st.data[st.message.start:]
	http.hideHeaders(st.message)
	msg := st.message.raw

	assert.True(t, ok)
	assert.Equal(t, "*", string(st.message.headers["authorization"]))

	authPattern, _ := regexp.Compile(`(?m)^[Aa]uthorization:\*+`)
	authObscured := authPattern.Match(msg)
	assert.True(t, authObscured)

	assert.Equal(t, "*", string(st.message.headers["proxy-authorization"]))

	proxyPattern, _ := regexp.Compile(`(?m)^[Pp]roxy-[Aa]uthorization:\*+`)
	proxyObscured := proxyPattern.Match(msg)
	assert.True(t, proxyObscured)
}

func TestHttpParser_RedactAuthorization_raw(t *testing.T) {

	http := httpModForTests()
	http.redactAuthorization = true
	http.parserConfig.sendHeaders = false
	http.parserConfig.sendAllHeaders = false

	data := []byte("POST / HTTP/1.1\r\n" +
		"user-agent: curl/7.35.0\r\n" + "host: localhost:9000\r\n" +
		"accept: */*\r\n" +
		"authorization: Company 1\r\n" +
		"content-length: 0\r\n" +
		"connection: close\r\n" +
		"\r\n")

	st := &stream{data: data, message: new(message)}

	ok, complete := testParseStream(http, st, 0)

	st.message.raw = st.data[st.message.start:]
	http.hideHeaders(st.message)
	msg := st.message.raw

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	rawMessageObscured := bytes.Index(msg, []byte("uthorization:*"))
	if rawMessageObscured < 0 {
		t.Errorf("Obscured authorization string not found: " + string(msg[:]))
	}
}

func TestHttpParser_RedactAuthorization_Proxy_raw(t *testing.T) {

	http := httpModForTests()
	http.redactAuthorization = true
	http.parserConfig.sendHeaders = false
	http.parserConfig.sendAllHeaders = false

	data := []byte("POST / HTTP/1.1\r\n" +
		"user-agent: curl/7.35.0\r\n" + "host: localhost:9000\r\n" +
		"accept: */*\r\n" +
		"proxy-authorization: cHJveHk6MWM3MGRjM2JhZDIwCg==\r\n" +
		"content-length: 0\r\n" +
		"connection: close\r\n" +
		"\r\n")

	st := &stream{data: data, message: new(message)}

	ok, complete := testParseStream(http, st, 0)

	st.message.raw = st.data[st.message.start:]
	http.hideHeaders(st.message)
	msg := st.message.raw

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	rawMessageObscured := bytes.Index(msg, []byte("uthorization:*"))
	if rawMessageObscured < 0 {
		t.Errorf("Failed to redact proxy-authorization header: " + string(msg[:]))
	}
}

func Test_splitCookiesHeader(t *testing.T) {
	type io struct {
		Input  string
		Output map[string]string
	}

	tests := []io{
		{
			Input: "sessionToken=abc123; Expires=Wed, 09 Jun 2021 10:18:14 GMT",
			Output: map[string]string{
				"sessiontoken": "abc123",
				"expires":      "Wed, 09 Jun 2021 10:18:14 GMT",
			},
		},

		{
			Input: "sessionToken=abc123; invalid",
			Output: map[string]string{
				"sessiontoken": "abc123",
			},
		},

		{
			Input: "sessionToken=abc123; ",
			Output: map[string]string{
				"sessiontoken": "abc123",
			},
		},

		{
			Input: "sessionToken=abc123;;;; ",
			Output: map[string]string{
				"sessiontoken": "abc123",
			},
		},

		{
			Input: "sessionToken=abc123; multiple=a=d=2 ",
			Output: map[string]string{
				"sessiontoken": "abc123",
				"multiple":     "a=d=2",
			},
		},

		{
			Input: "sessionToken=\"abc123\"; multiple=\"a=d=2 \"",
			Output: map[string]string{
				"sessiontoken": "abc123",
				"multiple":     "a=d=2 ",
			},
		},

		{
			Input: "sessionToken\t=   abc123; multiple=a=d=2 ",
			Output: map[string]string{
				"sessiontoken": "abc123",
				"multiple":     "a=d=2",
			},
		},

		{
			Input:  ";",
			Output: map[string]string{},
		},

		{
			Input:  "",
			Output: map[string]string{},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, splitCookiesHeader(test.Input))
	}
}

// If a TCP gap (lost packets) happen while we're waiting for
// headers, drop the stream.
func Test_gap_in_headers(t *testing.T) {

	http := httpModForTests()

	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n")

	st := &stream{data: data1, message: new(message)}
	ok, complete := testParseStream(http, st, 0)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	ok, complete = http.messageGap(st, 5)
	assert.Equal(t, false, ok)
	assert.Equal(t, false, complete)
}

// If a TCP gap (lost packets) happen while we're waiting for
// parts of the body, it's ok.
func Test_gap_in_body(t *testing.T) {

	http := httpModForTests()

	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 40\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n" +
		"xxxxxxxxxxxxxxxxxxxx")

	st := &stream{data: data1, message: new(message)}
	ok, complete := testParseStream(http, st, 0)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	ok, complete = http.messageGap(st, 10)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	ok, complete = http.messageGap(st, 10)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, complete)
}

// If a TCP gap (lost packets) happen while we're waiting for
// parts of the body, it's ok.
func Test_gap_in_body_http1dot0(t *testing.T) {

	http := httpModForTests()

	data1 := []byte("HTTP/1.0 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n" +
		"xxxxxxxxxxxxxxxxxxxx")

	st := &stream{data: data1, message: new(message)}
	ok, complete := testParseStream(http, st, 0)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	ok, complete = http.messageGap(st, 10)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

}

func testCreateTCPTuple() *common.TCPTuple {
	t := &common.TCPTuple{
		IPLength: 4,
		SrcIP:    net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
		SrcPort: 6512, DstPort: 80,
	}
	t.ComputeHashebles()
	return t
}

// Helper function to read from the Publisher Queue
func expectTransaction(t *testing.T, http *httpPlugin) common.MapStr {
	client := http.results.(*publish.ChanTransactions)
	select {
	case trans := <-client.Channel:
		return trans
	default:
		t.Error("No transaction")
	}
	return nil
}

func Test_gap_in_body_http1dot0_fin(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http",
			"httpdetailed"})
	}
	http := httpModForTests()

	data1 := []byte("GET / HTTP/1.0\r\n\r\n")

	data2 := []byte("HTTP/1.0 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n" +
		"xxxxxxxxxxxxxxxxxxxx")

	tcptuple := testCreateTCPTuple()
	req := protos.Packet{Payload: data1}
	resp := protos.Packet{Payload: data2}

	private := protos.ProtocolData(new(httpConnectionData))

	private = http.Parse(&req, tcptuple, 0, private)
	private = http.ReceivedFin(tcptuple, 0, private)

	private = http.Parse(&resp, tcptuple, 1, private)

	logp.Debug("http", "Now sending gap..")

	private, drop := http.GapInStream(tcptuple, 1, 10, private)
	assert.Equal(t, false, drop)

	http.ReceivedFin(tcptuple, 1, private)

	trans := expectTransaction(t, http)
	assert.NotNil(t, trans)
	assert.Equal(t, trans["notes"], []string{"Packet loss while capturing the response"})
}

func TestHttp_configsSettingAll(t *testing.T) {

	http := httpModForTests()
	config := defaultConfig

	// Assign config vars
	config.Ports = []int{80, 8080}

	config.SendRequest = true
	config.SendResponse = true
	config.HideKeywords = []string{"a", "b"}
	config.RedactAuthorization = true
	config.SendAllHeaders = true
	config.SplitCookie = true
	config.RealIPHeader = "X-Forwarded-For"

	// Set config
	http.setFromConfig(&config)

	// Check if http config is set correctly
	assert.Equal(t, config.Ports, http.ports)
	assert.Equal(t, config.Ports, http.GetPorts())
	assert.Equal(t, config.SendRequest, http.sendRequest)
	assert.Equal(t, config.SendResponse, http.sendResponse)
	assert.Equal(t, config.HideKeywords, http.hideKeywords)
	assert.Equal(t, config.RedactAuthorization, http.redactAuthorization)
	assert.True(t, http.parserConfig.sendHeaders)
	assert.True(t, http.parserConfig.sendAllHeaders)
	assert.Equal(t, config.SplitCookie, http.splitCookie)
	assert.Equal(t, strings.ToLower(config.RealIPHeader), http.parserConfig.realIPHeader)
}

func TestHttp_configsSettingHeaders(t *testing.T) {

	http := httpModForTests()
	config := defaultConfig

	// Assign config vars
	config.SendHeaders = []string{"a", "b", "c"}

	// Set config
	http.setFromConfig(&config)

	// Check if http config is set correctly
	assert.True(t, http.parserConfig.sendHeaders)
	assert.Equal(t, len(config.SendHeaders), len(http.parserConfig.headersWhitelist))

	for _, val := range http.parserConfig.headersWhitelist {
		assert.True(t, val)
	}
}

func benchmarkHTTPMessage(b *testing.B, data []byte) {
	http := httpModForTests()
	parser := newParser(&http.parserConfig)

	for i := 0; i < b.N; i++ {
		stream := &stream{data: data, message: new(message)}
		ok, complete := parser.parse(stream, 0)
		if !ok || !complete {
			b.Errorf("failed to parse message")
		}
	}
}

func BenchmarkHTTPSimpleResponse(b *testing.B) {
	data := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n")

	benchmarkHTTPMessage(b, data)
}

func BenchmarkHTTPSimpleRequest(b *testing.B) {
	data := []byte("GET / HTTP/1.1\r\n" +
		"Host: www.google.ro\r\n" +
		"Connection: keep-alive\r\n" +
		"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_4) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/21.0.1180.75 Safari/537.1\r\n" +
		"Accept: */*\r\n" +
		"X-Chrome-Variations: CLa1yQEIj7bJAQiftskBCKS2yQEIp7bJAQiptskBCLSDygE=\r\n" +
		"Referer: http://www.google.ro/\r\n" +
		"Accept-Encoding: gzip,deflate,sdch\r\n" +
		"Accept-Language: en-US,en;q=0.8\r\n" +
		"Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.3\r\n" +
		"Cookie: PREF=ID=6b67d166417efec4:U=69097d4080ae0e15:FF=0:TM=1340891937:LM=1340891938:S=8t97UBiUwKbESvVX; NID=61=sf10OV-t02wu5PXrc09AhGagFrhSAB2C_98ZaI53-uH4jGiVG_yz9WmE3vjEBcmJyWUogB1ZF5puyDIIiB-UIdLd4OEgPR3x1LHNyuGmEDaNbQ_XaxWQqqQ59mX1qgLQ\r\n" +
		"\r\n" +
		"garbage")

	benchmarkHTTPMessage(b, data)
}

func BenchmarkHTTPSplitResponse(b *testing.B) {
	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n")
	data2 := []byte("Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n")

	http := httpModForTests()
	parser := newParser(&http.parserConfig)

	for i := 0; i < b.N; i++ {
		stream := &stream{data: data1, message: new(message)}
		ok, complete := parser.parse(stream, 0)
		if !ok || complete {
			b.Errorf("parse failure. Expected message to be incomplete, but no parse failures")
		}

		stream.data = append(stream.data, data2...)
		ok, complete = parser.parse(stream, 0)
		if !ok || !complete {
			b.Errorf("failed to parse message")
		}
	}
}

func BenchmarkHttpSimpleTransaction(b *testing.B) {
	data1 := "GET / HTTP/1.1\r\n" +
		"Host: www.google.ro\r\n" +
		"Connection: keep-alive\r\n" +
		"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_4) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/21.0.1180.75 Safari/537.1\r\n" +
		"Accept: */*\r\n" +
		"X-Chrome-Variations: CLa1yQEIj7bJAQiftskBCKS2yQEIp7bJAQiptskBCLSDygE=\r\n" +
		"Referer: http://www.google.ro/\r\n" +
		"Accept-Encoding: gzip,deflate,sdch\r\n" +
		"Accept-Language: en-US,en;q=0.8\r\n" +
		"Accept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.3\r\n" +
		"Cookie: PREF=ID=6b67d166417efec4:U=69097d4080ae0e15:FF=0:TM=1340891937:LM=1340891938:S=8t97UBiUwKbESvVX; NID=61=sf10OV-t02wu5PXrc09AhGagFrhSAB2C_98ZaI53-uH4jGiVG_yz9WmE3vjEBcmJyWUogB1ZF5puyDIIiB-UIdLd4OEgPR3x1LHNyuGmEDaNbQ_XaxWQqqQ59mX1qgLQ\r\n" +
		"\r\n"

	data2 := "HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n"

	http := httpModForTests()
	tcptuple := testCreateTCPTuple()
	req := protos.Packet{Payload: []byte(data1)}
	resp := protos.Packet{Payload: []byte(data2)}

	client := http.results.(*publish.ChanTransactions)

	for i := 0; i < b.N; i++ {
		private := protos.ProtocolData(&httpConnectionData{})

		private = http.Parse(&req, tcptuple, 0, private)
		private = http.ReceivedFin(tcptuple, 0, private)

		private = http.Parse(&resp, tcptuple, 1, private)
		http.ReceivedFin(tcptuple, 1, private)

		select {
		case <-client.Channel:
		default:
			b.Error("No transaction returned")
		}
	}
}
