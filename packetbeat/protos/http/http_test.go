package http

import (
	"bytes"
	"net"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/stretchr/testify/assert"
)

func HttpModForTests() *Http {
	var http Http
	results := publisher.ChanClient{make(chan common.MapStr, 10)}
	http.Init(true, results)
	return &http
}

func TestHttpParser_simpleResponse(t *testing.T) {

	http := HttpModForTests()

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

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	if stream.message.IsRequest {
		t.Errorf("Failed to parse HTTP response")
	}
	if stream.message.StatusCode != 200 {
		t.Errorf("Failed to parse status code: %d", stream.message.StatusCode)
	}
	if stream.message.StatusPhrase != "OK" {
		t.Errorf("Failed to parse response phrase: %s", stream.message.StatusPhrase)
	}
	if stream.message.ContentLength != 0 {
		t.Errorf("Failed to parse Content Length: %s", stream.message.Headers["content-length"])
	}
	if stream.message.version_major != 1 {
		t.Errorf("Failed to parse version major")
	}
	if stream.message.version_minor != 1 {
		t.Errorf("Failed to parse version minor")
	}
	if stream.message.Size != 262 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

func TestHttpParser_simpleResponseCaseInsensitive(t *testing.T) {

	http := HttpModForTests()

	data := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"EXPIRES: -1\r\n" +
		"cACHE-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"SERVER: gws\r\n" +
		"content-LeNgTh: 0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n")

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	if stream.message.IsRequest {
		t.Errorf("Failed to parse HTTP response")
	}
	if stream.message.StatusCode != 200 {
		t.Errorf("Failed to parse status code: %d", stream.message.StatusCode)
	}
	if stream.message.StatusPhrase != "OK" {
		t.Errorf("Failed to parse response phrase: %s", stream.message.StatusPhrase)
	}
	if stream.message.ContentLength != 0 {
		t.Errorf("Failed to parse Content Length: %s", stream.message.Headers["content-length"])
	}
	if stream.message.version_major != 1 {
		t.Errorf("Failed to parse version major")
	}
	if stream.message.version_minor != 1 {
		t.Errorf("Failed to parse version minor")
	}
	if stream.message.Size != 262 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

func TestHttpParser_simpleRequest(t *testing.T) {

	http := HttpModForTests()
	http.Send_headers = true
	http.Send_all_headers = true

	data := []byte(
		"GET / HTTP/1.1\r\n" +
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

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	if !bytes.Equal(stream.data[stream.parseOffset:], []byte("garbage")) {
		t.Errorf("The offset is wrong")
	}
	if !stream.message.IsRequest {
		t.Errorf("Failed to parse the HTTP request")
	}
	if stream.message.Method != "GET" {
		t.Errorf("Failed to parse HTTP method: %s", stream.message.Method)
	}
	if stream.message.RequestUri != "/" {
		t.Errorf("Failed to parse HTTP request uri: %s", stream.message.RequestUri)
	}
	if stream.message.Headers["host"] != "www.google.ro" {
		t.Errorf("Failed to parse HTTP Host header: %s", stream.message.Headers["host"])
	}
	if stream.message.version_major != 1 {
		t.Errorf("Failed to parse version major")
	}
	if stream.message.version_minor != 1 {
		t.Errorf("Failed to parse version minor")
	}
	if stream.message.Size != 669 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}

}

func TestHttpParser_Request_ContentLength_0(t *testing.T) {

	http := HttpModForTests()
	http.Send_headers = true
	http.Send_all_headers = true

	data := []byte("POST / HTTP/1.1\r\n" +
		"user-agent: curl/7.35.0\r\n" + "host: localhost:9000\r\n" +
		"accept: */*\r\n" +
		"authorization: Company 1\r\n" +
		"content-length: 0\r\n" +
		"connection: close\r\n" +
		"\r\n")

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

}

func TestHttpParser_splitResponse(t *testing.T) {

	http := HttpModForTests()

	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n")

	data2 := []byte(
		"Content-Encoding: gzip\r\n" +
			"Server: gws\r\n" +
			"Content-Length: 0\r\n" +
			"X-XSS-Protection: 1; mode=block\r\n" +
			"X-Frame-Options: SAMEORIGIN\r\n" +
			"\r\n")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	stream.data = append(stream.data, data2...)

	ok, complete = http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
}

func TestHttpParser_splitResponse_midHeaderName(t *testing.T) {
	http := HttpModForTests()
	http.Send_headers = true
	http.Send_all_headers = true

	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-En")

	data2 := []byte(
		"coding: gzip\r\n" +
			"Server: gws\r\n" +
			"Content-Length: 0\r\n" +
			"X-XSS-Protection: 1; mode=block\r\n" +
			"X-Frame-Options: SAMEORIGIN\r\n" +
			"\r\n")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	stream.data = append(stream.data, data2...)

	ok, complete = http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.StatusCode != 200 {
		t.Errorf("Failed to parse response code")
	}
	if stream.message.StatusPhrase != "OK" {
		t.Errorf("Failed to parse response phrase")
	}
	if stream.message.Headers["content-type"] != "text/html; charset=UTF-8" {
		t.Errorf("Failed to parse content type")
	}
	if stream.message.version_major != 1 {
		t.Errorf("Failed to parse version major")
	}
	if stream.message.version_minor != 1 {
		t.Errorf("Failed to parse version minor")
	}
}

func TestHttpParser_splitResponse_midHeaderValue(t *testing.T) {

	http := HttpModForTests()

	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: g")

	data2 := []byte(
		"zip\r\n" +
			"Server: gws\r\n" +
			"Content-Length: 0\r\n" +
			"X-XSS-Protection: 1; mode=block\r\n" +
			"X-Frame-Options: SAMEORIGIN\r\n" +
			"\r\n")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	stream.data = append(stream.data, data2...)

	ok, complete = http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
}

func TestHttpParser_splitResponse_midNewLine(t *testing.T) {

	http := HttpModForTests()
	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r")

	data2 := []byte(
		"\n" +
			"Server: gws\r\n" +
			"Content-Length: 0\r\n" +
			"X-XSS-Protection: 1; mode=block\r\n" +
			"X-Frame-Options: SAMEORIGIN\r\n" +
			"\r\n")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	stream.data = append(stream.data, data2...)

	ok, complete = http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
}

func TestHttpParser_ResponseWithBody(t *testing.T) {
	http := HttpModForTests()

	data := []byte("HTTP/1.1 200 OK\r\n" +
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
		"garbage")

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	if stream.message.ContentLength != 30 {
		t.Errorf("Wrong Content-Length =" + strconv.Itoa(stream.message.ContentLength))
	}

	if !bytes.Equal(stream.data[stream.parseOffset:], []byte("garbage")) {
		t.Errorf("The offset is wrong")
	}
}

func TestHttpParser_Response_HTTP_10_without_content_length(t *testing.T) {
	http := HttpModForTests()

	data := []byte("HTTP/1.0 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		"test")

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	if stream.message.ContentLength != 4 {
		t.Errorf("Wrong Content-Length =" + strconv.Itoa(stream.message.ContentLength))
	}

}

func TestHttpParser_splitResponse_midBody(t *testing.T) {
	http := HttpModForTests()

	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Encoding: gzip\r\n" +
		"Server: gws\r\n" +
		"Content-Length: 3")

	data2 := []byte("0\r\n" +
		"X-XSS-Protection: 1; mode=block\r\n" +
		"X-Frame-Options: SAMEORIGIN\r\n" +
		"\r\n" +
		"xxxxxxxxxx")

	data3 := []byte("xxxxxxxxxxxxxxxxxxxx")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	stream.data = append(stream.data, data2...)
	ok, complete = http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	stream.data = append(stream.data, data3...)
	ok, complete = http.messageParser(stream)
	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	if stream.message.ContentLength != 30 {
		t.Errorf("Wrong content-length")
	}

	if !bytes.Equal(stream.data[stream.parseOffset:], []byte("")) {
		t.Errorf("The offset is wrong")
	}
}

func TestHttpParser_RequestResponse(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := HttpModForTests()

	data := []byte(
		"GET / HTTP/1.1\r\n" +
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
			"\r\n")

	stream := &HttpStream{data: data, message: &HttpMessage{Ts: time.Now()}}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	stream.PrepareForNewMessage()
	stream.message = &HttpMessage{Ts: time.Now()}

	ok, complete = http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}
}

func TestHttpParser_RequestResponseBody(t *testing.T) {
	http := HttpModForTests()

	data1 := []byte(
		"GET / HTTP/1.1\r\n" +
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
			"xx")

	data2 := []byte(
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
			"\r\n")

	data := append(data1, data2...)

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	if stream.message.ContentLength != 2 {
		t.Errorf("Wrong content lenght")
	}

	if !bytes.Equal(stream.data[stream.message.start:stream.message.end], data1) {
		t.Errorf("First message not correctly extracted")
	}

	stream.PrepareForNewMessage()
	stream.message = &HttpMessage{Ts: time.Now()}

	ok, complete = http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}
}

func TestHttpParser_301_response(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http"})
	}
	http := HttpModForTests()

	data := []byte(
		"HTTP/1.1 301 Moved Permanently\r\n" +
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
			"</body></html>")

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	if stream.message.ContentLength != 290 {
		t.Errorf("Expecting content length 290")
	}
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
	stream := &HttpStream{
		data:         msgs[0],
		parseOffset:  0,
		bodyReceived: 0,
		parseState:   BODY_CHUNKED_START,
	}
	message := &HttpMessage{
		chunked_length: 5,
		ContentLength:  0,
	}

	cont, ok, complete := state_body_chunked_start(stream, message)
	if cont != false || ok != true || complete != false {
		t.Errorf("Wrong return values")
	}
	if stream.parseOffset != 0 {
		t.Errorf("Wrong parseOffset")
	}

	stream.data = append(stream.data, msgs[1]...)

	cont, ok, complete = state_body_chunked_start(stream, message)
	if cont != true {
		t.Errorf("Wrong return values")
	}
	if message.chunked_length != 3 {
		t.Errorf("Wrong chunked_length")
	}
	if stream.parseOffset != 4 {
		t.Errorf("Wrong parseOffset")
	}
	if stream.parseState != BODY_CHUNKED {
		t.Errorf("Wrong state")
	}

	cont, ok, complete = state_body_chunked(stream, message)
	if cont != true {
		t.Errorf("Wrong return values")
	}
	if stream.parseState != BODY_CHUNKED_START {
		t.Errorf("Wrong state")
	}
	if stream.parseOffset != 9 {
		t.Errorf("Wrong parseOffset")
	}

	cont, ok, complete = state_body_chunked_start(stream, message)
	if cont != true {
		t.Errorf("Wrong return values")
	}
	if message.chunked_length != 3 {
		t.Errorf("Wrong chunked_length")
	}
	if stream.parseOffset != 13 {
		t.Errorf("Wrong parseOffset")
	}
	if stream.parseState != BODY_CHUNKED {
		t.Errorf("Wrong state")
	}

	cont, ok, complete = state_body_chunked(stream, message)
	if cont != false || ok != true || complete != false {
		t.Errorf("Wrong return values")
	}
	if stream.parseState != BODY_CHUNKED {
		t.Errorf("Wrong state")
	}
	if stream.parseOffset != 13 {
		t.Errorf("Wrong parseOffset")
	}
	if stream.bodyReceived != 0 {
		t.Errorf("Wrong bodyReceived")
	}

	stream.data = append(stream.data, msgs[2]...)
	cont, ok, complete = state_body_chunked(stream, message)
	if cont != true {
		t.Errorf("Wrong return values")
	}
	if stream.parseState != BODY_CHUNKED_START {
		t.Errorf("Wrong state")
	}
	if stream.parseOffset != 18 {
		t.Errorf("Wrong parseOffset")
	}

	cont, ok, complete = state_body_chunked_start(stream, message)
	if cont != false || ok != true || complete != true {
		t.Error("Wrong return values", cont, ok, complete)
	}
}

func TestEatBodyChunkedWaitCRLF(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	msgs := [][]byte{
		[]byte("03\r\n123\r\n0\r\n\r"),
		[]byte("\n"),
	}
	stream := &HttpStream{
		data:         msgs[0],
		parseOffset:  0,
		bodyReceived: 0,
		parseState:   BODY_CHUNKED_START,
	}
	message := &HttpMessage{
		chunked_length: 5,
		ContentLength:  0,
	}

	cont, ok, complete := state_body_chunked_start(stream, message)
	if cont != true || ok != true || complete != false {
		t.Error("Wrong return values", cont, ok, complete)
	}
	if stream.parseState != BODY_CHUNKED {
		t.Error("Unexpected state", stream.parseState)
	}

	cont, ok, complete = state_body_chunked(stream, message)
	if cont != true || ok != true || complete != false {
		t.Error("Wrong return values", cont, ok, complete)
	}
	if stream.parseState != BODY_CHUNKED_START {
		t.Error("Unexpected state", stream.parseState)
	}

	cont, ok, complete = state_body_chunked_start(stream, message)
	if cont != false || ok != true || complete != false {
		t.Error("Wrong return values", cont, ok, complete)
	}
	if stream.parseState != BODY_CHUNKED_WAIT_FINAL_CRLF {
		t.Error("Unexpected state", stream.parseState)
	}

	logp.Debug("http", "parseOffset", stream.parseOffset)

	ok, complete = state_body_chunked_wait_final_crlf(stream, message)
	if ok != true || complete != false {
		t.Error("Wrong return values", ok, complete)

	}
	stream.data = append(stream.data, msgs[1]...)

	ok, complete = state_body_chunked_wait_final_crlf(stream, message)
	if ok != true || complete != true {
		t.Error("Wrong return values", ok, complete)
	}
	if message.end != 14 {
		t.Error("Wrong message end", message.end)
	}
}

func TestHttpParser_censorPasswordURL(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := HttpModForTests()
	http.Hide_keywords = []string{"password", "pass"}
	http.Send_headers = true
	http.Send_all_headers = true

	data1 := []byte(
		"GET http://localhost:8080/test?password=secret HTTP/1.1\r\n" +
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
			"username=ME&pass=secret")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	msg := stream.data[stream.message.start:stream.message.end]
	path, params, err := http.extractParameters(stream.message, msg)
	if err != nil {
		t.Errorf("Fail to parse parameters")
	}

	if path != "/test" {
		t.Errorf("Wrong path: %s", path)
	}

	if strings.Contains(params, "secret") {
		t.Errorf("Failed to censor the password: %s", params)
	}
}

func TestHttpParser_censorPasswordPOST(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := HttpModForTests()
	http.Hide_keywords = []string{"password"}
	http.Send_headers = true
	http.Send_all_headers = true

	data1 := []byte(
		"POST /users/login HTTP/1.1\r\n" +
			"HOST: www.example.com\r\n" +
			"Content-Type: application/x-www-form-urlencoded\r\n" +
			"Content-Length: 28\r\n" +
			"\r\n" +
			"username=ME&password=secret\r\n")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	msg := stream.data[stream.message.start:stream.message.end]
	path, params, err := http.extractParameters(stream.message, msg)
	if err != nil {
		t.Errorf("Fail to parse parameters")
	}

	if path != "/users/login" {
		t.Errorf("Wrong path: %s", path)
	}

	if strings.Contains(params, "secret") {
		t.Errorf("Failed to censor the password: %s", msg)
	}
}
func TestHttpParser_censorPasswordGET(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"http", "httpdetailed"})
	}

	http := HttpModForTests()
	http.Hide_keywords = []string{"password"}
	http.Send_headers = true
	http.Send_all_headers = true
	http.Send_request = false
	http.Send_response = false

	data1 := []byte(
		"GET /users/login HTTP/1.1\r\n" +
			"HOST: www.example.com\r\n" +
			"Content-Type: application/x-www-form-urlencoded\r\n" +
			"Content-Length: 53\r\n" +
			"\r\n" +
			"password=my_secret_pass&Password=my_secret_password_2\r\n")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	msg := stream.data[stream.message.start:stream.message.end]
	path, params, err := http.extractParameters(stream.message, msg)
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

	http := HttpModForTests()
	http.Redact_authorization = true
	http.Send_headers = true
	http.Send_all_headers = true

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

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, _ := http.messageParser(stream)

	msg := stream.data[stream.message.start:]
	http.hideHeaders(stream.message, msg)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if stream.message.Headers["authorization"] != "*" {
		t.Errorf("Failed to redact authorization header: " + stream.message.Headers["authorization"])
	}
	authPattern, _ := regexp.Compile(`(?m)^[Aa]uthorization:\*+`)
	authObscured := authPattern.Match(msg)
	if !authObscured {
		t.Errorf("Obscured authorization string not found: " + string(msg[:]))
	}

	if stream.message.Headers["proxy-authorization"] != "*" {
		t.Errorf("Failed to redact proxy authorization header: " + stream.message.Headers["proxy-authorization"])
	}
	proxyPattern, _ := regexp.Compile(`(?m)^[Pp]roxy-[Aa]uthorization:\*+`)
	proxyObscured := proxyPattern.Match(msg)
	if !proxyObscured {
		t.Errorf("Obscured proxy-authorization string not found: " + string(msg[:]))
	}

}

func TestHttpParser_RedactAuthorization_raw(t *testing.T) {

	http := HttpModForTests()
	http.Redact_authorization = true
	http.Send_headers = false
	http.Send_all_headers = false

	data := []byte("POST / HTTP/1.1\r\n" +
		"user-agent: curl/7.35.0\r\n" + "host: localhost:9000\r\n" +
		"accept: */*\r\n" +
		"authorization: Company 1\r\n" +
		"content-length: 0\r\n" +
		"connection: close\r\n" +
		"\r\n")

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	msg := stream.data[stream.message.start:]
	http.hideHeaders(stream.message, msg)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	raw_message_obscured := bytes.Index(msg, []byte("uthorization:*"))
	if raw_message_obscured < 0 {
		t.Errorf("Obscured authorization string not found: " + string(msg[:]))
	}
}

func TestHttpParser_RedactAuthorization_Proxy_raw(t *testing.T) {

	http := HttpModForTests()
	http.Redact_authorization = true
	http.Send_headers = false
	http.Send_all_headers = false

	data := []byte("POST / HTTP/1.1\r\n" +
		"user-agent: curl/7.35.0\r\n" + "host: localhost:9000\r\n" +
		"accept: */*\r\n" +
		"proxy-authorization: cHJveHk6MWM3MGRjM2JhZDIwCg==\r\n" +
		"content-length: 0\r\n" +
		"connection: close\r\n" +
		"\r\n")

	stream := &HttpStream{data: data, message: new(HttpMessage)}

	ok, complete := http.messageParser(stream)

	msg := stream.data[stream.message.start:]
	http.hideHeaders(stream.message, msg)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if !complete {
		t.Errorf("Expecting a complete message")
	}

	raw_message_obscured := bytes.Index(msg, []byte("uthorization:*"))
	if raw_message_obscured < 0 {
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

	http := HttpModForTests()

	data1 := []byte("HTTP/1.1 200 OK\r\n" +
		"Date: Tue, 14 Aug 2012 22:31:45 GMT\r\n" +
		"Expires: -1\r\n" +
		"Cache-Control: private, max-age=0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n")

	stream := &HttpStream{data: data1, message: new(HttpMessage)}
	ok, complete := http.messageParser(stream)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	ok, complete = http.messageGap(stream, 5)
	assert.Equal(t, false, ok)
	assert.Equal(t, false, complete)
}

// If a TCP gap (lost packets) happen while we're waiting for
// parts of the body, it's ok.
func Test_gap_in_body(t *testing.T) {

	http := HttpModForTests()

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

	stream := &HttpStream{data: data1, message: new(HttpMessage)}
	ok, complete := http.messageParser(stream)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	ok, complete = http.messageGap(stream, 10)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	ok, complete = http.messageGap(stream, 10)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, complete)
}

// If a TCP gap (lost packets) happen while we're waiting for
// parts of the body, it's ok.
func Test_gap_in_body_http1dot0(t *testing.T) {

	http := HttpModForTests()

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

	stream := &HttpStream{data: data1, message: new(HttpMessage)}
	ok, complete := http.messageParser(stream)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	ok, complete = http.messageGap(stream, 10)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

}

func testTcpTuple() *common.TcpTuple {
	t := &common.TcpTuple{
		Ip_length: 4,
		Src_ip:    net.IPv4(192, 168, 0, 1), Dst_ip: net.IPv4(192, 168, 0, 2),
		Src_port: 6512, Dst_port: 80,
	}
	t.ComputeHashebles()
	return t
}

// Helper function to read from the Publisher Queue
func expectTransaction(t *testing.T, http *Http) common.MapStr {
	client := http.results.(publisher.ChanClient)
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
	http := HttpModForTests()

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

	tcptuple := testTcpTuple()
	req := protos.Packet{Payload: data1}
	resp := protos.Packet{Payload: data2}

	private := protos.ProtocolData(new(httpPrivateData))

	private = http.Parse(&req, tcptuple, 0, private)
	private = http.ReceivedFin(tcptuple, 0, private)

	private = http.Parse(&resp, tcptuple, 1, private)

	logp.Debug("http", "Now sending gap..")

	private, drop := http.GapInStream(tcptuple, 1, 10, private)
	assert.Equal(t, false, drop)

	private = http.ReceivedFin(tcptuple, 1, private)

	trans := expectTransaction(t, http)
	assert.NotNil(t, trans)
	assert.Equal(t, trans["notes"], []string{"Packet loss while capturing the response"})
}

func TestHttp_configsSettingAll(t *testing.T) {

	http := HttpModForTests()
	config := new(config.Http)

	// Assign config vars
	config.Ports = []int{80, 8080}

	trueVar := true
	config.SendRequest = &trueVar
	config.SendResponse = &trueVar
	config.Hide_keywords = []string{"a", "b"}
	config.Redact_authorization = &trueVar
	config.Send_all_headers = &trueVar
	config.Split_cookie = &trueVar
	realIpHeader := "X-Forwarded-For"
	config.Real_ip_header = &realIpHeader

	// Set config
	http.SetFromConfig(*config)

	// Check if http config is set correctly
	assert.Equal(t, config.Ports, http.Ports)
	assert.Equal(t, config.Ports, http.GetPorts())
	assert.Equal(t, *config.SendRequest, http.Send_request)
	assert.Equal(t, *config.SendResponse, http.Send_response)
	assert.Equal(t, config.Hide_keywords, http.Hide_keywords)
	assert.Equal(t, *config.Redact_authorization, http.Redact_authorization)
	assert.True(t, http.Send_headers)
	assert.True(t, http.Send_all_headers)
	assert.Equal(t, *config.Split_cookie, http.Split_cookie)
	assert.Equal(t, strings.ToLower(*config.Real_ip_header), http.Real_ip_header)
}

func TestHttp_configsSettingHeaders(t *testing.T) {

	http := HttpModForTests()
	config := new(config.Http)

	// Assign config vars
	config.Send_headers = []string{"a", "b", "c"}

	// Set config
	http.SetFromConfig(*config)

	// Check if http config is set correctly
	assert.True(t, http.Send_headers)
	assert.Equal(t, len(config.Send_headers), len(http.Headers_whitelist))

	for _, val := range http.Headers_whitelist {
		assert.True(t, val)
	}

}
