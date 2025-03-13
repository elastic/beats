// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ios

import (
	"flag"
	"os"
	"testing"

	"github.com/elastic/beats/v7/testing/testflag"
)

func TestMain(m *testing.M) {
	testflag.MustSetStrictPermsFalse()

	flag.Parse()

	os.Exit(m.Run())
}
