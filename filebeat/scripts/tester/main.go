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
		line := s.Text()
	}
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
	path := flag.String("path", "", "Path to pipeline")
	log := flag.String("log", "", "Single log line to test")
	logfile := flag.String("logs", "", "Path to log file")
	multiPattern := flag.String("multiline-pattern", "", "Multiline pattern")
	multiNegate := flag.Bool("multiline-negate", false, "Multiline negate")
	flag.Parse()

	if *path == "" {
		fmt.Println("Error: -path is required")
		os.Exit(1)
	}

	if *log == "" && *logfile == "" {
		fmt.Println("Error: -log or -logs has to be specified")
		os.Exit(1)
	}

	var logs []string
	if logfile != "" {
		logs, err = getLogsFromFile(logfile, multiPattern, multiNegate)
		if err != nil {
			fmt.Println("Error while reading logs from file:", err)
			os.Exit(2)
		}
	} else {
		logs = []string{*log}
	}

	pipeline, err := readPipeline(*path)
	if err != nil {
		fmt.Println("Error while reading pipeline:", err)
		os.Exit(2)
	}

	resp, err := runSimulate(*esURL, pipeline, logs)
	if err != nil {
		fmt.Println("Error while sending request to Elasticsearch:", err)
		os.Exit(2)
	}

	showResp(resp)
}
