// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package types contains request/response types and codes for the server.
package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/topdown"
	"github.com/open-policy-agent/opa/util"
)

// Error codes returned by OPA's REST API.
const (
	CodeInternal          = "internal_error"
	CodeEvaluation        = "evaluation_error"
	CodeUnauthorized      = "unauthorized"
	CodeInvalidParameter  = "invalid_parameter"
	CodeInvalidOperation  = "invalid_operation"
	CodeResourceNotFound  = "resource_not_found"
	CodeResourceConflict  = "resource_conflict"
	CodeUndefinedDocument = "undefined_document"
)

// ErrorV1 models an error response sent to the client.
type ErrorV1 struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Errors  []error `json:"errors,omitempty"`
}

// NewErrorV1 returns a new ErrorV1 object.
func NewErrorV1(code, f string, a ...interface{}) *ErrorV1 {
	return &ErrorV1{
		Code:    code,
		Message: fmt.Sprintf(f, a...),
	}
}

// This shall only used for debugging purpose.
func (e *ErrorV1) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WithError updates e to include a detailed error.
func (e *ErrorV1) WithError(err error) *ErrorV1 {
	e.Errors = append(e.Errors, err)
	return e
}

// WithASTErrors updates e to include detailed AST errors.
func (e *ErrorV1) WithASTErrors(errors []*ast.Error) *ErrorV1 {
	e.Errors = make([]error, len(errors))
	for i := range e.Errors {
		e.Errors[i] = errors[i]
	}
	return e
}

// Bytes marshals e with indentation for readability.
func (e *ErrorV1) Bytes() []byte {
	if bs, err := json.MarshalIndent(e, "", "  "); err == nil {
		return bs
	}
	return nil
}

// Messages included in error responses.
const (
	MsgCompileModuleError         = "error(s) occurred while compiling module(s)"
	MsgParseQueryError            = "error(s) occurred while parsing query"
	MsgCompileQueryError          = "error(s) occurred while compiling query"
	MsgEvaluationError            = "error(s) occurred while evaluating query"
	MsgUnauthorizedUndefinedError = "authorization policy missing or undefined"
	MsgUnauthorizedError          = "request rejected by administrative policy"
	MsgUndefinedError             = "document missing or undefined"
	MsgPluginConfigError          = "error(s) occurred while configuring plugin(s)"
)

// PatchV1 models a single patch operation against a document.
type PatchV1 struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// PolicyListResponseV1 models the response message for the Policy API list operation.
type PolicyListResponseV1 struct {
	Result []PolicyV1 `json:"result"`
}

// PolicyGetResponseV1 models the response message for the Policy API get operation.
type PolicyGetResponseV1 struct {
	Result PolicyV1 `json:"result"`
}

// PolicyPutResponseV1 models the response message for the Policy API put operation.
type PolicyPutResponseV1 struct {
	Metrics MetricsV1 `json:"metrics,omitempty"`
}

// PolicyDeleteResponseV1 models the response message for the Policy API delete operation.
type PolicyDeleteResponseV1 struct {
	Metrics MetricsV1 `json:"metrics,omitempty"`
}

// PolicyV1 models a policy module in OPA.
type PolicyV1 struct {
	ID  string      `json:"id"`
	Raw string      `json:"raw"`
	AST *ast.Module `json:"ast"`
}

// Equal returns true if p is equal to other.
func (p PolicyV1) Equal(other PolicyV1) bool {
	return p.ID == other.ID && p.Raw == other.Raw && p.AST.Equal(other.AST)
}

// ProvenanceV1 models a collection of build/version information.
type ProvenanceV1 struct {
	Version   string                        `json:"version"`
	Vcs       string                        `json:"build_commit"`
	Timestamp string                        `json:"build_timestamp"`
	Hostname  string                        `json:"build_hostname"`
	Revision  string                        `json:"revision,omitempty"` // Deprecated: Prefer `Bundles`
	Bundles   map[string]ProvenanceBundleV1 `json:"bundles,omitempty"`
}

// ProvenanceBundleV1 models a bundle at some point in time
type ProvenanceBundleV1 struct {
	Revision string `json:"revision"`
}

// DataRequestV1 models the request message for Data API POST operations.
type DataRequestV1 struct {
	Input *interface{} `json:"input"`
}

// DataResponseV1 models the response message for Data API read operations.
type DataResponseV1 struct {
	DecisionID  string        `json:"decision_id,omitempty"`
	Provenance  *ProvenanceV1 `json:"provenance,omitempty"`
	Explanation TraceV1       `json:"explanation,omitempty"`
	Metrics     MetricsV1     `json:"metrics,omitempty"`
	Result      *interface{}  `json:"result,omitempty"`
}

// MetricsV1 models a collection of performance metrics.
type MetricsV1 map[string]interface{}

// QueryResponseV1 models the response message for Query API operations.
type QueryResponseV1 struct {
	Explanation TraceV1               `json:"explanation,omitempty"`
	Metrics     MetricsV1             `json:"metrics,omitempty"`
	Result      AdhocQueryResultSetV1 `json:"result,omitempty"`
}

// AdhocQueryResultSetV1 models the result of a Query API query.
type AdhocQueryResultSetV1 []map[string]interface{}

// ExplainModeV1 defines supported values for the "explain" query parameter.
type ExplainModeV1 string

// Explanation mode enumeration.
const (
	ExplainOffV1   ExplainModeV1 = "off"
	ExplainFullV1  ExplainModeV1 = "full"
	ExplainNotesV1 ExplainModeV1 = "notes"
	ExplainFailsV1 ExplainModeV1 = "fails"
)

// TraceV1 models the trace result returned for queries that include the
// "explain" parameter.
type TraceV1 json.RawMessage

// MarshalJSON unmarshals the TraceV1 to a JSON representation.
func (t TraceV1) MarshalJSON() ([]byte, error) {
	return t, nil
}

// UnmarshalJSON unmarshals the TraceV1 from a JSON representation.
func (t *TraceV1) UnmarshalJSON(b []byte) error {
	*t = TraceV1(b[:])
	return nil
}

// TraceV1Raw models the trace result returned for queries that include the
// "explain" parameter. The trace is modelled as series of trace events that
// identify the expression, local term bindings, query hierarchy, etc.
type TraceV1Raw []TraceEventV1

// UnmarshalJSON unmarshals the TraceV1Raw from a JSON representation.
func (t *TraceV1Raw) UnmarshalJSON(b []byte) error {
	var trace []TraceEventV1
	if err := json.Unmarshal(b, &trace); err != nil {
		return err
	}
	*t = TraceV1Raw(trace)
	return nil
}

// TraceV1Pretty models the trace result returned for queries that include the "explain"
// parameter. The trace is modelled as a human readable array of strings representing the
// evaluation of the query.
type TraceV1Pretty []string

// UnmarshalJSON unmarshals the TraceV1Pretty from a JSON representation.
func (t *TraceV1Pretty) UnmarshalJSON(b []byte) error {
	var s []string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*t = TraceV1Pretty(s)
	return nil
}

// NewTraceV1 returns a new TraceV1 object.
func NewTraceV1(trace []*topdown.Event, pretty bool) (result TraceV1, err error) {
	if pretty {
		return newPrettyTraceV1(trace)
	}
	return newRawTraceV1(trace)
}

func newRawTraceV1(trace []*topdown.Event) (TraceV1, error) {
	result := TraceV1Raw(make([]TraceEventV1, len(trace)))
	for i := range trace {
		result[i] = TraceEventV1{
			Op:       strings.ToLower(string(trace[i].Op)),
			QueryID:  trace[i].QueryID,
			ParentID: trace[i].ParentID,
			Locals:   NewBindingsV1(trace[i].Locals),
			Message:  trace[i].Message,
		}
		if trace[i].Node != nil {
			result[i].Type = ast.TypeName(trace[i].Node)
			result[i].Node = trace[i].Node
		}
	}

	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return TraceV1(json.RawMessage(b)), nil
}

func newPrettyTraceV1(trace []*topdown.Event) (TraceV1, error) {
	var buf bytes.Buffer
	topdown.PrettyTraceWithLocation(&buf, trace)

	str := strings.Trim(buf.String(), "\n")
	b, err := json.Marshal(strings.Split(str, "\n"))
	if err != nil {
		return nil, err
	}
	return TraceV1(json.RawMessage(b)), nil
}

// TraceEventV1 represents a step in the query evaluation process.
type TraceEventV1 struct {
	Op       string      `json:"op"`
	QueryID  uint64      `json:"query_id"`
	ParentID uint64      `json:"parent_id"`
	Type     string      `json:"type"`
	Node     interface{} `json:"node"`
	Locals   BindingsV1  `json:"locals"`
	Message  string      `json:"message,omitempty"`
}

// UnmarshalJSON deserializes a TraceEventV1 object. The Node field is
// deserialized based on the type hint from the type property in the JSON
// object.
func (te *TraceEventV1) UnmarshalJSON(bs []byte) error {

	keys := map[string]json.RawMessage{}

	if err := util.UnmarshalJSON(bs, &keys); err != nil {
		return err
	}

	if err := util.UnmarshalJSON(keys["type"], &te.Type); err != nil {
		return err
	}

	if err := util.UnmarshalJSON(keys["op"], &te.Op); err != nil {
		return err
	}

	if err := util.UnmarshalJSON(keys["query_id"], &te.QueryID); err != nil {
		return err
	}

	if err := util.UnmarshalJSON(keys["parent_id"], &te.ParentID); err != nil {
		return err
	}

	switch te.Type {
	case "body":
		var body ast.Body
		if err := util.UnmarshalJSON(keys["node"], &body); err != nil {
			return err
		}
		te.Node = body
	case "expr":
		var expr ast.Expr
		if err := util.UnmarshalJSON(keys["node"], &expr); err != nil {
			return err
		}
		te.Node = &expr
	case "rule":
		var rule ast.Rule
		if err := util.UnmarshalJSON(keys["node"], &rule); err != nil {
			return err
		}
		te.Node = &rule
	}

	return util.UnmarshalJSON(keys["locals"], &te.Locals)
}

// BindingsV1 represents a set of term bindings.
type BindingsV1 []*BindingV1

// BindingV1 represents a single term binding.
type BindingV1 struct {
	Key   *ast.Term `json:"key"`
	Value *ast.Term `json:"value"`
}

// NewBindingsV1 returns a new BindingsV1 object.
func NewBindingsV1(locals *ast.ValueMap) (result []*BindingV1) {
	result = make([]*BindingV1, 0, locals.Len())
	locals.Iter(func(key, value ast.Value) bool {
		result = append(result, &BindingV1{
			Key:   &ast.Term{Value: key},
			Value: &ast.Term{Value: value},
		})
		return false
	})
	return result
}

// CompileRequestV1 models the request message for Compile API operations.
type CompileRequestV1 struct {
	Input    *interface{} `json:"input"`
	Query    string       `json:"query"`
	Unknowns *[]string    `json:"unknowns"`
}

// CompileResponseV1 models the response message for Compile API operations.
type CompileResponseV1 struct {
	Result      *interface{} `json:"result,omitempty"`
	Explanation TraceV1      `json:"explanation,omitempty"`
	Metrics     MetricsV1    `json:"metrics,omitempty"`
}

// PartialEvaluationResultV1 represents the output of partial evaluation and is
// included in Compile API responses.
type PartialEvaluationResultV1 struct {
	Queries []ast.Body    `json:"queries,omitempty"`
	Support []*ast.Module `json:"support,omitempty"`
}

// QueryRequestV1 models the request message for Query API operations.
type QueryRequestV1 struct {
	Query string       `json:"query"`
	Input *interface{} `json:"input"`
}

// ConfigResponseV1 models the response message for Config API operations.
type ConfigResponseV1 struct {
	Result *interface{} `json:"result,omitempty"`
}

// HealthResponseV1 models the response message for Health API operations.
type HealthResponseV1 struct {
	Error string `json:"error,omitempty"`
}

const (
	// ParamQueryV1 defines the name of the HTTP URL parameter that specifies
	// values for the request query.
	ParamQueryV1 = "q"

	// ParamInputV1 defines the name of the HTTP URL parameter that specifies
	// values for the "input" document.
	ParamInputV1 = "input"

	// ParamPrettyV1 defines the name of the HTTP URL parameter that indicates
	// the client wants to receive a pretty-printed version of the response.
	ParamPrettyV1 = "pretty"

	// ParamExplainV1 defines the name of the HTTP URL parameter that indicates the
	// client wants to receive explanations in addition to the result.
	ParamExplainV1 = "explain"

	// ParamMetricsV1 defines the name of the HTTP URL parameter that indicates
	// the client wants to receive performance metrics in addition to the
	// result.
	ParamMetricsV1 = "metrics"

	// ParamInstrumentV1 defines the name of the HTTP URL parameter that
	// indicates the client wants to receive instrumentation data for
	// diagnosing performance issues.
	ParamInstrumentV1 = "instrument"

	// ParamPartialV1 defines the name of the HTTP URL parameter that indicates
	// the client wants the partial evaluation optimization to be used during
	// query evaluation. This parameter is DEPRECATED.
	ParamPartialV1 = "partial"

	// ParamProvenanceV1 defines the name of the HTTP URL parameter that indicates
	// the client wants build and version information in addition to the result.
	ParamProvenanceV1 = "provenance"

	// ParamBundleActivationV1 defines the name of the HTTP URL parameter that
	// indicates the client wants to include bundle activation in the results
	// of the health API.
	// Deprecated: Use ParamBundlesActivationV1 instead.
	ParamBundleActivationV1 = "bundle"

	// ParamBundlesActivationV1 defines the name of the HTTP URL parameter that
	// indicates the client wants to include bundle activation in the results
	// of the health API.
	ParamBundlesActivationV1 = "bundles"

	// ParamPluginsV1 defines the name of the HTTP URL parameter that
	// indicates the client wants to include bundle status in the results
	// of the health API.
	ParamPluginsV1 = "plugins"

	// ParamExcludePluginV1 defines the name of the HTTP URL parameter that
	// indicates the client wants to exclude plugin status in the results
	// of the health API for the specified plugin(s)
	ParamExcludePluginV1 = "exclude-plugin"

	// ParamStrictBuiltinErrors names the HTTP URL parameter that indicates the client
	// wants built-in function errors to be treated as fatal.
	ParamStrictBuiltinErrors = "strict-builtin-errors"
)

// BadRequestErr represents an error condition raised if the caller passes
// invalid parameters.
type BadRequestErr string

// BadPatchOperationErr returns BadRequestErr indicating the patch operation was
// invalid.
func BadPatchOperationErr(op string) error {
	return BadRequestErr(fmt.Sprintf("bad patch operation: %v", op))
}

// BadPatchPathErr returns BadRequestErr indicating the patch path was invalid.
func BadPatchPathErr(path string) error {
	return BadRequestErr(fmt.Sprintf("bad patch path: %v", path))
}

func (err BadRequestErr) Error() string {
	return string(err)
}

// IsBadRequest returns true if err is a BadRequestErr.
func IsBadRequest(err error) bool {
	_, ok := err.(BadRequestErr)
	return ok
}
