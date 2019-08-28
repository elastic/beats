// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shared

//go:generate go run gen-ftd-ecs-mapping.go stringset.go -output ecs-mapping-processor.yml security-mappings.csv
//go:generate go run gen-ecs-mapping-docs.go stringset.go -output ecs-mapping-docs.asciidoc security-mappings.csv
