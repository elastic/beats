// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package amcache

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticamcacheapplication "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/amcache/elastic_amcache_application"
	elasticamcacheapplicationfile "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/amcache/elastic_amcache_application_file"
	elasticamcacheapplicationshortcut "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/amcache/elastic_amcache_application_shortcut"
	elasticamcachedevicepnp "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/amcache/elastic_amcache_device_pnp"
	elasticamcachedriverbinary "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/amcache/elastic_amcache_driver_binary"
	elasticamcachedriverpackage "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/amcache/elastic_amcache_driver_package"
	elasticamcacheapplicationsview "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/views/generated/amcache/elastic_amcache_applications_view"
)

func init() {
	state := tables.GetAmcacheState()

	elasticamcacheapplication.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elasticamcacheapplication.Result, error) {
		t := tables.GetAmcacheTableByName(tables.TableNameApplication)
		entries, err := state.GetCachedEntries(*t, filters.GetConstraintFilters(queryContext), log)
		if err != nil {
			return nil, err
		}
		return entriesToApplicationResults(entries)
	})

	elasticamcacheapplicationfile.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elasticamcacheapplicationfile.Result, error) {
		t := tables.GetAmcacheTableByName(tables.TableNameApplicationFile)
		entries, err := state.GetCachedEntries(*t, filters.GetConstraintFilters(queryContext), log)
		if err != nil {
			return nil, err
		}
		return entriesToApplicationFileResults(entries)
	})

	elasticamcacheapplicationshortcut.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elasticamcacheapplicationshortcut.Result, error) {
		t := tables.GetAmcacheTableByName(tables.TableNameApplicationShortcut)
		entries, err := state.GetCachedEntries(*t, filters.GetConstraintFilters(queryContext), log)
		if err != nil {
			return nil, err
		}
		return entriesToApplicationShortcutResults(entries)
	})

	elasticamcachedriverbinary.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elasticamcachedriverbinary.Result, error) {
		t := tables.GetAmcacheTableByName(tables.TableNameDriverBinary)
		entries, err := state.GetCachedEntries(*t, filters.GetConstraintFilters(queryContext), log)
		if err != nil {
			return nil, err
		}
		return entriesToDriverBinaryResults(entries)
	})

	elasticamcachedevicepnp.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elasticamcachedevicepnp.Result, error) {
		t := tables.GetAmcacheTableByName(tables.TableNameDevicePnp)
		entries, err := state.GetCachedEntries(*t, filters.GetConstraintFilters(queryContext), log)
		if err != nil {
			return nil, err
		}
		return entriesToDevicePnpResults(entries)
	})

	elasticamcachedriverpackage.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elasticamcachedriverpackage.Result, error) {
		t := tables.GetAmcacheTableByName(tables.TableNameDriverPackage)
		entries, err := state.GetCachedEntries(*t, filters.GetConstraintFilters(queryContext), log)
		if err != nil {
			return nil, err
		}
		return entriesToDriverPackageResults(entries)
	})

	elasticamcacheapplicationsview.RegisterHooksFunc(registerAmcacheViewHooks)
}

// registerAmcacheViewHooks registers the amcache applications view create/delete hooks and cleanup hook with the hook manager.
// All amcache glue (tables + view + cleanup) is isolated in this file.
func registerAmcacheViewHooks(hm *hooks.HookManager) {
	elasticamcacheapplicationsview.RegisterDefaultViewHook(hm)
	hm.Register(hooks.NewHook("CleanupAmcacheInstance", nil, cleanupInstanceHook, nil))
}

func cleanupInstanceHook(socket *string, log *logger.Logger, hookData any) error {
	state := tables.GetAmcacheState()
	state.Close()
	return nil
}

func entriesToApplicationResults(entries []tables.Entry) ([]elasticamcacheapplication.Result, error) {
	out := make([]elasticamcacheapplication.Result, 0, len(entries))
	for _, e := range entries {
		app := e.(*tables.ApplicationEntry)
		out = append(out, elasticamcacheapplication.Result{
			Timestamp:          app.Timestamp,
			DateTime:           app.DateTime,
			ProgramId:          app.ProgramId,
			ProgramInstanceId:  app.ProgramInstanceId,
			Name:               app.Name,
			Version:            app.Version,
			Publisher:          app.Publisher,
			Language:           app.Language,
			InstallDate:        app.InstallDate,
			Source:             app.Source,
			RootDirPath:        app.RootDirPath,
			HiddenArp:          app.HiddenArp,
			UninstallString:    app.UninstallString,
			RegistryKeyPath:    app.RegistryKeyPath,
			StoreAppType:       app.StoreAppType,
			InboxModernApp:     app.InboxModernApp,
			ManifestPath:       app.ManifestPath,
			PackageFullName:    app.PackageFullName,
			MsiPackageCode:     app.MsiPackageCode,
			MsiProductCode:     app.MsiProductCode,
			MsiInstallDate:     app.MsiInstallDate,
			BundleManifestPath: app.BundleManifestPath,
			UserSid:            app.UserSid,
			Sha1:               app.Sha1,
		})
	}
	return out, nil
}

func entriesToApplicationFileResults(entries []tables.Entry) ([]elasticamcacheapplicationfile.Result, error) {
	out := make([]elasticamcacheapplicationfile.Result, 0, len(entries))
	for _, e := range entries {
		app := e.(*tables.ApplicationFileEntry)
		out = append(out, elasticamcacheapplicationfile.Result{
			Timestamp:             app.Timestamp,
			DateTime:              app.DateTime,
			ProgramId:             app.ProgramId,
			FileId:                app.FileId,
			LowerCaseLongPath:     app.LowerCaseLongPath,
			Name:                  app.Name,
			OriginalFileName:      app.OriginalFileName,
			Publisher:             app.Publisher,
			Version:               app.Version,
			BinFileVersion:        app.BinFileVersion,
			BinaryType:            app.BinaryType,
			ProductName:           app.ProductName,
			ProductVersion:        app.ProductVersion,
			LinkDate:              app.LinkDate,
			BinProductVersion:     app.BinProductVersion,
			Size:                  app.Size,
			Language:              app.Language,
			Usn:                   app.Usn,
			AppxPackageFullName:   app.AppxPackageFullName,
			IsOsComponent:         app.IsOsComponent,
			AppxPackageRelativeId: app.AppxPackageRelativeId,
			Sha1:                  app.Sha1,
		})
	}
	return out, nil
}

func entriesToApplicationShortcutResults(entries []tables.Entry) ([]elasticamcacheapplicationshortcut.Result, error) {
	out := make([]elasticamcacheapplicationshortcut.Result, 0, len(entries))
	for _, e := range entries {
		app := e.(*tables.ApplicationShortcutEntry)
		out = append(out, elasticamcacheapplicationshortcut.Result{
			Timestamp:          app.Timestamp,
			DateTime:           app.DateTime,
			ShortcutPath:       app.ShortcutPath,
			ShortcutTargetPath: app.ShortcutTargetPath,
			ShortcutAumid:      app.ShortcutAumid,
			ShortcutProgramId:  app.ShortcutProgramId,
		})
	}
	return out, nil
}

func entriesToDriverBinaryResults(entries []tables.Entry) ([]elasticamcachedriverbinary.Result, error) {
	out := make([]elasticamcachedriverbinary.Result, 0, len(entries))
	for _, e := range entries {
		app := e.(*tables.DriverBinaryEntry)
		out = append(out, elasticamcachedriverbinary.Result{
			Timestamp:               app.Timestamp,
			DateTime:                app.DateTime,
			DriverName:              app.DriverName,
			Inf:                     app.Inf,
			DriverVersion:           app.DriverVersion,
			Product:                 app.Product,
			ProductVersion:          app.ProductVersion,
			WdfVersion:              app.WdfVersion,
			DriverCompany:           app.DriverCompany,
			DriverPackageStrongName: app.DriverPackageStrongName,
			Service:                 app.Service,
			DriverInBox:             app.DriverInBox,
			DriverSigned:            app.DriverSigned,
			DriverIsKernelMode:      app.DriverIsKernelMode,
			DriverId:                app.DriverId,
			DriverLastWriteTime:     app.DriverLastWriteTime,
			DriverType:              app.DriverType,
			DriverTimeStamp:         app.DriverTimeStamp,
			DriverCheckSum:          app.DriverCheckSum,
			ImageSize:               app.ImageSize,
		})
	}
	return out, nil
}

func entriesToDevicePnpResults(entries []tables.Entry) ([]elasticamcachedevicepnp.Result, error) {
	out := make([]elasticamcachedevicepnp.Result, 0, len(entries))
	for _, e := range entries {
		app := e.(*tables.DevicePnpEntry)
		out = append(out, elasticamcachedevicepnp.Result{
			Timestamp:               app.Timestamp,
			DateTime:                app.DateTime,
			Model:                   app.Model,
			Manufacturer:            app.Manufacturer,
			DriverName:              app.DriverName,
			ParentId:                app.ParentId,
			MatchingId:              app.MatchingID,
			Class:                   app.Class,
			ClassGuid:               app.ClassGuid,
			Description:             app.Description,
			Enumerator:              app.Enumerator,
			Service:                 app.Service,
			InstallState:            app.InstallState,
			DeviceState:             app.DeviceState,
			Inf:                     app.Inf,
			DriverVerDate:           app.DriverVerDate,
			InstallDate:             app.InstallDate,
			FirstInstallDate:        app.FirstInstallDate,
			DriverPackageStrongName: app.DriverPackageStrongName,
			DriverVerVersion:        app.DriverVerVersion,
			ContainerId:             app.ContainerId,
			ProblemCode:             app.ProblemCode,
			Provider:                app.Provider,
			DriverId:                app.DriverId,
			BusReportedDescription:  app.BusReportedDescription,
			HwId:                    app.HWID,
			ExtendedInfs:            app.ExtendedInfs,
			Compid:                  app.COMPID,
			StackId:                 app.STACKID,
			UpperClassFilters:       app.UpperClassFilters,
			LowerClassFilters:       app.LowerClassFilters,
			UpperFilters:            app.UpperFilters,
			LowerFilters:            app.LowerFilters,
			DeviceInterfaceClasses:  app.DeviceInterfaceClasses,
			LocationPaths:           app.LocationPaths,
		})
	}
	return out, nil
}

func entriesToDriverPackageResults(entries []tables.Entry) ([]elasticamcachedriverpackage.Result, error) {
	out := make([]elasticamcachedriverpackage.Result, 0, len(entries))
	for _, e := range entries {
		app := e.(*tables.DriverPackageEntry)
		out = append(out, elasticamcachedriverpackage.Result{
			Timestamp:    app.Timestamp,
			DateTime:     app.DateTime,
			ClassGuid:    app.ClassGuid,
			Class:        app.Class,
			Directory:    app.Directory,
			Date:         app.Date,
			Version:      app.Version,
			Provider:     app.Provider,
			SubmissionId: app.SubmissionId,
			DriverInBox:  app.DriverInBox,
			Inf:          app.Inf,
			FlightIds:    app.FlightIds,
			RecoveryIds:  app.RecoveryIds,
			IsActive:     app.IsActive,
			Hwids:        app.Hwids,
			Sysfile:      app.SYSFILE,
		})
	}
	return out, nil
}
