package device_health

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func getNetworkApplianceVPNSiteToSite(client *meraki_api.Client, networks *meraki_api.ResponseOrganizationsGetOrganizationNetworks) (map[NetworkID]*meraki_api.ResponseApplianceGetNetworkApplianceVpnSiteToSiteVpn, error) {

	networkVPNSiteToSites := make(map[NetworkID]*meraki_api.ResponseApplianceGetNetworkApplianceVpnSiteToSiteVpn)

	for _, network := range *networks {

		networkVPNSiteToSite, res, err := client.Appliance.GetNetworkApplianceVpnSiteToSiteVpn(network.ID)
		if err != nil {
			//Error: "This endpoint only supports MX networks"
			// We just swallow this error but do not append to the list
			if !(strings.Contains(string(res.Body()), "MX network")) {
				//Any other problem we are going to return an error
				return nil, fmt.Errorf("Appliance.GetNetworkApplianceVpnSiteToSiteVpn failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
			}
		} else {
			networkVPNSiteToSites[NetworkID(network.ID)] = networkVPNSiteToSite

		}

	}

	return networkVPNSiteToSites, nil
}

func reportNetwrokApplianceVPNSiteToSite(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, networkVPNSiteToSites map[NetworkID]*meraki_api.ResponseApplianceGetNetworkApplianceVpnSiteToSiteVpn) {
	metrics := []mapstr.M{}

	for network_id, networkVPNSiteToSite := range networkVPNSiteToSites {
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
				metric["network.appliance.vpn.site_to_site.network_id"] = network_id
				metric["network.appliance.vpn.site_to_site.mode"] = networkVPNSiteToSite.Mode

				if networkVPNSiteToSite.Mode != "none" {
					if networkVPNSiteToSite.Hubs != nil {
						for k, hub := range *networkVPNSiteToSite.Hubs {
							metric[fmt.Sprintf("network.appliance.vpn.site_to_site.hub.%d.hub_id", k)] = hub.HubID
							metric[fmt.Sprintf("network.appliance.vpn.site_to_site.hub.%d.use_default_route", k)] = *hub.UseDefaultRoute
						}
					}

					if networkVPNSiteToSite.Subnets != nil {
						for k, subnet := range *networkVPNSiteToSite.Subnets {
							metric[fmt.Sprintf("network.appliance.vpn.site_to_site.subnet.%d.local_subnet", k)] = subnet.LocalSubnet
							metric[fmt.Sprintf("network.appliance.vpn.site_to_site.subnet.%d.use_vpn", k)] = *subnet.UseVpn
						}
					}
				}
				metrics = append(metrics, metric)
			}
		}

	}

	ReportMetricsForOrganization(reporter, organizationID, metrics)

}
