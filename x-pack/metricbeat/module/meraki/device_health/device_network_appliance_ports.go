package device_health

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func getNetworkAppliancePorts(client *meraki_api.Client, networks *meraki_api.ResponseOrganizationsGetOrganizationNetworks) (map[NetworkID]*meraki_api.ResponseApplianceGetNetworkAppliancePorts, error) {

	networkPorts := make(map[NetworkID]*meraki_api.ResponseApplianceGetNetworkAppliancePorts)

	for _, network := range *networks {

		networkPort, res, err := client.Appliance.GetNetworkAppliancePorts(network.ID)
		if err != nil {
			//Error: "This endpoint only supports MX networks" or "VLANs are not enabled for this network"
			// We just ignore theses error but do not append to the list
			if !(strings.Contains(string(res.Body()), "VLANs are not enabled")) && !(strings.Contains(string(res.Body()), "MX networks")) {
				//Any other problem we are going to return an error
				return nil, fmt.Errorf("Appliance.GetNetworkAppliancePorts failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
			}
		} else {
			networkPorts[NetworkID(network.ID)] = networkPort

		}

	}

	return networkPorts, nil
}

func reportNetwrokAppliancePorts(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, networkPorts map[NetworkID]*meraki_api.ResponseApplianceGetNetworkAppliancePorts) {
	metrics := []mapstr.M{}

	for network_id, networkPort := range networkPorts {
		for _, device := range devices {
			if device.NetworkID == string(network_id) {
				metric := mapstr.M{
					"device.address":      device.Address,
					"device.firmware":     device.Firmware,
					"device.imei":         device.Imei,
					"device.lan_ip":       device.LanIP,
					"device.location":     device.Location,
					"device.mac":          device.Mac,
					"device.model":        device.Model,
					"device.name":         device.Name,
					"device.network_id":   device.NetworkID,
					"device.notes":        device.Notes,
					"device.product_type": device.ProductType,
					"device.serial":       device.Serial,
					"device.tags":         device.Tags,
				}

				for i, port := range *networkPort {
					metric[fmt.Sprintf("appliance.network.port.item_%d.number", i)] = port.Number
					metric[fmt.Sprintf("appliance.network.port.item_%d.enabled", i)] = port.Enabled
					metric[fmt.Sprintf("appliance.network.port.item_%d.type", i)] = port.Type
					metric[fmt.Sprintf("appliance.network.port.item_%d.drop_untagged_traffic", i)] = port.DropUntaggedTraffic
					metric[fmt.Sprintf("appliance.network.port.item_%d.vlan", i)] = port.VLAN
					metric[fmt.Sprintf("appliance.network.port.item_%d.allowed_vlans", i)] = port.AllowedVLANs
					metric[fmt.Sprintf("appliance.network.port.item_%d.access_policy", i)] = port.AccessPolicy
				}

				metrics = append(metrics, metric)

			}
		}

	}

	ReportMetricsForOrganization(reporter, organizationID, metrics)

}
