// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"

	"github.com/elastic/beats/x-pack/functionbeat/provider/gcp"
)

func main() {
	ctx := context.Background()
	m := pubsub.Message{
		ID:   "string",
		Data: []byte("lovacska"),
		Attributes: map[string]string{
			"attr": "val",
		},
		PublishTime: time.Now(),
	}
	gcp.RunPubSub(ctx, m)
	fmt.Println("ba")
}
