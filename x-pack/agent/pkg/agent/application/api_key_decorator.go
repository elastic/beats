// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"encoding/base64"
	"fmt"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
)

const (
	esOutputKey       = "output.elasticsearch"
	esOutputAPIKeyKey = "output.elasticsearch.api_key"
	apiKeyKey         = "api_key"
)

func injectESOutputAPIKey(
	APIKey string,
) (decoratorFunc, error) {

	APIStrdec, err := base64.StdEncoding.DecodeString(APIKey)
	if err != nil {
		return nil, errors.New(errors.TypeConfig, "fail to decode API key")
	}

	APIStr := string(APIStrdec)

	return func(_ string, _ *transpiler.AST, programs []program.Program) ([]program.Program, error) {
		if len(programs) == 0 {
			return nil, errors.New(errors.TypeConfig, "no program received")
		}

		first := programs[0]

		// Short circuits
		// All the programs in a group have the same output so we only need to do this check once.
		// If we don't have an elasticsearch output defined we skip.
		if _, found := transpiler.Lookup(first.Config, esOutputKey); !found {
			return programs, nil
		}

		// Do not replace explicitely defined API key in the policy.
		if _, found := transpiler.Lookup(first.Config, esOutputAPIKeyKey); found {
			return programs, nil
		}

		for _, program := range programs {
			n, found := transpiler.Lookup(program.Config, esOutputKey)
			if !found {
				return nil, errors.New("waat")
			}

			d, ok := n.Value().(*transpiler.Dict)
			if !ok {
				return nil, errors.New(
					errors.TypeConfig,
					fmt.Sprintf("incompatible type expected Dictionary and received %T", n),
				)
			}

			kv, err := transpiler.NewKV(apiKeyKey, APIStr)
			if err != nil {
				return nil, errors.New(err, errors.TypeConfig, "fail to add api_key to output")
			}

			d.AddKey(kv)
		}

		return programs, nil
	}, nil
}
