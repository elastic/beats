package consumer

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/cloudfoundry/noaa/consumer/internal"

	noaa_errors "github.com/cloudfoundry/noaa/errors"
	"github.com/gorilla/websocket"
)

var (
	// KeepAlive sets the interval between keep-alive messages sent by the client to loggregator.
	KeepAlive = 25 * time.Second

	boundaryRegexp       = regexp.MustCompile("boundary=(.*)")
	ErrNotOK             = errors.New("unknown issue when making HTTP request to Loggregator")
	ErrNotFound          = ErrNotOK // NotFound isn't an accurate description of how this is used; please use ErrNotOK instead
	ErrBadResponse       = errors.New("bad server response")
	ErrBadRequest        = errors.New("bad client request")
	ErrLostConnection    = errors.New("remote server terminated connection unexpectedly")
	ErrMaxRetriesReached = errors.New("maximum number of connection retries reached")
)

//go:generate hel --type DebugPrinter --output mock_debug_printer_test.go

// DebugPrinter is a type which handles printing debug information.
type DebugPrinter interface {
	Print(title, dump string)
}

type nullDebugPrinter struct {
}

func (nullDebugPrinter) Print(title, body string) {
}

// Consumer represents the actions that can be performed against trafficcontroller.
// See sync.go and async.go for trafficcontroller access methods.
type Consumer struct {
	// minRetryDelay, maxRetryDelay, and maxRetryCount must be the first words in
	// this struct in order to be used atomically by 32-bit systems.
	// https://golang.org/src/sync/atomic/doc.go?#L50
	minRetryDelay, maxRetryDelay, maxRetryCount int64

	trafficControllerUrl string
	idleTimeout          time.Duration
	callback             func()
	callbackLock         sync.RWMutex
	debugPrinter         DebugPrinter
	client               *http.Client
	dialer               websocket.Dialer

	conns     []*connection
	connsLock sync.Mutex

	refreshTokens  bool
	refresherMutex sync.RWMutex
	tokenRefresher TokenRefresher
}

// New creates a new consumer to a trafficcontroller.
func New(trafficControllerUrl string, tlsConfig *tls.Config, proxy func(*http.Request) (*url.URL, error)) *Consumer {
	if proxy == nil {
		proxy = http.ProxyFromEnvironment
	}

	return &Consumer{
		trafficControllerUrl: trafficControllerUrl,
		debugPrinter:         nullDebugPrinter{},
		client: &http.Client{
			Transport: &http.Transport{
				Proxy:               proxy,
				TLSClientConfig:     tlsConfig,
				TLSHandshakeTimeout: internal.Timeout,
				DisableKeepAlives:   true,
			},
			Timeout: internal.Timeout,
		},
		minRetryDelay: int64(DefaultMinRetryDelay),
		maxRetryDelay: int64(DefaultMaxRetryDelay),
		maxRetryCount: int64(DefaultMaxRetryCount),
		dialer: websocket.Dialer{
			HandshakeTimeout: internal.Timeout,
			Proxy:            proxy,
			TLSClientConfig:  tlsConfig,
		},
	}
}

type httpError struct {
	statusCode int
	error      error
}

func checkForErrors(resp *http.Response) *httpError {
	if resp.StatusCode == http.StatusUnauthorized {
		data, _ := ioutil.ReadAll(resp.Body)
		return &httpError{
			statusCode: resp.StatusCode,
			error:      noaa_errors.NewUnauthorizedError(string(data)),
		}
	}

	if resp.StatusCode == http.StatusBadRequest {
		return &httpError{
			statusCode: resp.StatusCode,
			error:      ErrBadRequest,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &httpError{
			statusCode: resp.StatusCode,
			error:      ErrNotOK,
		}
	}
	return nil
}
