package main

import (
	"bytes"
	"strconv"
	"testing"
	"time"
	//"fmt"
)

func HttpModForTests() *Http {
	var http Http
	http.Init(true)
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

	stream := &HttpStream{tcpStream: nil, data: data, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data1, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data1, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data1, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data1, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data, message: new(HttpMessage)}

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

	stream := &HttpStream{tcpStream: nil, data: data1, message: new(HttpMessage)}

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
	LogInit(LOG_CRIT, "" /*toSyslog*/, false, []string{})

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

	stream := &HttpStream{tcpStream: nil, data: data, message: &HttpMessage{Ts: time.Now()}}

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

	stream := &HttpStream{tcpStream: nil, data: data, message: new(HttpMessage)}

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
		LogInit(LOG_DEBUG, "", false, []string{"http"})
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

	stream := &HttpStream{tcpStream: nil, data: data, message: new(HttpMessage)}

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
		LogInit(LOG_DEBUG, "", false, []string{"http", "httpdetailed"})
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
		LogInit(LOG_DEBUG, "", false, []string{"http", "httpdetailed"})
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

	DEBUG("http", "parseOffset", stream.parseOffset)

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
