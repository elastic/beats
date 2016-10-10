package http

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/monitors/active/dialchain"
	"github.com/elastic/beats/heartbeat/reason"
)

func newHTTPMonitorHostJob(
	addr string,
	config *Config,
	transport *http.Transport,
	enc contentEncoder,
	body []byte,
	validator RespCheck,
) (monitors.Job, error) {
	typ := config.Name
	jobName := fmt.Sprintf("%v@%v", typ, addr)

	client := &http.Client{
		CheckRedirect: makeCheckRedirect(config.MaxRedirects),
		Transport:     transport,
		Timeout:       config.Timeout,
	}
	request, err := buildRequest(addr, config, enc)
	if err != nil {
		return nil, err
	}

	hostname, port, err := splitHostnamePort(request)
	if err != nil {
		return nil, err
	}

	timeout := config.Timeout
	fields := common.MapStr{
		"scheme": request.URL.Scheme,
		"host":   hostname,
		"port":   port,
		"url":    request.URL.String(),
	}

	return monitors.MakeSimpleJob(jobName, typ, func() (common.MapStr, error) {
		event, err := execPing(client, request, body, timeout, validator)
		if event == nil {
			event = common.MapStr{}
		}
		event.Update(fields)
		return event, err
	}), nil
}

func newHTTPMonitorIPsJob(
	config *Config,
	addr string,
	tls *transport.TLSConfig,
	enc contentEncoder,
	body []byte,
	validator RespCheck,
) (monitors.Job, error) {
	typ := config.Name
	jobName := fmt.Sprintf("%v@%v", typ, addr)

	req, err := buildRequest(addr, config, enc)
	if err != nil {
		return nil, err
	}

	hostname, port, err := splitHostnamePort(req)
	if err != nil {
		return nil, err
	}

	pingFactory := createPingFactory(config, hostname, port, tls, req, body, validator)
	if ip := net.ParseIP(hostname); ip != nil {
		return monitors.MakeByIPJob(jobName, typ, ip, pingFactory)
	}
	return monitors.MakeByHostJob(jobName, typ, hostname, config.Mode, pingFactory)
}

func createPingFactory(
	config *Config,
	hostname string,
	port uint16,
	tls *transport.TLSConfig,
	request *http.Request,
	body []byte,
	validator RespCheck,
) func(*net.IPAddr) monitors.TaskRunner {
	fields := common.MapStr{
		"scheme": request.URL.Scheme,
		"port":   port,
		"url":    request.URL.String(),
	}

	timeout := config.Timeout
	isTLS := request.URL.Scheme == "https"
	checkRedirect := makeCheckRedirect(config.MaxRedirects)

	return monitors.MakePingIPFactory(fields, func(ip *net.IPAddr) (common.MapStr, error) {
		addr := net.JoinHostPort(ip.String(), strconv.Itoa(int(port)))
		d := &dialchain.DialerChain{
			Net: dialchain.ConstAddrDialer("tcp_connect_rtt", addr, timeout),
		}
		if isTLS {
			d.AddLayer(dialchain.TLSLayer("tls_handshake_rtt", tls, timeout))
		}

		measures := common.MapStr{}
		dialer, err := d.BuildWithMeasures(measures)
		if err != nil {
			return nil, err
		}

		var httpStart, httpEnd time.Time

		client := &http.Client{
			CheckRedirect: checkRedirect,
			Timeout:       timeout,
			Transport: &SimpleTransport{
				Dialer:       dialer,
				OnStartWrite: func() { httpStart = time.Now() },
				OnStartRead:  func() { httpEnd = time.Now() },
			},
		}

		event, err := execPing(client, request, body, timeout, validator)
		if event == nil {
			event = measures
		} else {
			event.Update(measures)
		}

		if !httpEnd.IsZero() {
			event["http_rtt"] = look.RTT(httpEnd.Sub(httpStart))
		}
		return event, err
	})
}

func buildRequest(addr string, config *Config, enc contentEncoder) (*http.Request, error) {
	method := strings.ToUpper(config.Check.Method)
	request, err := http.NewRequest(method, addr, nil)
	if err != nil {
		return nil, err
	}
	request.Close = true

	if config.Username != "" {
		request.SetBasicAuth(config.Username, config.Password)
	}
	for k, v := range config.Check.SendHeaders {
		request.Header.Add(k, v)
	}

	if enc != nil {
		enc.AddHeaders(&request.Header)
	}

	return request, nil
}

func execPing(
	client *http.Client,
	req *http.Request,
	body []byte,
	timeout time.Duration,
	validator func(*http.Response) error,
) (common.MapStr, reason.Reason) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req = req.WithContext(ctx)
	if len(body) > 0 {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		req.ContentLength = int64(len(body))
	}

	start := time.Now()
	resp, err := client.Do(req)
	end := time.Now()
	if err != nil {
		return nil, reason.IOFailed(err)
	}
	defer resp.Body.Close()

	if err := validator(resp); err != nil {
		return nil, reason.ValidateFailed(err)
	}

	rtt := end.Sub(start)
	event := common.MapStr{
		"response": common.MapStr{
			"status": resp.StatusCode,
		},
		"rtt": look.RTT(rtt),
	}
	return event, nil
}

func splitHostnamePort(requ *http.Request) (string, uint16, error) {
	host, port, err := net.SplitHostPort(requ.URL.Host)
	if err != nil {
		return "", 0, err
	}

	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("'%v' is no valid port number in '%v'", port, requ.URL.Host)
	}

	return host, uint16(p), nil
}

func makeCheckRedirect(max int) func(*http.Request, []*http.Request) error {
	if max == 0 {
		return func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return func(_ *http.Request, via []*http.Request) error {
		if max == len(via) {
			return http.ErrUseLastResponse
		}
		return nil
	}
}
