// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/reader"
	"github.com/elastic/beats/libbeat/reader/multiline"
	"github.com/elastic/beats/libbeat/reader/readfile"
	"github.com/elastic/beats/libbeat/reader/readfile/encoding"
)

type logReaderConfig struct {
	multiPattern string
	multiNegate  bool
	maxBytes     int
	matchMode    string
	encoding     string
}

func main() {
	esURL := flag.String("elasticsearch", "http://localhost:9200", "Elasticsearch URL")
	path := flag.String("pipeline", "", "Path to pipeline")
	modulesPath := flag.String("modules", "./modules", "Path to modules")

	log := flag.String("log", "", "Single log line to test")
	logfile := flag.String("logfile", "", "Path to log file")

	multiPattern := flag.String("multiline.pattern", "", "Multiline pattern")
	multiNegate := flag.Bool("multiline.negate", false, "Multiline negate")
	multiMode := flag.String("multiline.mode", "before", "Multiline mode")
	maxBytes := flag.Int("maxbytes", 10485760, "Number of max bytes to be read")
	fileEncoding := flag.String("encoding", "utf8", "Encoding of logfile")

	verbose := flag.Bool("verbose", false, "Call Simulate API with verbose option")
	simulateVerbose := flag.Bool("simulate.verbose", false, "Print full output of Simulate API with verbose option")
	flag.Parse()

	if *path == "" {
		os.Stderr.WriteString("Error: -pipeline is required\n")
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
		c := logReaderConfig{
			multiPattern: *multiPattern,
			multiNegate:  *multiNegate,
			matchMode:    *multiMode,
			maxBytes:     *maxBytes,
			encoding:     *fileEncoding,
		}
		logs, err = getLogsFromFile(*logfile, &c)
		if err != nil {
			os.Stderr.WriteString(fmt.Sprintf("Error while reading logs from file: %v\n", err))
			os.Exit(2)
		}
	} else {
		logs = []string{*log}
	}

	paths, err := getPipelinePath(*path, *modulesPath)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(3)
	}
	if len(paths) == 0 {
		os.Stderr.WriteString("No pipeline file was found\n")
		os.Exit(3)
	}

	for _, path := range paths {
		err = testPipeline(*esURL, path, logs, *verbose, *simulateVerbose)
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(4)
		}
	}
}
func getLogsFromFile(logfile string, conf *logReaderConfig) ([]string, error) {
	f, err := os.Open(logfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	encFactory, ok := encoding.FindEncoding(conf.encoding)
	if !ok {
		return nil, fmt.Errorf("unable to find encoding: %s", conf.encoding)
	}

	enc, err := encFactory(f)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encoding: %v", err)
	}

	var r reader.Reader
	r, err = readfile.NewEncodeReader(f, readfile.Config{
		Codec:      enc,
		BufferSize: 4096,
		Terminator: readfile.LineFeed,
	})
	if err != nil {
		return nil, err
	}

	r = readfile.NewStripNewline(r, readfile.LineFeed)

	if conf.multiPattern != "" {
		p, err := match.Compile(conf.multiPattern)
		if err != nil {
			return nil, err
		}

		c := multiline.Config{
			Negate:  conf.multiNegate,
			Match:   conf.matchMode,
			Pattern: &p,
		}
		r, err = multiline.New(r, "\n", 1<<20, &c)
		if err != nil {
			return nil, err
		}
	}
	r = readfile.NewLimitReader(r, conf.maxBytes)

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

func getPipelinePath(path, modulesPath string) ([]string, error) {
	var paths []string
	stat, err := os.Stat(path)
	if err != nil {
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Cannot find pipeline in %s\n", path)
		}
		module := parts[0]
		fileset := parts[1]

		pathToPipeline := filepath.Join(modulesPath, module, fileset, "ingest", "pipeline.json")
		_, err := os.Stat(pathToPipeline)
		if err != nil {
			return nil, fmt.Errorf("Cannot find pipeline in %s: %v %v\n", path, err, pathToPipeline)
		}
		return []string{pathToPipeline}, nil
	}

	if stat.IsDir() {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			isPipelineFile := strings.HasSuffix(f.Name(), ".json")
			if isPipelineFile {
				fullPath := filepath.Join(path, f.Name())
				paths = append(paths, fullPath)
			}
		}
		if len(paths) == 0 {
			return paths, fmt.Errorf("Cannot find pipeline in %s", path)
		}
		return paths, nil
	}

	isPipelineFile := strings.HasSuffix(path, ".json")
	if isPipelineFile {
		return []string{path}, nil
	}

	return paths, nil

}

func testPipeline(esURL, path string, logs []string, verbose, simulateVerbose bool) error {
	pipeline, err := readPipeline(path)
	if err != nil {
		return fmt.Errorf("Error while reading pipeline: %v\n", err)
	}

	resp, err := runSimulate(esURL, pipeline, logs, simulateVerbose)
	if err != nil {
		return fmt.Errorf("Error while sending request to Elasticsearch: %v\n", err)
	}

	err = showResp(resp, verbose, simulateVerbose)
	if err != nil {
		return fmt.Errorf("Error while reading response from Elasticsearch: %v\n", err)
	}
	return nil
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

func runSimulate(url string, pipeline map[string]interface{}, logs []string, verbose bool) (*http.Response, error) {
	var sources []common.MapStr
	now := time.Now().UTC()
	for _, l := range logs {
		s := common.MapStr{
			"@timestamp": common.Time(now),
			"message":    l,
		}
		sources = append(sources, s)
	}

	var docs []common.MapStr
	for _, s := range sources {
		d := common.MapStr{
			"_index":  "index",
			"_type":   "_doc",
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

	simulateURL := url + "/_ingest/pipeline/_simulate"
	if verbose {
		simulateURL += "?verbose"
	}

	return client.Post(simulateURL, "application/json", strings.NewReader(payload))
}

func showResp(resp *http.Response, verbose, simulateVerbose bool) error {
	if resp.StatusCode != 200 {
		return fmt.Errorf("response code is %d not 200", resp.StatusCode)
	}

	b := new(bytes.Buffer)
	b.ReadFrom(resp.Body)
	var r common.MapStr
	err := json.Unmarshal(b.Bytes(), &r)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println(r.StringToPrint())
	} else {
		docErrors, err := getDocErrors(r, simulateVerbose)
		if err != nil {
			return err
		}

		for _, d := range docErrors {
			fmt.Println(d.StringToPrint())
		}
	}
	return nil
}

func getDocErrors(r common.MapStr, simulateVerbose bool) ([]common.MapStr, error) {
	d, err := r.GetValue("docs")
	if err != nil {
		return nil, err
	}

	docs := d.([]interface{})
	if simulateVerbose {
		return getErrorsSimulateVerbose(docs)
	}

	return getRegularErrors(docs)
}

func getRegularErrors(docs []interface{}) ([]common.MapStr, error) {
	var errors []common.MapStr
	for _, d := range docs {
		dd := d.(map[string]interface{})
		doc := common.MapStr(dd)
		hasError, err := doc.HasKey("doc._source.error")
		if err != nil {
			return nil, err
		}

		if hasError {
			errors = append(errors, doc)
		}
	}
	return errors, nil
}

func getErrorsSimulateVerbose(docs []interface{}) ([]common.MapStr, error) {
	var errors []common.MapStr
	for _, d := range docs {
		pr := d.(map[string]interface{})
		p := common.MapStr(pr)

		rr, err := p.GetValue("processor_results")
		if err != nil {
			return nil, err
		}
		res := rr.([]interface{})
		hasError := false
		for _, r := range res {
			rres := r.(map[string]interface{})
			result := common.MapStr(rres)
			hasError, _ = result.HasKey("error")
			if hasError {
				errors = append(errors, p)
			}
		}
	}
	return errors, nil
}
