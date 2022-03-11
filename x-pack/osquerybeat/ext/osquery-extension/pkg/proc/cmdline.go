// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"bytes"
	"io/ioutil"
	"strings"
)

func ReadCmdLine(root string, pid string) (string, error) {
	fn := getProcAttr(root, pid, "cmdline")

	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return "", err
	}

	b = bytes.ReplaceAll(b, []byte{0}, []byte{' '})

	return strings.TrimSpace(string(b)), nil
}
