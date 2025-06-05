// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package events

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestCreateEventWithRandomTransactionId(t *testing.T) {
	info := auto_ops_testing.CreateClusterInfo("8.15.3")
	metricSetFields := mapstr.M{
		"field1":      "value1",
		"obj1.field1": "value2",
		"obj2.field1": "value3",
	}

	event := CreateEventWithRandomTransactionId(&info, metricSetFields)

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event, info)

	// metrics exist
	require.True(t, len(*event.MetricSetFields.FlattenKeys()) == 3)
	require.Equal(t, "value1", auto_ops_testing.GetObjectValue(event.MetricSetFields, "field1"))
	require.Equal(t, "value2", auto_ops_testing.GetObjectValue(event.MetricSetFields, "obj1.field1"))
	require.Equal(t, "value3", auto_ops_testing.GetObjectValue(event.MetricSetFields, "obj2.field1"))
}

func TestCreateEvent(t *testing.T) {
	info := auto_ops_testing.CreateClusterInfo("8.15.3")
	metricSetFields := mapstr.M{
		"field1":      "value1",
		"obj1.field1": "value2",
		"obj2.field1": "value3",
	}
	transactionId := "my-id-is-totally-random"

	event := CreateEvent(&info, metricSetFields, transactionId)

	auto_ops_testing.CheckEventWithTransactionId(t, event, info, transactionId)

	// metrics exist
	require.True(t, len(*event.MetricSetFields.FlattenKeys()) == 3)
	require.Equal(t, "value1", auto_ops_testing.GetObjectValue(event.MetricSetFields, "field1"))
	require.Equal(t, "value2", auto_ops_testing.GetObjectValue(event.MetricSetFields, "obj1.field1"))
	require.Equal(t, "value3", auto_ops_testing.GetObjectValue(event.MetricSetFields, "obj2.field1"))
}
