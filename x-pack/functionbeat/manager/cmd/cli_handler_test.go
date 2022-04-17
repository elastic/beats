// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/menderesk/beats/v7/x-pack/functionbeat/function/provider"
)

type mockCLIManager struct {
	mock.Mock
}

func (m *mockCLIManager) Deploy(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *mockCLIManager) Update(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *mockCLIManager) Remove(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *mockCLIManager) Export(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *mockCLIManager) Package(outputPattern string) error {
	args := m.Called(outputPattern)
	return args.Error(0)
}

func outputs() (io.Writer, io.Writer) {
	errOut := new(bytes.Buffer)
	output := new(bytes.Buffer)
	return errOut, output
}

func functionByProvider() map[string]string {
	return map[string]string{
		"super":    "mockProvider",
		"saiyajin": "mockProvider",
	}
}

func wrapCLIManager(m provider.CLIManager) map[string]provider.CLIManager {
	return map[string]provider.CLIManager{
		"mockProvider": m,
	}
}

func TestCliHandler(t *testing.T) {
	t.Run("deploy", testDeploy)
	t.Run("update", testUpdate)
	t.Run("remove", testRemove)
}

func testDeploy(t *testing.T) {
	t.Run("return error when no functions are specified", func(t *testing.T) {
		errOut, output := outputs()
		handler := newCLIHandler(wrapCLIManager(&mockCLIManager{}), functionByProvider(), errOut, output)
		err := handler.Deploy([]string{})
		assert.Equal(t, errNoFunctionGiven, err)
	})

	t.Run("return an error if the manager return an error", func(t *testing.T) {
		errOut, output := outputs()
		myErr := errors.New("my error")
		m := &mockCLIManager{}
		m.On("Deploy", "saiyajin").Return(myErr)
		handler := newCLIHandler(wrapCLIManager(m), functionByProvider(), errOut, output)
		err := handler.Deploy([]string{"saiyajin"})
		assert.Error(t, err)
	})

	t.Run("call the method for all the functions", func(t *testing.T) {
		errOut, output := outputs()
		m := &mockCLIManager{}
		m.On("Deploy", "super").Return(nil)
		m.On("Deploy", "saiyajin").Return(nil)
		handler := newCLIHandler(wrapCLIManager(m), functionByProvider(), errOut, output)
		err := handler.Deploy([]string{"super", "saiyajin"})
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})
}

func testUpdate(t *testing.T) {
	t.Run("return error when no functions are specified", func(t *testing.T) {
		errOut, output := outputs()
		handler := newCLIHandler(wrapCLIManager(&mockCLIManager{}), functionByProvider(), errOut, output)
		err := handler.Update([]string{})
		assert.Equal(t, errNoFunctionGiven, err)
	})

	t.Run("return an error if the manager return an error", func(t *testing.T) {
		errOut, output := outputs()
		myErr := errors.New("my error")
		m := &mockCLIManager{}
		m.On("Update", "saiyajin").Return(myErr)
		handler := newCLIHandler(wrapCLIManager(m), functionByProvider(), errOut, output)
		err := handler.Update([]string{"saiyajin"})
		assert.Error(t, err)
	})

	t.Run("call the method for all the functions", func(t *testing.T) {
		errOut, output := outputs()
		m := &mockCLIManager{}
		m.On("Update", "super").Return(nil)
		m.On("Update", "saiyajin").Return(nil)
		handler := newCLIHandler(wrapCLIManager(m), functionByProvider(), errOut, output)
		err := handler.Update([]string{"super", "saiyajin"})
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})
}

func testRemove(t *testing.T) {
	t.Run("return error when no functions are specified", func(t *testing.T) {
		errOut, output := outputs()
		handler := newCLIHandler(wrapCLIManager(&mockCLIManager{}), functionByProvider(), errOut, output)
		err := handler.Remove([]string{})
		assert.Equal(t, errNoFunctionGiven, err)
	})

	t.Run("return an error if the manager return an error", func(t *testing.T) {
		errOut, output := outputs()
		myErr := errors.New("my error")
		m := &mockCLIManager{}
		m.On("Remove", "saiyajin").Return(myErr)
		handler := newCLIHandler(wrapCLIManager(m), functionByProvider(), errOut, output)
		err := handler.Remove([]string{"saiyajin"})
		assert.Error(t, err)
	})

	t.Run("call the method for all the functions", func(t *testing.T) {
		errOut, output := outputs()
		m := &mockCLIManager{}
		m.On("Remove", "super").Return(nil)
		m.On("Remove", "saiyajin").Return(nil)
		handler := newCLIHandler(wrapCLIManager(m), functionByProvider(), errOut, output)
		err := handler.Remove([]string{"super", "saiyajin"})
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})
}
