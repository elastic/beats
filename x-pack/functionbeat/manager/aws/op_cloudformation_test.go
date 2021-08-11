// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type mockCloudformationStack struct {
	cloudformationiface.ClientAPI

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

func (m *mockCloudformationStack) CreateStackRequest(
	input *cloudformation.CreateStackInput,
) cloudformation.CreateStackRequest {
	if m.onCreateStackInput != nil {
		m.onCreateStackInput(input)
	}

	httpReq, _ := http.NewRequest("", "", nil)
	if m.err != nil {
		return cloudformation.CreateStackRequest{
			Request: &aws.Request{Data: m.respCreateStackOutput, Error: m.err, HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
		}
	}

	return cloudformation.CreateStackRequest{
		Request: &aws.Request{Data: m.respCreateStackOutput, HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
	}
}

func (m *mockCloudformationStack) DeleteStackRequest(
	input *cloudformation.DeleteStackInput,
) cloudformation.DeleteStackRequest {
	if m.onDeleteStackInput != nil {
		m.onDeleteStackInput(input)
	}

	httpReq, _ := http.NewRequest("", "", nil)
	if m.err != nil {
		return cloudformation.DeleteStackRequest{
			Request: &aws.Request{Data: m.respDeleteStackOutput, Error: m.err, HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
		}
	}

	return cloudformation.DeleteStackRequest{
		Request: &aws.Request{Data: m.respDeleteStackOutput, HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
	}
}

func (m *mockCloudformationStack) DescribeStacksRequest(
	input *cloudformation.DescribeStacksInput,
) cloudformation.DescribeStacksRequest {
	if m.onDescribeStacksInput != nil {
		m.onDescribeStacksInput(input)
	}

	httpReq, _ := http.NewRequest("", "", nil)
	if m.err != nil {
		return cloudformation.DescribeStacksRequest{
			Request: &aws.Request{Data: m.respDescribeStacksOutput, Error: m.err, HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
		}
	}

	return cloudformation.DescribeStacksRequest{
		Request: &aws.Request{Data: m.respDescribeStacksOutput, HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
	}
}

func (m *mockCloudformationStack) UpdateStackRequest(
	input *cloudformation.UpdateStackInput,
) cloudformation.UpdateStackRequest {
	if m.onUpdateStackInput != nil {
		m.onUpdateStackInput(input)
	}

	httpReq, _ := http.NewRequest("", "", nil)
	if m.err != nil {
		return cloudformation.UpdateStackRequest{
			Request: &aws.Request{Data: m.respUpdateStackOutput, Error: m.err, HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
		}
	}

	return cloudformation.UpdateStackRequest{
		Request: &aws.Request{Data: m.respUpdateStackOutput, HTTPRequest: httpReq, Retryer: aws.NoOpRetryer{}},
	}
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
				Stacks: []cloudformation.Stack{cloudformation.Stack{StackId: &stackID}},
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
