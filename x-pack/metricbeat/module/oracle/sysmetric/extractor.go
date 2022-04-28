package sysmetric

import (
	"context"
	"database/sql"
)

// sysmetricExtractMethod contains the methods needed to extract the necessary information about the performance of the database
type sysmetricExtractMethod interface {
	sysmetricMetric(context.Context) ([]sysmetricMetric, error)
}

// extractedData contains the necessary system metric information. 
type extractedData struct {
	sysmetricMetrics        []sysmetricMetric
}

// sysmetricExtractor is the implementor of sysmetricExtractMethod. It's implementation are on different Go files
// which refers to the origin of the data for organization purposes.
type sysmetricExtractor struct {
	db *sql.DB
	patterns []string
}
