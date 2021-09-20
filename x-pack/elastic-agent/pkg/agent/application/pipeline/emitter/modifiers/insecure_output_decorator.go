// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package modifiers

import (
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	sslKey              = "ssl"
	verificationModeKey = "verification_mode"
	sslVerificationKey  = "ssl.verification_mode"
)

// InjectInsecureOutput injects a verification none into output configuration.
func InjectInsecureOutput(fleetConfig *configuration.FleetAgentConfig) func(*logger.Logger, *transpiler.AST) error {
	return func(_ *logger.Logger, rootAst *transpiler.AST) error {
		// if verification mode is not set abort
		if fleetConfig == nil ||
			fleetConfig.Client.Transport.TLS == nil ||
			fleetConfig.Client.Transport.TLS.VerificationMode == tlscommon.VerifyFull {
			// no change
			return nil
		}

		// look for outputs
		// inject verification mode to each output
		outputsNode, ok := transpiler.Lookup(rootAst, outputsKey)
		if !ok {
			// no outputs from configuration; skip
			return nil
		}

		outputsList, ok := outputsNode.Value().(*transpiler.Dict)
		if !ok {
			return nil
		}

		outputsNodeCollection, ok := outputsList.Value().([]transpiler.Node)
		if !ok {
			return nil
		}

		modeString := fleetConfig.Client.Transport.TLS.VerificationMode.String()

		for _, outputNode := range outputsNodeCollection {
			outputKV, ok := outputNode.(*transpiler.Key)
			if !ok {
				continue
			}

			output, ok := outputKV.Value().(*transpiler.Dict)
			if !ok {
				continue
			}

			// do not overwrite already specified config
			_, found := output.Find(sslVerificationKey)
			if found {
				continue
			}

			// it may be broken down
			if sslNode, found := output.Find(sslKey); found {
				if _, found := sslNode.Find(verificationModeKey); found {
					continue
				}
			}

			output.Insert(
				transpiler.NewKey(sslVerificationKey, transpiler.NewStrVal(modeString)),
			)
		}

		return nil
	}
}
