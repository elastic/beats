// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

func TestMultiOutput(t *testing.T) {
	tr := &testReporter{}
	l, _ := logger.New("test")
	t.Run("no match", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&outputCapability{
				Type:   "allow",
				Output: "something_else",
			}},
		}

		initialOutputs := []string{"elasticsearch", "logstash"}
		expectedOutputs := []string{"elasticsearch", "logstash"}
		runMultiOutputTest(t, l, rd, expectedOutputs, initialOutputs)
	})

	t.Run("filters logstash", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&outputCapability{
				Type:   "deny",
				Output: "logstash",
			}},
		}

		initialOutputs := []string{"elasticsearch", "logstash"}
		expectedOutputs := []string{"elasticsearch"}
		runMultiOutputTest(t, l, rd, expectedOutputs, initialOutputs)
	})

	t.Run("allows logstash only", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{
				&outputCapability{
					Type:   "allow",
					Output: "logstash",
				},
				&outputCapability{
					Type:   "deny",
					Output: "*",
				},
			},
		}

		initialOutputs := []string{"elasticsearch", "logstash"}
		expectedOutputs := []string{"logstash"}
		runMultiOutputTest(t, l, rd, expectedOutputs, initialOutputs)
	})

	t.Run("allows everything", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&outputCapability{
				Type:   "allow",
				Output: "*",
			}},
		}

		initialOutputs := []string{"elasticsearch", "logstash"}
		expectedOutputs := []string{"elasticsearch", "logstash"}
		runMultiOutputTest(t, l, rd, expectedOutputs, initialOutputs)
	})

	t.Run("deny everything", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&outputCapability{
				Type:   "deny",
				Output: "*",
			}},
		}

		initialOutputs := []string{"elasticsearch", "logstash"}
		expectedOutputs := []string{}
		runMultiOutputTest(t, l, rd, expectedOutputs, initialOutputs)
	})

	t.Run("keep format", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&outputCapability{
				Type:   "deny",
				Output: "logstash",
			}},
		}

		initialOutputs := []string{"elasticsearch", "logstash"}
		expectedOutputs := []string{"elasticsearch"}

		cap, err := newOutputsCapability(l, rd, tr)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		outputs := getOutputs(initialOutputs...)
		assert.NotNil(t, outputs)

		res, err := cap.Apply(outputs)
		assert.NoError(t, err, "should not be failing")
		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")

		ast, ok := res.(*transpiler.AST)
		assert.True(t, ok, "expecting an ast")

		outputsIface, found := transpiler.Lookup(ast, outputKey)
		assert.True(t, found, "output  not found")

		outputsDict, ok := outputsIface.Value().(*transpiler.Dict)
		assert.True(t, ok, "expecting a Dict for outputs")

		for _, in := range expectedOutputs {
			var typeFound bool
			nodes := outputsDict.Value().([]transpiler.Node)
			for _, outputKeyNode := range nodes {
				outputNode, ok := outputKeyNode.(*transpiler.Key).Value().(*transpiler.Dict)
				assert.True(t, ok, "output type key not string")

				typeNode, found := outputNode.Find("type")
				assert.True(t, found, "type not found")

				typeNodeStr, ok := typeNode.Value().(*transpiler.StrVal)
				assert.True(t, ok, "type node not strval")
				outputType, ok := typeNodeStr.Value().(string)
				assert.True(t, ok, "output type key not string")
				if outputType == in {
					typeFound = true
					break
				}
			}

			assert.True(t, typeFound, fmt.Sprintf("output '%s' type key not found", in))
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		rd := &ruleDefinitions{
			Capabilities: []ruler{&outputCapability{
				Type:   "deny",
				Output: "logstash",
			}},
		}

		cap, err := newOutputsCapability(l, rd, tr)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")

		apiAction := fleetapi.ActionUpgrade{}
		outAfter, err := cap.Apply(apiAction)

		assert.NoError(t, err, "should not be failing")
		assert.NotEqual(t, ErrBlocked, err, "should not be blocking")
		assert.Equal(t, apiAction, outAfter, "action should not be altered")
	})
}

func TestOutput(t *testing.T) {
	tr := &testReporter{}
	l, _ := logger.New("test")
	t.Run("invalid rule", func(t *testing.T) {
		r := &upgradeCapability{}
		cap, err := newOutputCapability(l, r, tr)
		assert.NoError(t, err, "no error expected")
		assert.Nil(t, cap, "cap should not be created")
	})

	t.Run("empty eql", func(t *testing.T) {
		r := &outputCapability{
			Type:   "allow",
			Output: "",
		}
		cap, err := newOutputCapability(l, r, tr)
		assert.NoError(t, err, "error not expected, provided eql is valid")
		assert.NotNil(t, cap, "cap should be created")
	})

	t.Run("valid action - 1/1 match", func(t *testing.T) {
		r := &outputCapability{
			Type:   "allow",
			Output: "logstash",
		}

		initialOutputs := []string{"logstash"}
		expectedOutputs := []string{"logstash"}
		runOutputTest(t, l, r, expectedOutputs, initialOutputs)
	})

	t.Run("valid action - 0/1 match", func(t *testing.T) {
		r := &outputCapability{
			Type:   "allow",
			Output: "elasticsearch",
		}

		initialOutputs := []string{"logstash"}
		expectedOutputs := []string{"logstash"}
		runOutputTest(t, l, r, expectedOutputs, initialOutputs)
	})

	t.Run("valid action - deny logstash", func(t *testing.T) {
		r := &outputCapability{
			Type:   "deny",
			Output: "logstash",
		}

		initialOutputs := []string{"logstash", "elasticsearch"}
		expectedOutputs := []string{"elasticsearch"}
		runOutputTest(t, l, r, expectedOutputs, initialOutputs)
	})

	t.Run("valid action - multiple outputs 1 explicitely allowed", func(t *testing.T) {
		r := &outputCapability{
			Type:   "allow",
			Output: "logstash",
		}

		initialOutputs := []string{"logstash", "elasticsearch"}
		expectedOutputs := []string{"logstash", "elasticsearch"}
		runOutputTest(t, l, r, expectedOutputs, initialOutputs)
	})
}

func runMultiOutputTest(t *testing.T, l *logger.Logger, rd *ruleDefinitions, expectedOutputs []string, initialOutputs []string) {
	tr := &testReporter{}
	cap, err := newOutputsCapability(l, rd, tr)
	assert.NoError(t, err, "error not expected, provided eql is valid")
	assert.NotNil(t, cap, "cap should be created")

	cfg := getOutputsMap(initialOutputs...)
	assert.NotNil(t, cfg)

	outAfter, err := cap.Apply(cfg)

	assert.NoError(t, err, "should not be failing")
	assert.NotEqual(t, ErrBlocked, err, "should not be blocking")

	newMap, ok := outAfter.(map[string]interface{})
	assert.True(t, ok, "out ast should be a map")
	assert.NotNil(t, newMap)

	outputsNode, found := newMap[outputKey]
	assert.True(t, found, "outputs not found")

	outputsList, ok := outputsNode.(map[string]interface{})
	assert.True(t, ok, "outputs not a list")

	typesMap := make(map[string]bool)
	for _, nodeIface := range outputsList {
		node, ok := nodeIface.(map[string]interface{})
		if !ok {
			continue
		}

		typeNode, ok := node["type"]
		if !ok {
			continue
		}

		outputType, ok := typeNode.(string)
		if !ok {
			continue
		}
		typesMap[outputType] = true
	}

	assert.Equal(t, len(expectedOutputs), len(typesMap))
	for _, ei := range expectedOutputs {
		_, found = typesMap[ei]
		assert.True(t, found, fmt.Sprintf("'%s' not found", ei))
		delete(typesMap, ei)
	}

	for k := range typesMap {
		assert.Fail(t, fmt.Sprintf("'%s' found but was not expected", k))
	}
}

func runOutputTest(t *testing.T, l *logger.Logger, r *outputCapability, expectedOutputs []string, initialOutputs []string) {
	tr := &testReporter{}
	cap, err := newOutputCapability(l, r, tr)
	assert.NoError(t, err, "error not expected, provided eql is valid")
	assert.NotNil(t, cap, "cap should be created")

	outputs := getOutputsMap(initialOutputs...)
	assert.NotNil(t, outputs)

	newMap, err := cap.Apply(outputs)
	assert.NoError(t, err, "should not be failing")
	assert.NotEqual(t, ErrBlocked, err, "should not be blocking")

	outputsNode, found := newMap[outputKey]
	assert.True(t, found, "outputs not found")

	outputsList, ok := outputsNode.(map[string]interface{})
	assert.True(t, ok, "outputs not a map")

	typesMap := make(map[string]bool)
	for _, nodeIface := range outputsList {
		node, ok := nodeIface.(map[string]interface{})
		if !ok {
			continue
		}

		typeNode, ok := node[typeKey]
		if !ok {
			continue
		}

		outputType, ok := typeNode.(string)
		if !ok {
			continue
		}

		conditionNode, ok := node[conditionKey]
		if !ok {
			// was not allowed nor denied -> allowing
			typesMap[outputType] = true
			continue
		}

		isAllowed, ok := conditionNode.(bool)
		if !ok {
			assert.Fail(t, fmt.Sprintf("condition should be bool but it's not for output '%s'", outputType))
			continue
		}

		if isAllowed {
			typesMap[outputType] = true
		}
	}

	assert.Equal(t, len(expectedOutputs), len(typesMap))
	for _, ei := range expectedOutputs {
		_, found = typesMap[ei]
		assert.True(t, found, fmt.Sprintf("'%s' not found", ei))
		delete(typesMap, ei)
	}

	for k := range typesMap {
		assert.Fail(t, fmt.Sprintf("'%s' found but was not expected", k))
	}
}

func getOutputs(tt ...string) *transpiler.AST {
	astMap := getOutputsMap(tt...)
	ast, _ := transpiler.NewAST(astMap)
	return ast
}

func getOutputsMap(tt ...string) map[string]interface{} {
	cfgMap := make(map[string]interface{})
	outputs := make(map[string]interface{})

	for i, t := range tt {
		outputs[fmt.Sprintf("id%d", i)] = map[string]interface{}{
			"type":  t,
			"hosts": []string{"testing"},
		}
	}

	cfgMap[outputKey] = outputs
	return cfgMap
}
