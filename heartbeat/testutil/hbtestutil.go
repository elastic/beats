package testutil

import (
	"io"
	"net/http"
	"net/url"
	"strconv"

	"net/http/httptest"

	"github.com/elastic/beats/heartbeat/skima"
)

var HelloWorldBody = "hello, world!"

var HelloWorldHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, HelloWorldBody)
})

var BadGatewayBody = "Bad Gateway"

var BadGatewayHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadGateway)
	io.WriteString(w, BadGatewayBody)
})

func ServerPort(server *httptest.Server) (uint16, error) {
	u, err := url.Parse(server.URL)
	if err != nil {
		return 0, err
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		return 0, err
	}
	return uint16(p), nil
}

func MonitorChecks(id string, ip string, scheme string, status string) skima.Validator {
	return skima.Schema(skima.Map{
		"monitor": skima.Map{
			"duration.us": skima.IsDuration,
			"id":          id,
			"ip":          ip,
			"scheme":      scheme,
			"status":      status,
		},
	})
}

func TcpChecks(port uint16) skima.Validator {
	return skima.Schema(skima.Map{
		"tcp": skima.Map{
			"port":           port,
			"rtt.connect.us": skima.IsDuration,
		},
	})
}
