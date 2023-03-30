// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// ErrInvalidNamespace is error returned when namespace value provided is invalid.
var ErrInvalidNamespace = errors.New("provided namespace is invalid", errors.TypeConfig)

// ErrInvalidDataset is error returned when datastream name value provided is invalid.
var ErrInvalidDataset = errors.New("provided datastream dataset is invalid", errors.TypeConfig)

// ErrInvalidIndex occurs when concatenation of {data_stream.type}-{data_stream.dataset}-{data_stream.namespace} does not meet index criteria.
var ErrInvalidIndex = errors.New("provided combination of type, datastream dataset and namespace is invalid", errors.TypeConfig)

// StreamChecker checks for invalid values in stream namespace and dataset.
func StreamChecker(log *logger.Logger, ast *transpiler.AST) error {
	inputsNode, found := transpiler.Lookup(ast, "inputs")
	if !found {
		return nil
	}

	inputsNodeList, ok := inputsNode.Value().(*transpiler.List)
	if !ok {
		return nil
	}

	inputsNodeListCollection, ok := inputsNodeList.Value().([]transpiler.Node)
	if !ok {
		return errors.New("inputs is not a list", errors.TypeConfig)
	}

	for _, inputNode := range inputsNodeListCollection {
		namespace := "default"
		datasetName := "generic"
		// fail only if data_stream.namespace or data_stream[namespace] is found and invalid
		// not provided values are ok and will be fixed by rules
		if nsNode, found := inputNode.Find("data_stream.namespace"); found {
			nsKey, ok := nsNode.(*transpiler.Key)
			if ok {
				namespace = nsKey.Value().(transpiler.Node).String()
			}
		} else {
			dsNode, found := inputNode.Find("data_stream")
			if found {
				// got a datastream
				datasetMap, ok := dsNode.Value().(*transpiler.Dict)
				if ok {
					nsNode, found := datasetMap.Find("namespace")
					if found {
						nsKey, ok := nsNode.(*transpiler.Key)
						if ok {
							namespace = nsKey.Value().(transpiler.Node).String()
						}
					}
				}
			}
		}

		if !matchesNamespaceContraints(namespace) {
			return ErrInvalidNamespace
		}

		// get the type, longest type for now is metrics
		datasetType := "metrics"
		if nsNode, found := inputNode.Find("data_stream.type"); found {
			nsKey, ok := nsNode.(*transpiler.Key)
			if ok {
				newDataset := nsKey.Value().(transpiler.Node).String()
				datasetType = newDataset
			}
		} else {
			dsNode, found := inputNode.Find("data_stream")
			if found {
				// got a dataset
				datasetMap, ok := dsNode.Value().(*transpiler.Dict)
				if ok {
					nsNode, found := datasetMap.Find("type")
					if found {
						nsKey, ok := nsNode.(*transpiler.Key)
						if ok {
							newDataset := nsKey.Value().(transpiler.Node).String()
							datasetType = newDataset
						}
					}
				}
			}
		}

		if !matchesTypeConstraints(datasetType) {
			return ErrInvalidIndex
		}

		streamsNode, ok := inputNode.Find("streams")
		if ok {
			streamsList, ok := streamsNode.Value().(*transpiler.List)
			if ok {
				streamNodes, ok := streamsList.Value().([]transpiler.Node)
				if !ok {
					return errors.New("streams is not a list", errors.TypeConfig)
				}

				for _, streamNode := range streamNodes {
					streamMap, ok := streamNode.(*transpiler.Dict)
					if !ok {
						continue
					}

					// fix this only if in compact form
					if dsNameNode, found := streamMap.Find("data_stream.dataset"); found {
						dsKey, ok := dsNameNode.(*transpiler.Key)
						if ok {
							datasetName = dsKey.Value().(transpiler.Node).String()
							break
						}
					} else {
						datasetNode, found := streamMap.Find("data_stream")
						if found {
							datasetMap, ok := datasetNode.Value().(*transpiler.Dict)
							if !ok {
								continue
							}

							dsNameNode, found := datasetMap.Find("dataset")
							if found {
								dsKey, ok := dsNameNode.(*transpiler.Key)
								if ok {
									datasetName = dsKey.Value().(transpiler.Node).String()
									break
								}
							}
						}
					}
				}
			}
		}
		if !matchesDatasetConstraints(datasetName) {
			return ErrInvalidDataset
		}
	}

	return nil
}

// The only two requirement are that it has only characters allowed in an Elasticsearch index name
// Index names must meet the following criteria:
//
//	Not longer than 100 bytes
//	Lowercase only
//	Cannot include \, /, *, ?, ", <, >, |, ` ` (space character), ,, #
func matchesNamespaceContraints(namespace string) bool {
	// length restriction is in bytes, not characters
	if len(namespace) <= 0 || len(namespace) > 100 {
		return false
	}

	return isCharactersetValid(namespace)
}

// matchesTypeConstraints fails for following rules. As type is first element of resulting index prefix restrictions need to be applied.
//
//	Not longer than 20 bytes
//	Lowercase only
//	Cannot start with -, _, +
//	Cannot include \, /, *, ?, ", <, >, |, ` ` (space character), ,, #
func matchesTypeConstraints(dsType string) bool {
	// length restriction is in bytes, not characters
	if len(dsType) <= 0 || len(dsType) > 20 {
		return false
	}

	if strings.HasPrefix(dsType, "-") || strings.HasPrefix(dsType, "_") || strings.HasPrefix(dsType, "+") {
		return false
	}

	return isCharactersetValid(dsType)
}

// matchesDatasetConstraints fails for following rules
//
//	Not longer than 100 bytes
//	Lowercase only
//	Cannot include \, /, *, ?, ", <, >, |, ` ` (space character), ,, #
func matchesDatasetConstraints(dataset string) bool {
	// length restriction is in bytes, not characters
	if len(dataset) <= 0 || len(dataset) > 100 {
		return false
	}

	return isCharactersetValid(dataset)
}

func isCharactersetValid(input string) bool {
	if strings.ToLower(input) != input {
		return false
	}

	if strings.ContainsAny(input, "\\/*?\"<>| ,#:") {
		return false
	}

	return true
}
