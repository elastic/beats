// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !requirefips

package collstats

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/mongodb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	mb.Registry.MustAddMetricSet("mongodb", "collstats", New,
		mb.WithHostParser(mongodb.ParseURL),
		mb.DefaultMetricSet(),
	)
}

// CollStatsOptions holds configuration options for collecting collection statistics
type CollStatsOptions struct {
	Scale int `config:"scale"` // Scale factor for size values (default: 1)
}

// CollectionInfo represents a collection in a database
type CollectionInfo struct {
	Database   string
	Collection string
	TopInfo    map[string]interface{} // Optional info from top command (mongod only)
}

// Metricset type defines all fields of the Metricset
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type Metricset struct {
	*mongodb.Metricset
	mongoVersion string           // cached MongoDB version
	options      CollStatsOptions // configuration options
}

// New creates a new instance of the Metricset
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := mongodb.NewMetricset(base)
	if err != nil {
		return nil, fmt.Errorf("could not create mongodb metricset: %w", err)
	}

	// Parse collstats-specific configuration
	var options CollStatsOptions
	if err := base.Module().UnpackConfig(&options); err != nil {
		return nil, fmt.Errorf("could not parse collstats config: %w", err)
	}

	// Set defaults
	if options.Scale <= 0 {
		options.Scale = 1 // no scaling; for example if Scale = 1024, then metrics come with the unit KiB
	}

	return &Metricset{
		Metricset:    ms,
		mongoVersion: "",
		options:      options,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *Metricset) Fetch(reporter mb.ReporterV2) error {
	client, err := mongodb.NewClient(m.Config, m.HostData().URI, m.Module().Config().Timeout, 0, m.Logger())
	if err != nil {
		return fmt.Errorf("could not create mongodb client: %w", err)
	}

	defer func() {
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			m.Logger().Warn("client disconnection did not happen gracefully")
		}
	}()

	// Get MongoDB version if not cached
	if m.mongoVersion == "" {
		version, err := getMongoDBVersion(client)
		if err != nil {
			m.Logger().Warnf("Failed to get MongoDB version, using legacy mode: %v", err)
			m.mongoVersion = "unknown"
		} else {
			m.mongoVersion = version
			m.Logger().Debugf("Detected MongoDB version: %s", version)
		}
	}

	// Try top command first (works on mongod), fall back to listCollections (works on mongos)
	collections, err := m.getCollectionsFromTop(client)
	if err != nil {
		return fmt.Errorf("top command failed (likely) mongos: %v", err)

		// NOTE(shmsr): This is a specialized feature that is supposed to be for mongos, we will be in adding it
		// after discussion in later commits. However, disabling the feature for now.
		//
		// m.Logger().Debugf("top command failed (likely mongos), falling back to listCollections: %v", err)
		// collections, err = m.getCollectionsList(client)
		// if err != nil {
		// 	return fmt.Errorf("failed to get collections using fallback method: %w", err)
		// }
	}

	collStatsErrGroup := &errgroup.Group{}
	collStatsErrGroup.SetLimit(10) // limit number of goroutines running at the same time

	for _, collInfo := range collections {
		collInfo := collInfo // make sure it works properly on older Go versions

		collStatsErrGroup.Go(func() error {
			database, collection := collInfo.Database, collInfo.Collection
			group := fmt.Sprintf("%s.%s", database, collection)
			m.Logger().Debugf("collstats: processing %s", group)

			// Use appropriate method based on MongoDB version
			collStats, err := m.fetchCollStatsWithVersion(client, database, collection)
			if err != nil {
				m.Logger().Debugf("collstats: fetch failed for %s.%s: %v", database, collection, err)
				reporter.Error(fmt.Errorf("fetching collStats failed: %w", err))

				// the error is captured by reporter. no need to return it (to avoid double reporting of the same error)
				return nil
			}

			// Create infoMap structure similar to what top command provides
			infoMap := map[string]interface{}{
				"stats": collStats,
			}

			// Include top command info if available (from getCollectionsFromTop)
			if collInfo.TopInfo != nil {
				for key, value := range collInfo.TopInfo {
					if key != "stats" { // don't override our collStats
						infoMap[key] = value
					}
				}
			}

			event, err := eventMapping(group, infoMap)
			if err != nil {
				m.Logger().Debugf("collstats: event mapping failed for %s.%s: %v", database, collection, err)
				reporter.Error(fmt.Errorf("mapping of the event data failed: %w", err))

				// the error is captured by reporter. no need to return it (to avoid double reporting of the same error)
				return nil
			}

			reporter.Event(mb.Event{
				MetricSetFields: event,
			})

			m.Logger().Debugf("collstats: completed %s.%s", database, collection)

			return nil
		})
	}

	if err := collStatsErrGroup.Wait(); err != nil {
		return fmt.Errorf("error processing mongodb collstats: %w", err)
	}
	return nil
}

// fetchCollStatsWithVersion selects the appropriate method based on MongoDB version
func (m *Metricset) fetchCollStatsWithVersion(client *mongo.Client, dbName, collectionName string) (map[string]interface{}, error) {
	// For MongoDB 6.2+, try aggregation first with fallback to command
	if isVersionAtLeast(m.mongoVersion, "6.2.0") {
		m.Logger().Debugf("collstats: using $collStats aggregation for %s.%s (scale=%d)", dbName, collectionName, m.options.Scale)
		stats, err := m.fetchCollStatsAggregation(client, dbName, collectionName)
		if err == nil {
			return m.applyOptionsToStats(stats)
		}
		m.Logger().Debugf("collstats: aggregation failed for %s.%s, falling back to collStats command: %v", dbName, collectionName, err)
	}

	// Use legacy command for older versions or as fallback
	m.Logger().Debugf("collstats: using collStats command for %s.%s (scale=%d)", dbName, collectionName, m.options.Scale)
	stats, err := m.fetchCollStatsCommand(client, dbName, collectionName)
	if err != nil {
		return nil, err
	}
	return m.applyOptionsToStats(stats)
}

// fetchCollStatsAggregation uses the $collStats aggregation stage (MongoDB 6.2+)
func (m *Metricset) fetchCollStatsAggregation(client *mongo.Client, dbName, collectionName string) (map[string]interface{}, error) {
	collection := client.Database(dbName).Collection(collectionName)
	ctx := context.Background()

	// Build $collStats stage dynamically (reference: mongosh capabilities)
	storageStatsOptions := bson.D{}
	if m.options.Scale > 1 {
		storageStatsOptions = append(storageStatsOptions, bson.E{Key: "scale", Value: m.options.Scale})
	}

	collStatsOptions := bson.D{{Key: "storageStats", Value: storageStatsOptions}}
	// Always request count (fast doc count path)
	collStatsOptions = append(collStatsOptions, bson.E{Key: "count", Value: bson.D{}})

	// Build aggregation pipeline with $collStats as first stage. The output structure differs
	// from the legacy collStats command (most fields are nested under storageStats). We will
	// flatten the result afterwards to keep the rest of the pipeline unchanged.
	pipeline := mongo.Pipeline{{{Key: "$collStats", Value: collStatsOptions}}}

	m.Logger().Debugf("collstats: running aggregation pipeline on %s.%s: %v", dbName, collectionName, pipeline)
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed for database=%s, collection=%s: %w", dbName, collectionName, err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("could not decode aggregation results: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results from collStats aggregation")
	}

	// Handle sharded collections by merging statistics from multiple shards
	// Flatten each result (handles both sharded and non-sharded outputs)
	for i := range results {
		results[i] = flattenAggregationResult(results[i])
	}

	if len(results) == 1 {
		return results[0], nil
	}

	mergedResult, err := mergeShardedCollStats(results)
	if err != nil {
		return nil, fmt.Errorf("failed to merge sharded collection stats: %w", err)
	}
	m.Logger().Debugf("collstats: merged sharded stats for %s.%s (shards=%d)", dbName, collectionName, len(results))
	return mergedResult, nil
}

// fetchCollStatsCommand uses the legacy collStats command (pre-6.2 and fallback)
func (m *Metricset) fetchCollStatsCommand(client *mongo.Client, dbName, collectionName string) (map[string]interface{}, error) {
	db := client.Database(dbName)

	// Build command with options
	command := bson.M{"collStats": collectionName}

	// Add scale parameter if not default
	if m.options.Scale != 1 {
		command["scale"] = m.options.Scale
	}

	m.Logger().Debugf("collstats: running collStats command on %s.%s: %v", dbName, collectionName, command)
	collStats := db.RunCommand(context.Background(), command)
	if err := collStats.Err(); err != nil {
		return nil, fmt.Errorf("collStats command failed: %w", err)
	}
	var statsRes map[string]interface{}
	if err := collStats.Decode(&statsRes); err != nil {
		return nil, fmt.Errorf("could not decode mongo response for database=%s, collection=%s: %w", dbName, collectionName, err)
	}

	return statsRes, nil
}

// applyOptionsToStats applies post-processing options to collected statistics
func (m *Metricset) applyOptionsToStats(stats map[string]interface{}) (map[string]interface{}, error) {
	if stats == nil {
		return stats, nil
	}

	// NOTE: Intentionally exclude shards.* and indexSizes.* for now.
	// This keeps parity with the legacy collstats output and avoids emitting many
	// dynamic sub-fields. If/when needed, we can add explicit support later.
	//
	// If these appear in server responses (e.g., legacy collStats on mongos), drop
	// them here to keep the output consistent.
	delete(stats, "shards")
	delete(stats, "indexSizes")

	// We rely on server-side scaling only; do NOT re-scale client side to avoid double scaling.
	// Ensure avgObjSize remains untouched (server always reports real value).
	return stats, nil
}

// mergeShardedCollStats merges collection statistics from multiple shards
func mergeShardedCollStats(shardResults []map[string]interface{}) (map[string]interface{}, error) {
	if len(shardResults) == 0 {
		return nil, fmt.Errorf("no shard results to merge")
	}

	// Start with the first shard's result as the base
	merged := make(map[string]interface{})
	for key, value := range shardResults[0] {
		merged[key] = value
	}

	// Fields that should be summed across shards
	sumFields := []string{
		"count", "size", "storageSize", "totalIndexSize", "totalSize",
		// indexSizes.* intentionally excluded (not collected currently)
		"numOrphanDocs",
	}

	// Fields that should be averaged across shards (weighted by count if available)
	avgFields := []string{
		"avgObjSize",
	}

	// Fields that should be taken from the maximum across shards
	maxFields := []string{
		"maxSize", "max",
	}

	// Fields that represent shard-specific information to be removed from merged result
	shardSpecificFields := []string{
		"shard", "host", "localTime",
	}

	// Initialize counters for summable fields
	sums := make(map[string]float64)
	counts := make(map[string]int)
	maxValues := make(map[string]float64)
	totalDocCount := float64(0)

	// Process each shard result
	for _, shardResult := range shardResults {

		// Sum the summable fields
		for _, field := range sumFields {
			if value, exists := shardResult[field]; exists {
				if numValue, ok := convertToFloat64(value); ok {
					sums[field] += numValue
					counts[field]++
				}
			}
		}

		// Track max values
		for _, field := range maxFields {
			if value, exists := shardResult[field]; exists {
				if numValue, ok := convertToFloat64(value); ok {
					if numValue > maxValues[field] {
						maxValues[field] = numValue
					}
				}
			}
		}

		// Track document count for averaging
		if count, exists := shardResult["count"]; exists {
			if numCount, ok := convertToFloat64(count); ok {
				totalDocCount += numCount
			}
		}
	}

	// Apply summed values to merged result
	for field, sum := range sums {
		if counts[field] > 0 {
			// Convert back to appropriate type (int64 for counts, float64 for sizes)
			switch field {
			case "count", "numOrphanDocs":
				merged[field] = int64(sum)
			default:
				merged[field] = sum
			}
		}
	}

	// Apply max values
	for field, maxVal := range maxValues {
		merged[field] = maxVal
	}

	// Calculate weighted averages
	for _, field := range avgFields {
		if totalDocCount == 0 {
			// Align with mongosh behavior: set 0 when there are no documents across shards
			merged[field] = 0
			continue
		}

		var weightedSum float64
		for _, shardResult := range shardResults {
			avgValue, exists := shardResult[field]
			if !exists {
				continue
			}
			count, countExists := shardResult["count"]
			if !countExists {
				continue
			}
			numAvg, ok := convertToFloat64(avgValue)
			if !ok {
				continue
			}
			numCount, ok := convertToFloat64(count)
			if !ok {
				continue
			}
			weightedSum += numAvg * numCount
		}
		merged[field] = weightedSum / totalDocCount
	}

	// Add shard count information only (shards breakdown excluded)
	merged["shardCount"] = len(shardResults)

	// Remove shard-specific fields from the merged result
	for _, field := range shardSpecificFields {
		delete(merged, field)
	}

	return merged, nil
}

// convertToFloat64 safely converts various numeric types to float64
func convertToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

// flattenAggregationResult flattens the $collStats aggregation output (storageStats sub-document)
// to resemble the legacy collStats command result expected by existing mapping logic.
func flattenAggregationResult(result map[string]interface{}) map[string]interface{} {
	if result == nil {
		return result
	}

	storageStatsRaw, ok := result["storageStats"]
	if !ok {
		return result // already flat (legacy or unexpected format)
	}

	storageStats, ok := storageStatsRaw.(map[string]interface{})
	if !ok {
		return result
	}

	// Copy selected known fields up if not already present.
	keysToLift := []string{
		"size", "count", "avgObjSize", "storageSize", "totalIndexSize", "totalSize",
		"max", "maxSize", "nindexes", "indexDetails" /* indexSizes excluded */, "scaleFactor",
		// Newly lifted optional fields
		"freeStorageSize", "capped", "numOrphanDocs",
	}
	for _, k := range keysToLift {
		if _, exists := result[k]; !exists {
			if v, ok2 := storageStats[k]; ok2 {
				result[k] = v
			}
		}
	}

	// Some deployments provide count only via storageStats; ensure top-level count.
	if _, exists := result["count"]; !exists {
		if v, ok := storageStats["count"]; ok {
			result["count"] = v
		}
	}

	// Add scaleFactor if absent
	// MongoDB 6.2+ $collStats aggregation with scale parameter includes scaleFactor in the output
	if _, exists := result["scaleFactor"]; !exists {
		// Check if scaleFactor exists in storageStats (when scale is used in aggregation)
		if sf, ok := storageStats["scaleFactor"]; ok {
			result["scaleFactor"] = sf
		} else {
			// Default to 1 if no scale was applied
			result["scaleFactor"] = 1
		}
	}

	return result
}

// getMongoDBVersion retrieves the MongoDB server version
func getMongoDBVersion(client *mongo.Client) (string, error) {
	db := client.Database("admin")
	result := db.RunCommand(context.Background(), bson.M{"buildInfo": 1})
	if err := result.Err(); err != nil {
		return "", fmt.Errorf("buildInfo command failed: %w", err)
	}

	var buildInfo map[string]interface{}
	if err := result.Decode(&buildInfo); err != nil {
		return "", fmt.Errorf("could not decode buildInfo: %w", err)
	}

	version, ok := buildInfo["version"]
	if !ok {
		return "", fmt.Errorf("version field not found in buildInfo")
	}

	versionStr, ok := version.(string)
	if !ok {
		return "", fmt.Errorf("version field not a valid string: %v (%T)", version, version)
	}

	return versionStr, nil
}

// isVersionAtLeast checks if the current version is at least the target version
func isVersionAtLeast(current, target string) bool {
	// Handle unknown or error cases
	if current == "" || current == "unknown" {
		return false
	}

	currentParts := parseVersion(current)
	targetParts := parseVersion(target)

	// Compare major, minor, patch
	for i := 0; i < 3 && i < len(currentParts) && i < len(targetParts); i++ {
		if currentParts[i] > targetParts[i] {
			return true
		}
		if currentParts[i] < targetParts[i] {
			return false
		}
	}

	return true
}

// parseVersion extracts major, minor, patch numbers from version string
func parseVersion(version string) []int {
	// See: https://www.mongodb.com/docs/manual/reference/command/buildInfo/#mongodb-data-buildInfo.version
	// """
	// This string will take the format <major>.<minor>.<patch> in the case of a release, but development
	// builds may contain additional information.
	// """
	//
	// Remove any pre-release or build metadata (e.g., "6.2.0-rc1" -> "6.2.0")
	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		version = version[:idx]
	}

	parts := strings.Split(version, ".")
	result := make([]int, 0, 3)

	for i, part := range parts {
		if i >= 3 {
			break
		}
		if num, err := strconv.Atoi(part); err == nil {
			result = append(result, num)
		} else {
			result = append(result, 0)
		}
	}

	// Pad with zeros if needed
	for len(result) < 3 {
		result = append(result, 0)
	}

	return result
}

// getCollectionsFromTop gets collections using the top command (mongod only)
func (m *Metricset) getCollectionsFromTop(client *mongo.Client) ([]CollectionInfo, error) {
	// This info is only stored in 'admin' database
	db := client.Database("admin")
	res := db.RunCommand(context.Background(), bson.D{bson.E{Key: "top"}})
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("'top' command failed: %w", err)
	}

	var result map[string]interface{}
	if err := res.Decode(&result); err != nil {
		return nil, fmt.Errorf("could not decode mongo response: %w", err)
	}

	if _, ok := result["totals"]; !ok {
		return nil, errors.New("collection 'totals' key not found in mongodb response")
	}

	totals, ok := result["totals"].(map[string]interface{})
	if !ok {
		return nil, errors.New("collection 'totals' is not a map")
	}

	var collections []CollectionInfo

	for group, info := range totals {
		if group == "note" {
			continue
		}

		infoMap, ok := info.(map[string]interface{})
		if !ok {
			m.Logger().Debugf("unexpected data returned by mongodb for group %s", group)
			continue
		}

		names, err := splitKey(group)
		if err != nil {
			m.Logger().Debugf("splitKey failed for group=%q: %v", group, err)
			continue
		}

		if len(names) != 2 {
			m.Logger().Debugf("invalid collection key format: %s", group)
			continue
		}

		database, collection := names[0], names[1]

		collections = append(collections, CollectionInfo{
			Database:   database,
			Collection: collection,
			TopInfo:    infoMap, // Include original top command info
		})
	}

	m.Logger().Debugf("Found %d collections from top command", len(collections))
	return collections, nil
}

// // getCollectionsList gets all collections from all databases (mongos compatible)
// // See: https://www.mongodb.com/docs/manual/reference/command/top/
// // The 'top' command must be run against a mongod instance and running top against a mongos
// // instance returns an error and hence a mongos compatible implementation is required.
// func (m *Metricset) getCollectionsList(client *mongo.Client) ([]CollectionInfo, error) {
// 	// Get list of database names
// 	dbNames, err := client.ListDatabaseNames(context.Background(), bson.D{})
// 	if err != nil {
// 		return nil, fmt.Errorf("could not retrieve database names: %w", err)
// 	}

// 	var collections []CollectionInfo

// 	for _, dbName := range dbNames {
// 		db := client.Database(dbName)
// 		collNames, err := db.ListCollectionNames(context.Background(), bson.D{})
// 		if err != nil {
// 			m.Logger().Debugf("Failed to list collections for database %s: %v", dbName, err)
// 			continue
// 		}

// 		for _, collName := range collNames {
// 			collections = append(collections, CollectionInfo{
// 				Database:   dbName,
// 				Collection: collName,
// 				TopInfo:    nil, // No top info available when using listCollections
// 			})
// 		}
// 	}

// 	m.Logger().Debugf("Found %d collections across %d databases using listCollections", len(collections), len(dbNames))
// 	return collections, nil
// }
