// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fileutil

import "os"

func FileExists(fp string) (ok bool, err error) {
	if _, err = os.Stat(fp); err == nil {
		ok = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return ok, err
}
