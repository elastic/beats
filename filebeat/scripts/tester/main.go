package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

func getLogsFromFile(logfile, multiPattern string, multiNegate bool) ([]string, error) {
	regex, err := regexp.Compile(multiPattern)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(logfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var logs []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		l := getCompleteLine(s, s.Text(), multiNegate, regex)
		logs = append(logs, l...)
	}
	return logs, nil
}

func getCompleteLine(s *bufio.Scanner, line string, multiNegate bool, regex *regexp.Regexp) []string {
	if regex.String() == "" {
		return []string{line}
	}
	return getMultiline(s, line, multiNegate, regex)
}

func getMultiline(s *bufio.Scanner, line string, multiNegate bool, regex *regexp.Regexp) []string {
	matches := regex.MatchString(line)
	fullLine := line
	if matches || !matches && multiNegate {
		if !s.Scan() {
			return []string{fullLine}
		}

		line = s.Text()
		matches = regex.MatchString(line)
		for !matches || matches && multiNegate {
			fullLine = strings.Join([]string{fullLine, line}, "\n")
			if !s.Scan() {
				return []string{fullLine}
			}
			line = s.Text()
			matches = regex.MatchString(line)
		}
		return []string{fullLine, line}

	}
	return []string{fullLine}

}

func readPipeline(path string) (map[string]interface{}, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var p map[string]interface{}
	err = json.Unmarshal(d, &p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func runSimulate(url string, pipeline map[string]interface{}, logs []string) (*http.Response, error) {
	var sources []map[string]string
	for _, l := range logs {
		s := map[string]string{
			"message": l,
		}
		sources = append(sources, s)
	}

	var docs []common.MapStr
	for _, s := range sources {
		d := common.MapStr{
			"_index":  "index",
			"_type":   "doc",
			"_id":     "id",
			"_source": s,
		}
		docs = append(docs, d)
	}

	p := common.MapStr{
		"pipeline": pipeline,
		"docs":     docs,
	}

	payload := p.String()
	client := http.Client{}

	return client.Post(url+"/_ingest/pipeline/_simulate", "application/json", strings.NewReader(payload))
}

func showResp(resp *http.Response) {
	b := new(bytes.Buffer)
	b.ReadFrom(resp.Body)
	var r common.MapStr
	_ = json.Unmarshal(b.Bytes(), &r)
	fmt.Println(r.StringToPrint())
}

func main() {
	esURL := flag.String("elasticsearch", "http://localhost:9200", "Elasticsearch URL")
	path := flag.String("pipeline", "", "Path to pipeline")
	log := flag.String("log", "", "Single log line to test")
	logfile := flag.String("logfile", "", "Path to log file")
	multiPattern := flag.String("multiline-pattern", "", "Multiline pattern")
	multiNegate := flag.Bool("multiline-negate", false, "Multiline negate")
	flag.Parse()

	if *path == "" {
		fmt.Println("Error: -path is required")
		os.Exit(1)
	}

	if *log == "" && *logfile == "" {
		os.Stderr.WriteString("Error: -log or -logs has to be specified\n")
		os.Exit(1)
	}

	if *multiPattern != "" && *logfile == "" {
		os.Stderr.WriteString("Error: -multiline-pattern is set but -logfile is not\n")
		os.Exit(1)
	}

	var logs []string
	var err error
	if *logfile != "" {
		logs, err = getLogsFromFile(*logfile, *multiPattern, *multiNegate)
		if err != nil {
			os.Stderr.WriteString(fmt.Sprintf("Error while reading logs from file: %v\n", err))
			os.Exit(2)
		}
	} else {
		logs = []string{*log}
	}

	pipeline, err := readPipeline(*path)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Error while reading pipeline: %v\n", err))
		os.Exit(2)
	}

	resp, err := runSimulate(*esURL, pipeline, logs)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Error while sending request to Elasticsearch: %v\n", err))
		os.Exit(2)
	}

	showResp(resp)
}
