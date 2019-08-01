// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tokenbucket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucket(t *testing.T) {
	dropSize := 1
	dropRate := 50 * time.Millisecond
	bucketSize := 3
	itemsToRun := 5
	workload := make(chan int, itemsToRun)

	b, err := NewTokenBucket(bucketSize, dropSize, dropRate)
	assert.NoError(t, err, "initiating a bucket failed")

	go runSomething(b, itemsToRun, workload)

	<-time.After(10 * time.Millisecond)

	assert.Equal(t, bucketSize, len(workload))

	<-time.After(dropRate)
	assert.Equal(t, bucketSize+1, len(workload))

	<-time.After(dropRate)
	assert.Equal(t, bucketSize+2, len(workload))
}

func runSomething(b *Bucket, items int, workload chan int) {
	for i := 0; i < items; i++ {
		b.Add()
		workload <- i
	}
}
