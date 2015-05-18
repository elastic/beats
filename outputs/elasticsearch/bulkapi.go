package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
)

type BulkMsg struct {
	Ts    time.Time
	Event common.MapStr
}

func (es *Elasticsearch) Bulk(index string, doc_type string,
	params map[string]string, body chan interface{}) (*QueryResult, error) {

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for obj := range body {
		enc.Encode(obj)
		logp.Debug("elasticsearch", "obj %s", obj)
	}

	if buf.Len() == 0 {
		logp.Debug("elasticsearch", "Empty channel. Wait for more data.")
		return nil, nil
	}

	logp.Debug("elasticsearch", "Insert bulk messages: %s\n", buf)

	path, err := MakePath(index, doc_type, "_bulk")
	if err != nil {
		return nil, err
	}

	url := es.Url + path
	if len(params) > 0 {
		url = url + "?" + UrlEncode(params)
	}

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	resp, err := es.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	obj, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result QueryResult
	err = json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return &result, fmt.Errorf("ES returned an error: %s", resp.Status)
	}
	return &result, err
}
