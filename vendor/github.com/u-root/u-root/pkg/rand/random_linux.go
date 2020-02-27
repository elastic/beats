// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rand

import (
	"context"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

var defaultContextReader = &getrandomReader{}

var backupReader = &urandomReader{}

type getrandomReader struct {
	once   sync.Once
	backup bool
}

// ReadContext implements a cancelable read from /dev/urandom.
func (r *getrandomReader) ReadContext(ctx context.Context, b []byte) (int, error) {
	r.once.Do(func() {
		if _, err := unix.Getrandom(b, unix.GRND_NONBLOCK); err == syscall.ENOSYS {
			r.backup = true
		}
	})
	if r.backup {
		return backupReader.ReadContext(ctx, b)
	}

	for {
		// getrandom(2) with GRND_NONBLOCK uses the urandom number
		// source, but only returns numbers if the crng has been
		// initialized.
		//
		// This is preferrable to /dev/urandom, as /dev/urandom will
		// make up fake random numbers until the crng has been
		// initialized.
		n, err := unix.Getrandom(b, unix.GRND_NONBLOCK)
		if err == nil {
			return n, err
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		default:
			if err != nil && err != syscall.EAGAIN && err != syscall.EINTR {
				return n, err
			}
		}
	}
}
