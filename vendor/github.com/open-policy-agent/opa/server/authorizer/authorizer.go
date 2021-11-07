// Copyright 2017 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package authorizer provides authorization handlers to the server.
package authorizer

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/server/identifier"
	"github.com/open-policy-agent/opa/server/types"
	"github.com/open-policy-agent/opa/server/writer"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/util"
)

// Basic provides policy-based authorization over incoming requests.
type Basic struct {
	inner    http.Handler
	compiler func() *ast.Compiler
	store    storage.Store
	runtime  *ast.Term
	decision func() ast.Ref
}

// Runtime returns an argument that sets the runtime on the authorizer.
func Runtime(term *ast.Term) func(*Basic) {
	return func(b *Basic) {
		b.runtime = term
	}
}

// Decision returns an argument that sets the path of the authorization decision
// to query.
func Decision(ref func() ast.Ref) func(*Basic) {
	return func(b *Basic) {
		b.decision = ref
	}
}

// NewBasic returns a new Basic object.
func NewBasic(inner http.Handler, compiler func() *ast.Compiler, store storage.Store, opts ...func(*Basic)) http.Handler {
	b := &Basic{
		inner:    inner,
		compiler: compiler,
		store:    store,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

func (h *Basic) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// TODO(tsandall): Pass AST value as input instead of Go value to avoid unnecessary
	// conversions.
	r, input, err := makeInput(r)
	if err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	rego := rego.New(
		rego.Query(h.decision().String()),
		rego.Compiler(h.compiler()),
		rego.Store(h.store),
		rego.Input(input),
		rego.Runtime(h.runtime),
	)

	rs, err := rego.Eval(r.Context())

	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	if len(rs) == 0 {
		// Authorizer was configured but no policy defined. This indicates an internal error or misconfiguration.
		writer.Error(w, http.StatusInternalServerError, types.NewErrorV1(types.CodeInternal, types.MsgUnauthorizedUndefinedError))
		return
	}

	switch allowed := rs[0].Expressions[0].Value.(type) {
	case bool:
		if allowed {
			h.inner.ServeHTTP(w, r)
			return
		}
	case map[string]interface{}:
		if decision, ok := allowed["allowed"]; ok {
			if allow, ok := decision.(bool); ok && allow {
				h.inner.ServeHTTP(w, r)
				return
			}
			if reason, ok := allowed["reason"]; ok {
				message, ok := reason.(string)
				if ok {
					writer.Error(w, http.StatusUnauthorized, types.NewErrorV1(types.CodeUnauthorized, message))
					return
				}
			}
		} else {
			writer.Error(w, http.StatusInternalServerError, types.NewErrorV1(types.CodeInternal, types.MsgUndefinedError))
			return
		}
	}
	writer.Error(w, http.StatusUnauthorized, types.NewErrorV1(types.CodeUnauthorized, types.MsgUnauthorizedError))
}

func makeInput(r *http.Request) (*http.Request, interface{}, error) {

	path, err := parsePath(r.URL.Path)
	if err != nil {
		return r, nil, err
	}

	method := strings.ToUpper(r.Method)
	query := r.URL.Query()

	var rawBody []byte

	if expectBody(r.Method, path) {
		rawBody, err = readBody(r)
		if err != nil {
			return r, nil, err
		}
	}

	input := map[string]interface{}{
		"path":    path,
		"method":  method,
		"params":  query,
		"headers": r.Header,
	}

	if len(rawBody) > 0 {
		var body interface{}
		if expectYAML(r) {
			if err := util.Unmarshal(rawBody, &body); err != nil {
				return r, nil, err
			}
		} else if err := util.UnmarshalJSON(rawBody, &body); err != nil {
			return r, nil, err
		}

		// We cache the parsed body on the context so the server does not have
		// to parse the input document twice.
		input["body"] = body
		ctx := SetBodyOnContext(r.Context(), body)
		r = r.WithContext(ctx)
	}

	identity, ok := identifier.Identity(r)
	if ok {
		input["identity"] = identity
	}

	return r, input, nil
}

var dataAPIVersions = map[string]bool{
	"v0": true,
	"v1": true,
}

func expectBody(method string, path []interface{}) bool {
	if method == http.MethodPost {
		if len(path) == 1 {
			s := path[0].(string)
			return s == ""
		} else if len(path) >= 2 {
			s1 := path[0].(string)
			s2 := path[1].(string)
			return dataAPIVersions[s1] && s2 == "data"
		}
	}
	return false
}

func expectYAML(r *http.Request) bool {
	// NOTE(tsandall): This check comes from the server's HTTP handler code. The docs
	// are a bit more strict, but the authorizer should be consistent w/ the original
	// server handler implementation.
	return strings.Contains(r.Header.Get("Content-Type"), "yaml")
}

func readBody(r *http.Request) ([]byte, error) {

	bs, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return nil, err
	}

	return bs, nil
}

func parsePath(path string) ([]interface{}, error) {
	if len(path) == 0 {
		return []interface{}{}, nil
	}
	parts := strings.Split(path[1:], "/")
	for i := range parts {
		var err error
		parts[i], err = url.PathUnescape(parts[i])
		if err != nil {
			return nil, err
		}
	}
	sl := make([]interface{}, len(parts))
	for i := range sl {
		sl[i] = parts[i]
	}
	return sl, nil
}

type authorizerCachedBody struct {
	parsed interface{}
}

type authorizerCachedBodyKey string

const ctxkey authorizerCachedBodyKey = "authorizerCachedBodyKey"

// SetBodyOnContext adds the parsed input value to the context. This function is only
// exposed for test purposes.
func SetBodyOnContext(ctx context.Context, x interface{}) context.Context {
	return context.WithValue(ctx, ctxkey, authorizerCachedBody{
		parsed: x,
	})
}

// GetBodyOnContext returns the parsed input from the request context if it exists.
// The authorizer saves the parsed input on the context when it runs.
func GetBodyOnContext(ctx context.Context) (interface{}, bool) {
	input, ok := ctx.Value(ctxkey).(authorizerCachedBody)
	if !ok {
		return nil, false
	}
	return input.parsed, true
}
