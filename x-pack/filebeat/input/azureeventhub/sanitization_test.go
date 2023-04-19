// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestSanitizeMessage(t *testing.T) {
	msg := "{\"records\":[{'test':\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"time\": \"2023-04-11T13:35:20Z\", \"resourceId\": \"/SUBSCRIPTIONS/REDACTED/RESOURCEGROUPS/ELASTIC-FUNCTION-TEST/PROVIDERS/MICROSOFT.WEB/SITES/REDACTED\", \"category\": \"FunctionAppLogs\", \"operationName\": \"Microsoft.Web/sites/functions/log\", \"level\": \"Informational\", \"location\": \"West Europe\", \"properties\": {'appName':'REDACTED','roleInstance':'REDACTED','message':'Elastic Test Function Trigger. ---- West Europe West Europe West Europe West Europe West Europe ','category':'Function.HttpTriggerJava.User','hostVersion':'4.16.5.5','functionInvocationId':'REDACTED','functionName':'HttpTriggerJava','hostInstanceId':'REDACTED','level':'Information','levelId':2,'processId':62}}]}"
	msgs := []string{
		"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"category\":\"FunctionAppLogs\",\"level\":\"Informational\",\"location\":\"West Europe\",\"operationName\":\"Microsoft.Web/sites/functions/log\",\"properties\":{\"appName\":\"REDACTED\",\"category\":\"Function.HttpTriggerJava.User\",\"functionInvocationId\":\"REDACTED\",\"functionName\":\"HttpTriggerJava\",\"hostInstanceId\":\"REDACTED\",\"hostVersion\":\"4.16.5.5\",\"level\":\"Information\",\"levelId\":2,\"message\":\"Elastic Test Function Trigger. ---- West Europe West Europe West Europe West Europe West Europe \",\"processId\":62,\"roleInstance\":\"REDACTED\"},\"resourceId\":\"/SUBSCRIPTIONS/REDACTED/RESOURCEGROUPS/ELASTIC-FUNCTION-TEST/PROVIDERS/MICROSOFT.WEB/SITES/REDACTED\",\"time\":\"2023-04-11T13:35:20Z\"}",
	}

	input := azureInput{log: logp.NewLogger(fmt.Sprintf("%s test for input", inputName))}
	input.config.SanitizeOptions = []string{"SINGLE_QUOTES", "NEW_LINES"}
	messages := input.parseMultipleMessages([]byte(msg))
	assert.NotNil(t, messages)
	assert.Equal(t, len(messages), 3)
	for _, ms := range messages {
		assert.Contains(t, msgs, ms)
	}
}

func TestSanitizeSingleQuotes(t *testing.T) {
	msg := "{\"records\":[{'test':\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"time\": \"2023-04-11T13:35:20Z\", \"resourceId\": \"/SUBSCRIPTIONS/REDACTED/RESOURCEGROUPS/ELASTIC-FUNCTION-TEST/PROVIDERS/MICROSOFT.WEB/SITES/REDACTED\", \"category\": \"FunctionAppLogs\", \"operationName\": \"Microsoft.Web/sites/functions/log\", \"level\": \"Informational\", \"location\": \"West Europe\", \"properties\": {'appName':'REDACTED','roleInstance':'REDACTED','message':'Elastic Test Function Trigger. ---- West Europe West Europe West Europe West Europe West Europe ','category':'Function.HttpTriggerJava.User','hostVersion':'4.16.5.5','functionInvocationId':'REDACTED','functionName':'HttpTriggerJava','hostInstanceId':'REDACTED','level':'Information','levelId':2,'processId':62}}]}"
	msgs := []string{
		"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"category\":\"FunctionAppLogs\",\"level\":\"Informational\",\"location\":\"West Europe\",\"operationName\":\"Microsoft.Web/sites/functions/log\",\"properties\":{\"appName\":\"REDACTED\",\"category\":\"Function.HttpTriggerJava.User\",\"functionInvocationId\":\"REDACTED\",\"functionName\":\"HttpTriggerJava\",\"hostInstanceId\":\"REDACTED\",\"hostVersion\":\"4.16.5.5\",\"level\":\"Information\",\"levelId\":2,\"message\":\"Elastic Test Function Trigger. ---- West Europe West Europe West Europe West Europe West Europe \",\"processId\":62,\"roleInstance\":\"REDACTED\"},\"resourceId\":\"/SUBSCRIPTIONS/REDACTED/RESOURCEGROUPS/ELASTIC-FUNCTION-TEST/PROVIDERS/MICROSOFT.WEB/SITES/REDACTED\",\"time\":\"2023-04-11T13:35:20Z\"}",
	}

	input := azureInput{log: logp.NewLogger(fmt.Sprintf("%s test for input", inputName))}
	input.config.SanitizeOptions = []string{"SINGLE_QUOTES"}
	messages := input.parseMultipleMessages([]byte(msg))
	assert.NotNil(t, messages)
	assert.Equal(t, len(messages), 3)
	for _, ms := range messages {
		assert.Contains(t, msgs, ms)
	}
}

func TestSanitizeNewLine(t *testing.T) {
	msg := "{\"records\":[{\"test\":\"this is some message\n\n\n\", \n\n\n \"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"
	msgs := []string{
		"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
	}

	input := azureInput{log: logp.NewLogger(fmt.Sprintf("%s test for input", inputName))}
	input.config.SanitizeOptions = []string{"NEW_LINES"}
	messages := input.parseMultipleMessages([]byte(msg))
	assert.NotNil(t, messages)
	assert.Equal(t, len(messages), 2)
	for _, ms := range messages {
		assert.Contains(t, msgs, ms)
	}
}
