// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hooks

import (
	"fmt"
	"time"

	"github.com/osquery/osquery-go"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// View is a struct that represents data required to create a view in osquery
type View struct {
	name            string
	requiredTables  []string
	createViewQuery string
}

// Name returns the name of the view
func (v *View) Name() string {
	return v.name
}

// NewView creates a new view
func NewView(name string, requiredTables []string, createViewQuery string) *View {
	return &View{
		name:            name,
		requiredTables:  requiredTables,
		createViewQuery: createViewQuery,
	}
}

// isTableReady checks if a table is ready in osquery
func isTableReady(client *osquery.ExtensionManagerClient, tableName string, log *logger.Logger) bool {
	resp, err := client.Query(fmt.Sprintf("pragma table_info(%s);", tableName))
	if err != nil {
		log.Errorf("error checking for table %s: %s\n", tableName, err)
		return false
	}
	if len(resp.Response) == 0 {
		log.Infof("table %s is not ready", tableName)
		return false
	}
	log.Infof("table %s is ready", tableName)
	return true
}

func allTablesReady(client *osquery.ExtensionManagerClient, requiredTables []string, log *logger.Logger) bool {
	for _, tableName := range requiredTables {
		if !isTableReady(client, tableName, log) {
			return false
		}
	}
	return true
}

// Create creates a view in osquery, it will wait for all required tables to be ready before creating the view
// it will return an error if the view cannot be created or if the required tables are not ready after 30 seconds
func (v *View) Create(socket *string, log *logger.Logger) error {
	client, err := osquery.NewClient(*socket, 2*time.Second)
	if err != nil {
		return fmt.Errorf("error creating osquery client: %w", err)
	}
	defer client.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		// check if all required tables are ready every second
		case <-ticker.C:
			if !allTablesReady(client, v.requiredTables, log) {
				log.Infof("all required tables are not ready for view %s", v.name)
				continue
			}

			// all tables are ready, create the view
			_, err = client.Query(v.createViewQuery)
			if err != nil {
				return fmt.Errorf("error creating view %s: %w", v.createViewQuery, err)
			}
			log.Infof("view %s created successfully", v.name)
			return nil

		// if all required tables are not ready after 30 seconds, return an error
		case <-time.After(30 * time.Second):
			return fmt.Errorf("timeout waiting for required tables to be ready for view %s", v.name)
		}
	}
}

func (v *View) Delete(socket *string, log *logger.Logger) error {
	client, err := osquery.NewClient(*socket, 2*time.Second)
	if err != nil {
		return fmt.Errorf("error creating osquery client: %w", err)
	}
	defer client.Close()

	_, err = client.Query(fmt.Sprintf("DROP VIEW IF EXISTS %s;", v.name))
	if err != nil {
		return fmt.Errorf("error deleting view %s: %w", v.name, err)
	}
	return nil
}
