// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"io"
	"log"
	"strconv"
	"testing"

	"cloud.google.com/go/bigquery"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentMonth(t *testing.T) {
	currentMonth := getCurrentMonth()
	_, err := strconv.ParseInt(currentMonth, 0, 64)
	assert.NoError(t, err)
}

func TestGenerateQuery(t *testing.T) {
	log.SetOutput(io.Discard)

	query := generateQuery("my-table", "jan", "cost")
	log.Println(query)

	// verify that table name quoting is in effect
	assert.Contains(t, query, "`my-table`")
	// verify the group by is preserved
	assert.Contains(t, query, "GROUP BY\n\tinvoice_month,\n\tproject_id,\n\tproject_name,\n\tbilling_account_id,\n\tcost_type")
	// verify the order by is preserved
	assert.Contains(t, query, "ORDER BY\n\tinvoice_month ASC,\n\tproject_id ASC,\n\tproject_name ASC,\n\tbilling_account_id ASC,\n\tcost_type ASC")
}

func TestCreateTagsMap(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name     string
		tagsItem bigquery.Value
		want     []tag
	}{
		{
			name:     "valid tags",
			tagsItem: "tag1.a:value1,tag2.b:value2",
			want: []tag{
				{Key: "tag1.a", Value: "value1"},
				{Key: "tag2.b", Value: "value2"},
			},
		},
		{
			name:     "valid tags no values",
			tagsItem: "tag1:,tag2:",
			want: []tag{
				{Key: "tag1", Value: ""},
				{Key: "tag2", Value: ""},
			},
		},
		{
			name:     "no tags",
			tagsItem: "",
			want:     nil,
		},
		{
			name:     "invalid format",
			tagsItem: "tag1 value1,tag2 value2",
			want:     nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tags := createTags(tt.tagsItem)
			assert.Equal(tt.want, tags)
		})
	}
}

func TestIsDetailedTable(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		tableName string
		expected  bool
	}{
		// Positive test cases
		{"project-name-123456.dataset.gcp_billing_export_resource_v1_011702_58A742_BQB4E8", true},
		{"GCP_BILLING_EXPORT_RESOURCE_V1", true},
		{"prefix_gcp_billing_export_resource_v1_suffix", true},

		// Negative test cases
		{"project-name-123456.dataset.gcp_billing_export_v1_011702_58A742_BQB4E8", false},
		{"gcp_billing_export", false},
		{"random_table_name", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isDetailedTable(tc.tableName)
		assert.Equal(tc.expected, result, "Expected output for table name '%s' to be '%v', got '%v'", tc.tableName, tc.expected, result)
	}
}

func TestCreateEvents(t *testing.T) {
	assert := assert.New(t)

	t.Run("standard table", func(t *testing.T) {
		row := row{
			InvoiceMonth:       "202001",
			ProjectId:          "project-123456",
			ProjectName:        "My Project 12345",
			BillingAccountId:   "011702_58A742_BQB4E8",
			CostType:           "regular",
			SkuId:              "F449-33EC-A5EF",
			SkuDescription:     "E2 Instance Ram running in Americas",
			ServiceId:          "6F81-5844-456A",
			ServiceDescription: "Compute Engine",
			Tags:               "tag1:value1,tag2.a.b/c:value2,tag3:",
			TotalExact:         123.45,
		}

		date := getCurrentDate()
		id := generateEventID(date, row)

		expected := mb.Event{
			ID: id,
			RootFields: mapstr.M{
				"cloud.provider":     "gcp",
				"cloud.project.id":   "project-123456",
				"cloud.project.name": "My Project 12345",
				"cloud.account.id":   "011702_58A742_BQB4E8",
			},
			MetricSetFields: mapstr.M{
				"invoice_month":       "202001",
				"project_id":          "project-123456",
				"project_name":        "My Project 12345",
				"billing_account_id":  "011702_58A742_BQB4E8",
				"cost_type":           "regular",
				"total":               123.45,
				"sku_id":              "F449-33EC-A5EF",
				"sku_description":     "E2 Instance Ram running in Americas",
				"service_id":          "6F81-5844-456A",
				"service_description": "Compute Engine",
				"tags": []tag{
					{Key: "tag1", Value: "value1"},
					{Key: "tag2.a.b/c", Value: "value2"},
					{Key: "tag3", Value: ""},
				},
			},
		}
		event := createEvents(row, "project-123456.dataset.gcp_billing_export_v1_011702_58A742_BQB4E8", "project-123456")
		assert.Equal(expected, event)
	})

	t.Run("detailed table", func(t *testing.T) {
		row := row{
			InvoiceMonth:       "202001",
			ProjectId:          "project-123456",
			ProjectName:        "My Project 12345",
			BillingAccountId:   "011702_58A742_BQB4E8",
			CostType:           "regular",
			SkuId:              "F449-33EC-A5EF",
			SkuDescription:     "E2 Instance Ram running in Americas",
			ServiceId:          "6F81-5844-456A",
			ServiceDescription: "Compute Engine",
			Tags:               "tag1:value1,tag2.a.b/c:value2,tag3:",
			TotalExact:         123.45,
			EffectivePrice:     123.45,
		}

		date := getCurrentDate()
		id := generateEventID(date, row)

		expected := mb.Event{
			ID: id,
			RootFields: mapstr.M{
				"cloud.provider":     "gcp",
				"cloud.project.id":   "project-123456",
				"cloud.project.name": "My Project 12345",
				"cloud.account.id":   "011702_58A742_BQB4E8",
			},
			MetricSetFields: mapstr.M{
				"invoice_month":       "202001",
				"project_id":          "project-123456",
				"project_name":        "My Project 12345",
				"billing_account_id":  "011702_58A742_BQB4E8",
				"cost_type":           "regular",
				"total":               123.45,
				"sku_id":              "F449-33EC-A5EF",
				"sku_description":     "E2 Instance Ram running in Americas",
				"service_id":          "6F81-5844-456A",
				"service_description": "Compute Engine",
				"effective_price":     123.45,
				"tags": []tag{
					{Key: "tag1", Value: "value1"},
					{Key: "tag2.a.b/c", Value: "value2"},
					{Key: "tag3", Value: ""},
				},
			},
		}
		event := createEvents(row, "project-123456.dataset.gcp_billing_export_resource_v1_011702_58A742_BQB4E8", "project-123456")
		assert.Equal(expected, event)
	})
}
