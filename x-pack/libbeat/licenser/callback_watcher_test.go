// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallbackWatcher(t *testing.T) {
	t.Run("when no callback is set do not execute anything", func(t *testing.T) {
		w := &CallbackWatcher{}
		w.OnNewLicense(License{})
		w.OnManagerStopped()
	})

	t.Run("proxy call to callback function", func(t *testing.T) {
		c := 0
		w := &CallbackWatcher{
			New:     func(license License) { c++ },
			Stopped: func() { c++ },
		}
		w.OnNewLicense(License{})
		w.OnManagerStopped()
		assert.Equal(t, 2, c)
	})
}
