// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package bundle

import (
	"time"

	"github.com/pkg/errors"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/metrics"
	"github.com/open-policy-agent/opa/server/types"
)

const (
	errCode = "bundle_error"
)

// Status represents the status of processing a bundle.
type Status struct {
	Name                     string          `json:"name"`
	ActiveRevision           string          `json:"active_revision,omitempty"`
	LastSuccessfulActivation time.Time       `json:"last_successful_activation,omitempty"`
	LastSuccessfulDownload   time.Time       `json:"last_successful_download,omitempty"`
	LastSuccessfulRequest    time.Time       `json:"last_successful_request,omitempty"`
	LastRequest              time.Time       `json:"last_request,omitempty"`
	Code                     string          `json:"code,omitempty"`
	Message                  string          `json:"message,omitempty"`
	Errors                   []error         `json:"errors,omitempty"`
	Metrics                  metrics.Metrics `json:"metrics,omitempty"`
}

// SetActivateSuccess updates the status object to reflect a successful
// activation.
func (s *Status) SetActivateSuccess(revision string) {
	s.LastSuccessfulActivation = time.Now().UTC()
	s.ActiveRevision = revision
}

// SetDownloadSuccess updates the status object to reflect a successful
// download.
func (s *Status) SetDownloadSuccess() {
	s.LastSuccessfulDownload = time.Now().UTC()
}

// SetRequest updates the status object to reflect a download attempt.
func (s *Status) SetRequest() {
	s.LastRequest = time.Now().UTC()
}

// SetError updates the status object to reflect a failure to download or
// activate. If err is nil, the error status is cleared.
func (s *Status) SetError(err error) {

	if err == nil {
		s.Code = ""
		s.Message = ""
		s.Errors = nil
		return
	}

	cause := errors.Cause(err)

	if astErr, ok := cause.(ast.Errors); ok {
		s.Code = errCode
		s.Message = types.MsgCompileModuleError
		s.Errors = make([]error, len(astErr))
		for i := range astErr {
			s.Errors[i] = astErr[i]
		}
	} else {
		s.Code = errCode
		s.Message = err.Error()
		s.Errors = nil
	}
}
