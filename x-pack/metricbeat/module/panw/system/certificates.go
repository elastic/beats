// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const certificatesQuery = "<show><sslmgr-store><config-certificate-info></config-certificate-info></sslmgr-store></show>"

func getCertificateEvents(m *MetricSet) ([]mb.Event, error) {

	var response CertificateResponse

	output, err := m.client.Op(certificatesQuery, vsys, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML response: %w", err)
	}

	if response.Result == "" {
		return nil, fmt.Errorf("empty result from XML response")
	}

	events, err := formatCertificateEvents(m, response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to format certificate events: %w", err)
	}

	return events, nil
}

func formatCertificateEvents(m *MetricSet, input string) ([]mb.Event, error) {
	timestamp := time.Now()

	certificates, err := parseCertificates(input)
	if err != nil {
		return nil, err
	}

	events := make([]mb.Event, 0, len(certificates))

	for _, certificate := range certificates {
		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"certificate.name":                certificate.CertName,
				"certificate.issuer":              certificate.Issuer,
				"certificate.issuer_subject_hash": certificate.IssuerSubjectHash,
				"certificate.issuer_key_hash":     certificate.IssuerKeyHash,
				"certificate.db_type":             certificate.DBType,
				"certificate.db_exp_date":         certificate.DBExpDate,
				"certificate.db_rev_date":         certificate.DBRevDate,
				"certificate.db_serial_no":        certificate.DBSerialNo,
				"certificate.db_file":             certificate.DBFile,
				"certificate.db_name":             certificate.DBName,
				"certificate.db_status":           certificate.DBStatus,
			},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
			},
		}

		events = append(events, event)
	}

	return events, nil
}

const (
	issuerPrefix            = "issuer: "
	issuerSubjectHashPrefix = "issuer-subjecthash: "
	issuerKeyHashPrefix     = "issuer-keyhash: "
	dbTypePrefix            = "db-type: "
	dbExpDatePrefix         = "db-exp-date: "
	dbRevDatePrefix         = "db-rev-date: "
	dbSerialNoPrefix        = "db-serialno: "
	dbFilePrefix            = "db-file: "
	dbNamePrefix            = "db-name: "
	dbStatusPrefix          = "db-status: "
)

func parseCertificates(input string) ([]Certificate, error) {
	lines := strings.Split(input, "\n")
	pattern := `^[0-9A-Fa-f]{1,40}:[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$`
	regex := regexp.MustCompile(pattern)

	certificates := make([]Certificate, 0)
	var currentSN Certificate

	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case regex.MatchString(line):
			if currentSN.CertName != "" {
				certificates = append(certificates, currentSN)
				currentSN = Certificate{}
			}
			currentSN.CertName = line
		case strings.HasPrefix(line, issuerPrefix):
			currentSN.Issuer = strings.TrimPrefix(line, issuerPrefix)
		case strings.HasPrefix(line, issuerSubjectHashPrefix):
			currentSN.IssuerSubjectHash = strings.TrimPrefix(line, issuerSubjectHashPrefix)
		case strings.HasPrefix(line, issuerKeyHashPrefix):
			currentSN.IssuerKeyHash = strings.TrimPrefix(line, issuerKeyHashPrefix)
			if strings.HasPrefix(currentSN.IssuerKeyHash, issuerKeyHashPrefix) {
				currentSN.IssuerKeyHash = ""
			}
		case strings.HasPrefix(line, dbTypePrefix):
			currentSN.DBType = strings.TrimPrefix(line, dbTypePrefix)
		case strings.HasPrefix(line, dbExpDatePrefix):
			currentSN.DBExpDate = strings.TrimPrefix(line, dbExpDatePrefix)
		case strings.HasPrefix(line, dbRevDatePrefix):
			currentSN.DBRevDate = strings.TrimPrefix(line, dbRevDatePrefix)
		case strings.HasPrefix(line, dbSerialNoPrefix):
			currentSN.DBSerialNo = strings.TrimPrefix(line, dbSerialNoPrefix)
		case strings.HasPrefix(line, dbFilePrefix):
			currentSN.DBFile = strings.TrimPrefix(line, dbFilePrefix)
		case strings.HasPrefix(line, dbNamePrefix):
			currentSN.DBName = strings.TrimPrefix(line, dbNamePrefix)
		case strings.HasPrefix(line, dbStatusPrefix):
			currentSN.DBStatus = strings.TrimPrefix(line, dbStatusPrefix)
		}
	}

	if currentSN.CertName != "" {
		certificates = append(certificates, currentSN)
	}

	if len(certificates) == 0 {
		return nil, errors.New("no valid certificates found")
	}

	return certificates, nil
}
