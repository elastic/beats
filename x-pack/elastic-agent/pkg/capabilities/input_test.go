// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

func TestMultiInput(t *testing.T) {
	tr := &testReporter{}
	l, _ := logger.New("test")
	t.Run("no match", func(t *testing.T) {

		rd := &ruleDefinitions{
			Capabilities: []ruler{&inputCapability{
				Type:  "allow",
				Input: "something_else",
			}},
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{"system/metrics", "system/logs"}
		runMultiInputTest(t, l, rd, expectedInputs, initialInputs)
	})

	t.Run("filters metrics", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&inputCapability{
				Type:  "deny",
				Input: "system/metrics",
			}},
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{"system/logs"}
		runMultiInputTest(t, l, rd, expectedInputs, initialInputs)
	})

	t.Run("allows metrics only", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&inputCapability{
					Type:  "allow",
					Input: "system/metrics",
				},
				&inputCapability{
					Type:  "deny",
					Input: "*",
				},
			},
		}

		initialInputs := []string{"system/metrics", "system/logs", "something_else"}
		expectedInputs := []string{"system/metrics"}
		runMultiInputTest(t, l, rd, expectedInputs, initialInputs)
	})

	t.Run("allows everything", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&inputCapability{
				Type:  "allow",
				Input: "*",
			}},
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{"system/metrics", "system/logs"}
		runMultiInputTest(t, l, rd, expectedInputs, initialInputs)
	})

	t.Run("deny everything", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&inputCapability{
				Type:  "deny",
				Input: "*",
			}},
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{}
		runMultiInputTest(t, l, rd, expectedInputs, initialInputs)
	})

	t.Run("deny everything with noise", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&inputCapability{
					Type:  "deny",
					Input: "*",
				},
				&inputCapability{
					Type:  "allow",
					Input: "something_else",
				},
			},
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{}
		runMultiInputTest(t, l, rd, expectedInputs, initialInputs)
	})

	t.Run("keep format", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&inputCapability{
				Type:  "deny",
				Input: "system/metrics",
			}},
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{"system/logs"}

		cap, err := newInputsCapability(l, rd, tr)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		inputs := getInputs(initialInputs...)
		assert.NotNil(t, inputs)

		res, err := cap.Apply(inputs)
		assert.NoError(t, err, "should not be failing")
		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")

		ast, ok := res.(*transpiler.AST)
		assert.True(t, ok, "expecting an ast")

		inputsIface, found := transpiler.Lookup(ast, "inputs")
		assert.True(t, found, "input  not found")

		inputsList, ok := inputsIface.Value().(*transpiler.List)
		assert.True(t, ok, "expecting a list for inputs")

		for _, in := range expectedInputs {
			var typeFound bool
			nodes := inputsList.Value().([]transpiler.Node)
			for _, inputNode := range nodes {
				typeNode, found := inputNode.Find("type")
				assert.True(t, found, "type not found")

				typeNodeStr, ok := typeNode.Value().(*transpiler.StrVal)
				assert.True(t, ok, "type node not strval")
				inputType, ok := typeNodeStr.Value().(string)
				assert.True(t, ok, "input type key not string")
				if inputType == in {
					typeFound = true
					break
				}
			}

			assert.True(t, typeFound, fmt.Sprintf("input '%s' type key not found", in))
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&inputCapability{
				Type:  "deny",
				Input: "system/metrics",
			}},
		}
		cap, err := newInputsCapability(l, rd, tr)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := fleetapi.ActionUpgrade{}
		outAfter, err := cap.Apply(apiAction)

		assert.NoError(t, err, "should not be failing")
		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})
}

func TestInput(t *testing.T) {
	l, _ := logger.New("test")
	tr := &testReporter{}
	t.Run("invalid rule", func(t *testing.T) {
		r := &upgradeCapability{}
		cap, err := newInputCapability(l, r, tr)
		assert.NoError(t, err, "no error expected")
		assert.Nil(t, cap, "cap should not be created")
	})

	t.Run("empty eql", func(t *testing.T) {
		r := &inputCapability{
			Type:  "allow",
			Input: "",
		}
		cap, err := newInputCapability(l, r, tr)
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
		runInputTest(t, l, r, expectedInputs, initialInputs)
	})

	t.Run("valid action - 0/1 match", func(t *testing.T) {
		r := &inputCapability{
			Type:  "allow",
			Input: "system/metrics",
		}

		initialInputs := []string{"system/logs"}
		expectedInputs := []string{"system/logs"}
		runInputTest(t, l, r, expectedInputs, initialInputs)
	})

	t.Run("valid action - deny metrics", func(t *testing.T) {
		r := &inputCapability{
			Type:  "deny",
			Input: "system/metrics",
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{"system/logs"}
		runInputTest(t, l, r, expectedInputs, initialInputs)
	})

	t.Run("valid action - multiple inputs 1 explicitely allowed", func(t *testing.T) {
		r := &inputCapability{
			Type:  "allow",
			Input: "system/metrics",
		}

		initialInputs := []string{"system/metrics", "system/logs"}
		expectedInputs := []string{"system/metrics", "system/logs"}
		runInputTest(t, l, r, expectedInputs, initialInputs)
	})

}

func runInputTest(t *testing.T, l *logger.Logger, r *inputCapability, expectedInputs []string, initialInputs []string) {
	tr := &testReporter{}
	cap, err := newInputCapability(l, r, tr)
	assert.NoError(t, err, "error not expected, provided eql is valid")
	assert.NotNil(t, cap, "cap should be created")

	inputs := getInputsMap(initialInputs...)
	assert.NotNil(t, inputs)

	newMap, err := cap.Apply(inputs)
	assert.NoError(t, err, "should not be failing")
	assert.NotEqual(t, ErrBlocked, err, "should not be blocking")

	inputsNode, found := newMap["inputs"]
	assert.True(t, found, "inputsnot found")

	inputsList, ok := inputsNode.([]map[string]interface{})
	assert.True(t, ok, "inputs not a list")

	typesMap := make(map[string]bool)
	for _, node := range inputsList {
		typeNode, ok := node["type"]
		if !ok {
			continue
		}

		inputType, ok := typeNode.(string)
		if !ok {
			continue
		}

		conditionNode, ok := node[conditionKey]
		if !ok {
			// was not allowed nor denied -> allowing
			typesMap[inputType] = true
			continue
		}

		isAllowed, ok := conditionNode.(bool)
		if !ok {
			assert.Fail(t, fmt.Sprintf("condition should be bool but it's not for input '%s'", inputType))
			continue
		}

		if isAllowed {
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

func runMultiInputTest(t *testing.T, l *logger.Logger, rd *ruleDefinitions, expectedInputs []string, initialInputs []string) {
	tr := &testReporter{}
	cap, err := newInputsCapability(l, rd, tr)
	assert.NoError(t, err, "error not expected, provided eql is valid")
	assert.NotNil(t, cap, "cap should be created")

	inputs := getInputsMap(initialInputs...)
	assert.NotNil(t, inputs)

	outAfter, err := cap.Apply(inputs)
	assert.NoError(t, err, "should not be failing")
	assert.NotEqual(t, ErrBlocked, err, "should not be blocking")

	newMap, ok := outAfter.(map[string]interface{})
	assert.True(t, ok, "out ast should be AST")
	assert.NotNil(t, newMap)

	inputsNode, found := newMap["inputs"]
	assert.True(t, found, "inputsnot found")

	inputsList, ok := inputsNode.([]map[string]interface{})
	assert.True(t, ok, "inputs not a list")

	typesMap := make(map[string]bool)
	for _, node := range inputsList {
		typeNode, ok := node["type"]
		if !ok {
			continue
		}

		inputType, ok := typeNode.(string)
		if !ok {
			continue
		}
		typesMap[inputType] = true
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

func getInputs(tt ...string) *transpiler.AST {
	astMap := getInputsMap(tt...)
	ast, _ := transpiler.NewAST(astMap)
	return ast
}

func getInputsMap(tt ...string) map[string]interface{} {
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

	return astMap
}

type testReporter struct{}

func (*testReporter) Update(state.Status, string) {}
func (*testReporter) Unregister()                 {}
