package status

import (
	"bufio"
	"io"
	"regexp"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
)

// Map body to MapStr
func eventMapping(body io.ReadCloser) common.MapStr {

	var (
		totalAccess int
		totalKB     int
		uptime      int
	)

	var re *regexp.Regexp

	// Reads file line by line
	scanner := bufio.NewScanner(body)

	// See https://github.com/radoondas/apachebeat/blob/master/collector/status.go#L114
	// Only as POC
	for scanner.Scan() {

		// Total Accesses: 16147
		re = regexp.MustCompile("Total Accesses: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches == nil {
			//
		} else {
			totalAccess, _ = strconv.Atoi(matches[1])
		}

		//Total kBytes: 12988
		re = regexp.MustCompile("Total kBytes: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches == nil {
			//
		} else {
			totalKB, _ = strconv.Atoi(matches[1])
		}

		// Uptime: 3229728
		re = regexp.MustCompile("Uptime: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches == nil {
			//
		} else {
			uptime, _ = strconv.Atoi(matches[1])
		}
	}

	event := common.MapStr{
		"total_access": totalAccess,
		"total_kb":     totalKB,
		"uptime":       uptime,
	}

	return event

}
