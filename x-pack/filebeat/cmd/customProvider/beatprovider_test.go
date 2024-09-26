// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package customProvider

import (
	"context"
	"fmt"
	"testing"
)

func TestBeatProvider(t *testing.T) {
	p := provider{}
	fmt.Println(p.Retrieve(context.Background(), "filebeat:/Users/khushijain/Documents/beats/x-pack/filebeat/filebeat.yml", nil))

}
