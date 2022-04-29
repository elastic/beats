// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestEmptyTransformContext(t *testing.T) {
	ctx := emptyTransformContext()
	assert.Equal(t, &cursor{}, ctx.cursor)
	assert.Equal(t, &mapstr.M{}, ctx.lastEvent)
	assert.Equal(t, &mapstr.M{}, ctx.firstEvent)
	assert.Equal(t, &response{}, ctx.lastResponse)
}

func TestEmptyTransformable(t *testing.T) {
	tr := transformable{}
	assert.Equal(t, mapstr.M{}, tr.body())
	assert.Equal(t, http.Header{}, tr.header())
}

func TestTransformableNilClone(t *testing.T) {
	var tr transformable
	cl := tr.Clone()
	assert.Equal(t, mapstr.M{}, cl.body())
	assert.Equal(t, http.Header{}, cl.header())
}

func TestTransformableClone(t *testing.T) {
	tr := transformable{}
	body := tr.body()
	_, _ = body.Put("key", "value")
	tr.setBody(body)
	cl := tr.Clone()
	assert.Equal(t, mapstr.M{"key": "value"}, cl.body())
	assert.Equal(t, http.Header{}, cl.header())
}

func TestNewTransformsFromConfig(t *testing.T) {
	registerTransform("test", setName, newSetRequestPagination)
	t.Cleanup(func() { registeredTransforms = newRegistry() })

	cases := []struct {
		name               string
		paramCfg           map[string]interface{}
		paramNamespace     string
		expectedTransforms transforms
		expectedErr        string
	}{
		{
			name: "fails if config has more than one action",
			paramCfg: map[string]interface{}{
				"set":  nil,
				"set2": nil,
			},
			expectedErr: "each transform must have exactly one action, but found 2 actions",
		},
		{
			name: "fails if not found in namespace",
			paramCfg: map[string]interface{}{
				"set": nil,
			},
			paramNamespace: "empty",
			expectedErr:    "the transform set does not exist. Valid transforms: test: (set)\n",
		},
		{
			name: "fails if constructor fails",
			paramCfg: map[string]interface{}{
				"set": map[string]interface{}{
					"target": "invalid",
				},
			},
			paramNamespace: "test",
			expectedErr:    "invalid target: invalid",
		},
		{
			name: "transform is correct",
			paramCfg: map[string]interface{}{
				"set": map[string]interface{}{
					"target": "body.foo",
				},
			},
			paramNamespace: "test",
			expectedTransforms: transforms{
				&set{
					targetInfo: targetInfo{Name: "foo", Type: "body"},
					valueType:  valueTypeString,
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(tc.paramCfg)
			gotTransforms, gotErr := newTransformsFromConfig(transformsConfig{cfg}, tc.paramNamespace, nil)
			if tc.expectedErr == "" {
				assert.NoError(t, gotErr)
				tr := gotTransforms[0].(*set) //nolint:errcheck // Bad linter! Panic is a check.

				tr.runFunc = nil // we do not want to check func pointer
				assert.EqualValues(t, tc.expectedTransforms, gotTransforms)
			} else {
				assert.EqualError(t, gotErr, tc.expectedErr)
			}
		})
	}
}

type fakeTransform struct{}

func (fakeTransform) transformName() string { return "fake" }

func TestNewBasicTransformsFromConfig(t *testing.T) {
	fakeConstr := func(*conf.C, *logp.Logger) (transform, error) {
		return fakeTransform{}, nil
	}

	registerTransform("test", setName, newSetRequestPagination)
	registerTransform("test", "fake", fakeConstr)
	t.Cleanup(func() { registeredTransforms = newRegistry() })

	cases := []struct {
		name           string
		paramCfg       map[string]interface{}
		paramNamespace string
		expectedErr    string
	}{
		{
			name: "succeeds if transform is basicTransform",
			paramCfg: map[string]interface{}{
				"set": map[string]interface{}{
					"target": "body.foo",
				},
			},
			paramNamespace: "test",
		},
		{
			name: "fails if transform is not a basicTransform",
			paramCfg: map[string]interface{}{
				"fake": nil,
			},
			paramNamespace: "test",
			expectedErr:    "transform fake is not a valid test transform",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(tc.paramCfg)
			_, gotErr := newBasicTransformsFromConfig(transformsConfig{cfg}, tc.paramNamespace, nil)
			if tc.expectedErr == "" {
				assert.NoError(t, gotErr)
			} else {
				assert.EqualError(t, gotErr, tc.expectedErr)
			}
		})
	}
}
