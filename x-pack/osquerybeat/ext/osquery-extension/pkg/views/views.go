// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package views

import (
	"fmt"
	"log"
	"time"

	"github.com/osquery/osquery-go"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type View struct {
	requiredTables  []string
	createViewQuery string
	created         bool
}

func NewView(requiredTables []string, createViewQuery string) *View {
	return &View{
		requiredTables:  requiredTables,
		createViewQuery: createViewQuery,
		created:         false,
	}
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

func CreateViews(socket *string, views []*View, log *logger.Logger) error {
	client, err := osquery.NewClient(*socket, 2*time.Second)
	if err != nil {
		return fmt.Errorf("error creating osquery client: %w", err)
	}

	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Track which views still need to be created
	pendingViews := make(map[int]bool)
	for i := range views {
		pendingViews[i] = true
	}

	for range ticker.C {
		// Only try to create views for 30 seconds
		if time.Since(startTime) > 30*time.Second {
			return fmt.Errorf("timeout waiting for required tables to be ready")
		}

		// Try to create each pending view
		for _, view := range views {
			if !view.created && AreTablesReady(client, view.requiredTables) {
				_, err := client.Query(view.createViewQuery)
				if err != nil {
					log.Errorf("Error creating view %s: %s\n", view.createViewQuery, err)
					continue
				}
				view.created = true
			}
		}

		// Exit if all views are created
		if len(pendingViews) == 0 {
			break
		}
	}
	return nil
}
