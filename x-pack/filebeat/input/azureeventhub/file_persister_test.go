// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/Azure/azure-event-hubs-go/v3/persist"

	"github.com/stretchr/testify/assert"
)

func TestFilePersister_Read(t *testing.T) {
	namespace := "namespace"
	name := "name"
	consumerGroup := "$Default"
	partitionID := "0"
	dir := path.Join(os.TempDir(), "read")
	persister, err := persist.NewFilePersister(dir)
	assert.NoError(t, err)
	ckp, err := persister.Read(namespace, name, consumerGroup, partitionID)
	assert.Error(t, err)
	assert.Equal(t, persist.NewCheckpointFromStartOfStream(), ckp)
}

func TestFilePersister_Write(t *testing.T) {
	namespace := "namespace"
	name := "name"
	consumerGroup := "$Default"
	partitionID := "0"
	dir := path.Join(os.TempDir(), "write")
	persister, err := persist.NewFilePersister(dir)
	assert.NoError(t, err)
	ckp := persist.NewCheckpoint("120", 22, time.Now())
	err = persister.Write(namespace, name, consumerGroup, partitionID, ckp)
	assert.NoError(t, err)
	ckp2, err := persister.Read(namespace, name, consumerGroup, partitionID)
	assert.NoError(t, err)
	assert.Equal(t, ckp.Offset, ckp2.Offset)
	assert.Equal(t, ckp.SequenceNumber, ckp2.SequenceNumber)
}
