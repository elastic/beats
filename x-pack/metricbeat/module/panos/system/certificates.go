// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getCertificateEvents(m *MetricSet) ([]mb.Event, error) {
	query := "<show><sslmgr-store><config-certificate-info></config-certificate-info></sslmgr-store></show>"
	var response CertificateResponse

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	events := formatCertificateEvents(m, response.Result)

	return events, nil
}

func formatCertificateEvents(m *MetricSet, input string) []mb.Event {
	currentTime := time.Now()

	certificates := parseCertificates(input)
	events := make([]mb.Event, 0, len(certificates))

	for _, certificate := range certificates {
		event := mb.Event{
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
				"@Timestamp":      currentTime,
			}}

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
