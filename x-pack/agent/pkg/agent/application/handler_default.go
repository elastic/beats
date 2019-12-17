// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import "github.com/elastic/beats/x-pack/agent/pkg/core/logger"

type handlerDefault struct {
	log *logger.Logger
}

func (h *handlerDefault) Handle(a action) error {
	h.log.Errorf("HandlerDefault: action '%+v' received", a)
	return nil
}
