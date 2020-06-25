// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apmtest

import (
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"go.elastic.co/apm"
	"go.elastic.co/apm/transport/transporttest"
)

// HTTPTestSuite is a test suite for HTTP instrumentation modules.
type HTTPTestSuite struct {
	suite.Suite

	// Handler holds an instrumented HTTP handler. Handler must
	// support the following routes:
	//
	//   GET /implicit_write (no explicit write on the response)
	//   GET /panic_before_write (panic without writing response)
	//   GET /panic_after_write (panic after writing response)
	//
	Handler http.Handler

	// Tracer is the apm.Tracer used to instrument Handler.
	//
	// HTTPTestSuite will close the tracer when all tests have
	// been completed.
	Tracer *apm.Tracer

	// Recorder is the transport used as the transport for Tracer.
	Recorder *transporttest.RecorderTransport

	server *httptest.Server
}

// SetupTest runs before each test.
func (s *HTTPTestSuite) SetupTest() {
	s.Recorder.ResetPayloads()
}

// SetupSuite runs before the tests in the suite are run.
func (s *HTTPTestSuite) SetupSuite() {
	s.server = httptest.NewServer(s.Handler)
}

// TearDownSuite runs after the tests in the suite are run.
func (s *HTTPTestSuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
	s.Tracer.Close()
}

// TestImplicitWrite tests the behaviour of instrumented handlers
// for routes which do not explicitly write a response, but instead
// leave it to the framework to write an empty 200 response.
func (s *HTTPTestSuite) TestImplicitWrite() {
	resp, err := http.Get(s.server.URL + "/implicit_write")
	require.NoError(s.T(), err)
	resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	s.Tracer.Flush(nil)
	ps := s.Recorder.Payloads()
	require.Len(s.T(), ps.Transactions, 1)

	tx := ps.Transactions[0]
	s.Equal("HTTP 2xx", tx.Result)
	s.Equal(resp.StatusCode, tx.Context.Response.StatusCode)
}

// TestPanicBeforeWrite tests the behaviour of instrumented handlers
// for routes which panic before any headers are written. The handler
// is expected to recover the panic and write an empty 500 response.
func (s *HTTPTestSuite) TestPanicBeforeWrite() {
	resp, err := http.Get(s.server.URL + "/panic_before_write")
	require.NoError(s.T(), err)
	resp.Body.Close()
	s.Equal(http.StatusInternalServerError, resp.StatusCode)

	s.Tracer.Flush(nil)
	ps := s.Recorder.Payloads()
	require.Len(s.T(), ps.Transactions, 1)
	require.Len(s.T(), ps.Errors, 1)

	tx := ps.Transactions[0]
	s.Equal("HTTP 5xx", tx.Result)
	s.Equal(resp.StatusCode, tx.Context.Response.StatusCode)

	e := ps.Errors[0]
	s.Equal(tx.ID, e.ParentID)
	s.Equal(resp.StatusCode, e.Context.Response.StatusCode)
}

// TestPanicAfterWrite tests the behaviour of instrumented handlers
// for routes which panic after writing headers. The handler is
// expected to recover the panic without otherwise affecting the
// response.
func (s *HTTPTestSuite) TestPanicAfterWrite() {
	resp, err := http.Get(s.server.URL + "/panic_after_write")
	require.NoError(s.T(), err)
	resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	s.Tracer.Flush(nil)
	ps := s.Recorder.Payloads()
	require.Len(s.T(), ps.Transactions, 1)
	require.Len(s.T(), ps.Errors, 1)

	tx := ps.Transactions[0]
	s.Equal("HTTP 2xx", tx.Result)
	s.Equal(resp.StatusCode, tx.Context.Response.StatusCode)

	e := ps.Errors[0]
	s.Equal(tx.ID, e.ParentID)
	s.Equal(resp.StatusCode, e.Context.Response.StatusCode)
}
