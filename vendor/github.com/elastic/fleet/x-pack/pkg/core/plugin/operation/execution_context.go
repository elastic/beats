// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"crypto/sha256"
	"fmt"
)

// ExecutionContext describes runnable binary
type ExecutionContext struct {
	BinaryName string
	Version    string
	Tags       map[Tag]string
	ID         string
}

// NewExecutionContext creates an execution context and generates an ID for this context
func NewExecutionContext(binaryName, version string, tags map[Tag]string) ExecutionContext {
	id := fmt.Sprintf("%s--%s", binaryName, version)
	if len(tags) > 0 {
		hash := fmt.Sprintf("%x", sha256.New().Sum([]byte(fmt.Sprint(tags))))
		id += fmt.Sprintf("--%x", hash)
	}

	return ExecutionContext{
		BinaryName: binaryName,
		Version:    version,
		Tags:       tags,
		ID:         id,
	}
}
