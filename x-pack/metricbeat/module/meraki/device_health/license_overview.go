package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func getLicenseStates(client *meraki_api.Client, organizationID string) ([]*CoterminationLicense, []*PerDeviceLicense, *SystemsManagerLicense, error) {
	val, res, err := client.Organizations.GetOrganizationLicensesOverview(organizationID)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("GetOrganizationLicensesOverview failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	var cotermLicenses []*CoterminationLicense
	var perDeviceLicenses []*PerDeviceLicense
	var systemsManagerLicense *SystemsManagerLicense

	// co-termination license metrics (all devices share a single expiration date and status) are reported as counts of licenses per-device
	if val.LicensedDeviceCounts != nil {
		// i don't know why this isn't typed in the SDK - slightly worrying
		for device, count := range (*val.LicensedDeviceCounts).(map[string]interface{}) {
			cotermLicenses = append(cotermLicenses, &CoterminationLicense{
				DeviceModel:    device,
				Count:          count,
				ExpirationDate: val.ExpirationDate,
				Status:         val.Status,
			})
		}
	}

	// per-device license metrics (each device has its own expiration date and status) are reported counts of licenses per-state
	if val.States != nil {
		if val.States.Active != nil {
			perDeviceLicenses = append(perDeviceLicenses, &PerDeviceLicense{
				State: "Active",
				Count: val.States.Active.Count,
			})
		}

		if val.States.Expired != nil {
			perDeviceLicenses = append(perDeviceLicenses, &PerDeviceLicense{
				State: "Expired",
				Count: val.States.Expired.Count,
			})
		}

		if val.States.RecentlyQueued != nil {
			perDeviceLicenses = append(perDeviceLicenses, &PerDeviceLicense{
				State: "RecentlyQueued",
				Count: val.States.RecentlyQueued.Count,
			})
		}

		if val.States.Expiring != nil {
			if val.States.Expiring.Critical != nil {
				perDeviceLicenses = append(perDeviceLicenses, &PerDeviceLicense{
					State:                   "Expiring",
					ExpirationState:         "Critical",
					Count:                   val.States.Expiring.Critical.ExpiringCount,
					ExpirationThresholdDays: val.States.Expiring.Critical.ThresholdInDays,
				})
			}

			if val.States.Expiring.Warning != nil {
				perDeviceLicenses = append(perDeviceLicenses, &PerDeviceLicense{
					State:                   "Expiring",
					ExpirationState:         "Warning",
					Count:                   val.States.Expiring.Warning.ExpiringCount,
					ExpirationThresholdDays: val.States.Expiring.Warning.ThresholdInDays,
				})
			}
		}

		if val.States.Unused != nil {
			perDeviceLicenses = append(perDeviceLicenses, &PerDeviceLicense{
				State:                  "Unused",
				Count:                  val.States.Unused.Count,
				SoonestActivationDate:  val.States.Unused.SoonestActivation.ActivationDate,
				SoonestActivationCount: val.States.Unused.SoonestActivation.ToActivateCount,
			})
		}

		if val.States.UnusedActive != nil {
			perDeviceLicenses = append(perDeviceLicenses, &PerDeviceLicense{
				State:                 "UnusedActive",
				Count:                 val.States.UnusedActive.Count,
				OldestActivationDate:  val.States.UnusedActive.OldestActivation.ActivationDate,
				OldestActivationCount: val.States.UnusedActive.OldestActivation.ActiveCount,
			})
		}
	}

	if val.LicenseTypes != nil {
		for _, t := range *val.LicenseTypes {
			perDeviceLicenses = append(perDeviceLicenses, &PerDeviceLicense{
				State: "Unassigned",
				Type:  t.LicenseType,
				Count: t.Counts.Unassigned,
			})
		}
	}

	// per-device metrics also contain systems manager metrics
	if val.SystemsManager != nil {
		systemsManagerLicense = &SystemsManagerLicense{
			TotalSeats:             val.SystemsManager.Counts.TotalSeats,
			ActiveSeats:            val.SystemsManager.Counts.ActiveSeats,
			UnassignedSeats:        val.SystemsManager.Counts.UnassignedSeats,
			OrgwideEnrolledDevices: val.SystemsManager.Counts.OrgwideEnrolledDevices,
		}
	}

	return cotermLicenses, perDeviceLicenses, systemsManagerLicense, nil
}

func reportLicenseMetrics(reporter mb.ReporterV2, organizationID string, cotermLicenses []*CoterminationLicense, perDeviceLicenses []*PerDeviceLicense, systemsManagerLicense *SystemsManagerLicense) {
	if len(cotermLicenses) != 0 {
		cotermLicenseMetrics := []mapstr.M{}
		for _, license := range cotermLicenses {
			cotermLicenseMetrics = append(cotermLicenseMetrics, mapstr.M{
				"license.device_model":    license.DeviceModel,
				"license.expiration_date": license.ExpirationDate,
				"license.status":          license.Status,
				"license.count":           license.Count,
			})
		}
		ReportMetricsForOrganization(reporter, organizationID, cotermLicenseMetrics)
	}

	if len(perDeviceLicenses) != 0 {
		perDeviceLicenseMetrics := []mapstr.M{}
		for _, license := range perDeviceLicenses {
			perDeviceLicenseMetrics = append(perDeviceLicenseMetrics, mapstr.M{
				"license.state":                     license.State,
				"license.count":                     license.Count,
				"license.expiration_state":          license.ExpirationState,
				"license.expiration_threshold_days": license.ExpirationThresholdDays,
				"license.soonest_activation_date":   license.SoonestActivationDate,
				"license.soonest_activation_count":  license.SoonestActivationCount,
				"license.oldest_activation_date":    license.OldestActivationDate,
				"license.oldest_activation_count":   license.OldestActivationCount,
				"license.type":                      license.Type,
			})
		}
		ReportMetricsForOrganization(reporter, organizationID, perDeviceLicenseMetrics)
	}

	if systemsManagerLicense != nil {

		ReportMetricsForOrganization(reporter, organizationID, []mapstr.M{
			{
				"license.systems_manager.active_seats":            systemsManagerLicense.ActiveSeats,
				"license.systems_manager.orgwideenrolled_devices": systemsManagerLicense.OrgwideEnrolledDevices,
				"license.systems_manager.total_seats":             systemsManagerLicense.TotalSeats,
				"license.systems_manager.unassigned_seats":        systemsManagerLicense.UnassignedSeats,
			},
		})
	}
}
