// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/telemetry"
	"github.com/elastic/beats/v7/x-pack/functionbeat/provider/aws/aws/transformer"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type message struct {
	RequestID string `json:"request_id"`
	Status    int    `json:"status"`
	Message   string `json:"message"`
}

// APIGatewayProxy receives events from the web service and forward them to elasticsearch.
type APIGatewayProxy struct {
	log *logp.Logger
}

// NewAPIGatewayProxy creates a new function to receives events from the web api gateway.
func NewAPIGatewayProxy(provider provider.Provider, config *conf.C) (provider.Function, error) {
	cfgwarn.Experimental("The api_gateway_proxy trigger is experimental.")
	return &APIGatewayProxy{log: logp.NewLogger("api gateway proxy")}, nil
}

// APIGatewayProxyDetails returns the details of the feature.
func APIGatewayProxyDetails() feature.Details {
	return feature.MakeDetails("API Gateway proxy trigger", "receive events from the api gateway proxy", feature.Experimental)
}

// Run starts the lambda function and wait for web triggers.
func (a *APIGatewayProxy) Run(_ context.Context, client pipeline.ISyncClient, telemetry telemetry.T) error {
	telemetry.AddTriggeredFunction()

	lambda.Start(a.createHandler(client))
	return nil
}

func (a *APIGatewayProxy) createHandler(
	client pipeline.ISyncClient,
) func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		a.log.Debugf("The handler receives a new event from the gateway (requestID: %s)", request.RequestContext.RequestID)
		event := transformer.APIGatewayProxyRequest(request)

		if err := client.Publish(event); err != nil {
			a.log.Errorf("could not publish event to the pipeline, error: %+v", err)
			return buildResponse(
				http.StatusInternalServerError,
				"an error occurred when sending the event.",
				request.RequestContext.RequestID,
			), err
		}
		client.Wait()
		return buildResponse(
			http.StatusOK,
			"event received successfully.",
			request.RequestContext.RequestID,
		), nil
	}
}

func buildResponse(
	statusCode int,
	responseMsg string,
	requestID string,
) events.APIGatewayProxyResponse {
	body, _ := json.Marshal(message{Status: statusCode, Message: responseMsg, RequestID: requestID})

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}
}

// Name return the name of the lambda function.
func (a *APIGatewayProxy) Name() string {
	return "api_gateway_proxy"
}
