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

	settings := monitors.MakeJobSetting(jobName).WithFields(common.MapStr{
		"monitor": common.MapStr{
			"scheme": request.URL.Scheme,
			"host":   hostname,
		},
		"http": common.MapStr{
			"url": request.URL.String(),
		},
		"tcp": common.MapStr{
			"port": port,
		},
	})

	return monitors.MakeSimpleJob(settings, func() (common.MapStr, error) {
		_, _, event, err := execPing(client, request, body, timeout, validator)
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

	settings := monitors.MakeHostJobSettings(jobName, hostname, config.Mode)
	settings = settings.WithFields(common.MapStr{
		"monitor": common.MapStr{
			"scheme": req.URL.Scheme,
		},
		"http": common.MapStr{
			"url": req.URL.String(),
		},
		"tcp": common.MapStr{
			"port": port,
		},
	})

	pingFactory := createPingFactory(config, hostname, port, tls, req, body, validator)
	return monitors.MakeByHostJob(settings, pingFactory)
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
	timeout := config.Timeout
	isTLS := request.URL.Scheme == "https"
	checkRedirect := makeCheckRedirect(config.MaxRedirects)

	return monitors.MakePingIPFactory(func(ip *net.IPAddr) (common.MapStr, error) {
		event := common.MapStr{}
		addr := net.JoinHostPort(ip.String(), strconv.Itoa(int(port)))
		d := &dialchain.DialerChain{
			Net: dialchain.MakeConstAddrDialer(addr, dialchain.TCPDialer(timeout)),
		}

		// TODO: add socks5 proxy?

		if isTLS {
			d.AddLayer(dialchain.TLSLayer(tls, timeout))
		}

		dialer, err := d.Build(event)
		if err != nil {
			return nil, err
		}

		var (
			writeStart, readStart, writeEnd time.Time
		)

		client := &http.Client{
			CheckRedirect: checkRedirect,
			Timeout:       timeout,
			Transport: &SimpleTransport{
				Dialer:       dialer,
				OnStartWrite: func() { writeStart = time.Now() },
				OnEndWrite:   func() { writeEnd = time.Now() },
				OnStartRead:  func() { readStart = time.Now() },
			},
		}

		_, end, result, err := execPing(client, request, body, timeout, validator)
		event.DeepUpdate(result)

		if !readStart.IsZero() {
			event.DeepUpdate(common.MapStr{
				"http": common.MapStr{
					"rtt": common.MapStr{
						"write_request":   look.RTT(writeEnd.Sub(writeStart)),
						"response_header": look.RTT(readStart.Sub(writeStart)),
					},
				},
			})
		}
		if !writeStart.IsZero() {
			event.Put("http.rtt.validate", look.RTT(end.Sub(writeStart)))
			event.Put("http.rtt.content", look.RTT(end.Sub(readStart)))
		}

		return event, err
	})
}

func buildRequest(addr string, config *Config, enc contentEncoder) (*http.Request, error) {
	method := strings.ToUpper(config.Check.Request.Method)
	request, err := http.NewRequest(method, addr, nil)
	if err != nil {
		return nil, err
	}
	request.Close = true

	if config.Username != "" {
		request.SetBasicAuth(config.Username, config.Password)
	}
	for k, v := range config.Check.Request.SendHeaders {
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
) (time.Time, time.Time, common.MapStr, reason.Reason) {
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
		return start, end, nil, reason.IOFailed(err)
	}
	defer resp.Body.Close()

	err = validator(resp)
	end = time.Now()

	rtt := end.Sub(start)
	event := common.MapStr{"http": common.MapStr{
		"response": common.MapStr{
			"status": resp.StatusCode,
		},
		"rtt": common.MapStr{
			"total": look.RTT(rtt),
		},
	}}

	if err != nil {
		return start, end, event, reason.ValidateFailed(err)
	}
	return start, end, event, nil
}

func splitHostnamePort(requ *http.Request) (string, uint16, error) {
	host := requ.URL.Host
	// Try to add a default port if needed
	if strings.LastIndex(host, ":") == -1 {
		switch requ.URL.Scheme {
		case urlSchemaHTTP:
			host += ":80"
		case urlSchemaHTTPS:
			host += ":443"
		}
	}
	host, port, err := net.SplitHostPort(host)
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
