// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import "fmt"

type handlerDefault struct{}

func (h *handlerDefault) Handle(a action) error {
	return fmt.Errorf("unhandled action of type %T", a)
}
