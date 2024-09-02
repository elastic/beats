package certificates

import (
	"encoding/xml"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panos"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/PaloAltoNetworks/pango"
)

const (
	metricsetName = "certificates"
	vsys          = ""
	query         = "<show><sslmgr-store><config-certificate-info></config-certificate-info></sslmgr-store></show>"
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
	cfgwarn.Beta("The panos certificates metricset is beta.")

	config := panos.Config{}
	logger := logp.NewLogger(base.FullyQualifiedName())

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

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
	log.Debugf("panos certificates.Fetch initialized client")

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

	events := getEvents(m, response.Result)

	for _, event := range events {
		report.Event(event)
	}

	return nil
}

func getEvents(m *MetricSet, input string) []mb.Event {
	currentTime := time.Now()

	certificates := parseCertificates(input)
	events := make([]mb.Event, 0, len(certificates))

	for _, certificate := range certificates {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"cert_name":           certificate.CertName,
				"issuer":              certificate.Issuer,
				"issuer_subject_hash": certificate.IssuerSubjectHash,
				"issuer_key_hash":     certificate.IssuerKeyHash,
				"db_type":             certificate.DBType,
				"db_exp_date":         certificate.DBExpDate,
				"db_rev_date":         certificate.DBRevDate,
				"db_serial_no":        certificate.DBSerialNo,
				"db_file":             certificate.DBFile,
				"db_name":             certificate.DBName,
				"db_status":           certificate.DBStatus,
			}}
		event.Timestamp = currentTime
		event.RootFields = mapstr.M{
			"observer.ip": m.config.HostIp,
			"host.ip":     m.config.HostIp,
		}
		events = append(events, event)
	}

	return events
}

func parseCertificates(input string) []Certificate {
	lines := strings.Split(input, "\n")
	pattern := `^[0-9A-Fa-f]{1,40}:[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$`
	regex := regexp.MustCompile(pattern)
	var certificates []Certificate
	var currentSN Certificate

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if regex.MatchString(line) {
			if currentSN.CertName != "" {
				certificates = append(certificates, currentSN)
				currentSN = Certificate{}
			}
			currentSN.CertName = line
		} else if strings.HasPrefix(line, "issuer:") {
			currentSN.Issuer = strings.TrimPrefix(line, "issuer: ")
		} else if strings.HasPrefix(line, "issuer-subjecthash:") {
			currentSN.IssuerSubjectHash = strings.TrimPrefix(line, "issuer-subjecthash: ")
		} else if strings.HasPrefix(line, "issuer-keyhash:") {
			currentSN.IssuerKeyHash = strings.TrimPrefix(line, "issuer-keyhash: ")
			if strings.HasPrefix(currentSN.IssuerKeyHash, "issuer-keyhash:") {
				currentSN.IssuerKeyHash = ""
			}
		} else if strings.HasPrefix(line, "db-type:") {
			currentSN.DBType = strings.TrimPrefix(line, "db-type: ")
		} else if strings.HasPrefix(line, "db-exp-date:") {
			currentSN.DBExpDate = strings.TrimPrefix(line, "db-exp-date: ")
		} else if strings.HasPrefix(line, "db-rev-date:") {
			currentSN.DBRevDate = strings.TrimPrefix(line, "db-rev-date: ")
		} else if strings.HasPrefix(line, "db-serialno:") {
			currentSN.DBSerialNo = strings.TrimPrefix(line, "db-serialno: ")
		} else if strings.HasPrefix(line, "db-file:") {
			currentSN.DBFile = strings.TrimPrefix(line, "db-file: ")
		} else if strings.HasPrefix(line, "db-name:") {
			currentSN.DBName = strings.TrimPrefix(line, "db-name: ")
		} else if strings.HasPrefix(line, "db-status:") {
			currentSN.DBStatus = strings.TrimPrefix(line, "db-status: ")
		}
	}

	if currentSN.Issuer != "" {
		certificates = append(certificates, currentSN)
	}

	return certificates
}
