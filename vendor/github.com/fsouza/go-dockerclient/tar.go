// Copyright 2014 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"fmt"
	"io"
)

func createTarStream(srcPath, dockerfilePath string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}
