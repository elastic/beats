// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shared

// These generators will output the following files for the FTD:
//  - ecs-mapping-processor.yml, an ingest pipeline processor that maps FTD
//    security event fields to ECS.
//  - ecs-mapping-docs.asciidoc, asciidoc tables to document those mappings.
//
// This files are not picked up by the FTD module. When generated, you need to
// manually update the pipeline in ingest/asa-ftd-pipeline.yml
// and the asciidoc tables into ../_meta/docs.asciidoc.

//go:generate go run gen-ftd-ecs-mapping.go stringset.go -output ecs-mapping-processor.yml security-mappings.csv
//go:generate go run gen-ecs-mapping-docs.go stringset.go -output ecs-mapping-docs.asciidoc security-mappings.csv
