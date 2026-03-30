// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// generate the application id map
//go:generate go run ./generate

package jumplists

import (
	"context"
	"errors"
	"math"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/interfaces"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	jumpliststypes "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/jumplists"
	elasticjumplists "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/jumplists/elastic_jumplists"
)

func init() {
	elasticjumplists.RegisterGenerateFunc(getResults)
}

type jumplistType string

const (
	jumplistTypeCustom    jumplistType = "custom"
	jumplistTypeAutomatic jumplistType = "automatic"
)

// jumplistMeta is metadata shared by every entry from one jump list file.
type jumplistMeta struct {
	*jumpliststypes.ApplicationID
	*jumpliststypes.UserProfile
	*jumpliststypes.JumplistMeta
}

// jumplistEntry is a single entry in a jump list.
type jumplistEntry struct {
	*DestListEntry
	*Lnk
}

// jumplist holds entries from one jump list source file.
type jumplist struct {
	*jumplistMeta
	entries []*jumplistEntry
}

// jumplistRow is one emitted row.
type jumplistRow struct {
	*jumplistMeta
	*jumplistEntry
}

// toRows converts a jump list to row objects.
func (j *jumplist) toRows() []jumplistRow {
	var rows []jumplistRow
	for _, entry := range j.entries {
		rows = append(rows, jumplistRow{
			jumplistMeta:  j.jumplistMeta,
			jumplistEntry: entry,
		})
	}
	return rows
}

// matchesFilters is a helper function that checks if a row matches the given filters.
func matchesFilters(row jumplistRow, filters []filters.Filter) bool {
	for _, filter := range filters {
		if !filter.Matches(row) {
			return false
		}
	}
	return true
}

type ClientInterface interface {
	interfaces.QueryExecutor
}

// getAllJumplists is a helper function that gets all the jumplists for all the user profiles.
func getAllJumplists(log *logger.Logger, client ClientInterface) ([]*jumplist, error) {
	var jumplists []*jumplist

	userProfiles, err := getUserProfiles(log, client)
	if err != nil {
		return nil, err
	}
	for _, userProfile := range userProfiles {
		jumplists = append(jumplists, userProfile.getJumplists(log)...)
	}

	return jumplists, nil
}

func getResults(_ context.Context, queryContext table.QueryContext, log *logger.Logger, resilientClient *client.ResilientClient) ([]elasticjumplists.Result, error) {
	if resilientClient == nil {
		return nil, errors.New("jumplists client is not configured")
	}

	jumplists, err := getAllJumplists(log, resilientClient)
	if err != nil {
		return nil, err
	}

	var results []elasticjumplists.Result
	constraintFilters := filters.GetConstraintFilters(queryContext)
	for _, jumpList := range jumplists {
		for _, row := range jumpList.toRows() {
			if matchesFilters(row, constraintFilters) {
				results = append(results, jumplistRowToResult(row))
			}
		}
	}
	return results, nil
}

func jumplistRowToResult(row jumplistRow) elasticjumplists.Result {
	result := elasticjumplists.Result{}

	result.ApplicationID = row.ApplicationID
	result.UserProfile = row.UserProfile
	result.JumplistMeta = row.JumplistMeta

	if row.DestListEntry != nil {
		result.DestListEntry = &jumpliststypes.DestListEntry{
			Hostname:              row.DestListEntry.Hostname,
			EntryNumber:           row.DestListEntry.EntryNumber,
			LastModifiedTime:      row.DestListEntry.LastModifiedTime,
			IsPinned:              row.DestListEntry.PinStatus,
			InteractionCount:      row.DestListEntry.InteractionCount,
			DestEntryPath:         row.DestListEntry.Path,
			DestEntryPathResolved: row.DestListEntry.ResolvedPath,
			MacAddress:            row.DestListEntry.MacAddress,
			CreationTime:          row.DestListEntry.CreationTime,
		}
	}

	if row.Lnk != nil {
		fileSize := int32(row.Lnk.FileSize)
		if row.Lnk.FileSize > math.MaxInt32 {
			fileSize = math.MaxInt32
		}

		volumeLabelOffset := int32(row.Lnk.VolumeLabelOffset)
		if row.Lnk.VolumeLabelOffset > math.MaxInt32 {
			volumeLabelOffset = math.MaxInt32
		}

		result.LnkMetadata = &jumpliststypes.LnkMetadata{
			LocalPath:              row.Lnk.LocalPath,
			FileSize:               fileSize,
			HotKey:                 row.Lnk.HotKey,
			IconIndex:              row.Lnk.IconIndex,
			ShowWindow:             row.Lnk.ShowWindow,
			IconLocation:           row.Lnk.IconLocation,
			CommandLineArguments:   row.Lnk.CommandLineArguments,
			TargetModificationTime: row.Lnk.TargetModificationDate,
			TargetLastAccessedTime: row.Lnk.TargetLastAccessedDate,
			TargetCreationTime:     row.Lnk.TargetCreationDate,
			VolumeSerialNumber:     row.Lnk.VolumeSerialNumber,
			VolumeType:             row.Lnk.VolumeType,
			VolumeLabel:            row.Lnk.VolumeLabel,
			VolumeLabelOffset:      volumeLabelOffset,
			Name:                   row.Lnk.Name,
		}
	}

	return result
}
