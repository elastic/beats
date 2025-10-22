// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"log"
	"time"
	"github.com/osquery/osquery-go"
)

type View struct {
	requiredTables  []string
	createViewQuery string
}

// AreTablesReady checks if all required tables are ready in osquery
func AreTablesReady(client *osquery.ExtensionManagerClient, tableNames []string) bool {
	for _, tableName := range tableNames {
		resp, err := client.Query(fmt.Sprintf("pragma table_info(%s);", tableName))
		if err != nil {
			log.Printf("Error checking for table %s: %s\n", tableName, err)
			return false
		}
		if len(resp.Response) == 0 {
			return false
		}
	}
	return true
}

func (v *View) CreateView(socket *string) error {
	client, err := osquery.NewClient(*socket, 2*time.Second)
	if err != nil {
		return fmt.Errorf("error creating osquery client: %w", err)
	}

	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Only try to create the view for 30 seconds
		if time.Since(startTime) > 30*time.Second {
			return fmt.Errorf("timeout waiting for required tables to be ready")
		}

		// Check if all required tables are ready
		if (!AreTablesReady(client, v.requiredTables)) {
			continue
		}

		// Create the view
		_, err := client.Query(v.createViewQuery)
		if err != nil {
			log.Printf("Error creating view %s: %s\n", v.createViewQuery, err)
			continue
		} else {
			// Successfully created the view
			log.Printf("Successfully created view with query: %s\n", v.createViewQuery)
			break
		}
	}
	return nil
}
