// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package executor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUndoer struct {
	mock.Mock
}

func (m *MockUndoer) Execute(_ Context) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockUndoer) Rollback(_ Context) error {
	args := m.Called()
	return args.Error(0)
}

type MockDoer struct {
	mock.Mock
}

func (m *MockDoer) Execute(_ Context) error {
	args := m.Called()
	return args.Error(0)
}

func TestExecutor(t *testing.T) {
	t.Run("executes all the tasks", testAll)
	t.Run("stop execution on first error", testError)
	t.Run("stop execution and allow rollback on undoer", testUndoer)
	t.Run("stop rollback if one rollback fail", testFailRollback)
	t.Run("an execution cannot be run twice", testCannotRunTwice)
	t.Run("cannot add operation to a completed execution", testCannotAddCompleted)
}

func testAll(t *testing.T) {
	ctx := struct{}{}
	executor := NewExecutor(nil)
	m1 := &MockDoer{}
	m1.On("Execute").Return(nil)

	m2 := &MockDoer{}
	m2.On("Execute").Return(nil)

	executor.Add(m1, m2)
	err := executor.Execute(ctx)
	if !assert.NoError(t, err) {
		return
	}

	m1.AssertExpectations(t)
	m2.AssertExpectations(t)
}

func testError(t *testing.T) {
	ctx := struct{}{}
	executor := NewExecutor(nil)
	m1 := &MockDoer{}
	m1.On("Execute").Return(nil)

	m2 := &MockDoer{}
	e := errors.New("something bad")
	m2.On("Execute").Return(e)

	m3 := &MockDoer{}
	executor.Add(m1, m2, m3)
	err := executor.Execute(ctx)
	if assert.Equal(t, e, err) {
		return
	}

	m1.AssertExpectations(t)
	m2.AssertExpectations(t)
	m3.AssertExpectations(t)
}

func testUndoer(t *testing.T) {
	ctx := struct{}{}
	executor := NewExecutor(nil)
	m1 := &MockUndoer{}
	m1.On("Execute").Return(nil)
	m1.On("Rollback").Return(nil)

	m2 := &MockDoer{}
	e := errors.New("something bad")
	m2.On("Execute").Return(e)

	m3 := &MockDoer{}
	executor.Add(m1, m2, m3)
	err := executor.Execute(ctx)
	if !assert.Equal(t, e, err) {
		return
	}

	err = executor.Rollback(ctx)
	if !assert.NoError(t, err) {
		return
	}

	m1.AssertExpectations(t)
	m2.AssertExpectations(t)
	m3.AssertExpectations(t)
}

func testFailRollback(t *testing.T) {
	ctx := struct{}{}
	e := errors.New("error on execution")
	e2 := errors.New("error on rollback")

	executor := NewExecutor(nil)
	m1 := &MockUndoer{}
	m1.On("Execute").Return(nil)

	m2 := &MockUndoer{}
	m2.On("Execute").Return(nil)
	m2.On("Rollback").Return(e2)

	m3 := &MockUndoer{}
	m3.On("Execute").Return(e)

	executor.Add(m1, m2, m3)

	err := executor.Execute(ctx)
	if !assert.Equal(t, e, err) {
		return
	}

	err = executor.Rollback(ctx)
	if !assert.Error(t, err) {
		return
	}

	m1.AssertExpectations(t)
	m2.AssertExpectations(t)
	m3.AssertExpectations(t)
}

func testCannotRunTwice(t *testing.T) {
	ctx := struct{}{}
	executor := NewExecutor(nil)
	m1 := &MockDoer{}
	m1.On("Execute").Return(nil)

	executor.Add(m1)
	err := executor.Execute(ctx)
	if !assert.NoError(t, err) {
		return
	}

	m1.AssertExpectations(t)

	assert.True(t, executor.IsCompleted())
	assert.Error(t, ErrAlreadyExecuted, executor.Execute(ctx))
}

func testCannotAddCompleted(t *testing.T) {
	executor := NewExecutor(nil)
	m1 := &MockDoer{}
	m1.On("Execute").Return(nil)

	executor.Add(m1)
	err := executor.Execute(struct{}{})
	if !assert.NoError(t, err) {
		return
	}

	m1.AssertExpectations(t)

	assert.Error(t, executor.Add(&MockDoer{}))
}
