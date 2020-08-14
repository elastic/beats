// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"fmt"
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
				newNamespace := nsKey.Value().(transpiler.Node).String()
				if !isValid(newNamespace) {
					return ErrInvalidNamespace
				}
				namespace = newNamespace
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
							newNamespace := nsKey.Value().(transpiler.Node).String()
							if !isValid(newNamespace) {
								return ErrInvalidNamespace
							}
							namespace = newNamespace
						}
					}
				}
			}
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
							newDataset := dsKey.Value().(transpiler.Node).String()
							if !isValid(newDataset) {
								return ErrInvalidDataset
							}
							datasetName = newDataset
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
									newDataset := dsKey.Value().(transpiler.Node).String()
									if !isValid(newDataset) {
										return ErrInvalidDataset
									}
									datasetName = newDataset
								}
							}
						}
					}
				}
			}
		}

		if indexName := fmt.Sprintf("%s-%s-%s", datasetType, datasetName, namespace); !matchesIndexContraints(indexName) {
			return ErrInvalidIndex
		}
	}

	return nil
}

// The only two requirement are that it has only characters allowed in an Elasticsearch index name
// and does NOT contain a `-`.
func isValid(namespace string) bool {
	return matchesIndexContraints(namespace) && !strings.Contains(namespace, "-")
}

// The only two requirement are that it has only characters allowed in an Elasticsearch index name
// Index names must meet the following criteria:
//     Lowercase only
//     Cannot include \, /, *, ?, ", <, >, |, ` ` (space character), ,, #
//     Cannot start with -, _, +
//     Cannot be . or ..
func matchesIndexContraints(namespace string) bool {
	// Cannot be . or ..
	if namespace == "." || namespace == ".." {
		return false
	}

	if len(namespace) <= 0 || len(namespace) > 255 {
		return false
	}

	// Lowercase only
	if strings.ToLower(namespace) != namespace {
		return false
	}

	// Cannot include \, /, *, ?, ", <, >, |, ` ` (space character), ,, #
	if strings.ContainsAny(namespace, "\\/*?\"<>| ,#") {
		return false
	}

	// Cannot start with -, _, +
	if strings.HasPrefix(namespace, "-") || strings.HasPrefix(namespace, "_") || strings.HasPrefix(namespace, "+") {
		return false
	}

	return true
}
