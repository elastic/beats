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
	}

	if buf.Len() == 0 {
		logp.Debug("elasticsearch", "Empty channel. Wait for more data.")
		return nil, nil
	}

	logp.Debug("elasticsearch", "Insert bulk messages:\n%s\n", buf)

	path, err := MakePath(index, doc_type, "_bulk")
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < es.MaxRetries; attempt++ {

		conn := es.connectionPool.GetConnection()
		logp.Debug("elasticsearch", "Use connection %s", conn.Url)

		url := conn.Url + path
		if len(params) > 0 {
			url = url + "?" + UrlEncode(params)
		}

		req, err := http.NewRequest("POST", url, &buf)
		if err != nil {
			return nil, err
		}

		logp.Debug("elasticsearch", "Sending bulk request to %s", url)

		req.Header.Add("Accept", "application/json")
		if conn.Username != "" || conn.Password != "" {
			req.SetBasicAuth(conn.Username, conn.Password)
		}

		resp, err := es.client.Do(req)
		if err != nil {
			// request fails
			logp.Warn("Request fails: %s", err)
			es.connectionPool.MarkDead(conn)
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

		if resp.StatusCode > 499 {
			// request fails
			es.connectionPool.MarkDead(conn)
			return &result, fmt.Errorf("ES returned an error: %s", resp.Status)
		}
		// request with success
		es.connectionPool.MarkLive(conn)

		return &result, err
	}

	return nil, fmt.Errorf("Request fails to be send after %d retries", es.MaxRetries)
}
