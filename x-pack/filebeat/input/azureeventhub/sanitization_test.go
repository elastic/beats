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

func TestParseMultipleMessagesSanitization(t *testing.T) {
	msg := "{\"records\":[{'test':\"this is some message\",\n\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"time\": \"2023-04-11T13:35:20Z\", \"resourceId\": \"/SUBSCRIPTIONS/REDACTED/RESOURCEGROUPS/ELASTIC-FUNCTION-TEST/PROVIDERS/MICROSOFT.WEB/SITES/REDACTED\", \"category\": \"FunctionAppLogs\", \"operationName\": \"Microsoft.Web/sites/functions/log\", \"level\": \"Informational\", \"location\": \"West Europe\", \"properties\": {'appName':'REDACTED','roleInstance':'REDACTED','message':'Elastic Test Function Trigger. ---- West Europe West Europe West Europe West Europe West Europe ','category':'Function.HttpTriggerJava.User','hostVersion':'4.16.5.5','functionInvocationId':'REDACTED','functionName':'HttpTriggerJava','hostInstanceId':'REDACTED','level':'Information','levelId':2,'processId':62}}]}"
	msgs := []string{
		"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"category\":\"FunctionAppLogs\",\"level\":\"Informational\",\"location\":\"West Europe\",\"operationName\":\"Microsoft.Web/sites/functions/log\",\"properties\":{\"appName\":\"REDACTED\",\"category\":\"Function.HttpTriggerJava.User\",\"functionInvocationId\":\"REDACTED\",\"functionName\":\"HttpTriggerJava\",\"hostInstanceId\":\"REDACTED\",\"hostVersion\":\"4.16.5.5\",\"level\":\"Information\",\"levelId\":2,\"message\":\"Elastic Test Function Trigger. ---- West Europe West Europe West Europe West Europe West Europe \",\"processId\":62,\"roleInstance\":\"REDACTED\"},\"resourceId\":\"/SUBSCRIPTIONS/REDACTED/RESOURCEGROUPS/ELASTIC-FUNCTION-TEST/PROVIDERS/MICROSOFT.WEB/SITES/REDACTED\",\"time\":\"2023-04-11T13:35:20Z\"}",
	}

	input := azureInput{
		log: logp.NewLogger(fmt.Sprintf("%s test for input", inputName)),
		config: azureInputConfig{
			SanitizeOptions: []string{"SINGLE_QUOTES", "NEW_LINES"},
		},
	}

	messages := input.parseMultipleMessages([]byte(msg))
	assert.NotNil(t, messages)
	assert.Equal(t, len(messages), 3)
	for _, ms := range messages {
		assert.Contains(t, msgs, ms)
	}
}

func TestSanitize(t *testing.T) {
	jsonByte := []byte("{'test':\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}")

	testCases := []struct {
		name     string
		opts     []string
		expected []byte
	}{
		{
			name:     "no options",
			opts:     []string{},
			expected: jsonByte,
		},
		{
			name:     "NEW_LINES option",
			opts:     []string{"NEW_LINES"},
			expected: []byte("{'test':\"this is 'some' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:     "SINGLE_QUOTES option",
			opts:     []string{"SINGLE_QUOTES"},
			expected: []byte("{\"test\":\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:     "both options",
			opts:     []string{"NEW_LINES", "SINGLE_QUOTES"},
			expected: []byte("{\"test\":\"this is 'some' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
	}

	// Run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := sanitize(jsonByte, tc.opts...)
			assert.Equal(t, tc.expected, res)
		})
	}
}
