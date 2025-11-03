// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package views

import (
	"fmt"
	"time"

	"github.com/osquery/osquery-go"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type View struct {
	tableName       string
	requiredTables  []string
	createViewQuery string
	created         bool
}

func (v *View) Name() string {
	return v.tableName
}

func NewView(tableName string, requiredTables []string, createViewQuery string) *View {
	return &View{
		tableName:       tableName,
		requiredTables:  requiredTables,
		createViewQuery: createViewQuery,
		created:         false,
	}
}

// AreTablesReady checks if all required tables are ready in osquery
func AreTablesReady(client *osquery.ExtensionManagerClient, tableNames []string, log *logger.Logger) bool {
	for _, tableName := range tableNames {
		resp, err := client.Query(fmt.Sprintf("pragma table_info(%s);", tableName))
		if err != nil {
			log.Errorf("Error checking for table %s: %s\n", tableName, err)
			return false
		}
		if len(resp.Response) == 0 {
			log.Infof("Table %s is not ready", tableName)
			return false
		}
	}
	log.Infof("All tables %s are ready", tableNames)
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

	allViewsCreated := func(views []*View) bool {
		for _, view := range views {
			if !view.created {
				return false
			}
		}
		return true
	}

	for range ticker.C {
		if allViewsCreated(views) {
			log.Infof("All views created successfully")
			return nil
		}

		if time.Since(startTime) > 30*time.Second {
			break
		}

		for _, view := range views {
			if view.created {
				continue
			}

			if AreTablesReady(client, view.requiredTables, log) {
				_, err := client.Query(view.createViewQuery)
				if err != nil {
					log.Errorf("Error creating view %s: %s\n", view.createViewQuery, err)
					continue
				}
				view.created = true
				log.Infof("View %s created successfully", view.tableName)
			}
		}
	}
	return fmt.Errorf("timeout waiting for required tables to be ready")
}

func CreateView(socket *string, view *View, log *logger.Logger) error {
	views := []*View{view}
	return CreateViews(socket, views, log)
}
