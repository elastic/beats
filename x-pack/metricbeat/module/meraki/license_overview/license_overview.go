package license_overview

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(meraki.ModuleName, "license_overview", New)
}

type config struct {
	BaseURL       string   `config:"apiBaseURL"`
	ApiKey        string   `config:"apiKey"`
	DebugMode     string   `config:"apiDebugMode"`
	Organizations []string `config:"organizations"`
	// todo: device filtering?
}

func defaultConfig() *config {
	return &config{
		BaseURL:   "https://api.meraki.com",
		DebugMode: "false",
	}
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	logger        *logp.Logger
	client        *meraki_api.Client
	organizations []string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The meraki license_overview metricset is beta.")

	logger := logp.NewLogger(base.FullyQualifiedName())

	config := defaultConfig()
	if err := base.Module().UnpackConfig(config); err != nil {
		return nil, err
	}

	logger.Debugf("loaded config: %v", config)
	client, err := meraki_api.NewClientWithOptions(config.BaseURL, config.ApiKey, config.DebugMode, "Metricbeat Elastic")
	if err != nil {
		logger.Error("creating Meraki dashboard API client failed: %w", err)
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		logger:        logger,
		client:        client,
		organizations: config.Organizations,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	for _, org := range m.organizations {

		cotermLicenses, perDeviceLicenses, systemsManagerLicense, err := getLicenseStates(m.client, org)
		if err != nil {
			return err
		}

		reportLicenseMetrics(reporter, org, cotermLicenses, perDeviceLicenses, systemsManagerLicense)
	}

	return nil
}

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
		meraki.ReportMetricsForOrganization(reporter, organizationID, cotermLicenseMetrics)
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
		meraki.ReportMetricsForOrganization(reporter, organizationID, perDeviceLicenseMetrics)
	}

	if systemsManagerLicense != nil {

		meraki.ReportMetricsForOrganization(reporter, organizationID, []mapstr.M{
			{
				"license.systems_manager.active_seats":            systemsManagerLicense.ActiveSeats,
				"license.systems_manager.orgwideenrolled_devices": systemsManagerLicense.OrgwideEnrolledDevices,
				"license.systems_manager.total_seats":             systemsManagerLicense.TotalSeats,
				"license.systems_manager.unassigned_seats":        systemsManagerLicense.UnassignedSeats,
			},
		})
	}
}
