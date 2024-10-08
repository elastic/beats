// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/mito/lib"
)

const headerContentEncoding = "Content-Encoding"

var (
	errBodyEmpty       = errors.New("body cannot be empty")
	errUnsupportedType = errors.New("only JSON objects are accepted")
	errNotCRC          = errors.New("event not processed as CRC request")
)

type handler struct {
	ctx context.Context

	metrics     *inputMetrics
	publish     func(beat.Event)
	log         *logp.Logger
	validator   apiValidator
	txBaseID    string        // Random value to make transaction IDs unique.
	txIDCounter atomic.Uint64 // Transaction ID counter that is incremented for each request.

	reqLogger    *zap.Logger
	host, scheme string

	program               *program
	messageField          string
	responseCode          int
	responseBody          string
	includeHeaders        []string
	preserveOriginalEvent bool
	crc                   *crcValidator
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	txID := h.nextTxID()
	h.log.Debugw("request", "url", r.URL, "tx_id", txID)
	status, err := h.validator.validateRequest(r)
	if err != nil {
		h.sendAPIErrorResponse(txID, w, r, h.log, status, err)
		return
	}

	wait, err := getTimeoutWait(r.URL, h.log)
	if err != nil {
		h.sendAPIErrorResponse(txID, w, r, h.log, http.StatusBadRequest, err)
		return
	}
	var (
		acked   chan struct{}
		timeout *time.Timer
	)
	if wait != 0 {
		acked = make(chan struct{})
		timeout = time.NewTimer(wait)
	}
	start := time.Now()
	acker := newBatchACKTracker(func() {
		h.metrics.batchACKTime.Update(time.Since(start).Nanoseconds())
		h.metrics.batchesACKedTotal.Inc()
		if acked != nil {
			close(acked)
		}
	})
	h.metrics.batchesReceived.Add(1)
	h.metrics.contentLength.Update(r.ContentLength)
	body, status, err := getBodyReader(r)
	if err != nil {
		h.sendAPIErrorResponse(txID, w, r, h.log, status, err)
		h.metrics.apiErrors.Add(1)
		return
	}
	defer body.Close()

	if h.reqLogger != nil {
		// If we are logging, keep a copy of the body for the logger.
		// This is stashed in the r.Body field. This is only safe
		// because we are closing the original body in a defer and
		// r.Body is not otherwise referenced by the non-logging logic
		// after the call to getBodyReader above.
		var buf bytes.Buffer
		body = io.NopCloser(io.TeeReader(body, &buf))
		r.Body = io.NopCloser(&buf)
	}

	objs, _, status, err := httpReadJSON(body, h.program)
	if err != nil {
		h.sendAPIErrorResponse(txID, w, r, h.log, status, err)
		h.metrics.apiErrors.Add(1)
		return
	}

	var headers map[string]interface{}
	if len(h.includeHeaders) != 0 {
		headers = getIncludedHeaders(r, h.includeHeaders)
	}

	var (
		respCode int
		respBody string
	)

	h.metrics.batchSize.Update(int64(len(objs)))
	for _, obj := range objs {
		var err error
		if h.crc != nil {
			respCode, respBody, err = h.crc.validate(obj)
			if err == nil {
				// CRC request processed
				break
			} else if !errors.Is(err, errNotCRC) {
				h.metrics.apiErrors.Add(1)
				h.sendAPIErrorResponse(txID, w, r, h.log, http.StatusBadRequest, err)
				return
			}
		}

		acker.Add()
		if err = h.publishEvent(obj, headers, acker); err != nil {
			h.metrics.apiErrors.Add(1)
			h.sendAPIErrorResponse(txID, w, r, h.log, http.StatusInternalServerError, err)
			return
		}
		h.metrics.eventsPublished.Add(1)
		respCode, respBody = h.responseCode, h.responseBody
	}

	acker.Ready()
	if acked == nil {
		h.sendResponse(w, respCode, respBody)
	} else {
		select {
		case <-acked:
			h.log.Debugw("request acked", "tx_id", txID)
			if !timeout.Stop() {
				<-timeout.C
			}
			h.sendResponse(w, respCode, respBody)
		case <-timeout.C:
			h.log.Debugw("request timed out", "tx_id", txID)
			h.sendAPIErrorResponse(txID, w, r, h.log, http.StatusGatewayTimeout, errTookTooLong)
		case <-h.ctx.Done():
			h.log.Debugw("request context cancelled", "tx_id", txID)
			h.sendAPIErrorResponse(txID, w, r, h.log, http.StatusGatewayTimeout, h.ctx.Err())
		}
		if h.reqLogger != nil {
			h.logRequest(txID, r, respCode, nil)
		}
	}
	h.metrics.batchProcessingTime.Update(time.Since(start).Nanoseconds())
	h.metrics.batchesPublished.Add(1)
}

var errTookTooLong = errors.New("could not publish event within timeout")

func getTimeoutWait(u *url.URL, log *logp.Logger) (time.Duration, error) {
	q := u.Query()
	switch len(q) {
	case 0:
		return 0, nil
	case 1:
		if _, ok := q["wait_for_completion_timeout"]; !ok {
			// Get the only key in q. We don't know what it is, so iterate
			// over the first one of one.
			var k string
			for k = range q {
				break
			}
			return 0, fmt.Errorf("unexpected URL query: %s", k)
		}
	default:
		delete(q, "wait_for_completion_timeout")
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return 0, fmt.Errorf("unexpected URL query: %s", strings.Join(keys, ", "))
	}
	p := q.Get("wait_for_completion_timeout")
	if p == "" {
		// This will never happen; it is already handled in the check switch above.
		return 0, nil
	}
	log.Debugw("wait_for_completion_timeout parameter", "value", p)
	t, err := time.ParseDuration(p)
	if err != nil {
		return 0, fmt.Errorf("could not parse wait_for_completion_timeout parameter: %w", err)
	}
	if t < 0 {
		return 0, fmt.Errorf("negative wait_for_completion_timeout parameter: %w", err)
	}
	return t, nil
}

func (h *handler) sendAPIErrorResponse(txID string, w http.ResponseWriter, r *http.Request, log *logp.Logger, status int, apiError error) {
	log.Errorw("request error", "tx_id", txID, "status_code", status, "error", apiError)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	var (
		mw  io.Writer = w
		buf bytes.Buffer
	)
	if h.reqLogger != nil {
		mw = io.MultiWriter(mw, &buf)
	}
	enc := json.NewEncoder(mw)
	enc.SetEscapeHTML(false)
	err := enc.Encode(map[string]interface{}{"message": apiError.Error()})
	if err != nil {
		log.Debugw("Failed to write HTTP response.", "error", err, "client.address", r.RemoteAddr)
	}
	if h.reqLogger != nil {
		h.logRequest(txID, r, status, buf.Bytes())
	}
}

func (h *handler) logRequest(txID string, r *http.Request, status int, respBody []byte) {
	// Populate and preserve scheme and host if they are missing;
	// they are required for httputil.DumpRequestOut.
	var scheme, host string
	if r.URL.Scheme == "" {
		scheme = r.URL.Scheme
		r.URL.Scheme = h.scheme
	}
	if r.URL.Host == "" {
		host = r.URL.Host
		r.URL.Host = h.host
	}
	extra := make([]zapcore.Field, 1, 4)
	extra[0] = zap.Int("status", status)
	addr, port, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		extra = append(extra,
			zap.String("source.ip", addr),
			zap.String("source.port", port),
		)
	}
	if len(respBody) != 0 {
		extra = append(extra,
			zap.ByteString("http.response.body.content", respBody),
		)
	}
	h.log.Debugw("new request trace transaction", "id", txID)
	// Limit request logging body size to 10kiB.
	const maxBodyLen = 10 * (1 << 10)
	httplog.LogRequest(h.reqLogger.With(zap.String("transaction.id", txID)), r, maxBodyLen, extra...)
	if scheme != "" {
		r.URL.Scheme = scheme
	}
	if host != "" {
		r.URL.Host = host
	}
}

func (h *handler) nextTxID() string {
	count := h.txIDCounter.Add(1)
	return h.formatTxID(count)
}

func (h *handler) formatTxID(count uint64) string {
	return h.txBaseID + "-" + strconv.FormatUint(count, 10)
}

func (h *handler) sendResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, message); err != nil {
		h.log.Debugw("Failed writing response to client.", "error", err)
	}
}

func (h *handler) publishEvent(obj, headers mapstr.M, acker *batchACKTracker) error {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Private:   acker,
	}
	if h.messageField == "." {
		event.Fields = obj
	} else {
		if _, err := event.PutValue(h.messageField, obj); err != nil {
			return fmt.Errorf("failed to put data into event key %q: %w", h.messageField, err)
		}
	}
	if h.preserveOriginalEvent {
		event.Fields["event"] = mapstr.M{
			"original": obj.String(),
		}
	}
	if len(headers) > 0 {
		event.Fields["headers"] = headers
	}

	h.publish(event)
	return nil
}

func httpReadJSON(body io.Reader, prg *program) (objs []mapstr.M, rawMessages []json.RawMessage, status int, err error) {
	if body == http.NoBody {
		return nil, nil, http.StatusNotAcceptable, errBodyEmpty
	}
	obj, rawMessage, err := decodeJSON(body, prg)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}
	return obj, rawMessage, http.StatusOK, err
}

func decodeJSON(body io.Reader, prg *program) (objs []mapstr.M, rawMessages []json.RawMessage, err error) {
	decoder := json.NewDecoder(body)
	for decoder.More() {
		var raw json.RawMessage
		if err = decoder.Decode(&raw); err != nil {
			if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
				break
			}
			return nil, nil, fmt.Errorf("malformed JSON object at stream position %d: %w", decoder.InputOffset(), err)
		}

		var obj interface{}
		if err = newJSONDecoder(bytes.NewReader(raw)).Decode(&obj); err != nil {
			return nil, nil, fmt.Errorf("malformed JSON object at stream position %d: %w", decoder.InputOffset(), err)
		}

		if prg != nil {
			obj, err = prg.eval(obj)
			if err != nil {
				return nil, nil, err
			}
			// Re-marshal to ensure the raw bytes agree with the constructed object.
			raw, err = json.Marshal(obj)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to remarshal object: %w", err)
			}
		}

		switch v := obj.(type) {
		case map[string]interface{}:
			objs = append(objs, v)
			rawMessages = append(rawMessages, raw)
		case []interface{}:
			nobjs, nrawMessages, err := decodeJSONArray(bytes.NewReader(raw))
			if err != nil {
				return nil, nil, fmt.Errorf("recursive error %d: %w", decoder.InputOffset(), err)
			}
			objs = append(objs, nobjs...)
			rawMessages = append(rawMessages, nrawMessages...)
		default:
			return nil, nil, fmt.Errorf("%w: %T", errUnsupportedType, v)
		}
	}
	for i := range objs {
		jsontransform.TransformNumbers(objs[i])
	}
	return objs, rawMessages, nil
}

type program struct {
	prg cel.Program
	ast *cel.Ast
}

func newProgram(src string, log *logp.Logger) (*program, error) {
	if src == "" {
		return nil, nil
	}

	registry, err := types.NewRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to create env: %w", err)
	}
	env, err := cel.NewEnv(
		cel.Declarations(decls.NewVar("obj", decls.Dyn)),
		cel.OptionalTypes(cel.OptionalTypesVersion(lib.OptionalTypesVersion)),
		cel.CustomTypeAdapter(&numberAdapter{registry}),
		cel.CustomTypeProvider(registry),
		lib.Debug(debug(log)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create env: %w", err)
	}

	ast, iss := env.Compile(src)
	if iss.Err() != nil {
		return nil, fmt.Errorf("failed compilation: %w", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed program instantiation: %w", err)
	}
	return &program{prg: prg, ast: ast}, nil
}

func debug(log *logp.Logger) func(string, any) {
	log = log.Named("http_endpoint_cel_debug")
	return func(tag string, value any) {
		level := "DEBUG"
		if _, ok := value.(error); ok {
			level = "ERROR"
		}
		log.Debugw(level, "tag", tag, "value", value)
	}
}

var _ types.Adapter = (*numberAdapter)(nil)

type numberAdapter struct {
	fallback types.Adapter
}

func (a *numberAdapter) NativeToValue(value any) ref.Val {
	switch value := value.(type) {
	case []any:
		for i, v := range value {
			value[i] = a.NativeToValue(v)
		}
	case map[string]any:
		for k, v := range value {
			value[k] = a.NativeToValue(v)
		}
	case json.Number:
		var errs []error
		i, err := value.Int64()
		if err == nil {
			return types.Int(i)
		}
		errs = append(errs, err)
		f, err := value.Float64()
		if err == nil {
			// Literalise floats that could have been an integer greater than
			// can be stored without loss of precision in a double.
			// This is any integer wider than the IEEE-754 double mantissa.
			// As a heuristic, allow anything that includes a decimal point
			// or uses scientific notation. We could be more careful, but
			// it is likely not important, and other languages use the same
			// rule.
			if f >= 0x1p53 && !strings.ContainsFunc(string(value), func(r rune) bool {
				return r == '.' || r == 'e' || r == 'E'
			}) {
				return types.String(value)
			}
			return types.Double(f)
		}
		errs = append(errs, err)
		return types.NewErr("%v", errors.Join(errs...))
	}
	return a.fallback.NativeToValue(value)
}

func (p *program) eval(obj interface{}) (interface{}, error) {
	out, _, err := p.prg.Eval(map[string]interface{}{"obj": obj})
	if err != nil {
		err = lib.DecoratedError{AST: p.ast, Err: err}
		return nil, fmt.Errorf("failed eval: %w", err)
	}

	v, err := out.ConvertToNative(reflect.TypeOf((*structpb.Value)(nil)))
	if err != nil {
		return nil, fmt.Errorf("failed proto conversion: %w", err)
	}
	switch v := v.(type) {
	case *structpb.Value:
		return v.AsInterface(), nil
	default:
		// This should never happen.
		return nil, fmt.Errorf("unexpected native conversion type: %T", v)
	}
}

func decodeJSONArray(raw *bytes.Reader) (objs []mapstr.M, rawMessages []json.RawMessage, err error) {
	dec := newJSONDecoder(raw)
	token, err := dec.Token()
	if err != nil {
		if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed reading JSON array: %w", err)
	}
	if token != json.Delim('[') {
		return nil, nil, fmt.Errorf("malformed JSON array, not starting with delimiter [ at position: %d", dec.InputOffset())
	}

	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF { //nolint:errorlint // This will never be a wrapped error.
				break
			}
			return nil, nil, fmt.Errorf("malformed JSON object at stream position %d: %w", dec.InputOffset(), err)
		}

		var obj interface{}
		if err := newJSONDecoder(bytes.NewReader(raw)).Decode(&obj); err != nil {
			return nil, nil, fmt.Errorf("malformed JSON object at stream position %d: %w", dec.InputOffset(), err)
		}

		m, ok := obj.(map[string]interface{})
		if ok {
			rawMessages = append(rawMessages, raw)
			objs = append(objs, m)
		}
	}
	return objs, rawMessages, nil
}

func getIncludedHeaders(r *http.Request, headerConf []string) (includedHeaders mapstr.M) {
	includedHeaders = mapstr.M{}
	for _, header := range headerConf {
		if value, found := r.Header[header]; found {
			includedHeaders[common.DeDot(header)] = value
		}
	}
	return includedHeaders
}

func newJSONDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec
}

// getBodyReader returns a reader that decodes the specified Content-Encoding.
func getBodyReader(r *http.Request) (body io.ReadCloser, status int, err error) {
	switch enc := r.Header.Get(headerContentEncoding); enc {
	case "gzip", "x-gzip":
		gzipReader, err := newPooledGzipReader(r.Body)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzipReader, 0, nil
	case "":
		// No encoding.
		return r.Body, 0, nil
	default:
		return nil, http.StatusUnsupportedMediaType, fmt.Errorf("unsupported Content-Encoding type %q", enc)
	}
}
