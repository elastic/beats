// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/types/known/structpb"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
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
	metrics   *inputMetrics
	publisher stateless.Publisher
	log       *logp.Logger
	validator apiValidator

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
	status, err := h.validator.validateRequest(r)
	if err != nil {
		h.sendAPIErrorResponse(w, r, h.log, status, err)
		return
	}

	start := time.Now()
	h.metrics.batchesReceived.Add(1)
	h.metrics.contentLength.Update(r.ContentLength)
	body, status, err := getBodyReader(r)
	if err != nil {
		h.sendAPIErrorResponse(w, r, h.log, status, err)
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
		h.sendAPIErrorResponse(w, r, h.log, status, err)
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
				h.sendAPIErrorResponse(w, r, h.log, http.StatusBadRequest, err)
				return
			}
		}

		if err = h.publishEvent(obj, headers); err != nil {
			h.metrics.apiErrors.Add(1)
			h.sendAPIErrorResponse(w, r, h.log, http.StatusInternalServerError, err)
			return
		}
		h.metrics.eventsPublished.Add(1)
		respCode, respBody = h.responseCode, h.responseBody
	}

	h.sendResponse(w, respCode, respBody)
	if h.reqLogger != nil {
		h.logRequest(r, respCode, nil)
	}
	h.metrics.batchProcessingTime.Update(time.Since(start).Nanoseconds())
	h.metrics.batchesPublished.Add(1)
}

func (h *handler) sendAPIErrorResponse(w http.ResponseWriter, r *http.Request, log *logp.Logger, status int, apiError error) {
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
		h.logRequest(r, status, buf.Bytes())
	}
}

func (h *handler) logRequest(r *http.Request, status int, respBody []byte) {
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
	// Limit request logging body size to 10kiB.
	const maxBodyLen = 10 * (1 << 10)
	httplog.LogRequest(h.reqLogger, r, maxBodyLen, extra...)
	if scheme != "" {
		r.URL.Scheme = scheme
	}
	if host != "" {
		r.URL.Host = host
	}
}

func (h *handler) sendResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, message); err != nil {
		h.log.Debugw("Failed writing response to client.", "error", err)
	}
}

func (h *handler) publishEvent(obj, headers mapstr.M) error {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields:    mapstr.M{},
	}
	if h.preserveOriginalEvent {
		event.Fields["event"] = mapstr.M{
			"original": obj.String(),
		}
	}
	if len(headers) > 0 {
		event.Fields["headers"] = headers
	}

	if _, err := event.PutValue(h.messageField, obj); err != nil {
		return fmt.Errorf("failed to put data into event key %q: %w", h.messageField, err)
	}

	h.publisher.Publish(event)
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
			return nil, nil, errUnsupportedType
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

func newProgram(src string) (*program, error) {
	if src == "" {
		return nil, nil
	}

	registry, err := types.NewRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to create env: %v", err)
	}
	env, err := cel.NewEnv(
		cel.Declarations(decls.NewVar("obj", decls.Dyn)),
		cel.OptionalTypes(cel.OptionalTypesVersion(lib.OptionalTypesVersion)),
		cel.CustomTypeAdapter(&numberAdapter{registry}),
		cel.CustomTypeProvider(registry),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create env: %v", err)
	}

	ast, iss := env.Compile(src)
	if iss.Err() != nil {
		return nil, fmt.Errorf("failed compilation: %v", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed program instantiation: %v", err)
	}
	return &program{prg: prg, ast: ast}, nil
}

var _ types.Adapter = (*numberAdapter)(nil)

type numberAdapter struct {
	fallback types.Adapter
}

func (a *numberAdapter) NativeToValue(value any) ref.Val {
	if n, ok := value.(json.Number); ok {
		var errs []error
		i, err := n.Int64()
		if err == nil {
			return types.Int(i)
		}
		errs = append(errs, err)
		f, err := n.Float64()
		if err == nil {
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

func errorMessage(msg string) map[string]interface{} {
	return map[string]interface{}{"error": map[string]interface{}{"message": msg}}
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
