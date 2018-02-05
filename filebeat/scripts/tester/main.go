package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
)

func getLogsFromFile(logfile, multiPattern string, multiNegate bool, matchMode string) ([]string, error) {
	f, err := os.Open(logfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	encFactory, ok := encoding.FindEncoding("utf8")
	if !ok {
		return nil, fmt.Errorf("unable to find 'utf8' encoding")
	}

	enc, err := encFactory(f)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encoding: %v", err)
	}

	var r reader.Reader
	r, err = reader.NewEncode(f, enc, 4096)
	if err != nil {
		return nil, err
	}

	r = reader.NewStripNewline(r)

	if multiPattern != "" {
		p := match.MustCompile(multiPattern)
		c := reader.MultilineConfig{
			Negate:  multiNegate,
			Match:   matchMode,
			Pattern: &p,
		}
		r, err = reader.NewMultiline(r, "\n", 1<<20, &c)
		if err != nil {
			return nil, err
		}
	}
	r = reader.NewLimit(r, 10485760)

	var logs []string
	for {
		msg, err := r.Next()
		if err != nil {
			break
		}
		logs = append(logs, string(msg.Content))
	}

	return logs, nil
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
	multiMode := flag.String("multiline-mode", "before", "Multiline mode")
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
		logs, err = getLogsFromFile(*logfile, *multiPattern, *multiNegate, *multiMode)
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
