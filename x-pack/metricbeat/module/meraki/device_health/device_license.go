package device_health

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/tommyers-elastic/dashboard-api-go/v3/sdk"
)

func reportOrganizationDeviceLicenses(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, orgLicenseBySerial *meraki_api.ResponseOrganizationsGetOrganizationLicenses) {

	metrics := []mapstr.M{}

	for _, license := range *orgLicenseBySerial {

		if device, ok := devices[Serial(license.DeviceSerial)]; ok {
			metric := mapstr.M{
				"device.address":                        device.Address,
				"device.firmware":                       device.Firmware,
				"device.imei":                           device.Imei,
				"device.lan_ip":                         device.LanIP,
				"device.location":                       device.Location,
				"device.mac":                            device.Mac,
				"device.model":                          device.Model,
				"device.name":                           device.Name,
				"device.network_id":                     device.NetworkID,
				"device.notes":                          device.Notes,
				"device.product_type":                   device.ProductType,
				"device.serial":                         device.Serial,
				"device.tags":                           device.Tags,
				"device.license.activation_date":        license.ActivationDate,
				"device.license.claim_date":             license.ClaimDate,
				"device.license.device_serial":          license.DeviceSerial,
				"device.license.duration_in_days":       license.DurationInDays,
				"device.license.expiration_date":        license.ExpirationDate,
				"device.license.head_license_id":        license.HeadLicenseID,
				"device.license.id":                     license.ID,
				"device.license.license_type":           license.LicenseType,
				"device.license.network_id":             license.NetworkID,
				"device.license.order_number":           license.OrderNumber,
				"device.license.seat_count":             license.SeatCount,
				"device.license.state":                  license.State,
				"device.license.total_duration_in_days": license.TotalDurationInDays,
				//"device.license.license_key":      license.LicenseKey, Not sure we want private key information on metric data in elastic
				//"device.license.11":    license.PermanentlyQueuedLicenses, //DEPRECATED List of permanently queued licenses attached to the license. Instead, use /organizations/{organizationId}/licenses?deviceSerial= to retrieved queued licenses for a given device.
			}

			metrics = append(metrics, metric)
		}
	}

	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
