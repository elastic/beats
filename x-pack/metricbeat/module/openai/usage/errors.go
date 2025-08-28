// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import "errors"

// ErrNoState indicates no previous state exists for the given API key
var ErrNoState = errors.New("no previous state found")
