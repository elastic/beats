// +build !integration

package http

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/elastic/beats/metricbeat/helper/server"

	"github.com/stretchr/testify/assert"
)

func GetHttpServer(host string, port int) (server.Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	h := &HttpServer{
		done:       make(chan struct{}),
		eventQueue: make(chan server.Event, 1),
		ctx:        ctx,
		stop:       cancel,
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: http.HandlerFunc(h.handleFunc),
	}
	h.server = httpServer

	return h, nil
}

func TestHttpServer(t *testing.T) {
	host := "127.0.0.1"
	port := 40050
	svc, err := GetHttpServer(host, port)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	svc.Start()
	defer svc.Stop()
	// make sure server is up before writing data into it.
	time.Sleep(2 * time.Second)
	writeToServer(t, "test1", host, port)
	msg := <-svc.GetEvents()

	assert.True(t, msg.GetEvent() != nil)
	ok, _ := msg.GetEvent().HasKey("data")
	assert.True(t, ok)
	bytes, _ := msg.GetEvent()["data"].([]byte)
	assert.True(t, string(bytes) == "test1")

}

func writeToServer(t *testing.T, message, host string, port int) {
	url := fmt.Sprintf("http://%s:%d/", host, port)
	var str = []byte(message)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(str))
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer resp.Body.Close()

}
