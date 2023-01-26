package ktest

import (
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus/ptest"
	"io/ioutil"
	"strings"
	"testing"
)

// GetTestCases Build test cases from the files and returns them
func GetTestCases(files []string) ptest.TestCases {
	var cases ptest.TestCases
	for i := 0; i < len(files); i++ {
		cases = append(cases,
			struct {
				MetricsFile  string
				ExpectedFile string
			}{
				MetricsFile:  files[i],
				ExpectedFile: files[i][1:] + ".expected",
			},
		)
	}
	return cases
}

// TestStateMetricsFamily
// This function reads the metric files and checks if the resource fetched metrics exist in it.
// It only checks the family metric, because if the metric doesn't have any data, we don't have a way
// to know the labels from the file.
// The test fails if the metric does not exist in any of the files.
// A warning is printed if the metric is not present in all of them.
// Nothing happens, otherwise.
func TestStateMetricsFamily(t *testing.T, files []string, mapping *p.MetricsMapping) {
	metricsFiles := map[string][]string{}
	for i := 0; i < len(files); i++ {
		content, err := ioutil.ReadFile(files[i])
		if err != nil {
			t.Fatalf("Unknown file %s.", files[i])
		}
		text := string(content)
		for metric, _ := range mapping.Metrics {
			if !strings.Contains(text, "# TYPE "+metric+" ") {
				metricsFiles[metric] = append(metricsFiles[metric], files[i])
			}
		}
	}
	for metric, filesList := range metricsFiles {
		if len(filesList) != len(files) {
			t.Logf("Warning: metric %s is not present in all files.", metric)
		} else {
			t.Errorf("Unknown metric: %s", metric)
		}
	}

}
