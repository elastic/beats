// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.

package lambda

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

type Function struct {
	handler Handler
}

func (fn *Function) Ping(req *messages.PingRequest, response *messages.PingResponse) error {
	*response = messages.PingResponse{}
	return nil
}

func (fn *Function) Invoke(req *messages.InvokeRequest, response *messages.InvokeResponse) error {
	defer func() {
		if err := recover(); err != nil {
			panicInfo := getPanicInfo(err)
			response.Error = &messages.InvokeResponse_Error{
				Message:    panicInfo.Message,
				Type:       getErrorType(err),
				StackTrace: panicInfo.StackTrace,
				ShouldExit: true,
			}
		}
	}()

	deadline := time.Unix(req.Deadline.Seconds, req.Deadline.Nanos).UTC()
	invokeContext, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	lc := &lambdacontext.LambdaContext{
		AwsRequestID:       req.RequestId,
		InvokedFunctionArn: req.InvokedFunctionArn,
		Identity: lambdacontext.CognitoIdentity{
			CognitoIdentityID:     req.CognitoIdentityId,
			CognitoIdentityPoolID: req.CognitoIdentityPoolId,
		},
	}
	if len(req.ClientContext) > 0 {
		if err := json.Unmarshal(req.ClientContext, &lc.ClientContext); err != nil {
			response.Error = lambdaErrorResponse(err)
			return nil
		}
	}
	invokeContext = lambdacontext.NewContext(invokeContext, lc)

	invokeContext = context.WithValue(invokeContext, "x-amzn-trace-id", req.XAmznTraceId)

	payload, err := fn.handler.Invoke(invokeContext, req.Payload)
	if err != nil {
		response.Error = lambdaErrorResponse(err)
		return nil
	}
	response.Payload = payload
	return nil
}

func getErrorType(err interface{}) string {
	errorType := reflect.TypeOf(err)
	if errorType.Kind() == reflect.Ptr {
		return errorType.Elem().Name()
	}
	return errorType.Name()
}

func lambdaErrorResponse(invokeError error) *messages.InvokeResponse_Error {
	var errorName string
	if errorType := reflect.TypeOf(invokeError); errorType.Kind() == reflect.Ptr {
		errorName = errorType.Elem().Name()
	} else {
		errorName = errorType.Name()
	}
	return &messages.InvokeResponse_Error{
		Message: invokeError.Error(),
		Type:    errorName,
	}
}
