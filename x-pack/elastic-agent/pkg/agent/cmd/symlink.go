// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"
)

const bash string = `#!/bin/sh
exec %s $@ %s
`

func createSymlink(oldPath, newPath string, argsOverrides ...string) error {
	args := strings.Join(argsOverrides, " ")
	fileContent := fmt.Sprintf(bash, newPath, args)
	return ioutil.WriteFile(oldPath, []byte(fileContent), 0750)
}
