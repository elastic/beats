// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute

import (
	"context"
	"sync"

	"google.golang.org/api/option"

	"google.golang.org/api/compute/v1"
)

var srv computeService

type computeService struct {
	*compute.Service
	sync.Mutex
}

func createOrReturnComputeService(ctx context.Context, opt option.ClientOption) (service *compute.Service, err error) {
	srv.Lock()
	defer srv.Unlock()

	if srv.Service == nil {
		srv.Service, err = compute.NewService(ctx, opt)
		if err != nil {
			return nil, err
		}
	}

	return srv.Service, nil
}
