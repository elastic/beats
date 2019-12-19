// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureeventhub

import "os"

var (
	azureConfig = azureInputConfig{
		SAKey:            os.Getenv("STORAGE_ACCOUNT_NAME"),
		SAName:           os.Getenv("STORAGE_ACCOUNT_KEY"),
		SAContainer:      ephContainerName,
		ConnectionString: os.Getenv("EVENTHUB_CONNECTION_STRING"),
		ConsumerGroup:    os.Getenv("EVENTHUB_CONSUMERGROUP"),
	}
)
