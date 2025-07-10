// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package pipereader

import (
	"context"
	"errors"
	"io"
	"os"
)

func openFifo(_ context.Context, _ string, _ int, _ os.FileMode) (io.ReadWriteCloser, error) {
	return nil, errors.New("fifo is not supported on windows")
}
