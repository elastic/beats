// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package syncgateway

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/helper"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "kubernetes" module.
	if err := mb.Registry.AddModule("syncgateway", ModuleBuilder()); err != nil {
		panic(err)
	}
}

// FetchPeriodThreshold is the threshold to use when checking if the cluster state call in this cache should be
// fetched again. It's used as a percentage so 0.9 means that cache is invalidated if at least 90% of period time has
// passed. This small threshold is to avoid race conditions where one of the X number of calls that happen after each
// period do not fetch new data while other does, becoming unsync by [period] seconds. It does not fully solve the
// problem because the period waits aren't guaranteed but it should be mostly enough.
const FetchPeriodThreshold = 0.9

type Module interface {
	mb.Module
	GetSyncgatewayResponse(http *helper.HTTP) (*SgResponse, error)
}

type exprVarCache struct {
	lock               sync.Mutex
	lastFetchTimestamp time.Time

	cachedData SgResponse
}

type module struct {
	mb.BaseModule

	expvarCache *exprVarCache
	cacheHash   uint64
}

func ModuleBuilder() func(base mb.BaseModule) (mb.Module, error) {
	expvarCache := &exprVarCache{}

	return func(base mb.BaseModule) (mb.Module, error) {
		m := module{
			BaseModule:  base,
			expvarCache: expvarCache,
		}
		return &m, nil
	}
}

func (m *module) GetSyncgatewayResponse(http *helper.HTTP) (*SgResponse, error) {
	m.expvarCache.lock.Lock()
	defer m.expvarCache.lock.Unlock()

	elapsedFromLastFetch := time.Since(m.expvarCache.lastFetchTimestamp)

	//Check the lifetime of current response data
	if elapsedFromLastFetch == 0 || elapsedFromLastFetch >= (time.Duration(float64(m.Config().Period)*FetchPeriodThreshold)) {
		//Fetch data again
		byt, err := http.FetchContent()
		if err != nil {
			return nil, err
		}

		input := SgResponse{}
		if err = json.Unmarshal(byt, &input); err != nil {
			return nil, errors.Wrap(err, "error unmarshalling JSON of SyncGateway expvar response")
		}

		m.expvarCache.cachedData = input
		m.expvarCache.lastFetchTimestamp = time.Now()
	}

	return &m.expvarCache.cachedData, nil
}
