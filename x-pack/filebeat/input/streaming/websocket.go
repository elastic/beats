// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type websocketStream struct {
	processor

	id          string
	cfg         config
	cursor      map[string]any
	tokenSource oauth2.TokenSource
	tokenExpiry <-chan time.Time
	time        func() time.Time
	keepAlive   *keepAlive
}

type loggingRoundTripper struct {
	rt  http.RoundTripper
	log *logp.Logger
}

// keepAlive manages the configuration and metrics for WebSocket keep-alive functionality
type keepAlive struct {
	cfg     keepAliveConfig
	metrics *inputMetrics
	log     *logp.Logger
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := l.rt.RoundTrip(req)
	// avoided logging request and and response body as it may contain sensitive information and can be huge
	if l.log.Core().Enabled(zapcore.DebugLevel) {
		l.log.Debugf("request: %v %v\nHeaders: %v\n", req.Method, req.URL, req.Header)
		if err == nil {
			l.log.Debugf("response: %v\nHeaders: %v\n", resp.Status, resp.Header)
		}
	}
	return resp, err
}

// heartBeat sends a ping message to the websocket server at regular intervals
func (k *keepAlive) heartBeat(ctx context.Context, conn *websocket.Conn, start time.Time) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	// set initial read deadline
	conn.SetReadDeadline(time.Now().Add(k.cfg.readControlDeadline))
	// set pong handler to update read deadline
	conn.SetPongHandler(func(pongData string) error {
		k.log.Debugw("received pong message from websocket server", "pong_data", pongData)
		k.metrics.pongMessageReceivedTime.Update(time.Since(start).Nanoseconds())
		return conn.SetReadDeadline(time.Now().Add(k.cfg.readControlDeadline))
	})

	// set heartbeat ping routine
	go func() {
		ticker := time.NewTicker(k.cfg.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				k.log.Debugw("heartbeat stopped")
				return
			case now := <-ticker.C:
				err := conn.WriteControl(websocket.PingMessage, nil, now.Add(k.cfg.WriteControlDeadline))
				if err != nil {
					k.log.Debugw("error sending ping control frame to websocket server", "error", err)
					k.metrics.writeControlErrors.Inc()
					k.metrics.errorsTotal.Inc()
				} else {
					k.log.Debugw("sent ping control frame to websocket server")
					k.metrics.pingMessageSendTime.Update(time.Since(start).Nanoseconds())
				}
			}
		}
	}()

	return cancel
}

// NewWebsocketFollower performs environment construction including CEL
// program and regexp compilation, and input metrics set-up for a websocket
// stream follower.
func NewWebsocketFollower(ctx context.Context, id string, cfg config, cursor map[string]any, pub inputcursor.Publisher, log *logp.Logger, now func() time.Time) (StreamFollower, error) {
	s := websocketStream{
		id:     id,
		cfg:    cfg,
		cursor: cursor,
		processor: processor{
			ns:      "websocket",
			pub:     pub,
			log:     log,
			redact:  cfg.Redact,
			metrics: newInputMetrics(id, nil),
		},
		// the token expiry handler will never trigger unless a valid expiry time is assigned
		tokenExpiry: nil,
	}
	s.metrics.url.Set(cfg.URL.String())
	s.metrics.errorsTotal.Set(0)
	// initialize the oauth2 token source if oauth2 is enabled and set access token in the config
	if cfg.Auth.OAuth2.isEnabled() {
		config := &clientcredentials.Config{
			AuthStyle:      cfg.Auth.OAuth2.getAuthStyle(),
			ClientID:       cfg.Auth.OAuth2.ClientID,
			ClientSecret:   cfg.Auth.OAuth2.ClientSecret,
			TokenURL:       cfg.Auth.OAuth2.TokenURL,
			Scopes:         cfg.Auth.OAuth2.Scopes,
			EndpointParams: cfg.Auth.OAuth2.EndpointParams,
		}
		// injecting a custom http client with loggingRoundTripper to debug-log request and response attributes for oauth2 token
		client := &http.Client{
			Transport: &loggingRoundTripper{http.DefaultTransport, log},
		}
		oauth2Ctx := context.WithValue(ctx, oauth2.HTTPClient, client)
		s.tokenSource = config.TokenSource(oauth2Ctx)
		// get the initial token
		token, err := s.tokenSource.Token()
		if err != nil {
			s.metrics.errorsTotal.Inc()
			s.Close()
			return nil, fmt.Errorf("failed to obtain oauth2 token: %w", err)
		}
		// set the initial token in the config if oauth2 is enabled
		// this allows seamless header creation in formHeader() for the initial connection
		s.cfg.Auth.OAuth2.accessToken = token.AccessToken
		// set the initial token expiry channel with buffer of 2 mins
		s.tokenExpiry = time.After(time.Until(token.Expiry) - cfg.Auth.OAuth2.TokenExpiryBuffer)
	}

	// set and assign KeepAlive if enabled
	if cfg.KeepAlive.Enable {
		// create a new keepAlive instance
		k := &keepAlive{
			cfg:     cfg.KeepAlive,
			metrics: s.metrics,
			log:     log,
		}
		s.keepAlive = k
	}

	patterns, err := regexpsFromConfig(cfg)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.Close()
		return nil, err
	}

	s.prg, s.ast, err = newProgram(ctx, cfg.Program, root, patterns, log)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.Close()
		return nil, err
	}

	return &s, nil
}

// FollowStream receives, processes and publishes events from the subscribed
// websocket stream.
func (s *websocketStream) FollowStream(ctx context.Context) error {
	var heartBeatCancel context.CancelFunc
	state := s.cfg.State

	if state == nil {
		state = make(map[string]any)
	}
	if s.cursor != nil {
		state["cursor"] = s.cursor
	}

	// initialize the input url with the help of the url_program.
	url, err := getURL(ctx, "websocket", s.cfg.URLProgram, s.cfg.URL.String(), state, s.cfg.Redact, s.log, s.now)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		return err
	}

	// websocket client
	c, resp, err := connectWebSocket(ctx, s.cfg, url, s.log)
	handleConnectionResponse(resp, s.metrics, s.log)
	if err != nil {
		s.metrics.errorsTotal.Inc()
		s.log.Errorw("failed to establish websocket connection", "error", err)
		return err
	}
	// Start the keep-alive routine if enabled and the connection is established successfully
	// this is for the initial connection only, the keep-alive will be restarted during the reconnect
	// logic/token refresh.
	if s.keepAlive != nil {
		heartBeatCancel = s.keepAlive.heartBeat(ctx, c, s.now().In(time.UTC))
	}

	// ensures this is the last connection closed when the function returns
	defer func() {
		if c != nil {
			if err := c.Close(); err != nil {
				s.metrics.errorsTotal.Inc()
				s.log.Errorw("encountered an error while closing the websocket connection", "error", err)
			}
		}
	}()

	for {
		select {
		// If the keep-alive is enabled, heartbeat will be automatically cancelled when
		// the parent context is done().
		case <-ctx.Done():
			s.log.Debugw("context cancelled, closing websocket connection")
			return ctx.Err()
		// s.tokenExpiry channel will only trigger if oauth2 is enabled and the token is about to expire
		case <-s.tokenExpiry:
			// Cancel the keep-alive routine if enabled since we need to establish a new
			// connection instance with the refreshed token.
			if s.keepAlive != nil {
				heartBeatCancel()
			}
			// get the new token
			token, err := s.tokenSource.Token()
			if err != nil {
				s.metrics.errorsTotal.Inc()
				s.log.Errorw("failed to obtain oauth2 token during token refresh", "error", err)
				return err
			}
			// gracefully close current connection
			if c != nil {
				if err := c.Close(); err != nil {
					s.metrics.errorsTotal.Inc()
					s.log.Errorw("encountered an error while closing the existing websocket connection during token refresh", "error", err)
				}
			}
			// set the new token in the config
			s.cfg.Auth.OAuth2.accessToken = token.AccessToken
			// set the new token expiry channel with 2 mins buffer
			s.tokenExpiry = time.After(time.Until(token.Expiry) - s.cfg.Auth.OAuth2.TokenExpiryBuffer)
			// establish a new connection with the new token
			c, resp, err = connectWebSocket(ctx, s.cfg, url, s.log)
			handleConnectionResponse(resp, s.metrics, s.log)
			if err != nil {
				s.metrics.errorsTotal.Inc()
				s.log.Errorw("failed to establish a new websocket connection on token refresh", "error", err)
				return err
			}
			// Restart the keep-alive routine on a token refresh if enabled and the
			// connection is established successfully.
			if s.keepAlive != nil {
				heartBeatCancel = s.keepAlive.heartBeat(ctx, c, s.now().In(time.UTC))
			}
		default:
			_, message, err := c.ReadMessage()
			if err != nil {
				s.metrics.errorsTotal.Inc()
				// Cancel the keep-alive routine if enabled since we need to establish a
				// new connection via our reconnect logic.
				if s.keepAlive != nil {
					heartBeatCancel()
				}
				if !s.cfg.Retry.BlanketRetries && !isRetryableError(err) {
					s.log.Errorw("failed to read websocket data", "error", err)
					return err
				}
				s.log.Debugw("websocket connection encountered an error, attempting to reconnect...", "error", err)
				if c != nil {
					if err := c.Close(); err != nil {
						s.metrics.errorsTotal.Inc()
						s.log.Errorw("encountered an error while closing the websocket connection", "error", err)
					}
				}
				// Since c is already a pointer, we can reassign it to the new connection
				// and the defer func will still handle it.
				c, resp, err = connectWebSocket(ctx, s.cfg, url, s.log)
				handleConnectionResponse(resp, s.metrics, s.log)
				if err != nil {
					s.metrics.errorsTotal.Inc()
					s.log.Errorw("failed to reconnect websocket connection", "error", err)
					return err
				}
				// Restart the keep-alive routine if enabled after a successful reconnection.
				if s.keepAlive != nil {
					heartBeatCancel = s.keepAlive.heartBeat(ctx, c, s.now().In(time.UTC))
				}

				continue
			}

			s.metrics.receivedBytesTotal.Add(uint64(len(message)))
			state["response"] = message
			s.log.Debugw("received websocket message", logp.Namespace(s.ns), "msg", string(message))
			err = s.process(ctx, state, s.cursor, s.now().In(time.UTC))
			if err != nil {
				s.metrics.errorsTotal.Inc()
				s.log.Errorw("failed to process and publish data", "error", err)
				return err
			}
		}
	}
}

// isRetryableError checks if the error is retryable based on the error type.
func isRetryableError(err error) bool {
	// check for specific network errors
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		switch {
		case netErr.Op == "dial" && netErr.Err.Error() == "i/o timeout",
			netErr.Op == "read" && netErr.Err.Error() == "i/o timeout",
			netErr.Op == "read" && netErr.Err.Error() == "connection reset by peer",
			netErr.Op == "read" && netErr.Err.Error() == "connection refused",
			netErr.Op == "read" && netErr.Err.Error() == "connection reset",
			netErr.Op == "read" && netErr.Err.Error() == "connection closed":
			return true
		}
	}

	// check for specific websocket close errors
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		switch closeErr.Code {
		case websocket.CloseGoingAway,
			websocket.CloseNormalClosure,
			websocket.CloseInternalServerErr,
			websocket.CloseTryAgainLater,
			websocket.CloseServiceRestart,
			websocket.CloseAbnormalClosure,
			websocket.CloseMessageTooBig,
			websocket.CloseNoStatusReceived,
			websocket.CloseTLSHandshake:
			return true
		}
	}

	// check for common error patterns
	if strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "temporary failure") ||
		strings.Contains(err.Error(), "server is busy") {
		return true
	}

	return false
}

// handleConnectionResponse logs the response body of the websocket connection.
func handleConnectionResponse(resp *http.Response, metrics *inputMetrics, log *logp.Logger) {
	if resp != nil && resp.Body != nil {
		var buf bytes.Buffer
		defer resp.Body.Close()

		if log.Core().Enabled(zapcore.DebugLevel) {
			const limit = 1e4
			if _, err := io.CopyN(&buf, resp.Body, limit); err != nil && !errors.Is(err, io.EOF) {
				metrics.errorsTotal.Inc()
				fmt.Fprintf(&buf, "failed to read websocket response body with error: (%s) \n", err)
			}
		}

		// discard the remaining part of the body and check for truncation.
		if n, err := io.Copy(io.Discard, resp.Body); err != nil {
			metrics.errorsTotal.Inc()
			fmt.Fprintf(&buf, "failed to discard remaining response body with error: (%s) ", err)
		} else if n != 0 && buf.Len() != 0 {
			buf.WriteString("... truncated")
		}

		log.Debugw("websocket connection response", "http.response.body.content", &buf)
	}
}

// connectWebSocket attempts to connect to the websocket server with exponential backoff if retry config is available else it connects without retry.
func connectWebSocket(ctx context.Context, cfg config, url string, log *logp.Logger) (*websocket.Conn, *http.Response, error) {
	var conn *websocket.Conn
	var response *http.Response
	var err error
	headers := formHeader(cfg)
	dialer, err := createWebSocketDialer(cfg)
	if err != nil {
		return nil, nil, err
	}
	if cfg.Retry != nil {
		retryConfig := cfg.Retry
		if !retryConfig.InfiniteRetries {
			for attempt := 1; attempt <= retryConfig.MaxAttempts; attempt++ {
				conn, response, err = dialer.DialContext(ctx, url, headers)
				if err == nil {
					return conn, response, nil
				}
				// in case of sudden network errors or server crashes the response will be nil and logging should be done gracefully
				if response != nil {
					//nolint:errorlint // it will never be a wrapped error at this point
					if err == websocket.ErrBadHandshake {
						log.Errorf("attempt %d: webSocket connection failed with bad handshake (status %d) retrying...\n", attempt, response.StatusCode)
					} else {
						log.Errorf("attempt %d: webSocket connection failed with error %v and (status %d), retrying...\n", attempt, err, response.StatusCode)
					}
				} else {
					log.Errorf("attempt %d: webSocket connection failed with error %v and no response, retrying...\n", attempt, err)
				}
				waitTime := calculateWaitTime(retryConfig.WaitMin, retryConfig.WaitMax, attempt)
				time.Sleep(waitTime)
			}
			return nil, nil, fmt.Errorf("failed to establish WebSocket connection after %d attempts with error %w and (status %d)", retryConfig.MaxAttempts, err, response.StatusCode)
		} else {
			for attempt := 1; ; attempt++ {
				conn, response, err = dialer.DialContext(ctx, url, headers)
				if err == nil {
					return conn, response, nil
				}
				// in case of sudden network errors or server crashes the response will be nil and logging should be done gracefully
				if response != nil {
					//nolint:errorlint // it will never be a wrapped error at this point
					if err == websocket.ErrBadHandshake {
						log.Errorf("attempt %d: webSocket connection failed with bad handshake (status %d) retrying...\n", attempt, response.StatusCode)
					} else {
						log.Errorf("attempt %d: webSocket connection failed with error %v and (status %d), retrying...\n", attempt, err, response.StatusCode)
					}
				} else {
					log.Errorf("attempt %d: webSocket connection failed with error %v and no response, retrying...\n", attempt, err)
				}
				waitTime := calculateWaitTime(retryConfig.WaitMin, retryConfig.WaitMax, attempt)
				time.Sleep(waitTime)
			}
		}
	}

	return dialer.DialContext(ctx, url, headers)
}

// calculateWaitTime calculates the wait time for the next attempt based on the exponential backoff algorithm.
func calculateWaitTime(waitMin, waitMax time.Duration, attempt int) time.Duration {
	// calculate exponential backoff
	base := float64(waitMin)
	backoff := base * math.Pow(2, float64(attempt-1))

	// calculate jitter proportional to the backoff
	maxJitter := float64(waitMax-waitMin) * math.Pow(2, float64(attempt-1))
	jitter := rand.Float64() * maxJitter

	waitTime := time.Duration(backoff + jitter)
	// caps the wait time to the maximum wait time
	if waitTime > waitMax {
		waitTime = waitMax
	}

	return waitTime
}

// now is time.Now with a modifiable time source.
func (s *websocketStream) now() time.Time {
	if s.time == nil {
		return time.Now()
	}
	return s.time()
}

func (s *websocketStream) Close() error {
	s.metrics.Close()
	return nil
}

func createWebSocketDialer(cfg config) (*websocket.Dialer, error) {
	var tlsConfig *tls.Config
	dialer := &websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
	}

	// load proxy configuration if available
	if cfg.Transport.Proxy.URL != nil {
		var proxy func(*http.Request) (*url.URL, error)
		proxyURL, err := httpcommon.NewProxyURIFromString(cfg.Transport.Proxy.URL.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}
		// create a custom HTTP Transport with proxy configuration
		proxyTransport := &http.Transport{
			Proxy:              http.ProxyURL(proxyURL.URI()),
			ProxyConnectHeader: cfg.Transport.Proxy.Headers.Headers(),
			DialContext: (&net.Dialer{
				Timeout: cfg.Transport.Timeout,
			}).DialContext,
		}
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return proxyTransport.DialContext(ctx, network, addr)
		}
		dialer.Proxy = proxy
	}
	// load TLS config if available
	if cfg.Transport.TLS != nil {
		TLSConfig, err := tlscommon.LoadTLSConfig(cfg.Transport.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		tlsConfig = TLSConfig.ToConfig()
		dialer.TLSClientConfig = tlsConfig
	}

	return dialer, nil
}
