// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

type mockCloudformationStack struct {
	respCreateStackOutput *cloudformation.CreateStackOutput
	onCreateStackInput    func(*cloudformation.CreateStackInput)

	respDeleteStackOutput *cloudformation.DeleteStackOutput
	onDeleteStackInput    func(*cloudformation.DeleteStackInput)

	respDescribeStacksOutput *cloudformation.DescribeStacksOutput
	onDescribeStacksInput    func(*cloudformation.DescribeStacksInput)

	respUpdateStackOutput *cloudformation.UpdateStackOutput
	onUpdateStackInput    func(*cloudformation.UpdateStackInput)
	err                   error
}

func (m *mockCloudformationStack) CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
	if m.onCreateStackInput != nil {
		m.onCreateStackInput(params)
	}

	if m.err != nil {
		return m.respCreateStackOutput, m.err
	}

	return m.respCreateStackOutput, nil
}

func (m *mockCloudformationStack) DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
	if m.onDeleteStackInput != nil {
		m.onDeleteStackInput(params)
	}

	if m.err != nil {
		return m.respDeleteStackOutput, m.err
	}

	return m.respDeleteStackOutput, nil
}

func (m *mockCloudformationStack) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if m.onDescribeStacksInput != nil {
		m.onDescribeStacksInput(params)
	}

	if m.err != nil {
		return m.respDescribeStacksOutput, m.err
	}

	return m.respDescribeStacksOutput, nil
}

func (m *mockCloudformationStack) UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
	if m.onUpdateStackInput != nil {
		m.onUpdateStackInput(params)
	}

	if m.err != nil {
		return m.respUpdateStackOutput, m.err
	}

	return m.respUpdateStackOutput, nil
}

func TestCreateStack(t *testing.T) {
	stackName := "new-stack"
	stackID := "new-stack-ID"
	templateURL := "https://localhost/stack.zip"
	log := logp.NewLogger("")

	t.Run("assert execution context", func(t *testing.T) {
		op := &opCreateCloudFormation{}
		err := op.Execute(struct{}{})
		assert.Error(t, err)
	})

	t.Run("create stack", func(t *testing.T) {
		mockSvc := &mockCloudformationStack{respCreateStackOutput: &cloudformation.CreateStackOutput{
			StackId: &stackID,
		}, onCreateStackInput: func(input *cloudformation.CreateStackInput) {
			assert.Equal(t, stackName, *input.StackName)
			assert.Equal(t, templateURL, *input.TemplateURL)
		}}

		ctx := &stackContext{}
		op := newOpCreateCloudFormation(log, mockSvc, templateURL, stackName)
		err := op.Execute(ctx)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, stackID, *ctx.ID)
	})

	t.Run("bubble any stack error back to the caller", func(t *testing.T) {
		anErr := errors.New("something is bad")
		mockSvc := &mockCloudformationStack{err: anErr}

		ctx := &stackContext{}
		op := newOpCreateCloudFormation(log, mockSvc, templateURL, stackName)
		err := op.Execute(ctx)
		assert.Equal(t, anErr, err)
	})
}

func TestDeleteStack(t *testing.T) {
	stackName := "new-stack"
	stackID := "new-stack-ID"
	log := logp.NewLogger("")

	t.Run("assert execution context", func(t *testing.T) {
		op := &opDeleteCloudFormation{}
		err := op.Execute(struct{}{})
		assert.Error(t, err)
	})

	t.Run("delete stack", func(t *testing.T) {
		mockSvc := &mockCloudformationStack{
			respDeleteStackOutput: &cloudformation.DeleteStackOutput{},
			onDeleteStackInput: func(
				input *cloudformation.DeleteStackInput,
			) {
				assert.Equal(t, stackName, *input.StackName)
			},
			respDescribeStacksOutput: &cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{types.Stack{StackId: &stackID}},
			},
		}
		ctx := &stackContext{}
		op := newOpDeleteCloudFormation(log, mockSvc, stackName)
		err := op.Execute(ctx)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, stackID, *ctx.ID)
	})

	t.Run("bubble any stack error back to the caller", func(t *testing.T) {
		anErr := errors.New("something is bad")
		mockSvc := &mockCloudformationStack{err: anErr}

		ctx := &stackContext{}
		op := newOpDeleteCloudFormation(log, mockSvc, stackName)
		err := op.Execute(ctx)
		assert.Equal(t, anErr, err)
	})
}

func TestUpdateStack(t *testing.T) {
	stackName := "new-stack"
	stackID := "new-stack-ID"
	templateURL := "https://localhost/stack.zip"
	log := logp.NewLogger("")

	t.Run("assert execution context", func(t *testing.T) {
		op := &opDeleteCloudFormation{}
		err := op.Execute(struct{}{})
		assert.Error(t, err)
	})

	t.Run("update stack", func(t *testing.T) {
		mockSvc := &mockCloudformationStack{
			onUpdateStackInput: func(
				input *cloudformation.UpdateStackInput,
			) {
				assert.Equal(t, stackName, *input.StackName)
				assert.Equal(t, templateURL, *input.TemplateURL)
			},
			respUpdateStackOutput: &cloudformation.UpdateStackOutput{
				StackId: &stackID,
			},
		}
		ctx := &stackContext{}
		op := newOpUpdateCloudFormation(log, mockSvc, templateURL, stackName)
		err := op.Execute(ctx)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, stackID, *ctx.ID)
	})

	t.Run("bubble any stack error back to the caller", func(t *testing.T) {
		anErr := errors.New("something is bad")
		mockSvc := &mockCloudformationStack{err: anErr}

		ctx := &stackContext{}
		op := newOpUpdateCloudFormation(log, mockSvc, templateURL, stackName)
		err := op.Execute(ctx)
		assert.Equal(t, anErr, err)
	})
}
