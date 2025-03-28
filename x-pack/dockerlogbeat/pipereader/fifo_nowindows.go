// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package pipereader

import (
	"context"
	"io"
	"os"

	"github.com/containerd/fifo"
)

func openFifo(ctx context.Context, file string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return fifo.OpenFifo(ctx, file, flag, perm)
}
