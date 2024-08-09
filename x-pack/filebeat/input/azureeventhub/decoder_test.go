// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestDecodeRecords(t *testing.T) {
	config := defaultConfig()
	log := logp.NewLogger(fmt.Sprintf("%s test for input", inputName))
	reg := monitoring.NewRegistry()

	decoder := messageDecoder{
		config:  config,
		log:     log,
		metrics: newInputMetrics("test", reg),
	}

	msgs := []string{
		"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is 2nd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is 3rd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
	}

	t.Run("Decode multiple records", func(t *testing.T) {
		msg := "{\"records\":[{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
			"{\"test\":\"this is 2nd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
			"{\"test\":\"this is 3rd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"

		messages := decoder.Decode([]byte(msg))

		assert.NotNil(t, messages)
		assert.Equal(t, len(messages), 3)
		for _, ms := range messages {
			assert.Contains(t, msgs, ms)
		}
	})

	t.Run("Decode array of events", func(t *testing.T) {
		msg1 := "[{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
			"{\"test\":\"this is 2nd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
			"{\"test\":\"this is 3rd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]"

		messages := decoder.Decode([]byte(msg1))

		assert.NotNil(t, messages)
		assert.Equal(t, len(messages), 3)
		for _, ms := range messages {
			assert.Contains(t, msgs, ms)
		}
	})

	t.Run("Decode one event only", func(t *testing.T) {
		msg2 := "{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"

		messages := decoder.Decode([]byte(msg2))

		assert.NotNil(t, messages)
		assert.Equal(t, len(messages), 1)
		for _, ms := range messages {
			assert.Contains(t, msgs, ms)
		}
	})

	t.Run("Decode array with one event", func(t *testing.T) {
		msg := "[{\"key1\":\"value1\",\"key2\":\"value2\",\"key3\":\"value3\",\"nestedKey\":{\"nestedKey1\":\"nestedValue1\"},\"arrayKey\":[\"arrayValue1\",\"arrayValue2\"]}]"
		expected := "{\"arrayKey\":[\"arrayValue1\",\"arrayValue2\"],\"key1\":\"value1\",\"key2\":\"value2\",\"key3\":\"value3\",\"nestedKey\":{\"nestedKey1\":\"nestedValue1\"}}"

		messages := decoder.Decode([]byte(msg))

		assert.NotNil(t, messages)
		assert.Equal(t, len(messages), 1)

		for _, actual := range messages {
			assert.Equal(t, expected, actual)
		}
	})
}

func TestDecodeRecordsWithSanitization(t *testing.T) {
	config := defaultConfig()
	config.SanitizeOptions = []string{"SINGLE_QUOTES", "NEW_LINES"}
	log := logp.NewLogger(fmt.Sprintf("%s test for input", inputName))
	reg := monitoring.NewRegistry()
	metrics := newInputMetrics("test", reg)
	defer metrics.Close()

	decoder := messageDecoder{
		config:  config,
		log:     log,
		metrics: metrics,
	}

	msg := "{\"records\":[{'test':\"this is some message\",\n\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"time\": \"2023-04-11T13:35:20Z\", \"resourceId\": \"/SUBSCRIPTIONS/REDACTED/RESOURCEGROUPS/ELASTIC-FUNCTION-TEST/PROVIDERS/MICROSOFT.WEB/SITES/REDACTED\", \"category\": \"FunctionAppLogs\", \"operationName\": \"Microsoft.Web/sites/functions/log\", \"level\": \"Informational\", \"location\": \"West Europe\", \"properties\": {'appName':'REDACTED','roleInstance':'REDACTED','message':'Elastic Test Function Trigger. ---- West Europe West Europe West Europe West Europe West Europe ','category':'Function.HttpTriggerJava.User','hostVersion':'4.16.5.5','functionInvocationId':'REDACTED','functionName':'HttpTriggerJava','hostInstanceId':'REDACTED','level':'Information','levelId':2,'processId':62}}]}"
	msgs := []string{
		"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is '2nd' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"category\":\"FunctionAppLogs\",\"level\":\"Informational\",\"location\":\"West Europe\",\"operationName\":\"Microsoft.Web/sites/functions/log\",\"properties\":{\"appName\":\"REDACTED\",\"category\":\"Function.HttpTriggerJava.User\",\"functionInvocationId\":\"REDACTED\",\"functionName\":\"HttpTriggerJava\",\"hostInstanceId\":\"REDACTED\",\"hostVersion\":\"4.16.5.5\",\"level\":\"Information\",\"levelId\":2,\"message\":\"Elastic Test Function Trigger. ---- West Europe West Europe West Europe West Europe West Europe \",\"processId\":62,\"roleInstance\":\"REDACTED\"},\"resourceId\":\"/SUBSCRIPTIONS/REDACTED/RESOURCEGROUPS/ELASTIC-FUNCTION-TEST/PROVIDERS/MICROSOFT.WEB/SITES/REDACTED\",\"time\":\"2023-04-11T13:35:20Z\"}",
	}

	messages := decoder.Decode([]byte(msg))

	assert.NotNil(t, messages)
	assert.Equal(t, len(messages), 3)
	for _, ms := range messages {
		assert.Contains(t, msgs, ms)
	}
}
