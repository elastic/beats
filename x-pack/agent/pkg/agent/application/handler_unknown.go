// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import "fmt"

type handlerUnknown struct{}

func (h *handlerUnknown) Handle(a action) error {
	return fmt.Errorf("unknown action reveived of type %T", a)
}
