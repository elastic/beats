package globalprotect_sessions

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panos"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/PaloAltoNetworks/pango"
)

const (
	metricsetName = "globalprotect_sessions"
	vsys          = ""
	query         = "<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(panos.ModuleName, metricsetName, New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config panos.Config
	logger *logp.Logger
	client *pango.Firewall
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The panos globalprotect_sessions metricset is beta.")

	config := panos.Config{}
	logger := logp.NewLogger(base.FullyQualifiedName())

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	logger.Debugf("panos_licenses metricset config: %v", config)

	client := &pango.Firewall{Client: pango.Client{Hostname: config.HostIp, ApiKey: config.ApiKey}}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		logger:        logger,
		client:        client,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	log := m.Logger()
	var response Response

	// Initialize the client
	if err := m.client.Initialize(); err != nil {
		log.Error("Failed to initialize client: %s", err)
		return err
	}
	log.Infof("panos_licenses.Fetch initialized client")

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		log.Error("Error: %s", err)
		return err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		log.Error("Error: %s", err)
		return err
	}

	events := getEvents(m, response.Result.Sessions)

	for _, event := range events {
		report.Event(event)
	}

	return nil
}

func getEvents(m *MetricSet, sessions []Session) []mb.Event {
	events := make([]mb.Event, 0, len(sessions))

	currentTime := time.Now()

	for _, session := range sessions {
		event := mb.Event{MetricSetFields: mapstr.M{
			"domain":                 session.Domain,
			"is_local":               session.IsLocal,
			"username":               session.Username,
			"primary_username":       session.PrimaryUsername,
			"region_for_config":      session.RegionForConfig,
			"source_region":          session.SourceRegion,
			"computer":               session.Computer,
			"client":                 session.Client,
			"vpn_type":               session.VPNType,
			"host_id":                session.HostID,
			"app_version":            session.AppVersion,
			"virtual_ip":             session.VirtualIP,
			"virtual_ipv6":           session.VirtualIPv6,
			"public_ip":              session.PublicIP,
			"public_ipv6":            session.PublicIPv6,
			"tunnel_type":            session.TunnelType,
			"public_connection_ipv6": session.PublicConnectionIPv6,
			"client_ip":              session.ClientIP,
			"login_time":             session.LoginTime,
			"login_time_utc":         session.LoginTimeUTC,
			"lifetime":               session.Lifetime,
			"request_login":          session.RequestLogin,
			"request_get_config":     session.RequestGetConfig,
			"request_sslvpn_connect": session.RequestSSLVPNConnect,
		}}
		event.Timestamp = currentTime
		event.RootFields = mapstr.M{
			"observer.ip":     m.config.HostIp,
			"host.ip":         m.config.HostIp,
			"observer.vendor": "Palo Alto",
			"observer.type":   "firewall",
		}

		events = append(events, event)
	}

	return events
}
