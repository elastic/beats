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

	insert, err := transpiler.NewAST(map[string]interface{}{
		apiKeyKey: string(APIStrdec),
	})
	if err != nil {
		return nil, errors.New(errors.TypeConfig, "error reading the API Key")
	}

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
			fmt.Printf("%+v\n", insert.Root())

			err := transpiler.Insert(program.Config, insert.Root(), esOutputKey)
			if err != nil {
				return nil, errors.New(
					err,
					errors.TypeConfig,
					"could not insert api key to the configuration",
				)
			}
		}

		return programs, nil
	}, nil
}
