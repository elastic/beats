package elasticsearch

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
)

type Client struct {
	elasticsearchConnection
	index string
}

type elasticsearchConnection struct {
	URL                string
	Username, Password string

	http      *http.Client
	connected bool
}

func NewClient(
	url, index string, tls *tls.Config,
	username, password string,
) *Client {
	client := &Client{
		elasticsearchConnection{
			URL:      url,
			Username: username,
			Password: password,
			http: &http.Client{
				Transport: &http.Transport{TLSClientConfig: tls},
			},
		},
		index,
	}
	return client
}

func (es *Client) Clone() *Client {
	client := &Client{
		elasticsearchConnection{
			URL:      es.URL,
			Username: es.Username,
			Password: es.Password,
			http: &http.Client{
				Transport: es.http.Transport,
			},
			connected: false,
		},
		es.index,
	}
	return client
}

func (es *Client) Connect(timeout time.Duration) error {
	var err error
	es.connected, err = es.Ping()
	if err != nil {
		return err
	}
	if !es.connected {
		return ErrNotConnected
	}
	return nil
}

func (es *Client) Ping() (bool, error) {
	es.http.Timeout = defaultEsOpenTimeout
	resp, err := es.http.Head(es.URL)
	if err != nil {
		return false, err
	}
	defer closing(resp.Body)

	status := resp.StatusCode
	return status < 300, nil
}

func (es *Client) IsConnected() bool {
	return es.connected
}

func (es *Client) Close() error {
	es.connected = false
	return nil
}

func (es *Client) PublishEvents(
	events []common.MapStr,
) (n int, err error) {
	if !es.connected {
		return 0, ErrNotConnected
	}

	request, err := es.startBulkRequest("", "", nil)
	if err != nil {
		logp.Err(
			"Failed to perform many index operations in a single API call: %s",
			err)
		return 0, err
	}

	for _, event := range events {
		ts := time.Time(event["timestamp"].(common.Time))
		index := fmt.Sprintf("%s-%d.%02d.%02d",
			es.index, ts.Year(), ts.Month(), ts.Day())
		meta := bulkMeta{
			Index: bulkMetaIndex{
				Index:   index,
				DocType: event["type"].(string),
			},
		}
		// meta := common.MapStr{
		// 	"index": map[string]interface{}{
		// 		"_index": index,
		// 		"_type":  event["type"].(string),
		// 	},
		// }
		err := request.Send(meta, event)
		if err != nil {
			logp.Err("Failed to encode event: %s", err)
		}
	}

	_, err = request.Flush()
	if err != nil {
		logp.Err(
			"Failed to perform many index operations in a single API call; %s",
			err)
		return 0, err
	}

	return len(events), nil
}

func (es *Client) PublishEvent(event common.MapStr) error {
	if !es.connected {
		return ErrNotConnected
	}

	ts := time.Time(event["timestamp"].(common.Time))
	index := fmt.Sprintf("%s-%d.%02d.%02d",
		es.index, ts.Year(), ts.Month(), ts.Day())
	logp.Debug("output_elasticsearch", "Publish event: %s", event)

	// insert the events one by one
	_, err := es.Index(index, event["type"].(string), "", nil, event)
	if err != nil {
		logp.Warn("Fail to insert a single event: %s", err)
	}
	return nil
}

func (es *Client) CreateIndex(index string, body interface{}) (*QueryResult, error) {
	return CreateIndex(es, index, body)
}

func (es *Client) Index(
	index, docType, id string,
	params map[string]string,
	body interface{},
) (*QueryResult, error) {
	return Index(es, index, docType, id, params, body)
}

func (es *Client) Refresh(index string) (*QueryResult, error) {
	return Refresh(es, index)
}

func (es *Client) Delete(index string, docType string, id string, params map[string]string) (*QueryResult, error) {
	return Delete(es, index, docType, id, params)
}

func (es *Client) SearchURI(
	index, docType string,
	params map[string]string,
) (*SearchResults, error) {
	return SearchURI(es, index, docType, params)
}

func (es *Client) CountSearchURI(
	index string, docType string,
	params map[string]string,
) (*CountResults, error) {
	return CountSearchURI(es, index, docType, params)
}

func (es *Client) Bulk(
	index, docType string,
	params map[string]string, body []interface{},
) (*QueryResult, error) {
	return Bulk(es, index, docType, params, body)
}

func (es *Client) startBulkRequest(
	index string,
	docType string,
	params map[string]string,
) (*bulkRequest, error) {
	path, err := makePath(index, docType, "_bulk")
	if err != nil {
		return nil, err
	}

	r := &bulkRequest{
		es:     es,
		path:   path,
		params: params,
	}
	r.enc = json.NewEncoder(&r.buf)
	return r, nil
}

func (es *Client) sendBulkRequest(
	method, path string,
	params map[string]string,
	buf *bytes.Buffer,
) ([]byte, error) {
	url := makeURL(es.URL, path, params)
	logp.Debug("elasticsearch", "Sending bulk request to %s", url)

	return es.execRequest(method, url, buf)
}

func (es *elasticsearchConnection) request(
	method, path string,
	params map[string]string,
	body interface{},
) ([]byte, error) {
	url := makeURL(es.URL, path, params)
	logp.Debug("elasticsearch", "%s %s %s", method, url, body)

	var obj []byte
	if body != nil {
		var err error
		obj, err = json.Marshal(body)
		if err != nil {
			return nil, ErrJsonEncodeFailed
		}
	}

	return es.execRequest(method, url, bytes.NewReader(obj))
}

func (es *elasticsearchConnection) execRequest(
	method, url string,
	body io.Reader,
) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		logp.Warn("Failed to create request", err)
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	if es.Username != "" || es.Password != "" {
		req.SetBasicAuth(es.Username, es.Password)
	}

	resp, err := es.http.Do(req)
	if err != nil {
		es.connected = false
		return nil, err
	}
	defer closing(resp.Body)

	status := resp.StatusCode
	if status >= 300 {
		es.connected = false
		return nil, fmt.Errorf("%v", resp.Status)
	}

	obj, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		es.connected = false
		return nil, err
	}
	return obj, nil
}

func closing(c io.Closer) {
	err := c.Close()
	if err != nil {
		logp.Warn("Close failed with: %v", err)
	}
}
