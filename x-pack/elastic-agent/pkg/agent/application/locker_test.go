// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLockFile = "test.lock"

func TestAppLocker(t *testing.T) {
	tmp, _ := ioutil.TempDir("", "locker")
	defer os.RemoveAll(tmp)

	locker1 := NewAppLocker(tmp, testLockFile)
	locker2 := NewAppLocker(tmp, testLockFile)

	require.NoError(t, locker1.TryLock())
	assert.Error(t, locker2.TryLock())
	require.NoError(t, locker1.Unlock())
	require.NoError(t, locker2.TryLock())
	assert.Error(t, locker1.TryLock())
	require.NoError(t, locker2.Unlock())
}
