// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/stretchr/testify/assert"
)

func TestInput(t *testing.T) {
	t.Run("invalid rule", func(t *testing.T) {
		r := &upgradeCapability{}
		cap, err := NewInputCapability(r)
		assert.NoError(t, err, "no error expected")
		assert.Nil(t, cap, "cap should not be created")
	})

	t.Run("empty eql", func(t *testing.T) {
		r := &inputCapability{
			Type:  "allow",
			Input: "",
		}
		cap, err := NewInputCapability(r)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")
	})

	t.Run("valid action - 1/1 match", func(t *testing.T) {
		r := &inputCapability{
			Type:  "allow",
			Input: "system/metrics",
		}

		initialInputs := []string{"system/metrics"}
		expectedInputs := []string{"system/metrics"}
		runInputTest(t, r, expectedInputs, initialInputs)
	})

	t.Run("valid action - 0/1 match", func(t *testing.T) {
		r := &inputCapability{
			Type:  "allow",
			Input: "system/metrics",
		}

		initialInputs := []string{"system/logs"}
		expectedInputs := []string{"system/logs"}
		runInputTest(t, r, expectedInputs, initialInputs)
	})

	t.Run("valid action - deny metrics", func(t *testing.T) {
		r := &inputCapability{
			Type:  "deny",
			Input: "system/metrics",
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{"system/logs"}
		runInputTest(t, r, expectedInputs, initialInputs)
	})

	t.Run("valid action - multiple inputs 1 explicitely allowed", func(t *testing.T) {
		r := &inputCapability{
			Type:  "allow",
			Input: "system/metrics",
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{"system/metrics", "system/logs"}
		runInputTest(t, r, expectedInputs, initialInputs)
	})

	t.Run("unknown action", func(t *testing.T) {
		r := &inputCapability{
			Type:  "allow",
			Input: "system/metrics",
		}
		cap, err := NewInputCapability(r)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := fleetapi.ActionUpgrade{}
		isBlocking, outAfter := cap.Apply(apiAction)

		assert.False(t, isBlocking, "should not be blocking")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})
}

func getInputs(tt ...string) *transpiler.AST {
	astMap := make(map[string]interface{})
	inputs := make([]map[string]interface{}, 0, len(tt))

	for _, t := range tt {
		mm := map[string]interface{}{
			"type":                  t,
			"use_output":            "testing",
			"data_stream.namespace": "default",
			"streams": []map[string]interface{}{
				{
					"metricset":           "cpu",
					"data_stream.dataset": "system.cpu",
				},
				{
					"metricset":           "memory",
					"data_stream.dataset": "system.memory",
				},
			},
		}
		inputs = append(inputs, mm)
	}

	astMap["inputs"] = inputs
	ast, _ := transpiler.NewAST(astMap)
	return ast
}

func runInputTest(t *testing.T, r *inputCapability, expectedInputs []string, initialInputs []string) {
	cap, err := NewInputCapability(r)
	assert.NoError(t, err, "error not expected, provided eql is valid")
	assert.NotNil(t, cap, "cap should be created")

	inputs := getInputs(initialInputs...)
	assert.NotNil(t, inputs)
	isBlocking, outAfter := cap.Apply(inputs)
	assert.False(t, isBlocking, "should not be blocking")

	newAST, ok := outAfter.(*transpiler.AST)
	assert.True(t, ok, "out ast should be AST")
	assert.NotNil(t, newAST)

	inputsNode, found := transpiler.Lookup(newAST, "inputs")
	assert.True(t, found, "inputsnot found")

	inputsList, ok := inputsNode.Value().(*transpiler.List)
	assert.True(t, ok, "inputs not a list")

	typesMap := make(map[string]bool)
	inputNodes := inputsList.Value().([]transpiler.Node)
	for _, node := range inputNodes {
		typeNode, ok := node.Find("type")
		if !ok {
			continue
		}

		inputTypeStr, ok := typeNode.Value().(*transpiler.StrVal)
		if !ok {
			continue
		}
		inputType := inputTypeStr.String()

		conditionNode, ok := node.Find(conditionKey)
		if !ok {
			// was not allowed nor denied -> allowing
			typesMap[inputType] = true
			continue
		}

		conditionTypeBool, ok := conditionNode.Value().(*transpiler.BoolVal)
		if !ok {
			assert.Fail(t, fmt.Sprintf("condition should be bool but it's not for input '%s'", inputType))
			continue
		}

		if isAllowed := conditionTypeBool.Value().(bool); isAllowed {
			inputType := inputTypeStr.String()
			typesMap[inputType] = true
		}
	}

	assert.Equal(t, len(expectedInputs), len(typesMap))
	for _, ei := range expectedInputs {
		_, found = typesMap[ei]
		assert.True(t, found, fmt.Sprintf("'%s' not found", ei))
		delete(typesMap, ei)
	}

	for k := range typesMap {
		assert.Fail(t, fmt.Sprintf("'%s' found but was not expected", k))
	}
}
