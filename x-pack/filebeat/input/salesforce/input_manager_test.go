// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import v2 "github.com/elastic/beats/v7/filebeat/input/v2"

// compile-time check if querier implements InputManager
var _ v2.InputManager = InputManager{}
