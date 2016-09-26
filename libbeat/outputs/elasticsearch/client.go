package elasticsearch

import (
	"bytes"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type Client struct {
	Connection
	tlsConfig *transport.TLSConfig

	index    outil.Selector
	pipeline *outil.Selector
	params   map[string]string

	// buffered bulk requests
	bulkRequ *bulkRequest

	// buffered json response reader
	json jsonReader

	// additional configs
	compressionLevel int
	proxyURL         *url.URL
}

type ClientSettings struct {
	URL                string
	Proxy              *url.URL
	TLS                *transport.TLSConfig
	Username, Password string
	Parameters         map[string]string
	Index              outil.Selector
	Pipeline           *outil.Selector
	Timeout            time.Duration
	CompressionLevel   int
}

type connectCallback func(client *Client) error

type Connection struct {
	URL      string
	Username string
	Password string

	http              *http.Client
	onConnectCallback func() error

	encoder bodyEncoder
	version string
}

// Metrics that can retrieved through the expvar web interface.
var (
	ackedEvents            = expvar.NewInt("libbeat.es.published_and_acked_events")
	eventsNotAcked         = expvar.NewInt("libbeat.es.published_but_not_acked_events")
	publishEventsCallCount = expvar.NewInt("libbeat.es.call_count.PublishEvents")

	statReadBytes   = expvar.NewInt("libbeat.es.publish.read_bytes")
	statWriteBytes  = expvar.NewInt("libbeat.es.publish.write_bytes")
	statReadErrors  = expvar.NewInt("libbeat.es.publish.read_errors")
	statWriteErrors = expvar.NewInt("libbeat.es.publish.write_errors")
)

var (
	nameItems  = []byte("items")
	nameStatus = []byte("status")
	nameError  = []byte("error")
)

var (
	errExpectedItemObject    = errors.New("expected item response object")
	errExpectedStatusCode    = errors.New("expected item status code")
	errUnexpectedEmptyObject = errors.New("empty object")
	errExcpectedObjectEnd    = errors.New("expected end of object")
)

func NewClient(
	s ClientSettings,
	onConnectCallback connectCallback,
) (*Client, error) {
	proxy := http.ProxyFromEnvironment
	if s.Proxy != nil {
		proxy = http.ProxyURL(s.Proxy)
	}

	pipeline := s.Pipeline
	if pipeline != nil && pipeline.IsEmpty() {
		pipeline = nil
	}

	logp.Info("Elasticsearch url: %s", s.URL)

	// TODO: add socks5 proxy support
	var dialer, tlsDialer transport.Dialer
	var err error

	dialer = transport.NetDialer(s.Timeout)
	tlsDialer, err = transport.TLSDialer(dialer, s.TLS, s.Timeout)
	if err != nil {
		return nil, err
	}

	iostats := &transport.IOStats{
		Read:        statReadBytes,
		Write:       statWriteBytes,
		ReadErrors:  statReadErrors,
		WriteErrors: statWriteErrors,
	}
	dialer = transport.StatsDialer(dialer, iostats)
	tlsDialer = transport.StatsDialer(tlsDialer, iostats)

	params := s.Parameters
	bulkRequ, err := newBulkRequest(s.URL, "", "", params, nil)
	if err != nil {
		return nil, err
	}

	var encoder bodyEncoder
	compression := s.CompressionLevel
	if compression == 0 {
		encoder = newJSONEncoder(nil)
	} else {
		encoder, err = newGzipEncoder(compression, nil)
		if err != nil {
			return nil, err
		}
	}

	client := &Client{
		Connection: Connection{
			URL:      s.URL,
			Username: s.Username,
			Password: s.Password,
			http: &http.Client{
				Transport: &http.Transport{
					Dial:    dialer.Dial,
					DialTLS: tlsDialer.Dial,
					Proxy:   proxy,
				},
				Timeout: s.Timeout,
			},
			encoder: encoder,
		},
		tlsConfig: s.TLS,
		index:     s.Index,
		pipeline:  pipeline,
		params:    params,

		bulkRequ: bulkRequ,

		compressionLevel: compression,
		proxyURL:         s.Proxy,
	}

	client.Connection.onConnectCallback = func() error {
		if onConnectCallback != nil {
			return onConnectCallback(client)
		}
		return nil
	}

	return client, nil
}

func (client *Client) Clone() *Client {
	// when cloning the connection callback and params are not copied. A
	// client's close is for example generated for topology-map support. With params
	// most likely containing the ingest node pipeline and default callback trying to
	// create install a template, we don't want these to be included in the clone.

	c, _ := NewClient(
		ClientSettings{
			URL:              client.URL,
			Index:            client.index,
			Pipeline:         client.pipeline,
			Proxy:            client.proxyURL,
			TLS:              client.tlsConfig,
			Username:         client.Username,
			Password:         client.Password,
			Parameters:       nil, // XXX: do not pass params?
			Timeout:          client.http.Timeout,
			CompressionLevel: client.compressionLevel,
		},
		nil, // XXX: do not pass connection callback?
	)
	return c
}

// PublishEvents sends all events to elasticsearch. On error a slice with all
// events not published or confirmed to be processed by elasticsearch will be
// returned. The input slice backing memory will be reused by return the value.
func (client *Client) PublishEvents(
	data []outputs.Data,
) ([]outputs.Data, error) {
	begin := time.Now()
	publishEventsCallCount.Add(1)

	if len(data) == 0 {
		return nil, nil
	}

	body := client.encoder
	body.Reset()

	// encode events into bulk request buffer, dropping failed elements from
	// events slice
	data = bulkEncodePublishRequest(body, client.index, client.pipeline, data)
	if len(data) == 0 {
		return nil, nil
	}

	requ := client.bulkRequ
	requ.Reset(body)
	status, result, sendErr := client.sendBulkRequest(requ)
	if sendErr != nil {
		logp.Err("Failed to perform any bulk index operations: %s", sendErr)
		return data, sendErr
	}

	debugf("PublishEvents: %d events have been  published to elasticsearch in %v.",
		len(data),
		time.Now().Sub(begin))

	// check response for transient errors
	var failedEvents []outputs.Data
	if status != 200 {
		failedEvents = data
	} else {
		client.json.init(result.raw)
		failedEvents = bulkCollectPublishFails(&client.json, data)
	}

	ackedEvents.Add(int64(len(data) - len(failedEvents)))
	eventsNotAcked.Add(int64(len(failedEvents)))
	if len(failedEvents) > 0 {
		if sendErr == nil {
			sendErr = mode.ErrTempBulkFailure
		}
		return failedEvents, sendErr
	}

	return nil, nil
}

// fillBulkRequest encodes all bulk requests and returns slice of events
// successfully added to bulk request.
func bulkEncodePublishRequest(
	body bulkWriter,
	index outil.Selector,
	pipeline *outil.Selector,
	data []outputs.Data,
) []outputs.Data {
	var mkMeta func(outil.Selector, *outil.Selector, outputs.Data) interface{}

	mkMeta = eventBulkMeta
	if pipeline != nil {
		mkMeta = eventIngestBulkMeta
	}

	okEvents := data[:0]
	for _, datum := range data {
		meta := mkMeta(index, pipeline, datum)
		if err := body.Add(meta, datum.Event); err != nil {
			logp.Err("Failed to encode event: %s", err)
			continue
		}
		okEvents = append(okEvents, datum)
	}
	return okEvents
}

func eventBulkMeta(
	index outil.Selector,
	_ *outil.Selector,
	data outputs.Data,
) interface{} {
	type bulkMetaIndex struct {
		Index   string `json:"_index"`
		DocType string `json:"_type"`
	}
	type bulkMeta struct {
		Index bulkMetaIndex `json:"index"`
	}

	event := data.Event
	meta := bulkMeta{
		Index: bulkMetaIndex{
			Index:   getIndex(event, index),
			DocType: event["type"].(string),
		},
	}
	return meta
}

func eventIngestBulkMeta(
	index outil.Selector,
	pipelineSel *outil.Selector,
	data outputs.Data,
) interface{} {
	type bulkMetaIndex struct {
		Index    string `json:"_index"`
		DocType  string `json:"_type"`
		Pipeline string `json:"pipeline"`
	}
	type bulkMeta struct {
		Index bulkMetaIndex `json:"index"`
	}

	event := data.Event
	pipeline, _ := pipelineSel.Select(event)
	if pipeline == "" {
		return eventBulkMeta(index, nil, data)
	}

	return bulkMeta{
		Index: bulkMetaIndex{
			Index:    getIndex(event, index),
			Pipeline: pipeline,
			DocType:  event["type"].(string),
		},
	}
}

// getIndex returns the full index name
// Index is either defined in the config as part of the output
// or can be overload by the event through setting index
func getIndex(event common.MapStr, index outil.Selector) string {

	ts := time.Time(event["@timestamp"].(common.Time)).UTC()

	// Check for dynamic index
	// XXX: is this used/needed?
	if _, ok := event["beat"]; ok {
		beatMeta, ok := event["beat"].(common.MapStr)
		if ok {
			// Check if index is set dynamically
			if dynamicIndex, ok := beatMeta["index"]; ok {
				if dynamicIndexValue, ok := dynamicIndex.(string); ok {
					return fmt.Sprintf("%s-%d.%02d.%02d",
						dynamicIndexValue, ts.Year(), ts.Month(), ts.Day())
				}
			}
		}
	}

	str, _ := index.Select(event)
	return str
}

// bulkCollectPublishFails checks per item errors returning all events
// to be tried again due to error code returned for that items. If indexing an
// event failed due to some error in the event itself (e.g. does not respect mapping),
// the event will be dropped.
func bulkCollectPublishFails(
	reader *jsonReader,
	data []outputs.Data,
) []outputs.Data {
	if err := reader.expectDict(); err != nil {
		logp.Err("Failed to parse bulk respose: expected JSON object")
		return nil
	}

	// find 'items' field in response
	for {
		kind, name, err := reader.nextFieldName()
		if err != nil {
			logp.Err("Failed to parse bulk response")
			return nil
		}

		if kind == dictEnd {
			logp.Err("Failed to parse bulk response: no 'items' field in response")
			return nil
		}

		// found items array -> continue
		if bytes.Equal(name, nameItems) {
			break
		}

		reader.ignoreNext()
	}

	// check items field is an array
	if err := reader.expectArray(); err != nil {
		logp.Err("Failed to parse bulk respose: expected items array")
		return nil
	}

	count := len(data)
	failed := data[:0]
	for i := 0; i < count; i++ {
		status, msg, err := itemStatus(reader)
		if err != nil {
			return nil
		}

		if status < 300 {
			continue // ok value
		}

		if status < 500 && status != 429 {
			// hard failure, don't collect
			logp.Warn("Can not index event (status=%v): %s", status, msg)
			continue
		}

		logp.Info("Bulk item insert failed (i=%v, status=%v): %s", i, status, msg)
		failed = append(failed, data[i])
	}

	return failed
}

func itemStatus(reader *jsonReader) (int, []byte, error) {
	// skip outer dictionary
	if err := reader.expectDict(); err != nil {
		return 0, nil, errExpectedItemObject
	}

	// find first field in outer dictionary (e.g. 'create')
	kind, _, err := reader.nextFieldName()
	if err != nil {
		logp.Err("Failed to parse bulk response item: %s", err)
		return 0, nil, err
	}
	if kind == dictEnd {
		err = errUnexpectedEmptyObject
		logp.Err("Failed to parse bulk response item: %s", err)
		return 0, nil, err
	}

	// parse actual item response code and error message
	status, msg, err := itemStatusInner(reader)

	// close dictionary. Expect outer dictionary to have only one element
	kind, _, err = reader.step()
	if err != nil {
		logp.Err("Failed to parse bulk response item: %s", err)
		return 0, nil, err
	}
	if kind != dictEnd {
		err = errExcpectedObjectEnd
		logp.Err("Failed to parse bulk response item: %s", err)
		return 0, nil, err
	}

	return status, msg, nil
}

func itemStatusInner(reader *jsonReader) (int, []byte, error) {
	if err := reader.expectDict(); err != nil {
		return 0, nil, errExpectedItemObject
	}

	status := -1
	var msg []byte
	for {
		kind, name, err := reader.nextFieldName()
		if err != nil {
			logp.Err("Failed to parse bulk response item: %s", err)
		}
		if kind == dictEnd {
			break
		}

		switch {
		case bytes.Equal(name, nameStatus): // name == "status"
			status, err = reader.nextInt()
			if err != nil {
				logp.Err("Failed to parse bulk reponse item: %s", err)
				return 0, nil, err
			}

		case bytes.Equal(name, nameError): // name == "error"
			msg, err = reader.ignoreNext() // collect raw string for "error" field
			if err != nil {
				return 0, nil, err
			}

		default: // ignore unknown fields
			_, err = reader.ignoreNext()
			if err != nil {
				return 0, nil, err
			}
		}
	}

	if status < 0 {
		return 0, nil, errExpectedStatusCode
	}
	return status, msg, nil
}

func (client *Client) PublishEvent(data outputs.Data) error {
	// insert the events one by one

	event := data.Event
	index := getIndex(event, client.index)
	typ := event["type"].(string)

	debugf("Publish event: %s", event)

	pipeline := ""
	if client.pipeline != nil {
		var err error
		pipeline, err = client.pipeline.Select(event)
		if err != nil {
			logp.Err("Failed to select pipeline: %v", err)
			return err
		}
		debugf("select pipeline: %v", pipeline)
	}

	var status int
	var err error
	if pipeline == "" {
		status, _, err = client.Index(index, typ, "", client.params, event)
	} else {
		status, _, err = client.Ingest(index, typ, pipeline, "", client.params, event)
	}

	// check indexing error
	if err != nil {
		logp.Warn("Fail to insert a single event: %s", err)
		if err == ErrJSONEncodeFailed {
			// don't retry unencodable values
			return nil
		}
	}
	switch {
	case status == 0: // event was not send yet
		return nil
	case status >= 500 || status == 429: // server error, retry
		return err
	case status >= 300 && status < 500:
		// won't be able to index event in Elasticsearch => don't retry
		return nil
	}

	return nil
}

// LoadTemplate loads a template into Elasticsearch overwriting the existing
// template if it exists. If you wish to not overwrite an existing template
// then use CheckTemplate prior to calling this method.
func (client *Client) LoadTemplate(templateName string, template map[string]interface{}) error {

	path := "/_template/" + templateName
	err := client.LoadJson(path, template)
	if err != nil {
		return fmt.Errorf("couldn't load template: %v", err)
	}
	logp.Info("Elasticsearch template with name '%s' loaded", templateName)
	return nil
}

func (client *Client) LoadJson(path string, json map[string]interface{}) error {
	status, _, err := client.request("PUT", path, "", nil, json)
	if err != nil {
		return fmt.Errorf("couldn't load json. Error: %s", err)
	}
	if status > 300 {
		return fmt.Errorf("couldn't load json. Status: %v", status)
	}

	return nil
}

// CheckTemplate checks if a given template already exist. It returns true if
// and only if Elasticsearch returns with HTTP status code 200.
func (client *Client) CheckTemplate(templateName string) bool {

	status, _, _ := client.request("HEAD", "/_template/"+templateName, "", nil, nil)

	if status != 200 {
		return false
	}

	return true
}

func (conn *Connection) Connect(timeout time.Duration) error {
	var err error
	conn.version, err = conn.Ping(timeout)
	if err != nil {
		return err
	}

	err = conn.onConnectCallback()
	if err != nil {
		return fmt.Errorf("Connection marked as failed because the onConnect callback failed: %v", err)
	}
	return nil
}

// Ping sends a GET request to the Elasticsearch
func (conn *Connection) Ping(timeout time.Duration) (string, error) {
	debugf("ES Ping(url=%v, timeout=%v)", conn.URL, timeout)

	conn.http.Timeout = timeout
	status, body, err := conn.execRequest("GET", conn.URL, nil)
	if err != nil {
		debugf("Ping request failed with: %v", err)
		return "", err
	}

	if status >= 300 {
		return "", fmt.Errorf("Non 2xx response code: %d", status)
	}

	var response struct {
		Version struct {
			Number string
		}
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("Failed to parse JSON response: %v", err)
	}

	debugf("Ping status code: %v", status)
	logp.Info("Connected to Elasticsearch version %s", response.Version.Number)
	return response.Version.Number, nil
}

func (conn *Connection) Close() error {
	return nil
}

func (conn *Connection) request(
	method, path string,
	pipeline string,
	params map[string]string,
	body interface{},
) (int, []byte, error) {
	url := makeURL(conn.URL, path, pipeline, params)
	debugf("%s %s %s %v", method, url, pipeline, body)

	if body == nil {
		return conn.execRequest(method, url, nil)
	}

	if err := conn.encoder.Marshal(body); err != nil {
		logp.Warn("Failed to json encode body (%v): %#v", err, body)
		return 0, nil, ErrJSONEncodeFailed
	}
	return conn.execRequest(method, url, conn.encoder.Reader())
}

func (conn *Connection) execRequest(
	method, url string,
	body io.Reader,
) (int, []byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		logp.Warn("Failed to create request", err)
		return 0, nil, err
	}
	if body != nil {
		conn.encoder.AddHeader(&req.Header)
	}
	return conn.execHTTPRequest(req)
}

func (conn *Connection) execHTTPRequest(req *http.Request) (int, []byte, error) {
	req.Header.Add("Accept", "application/json")
	if conn.Username != "" || conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	resp, err := conn.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer closing(resp.Body)

	status := resp.StatusCode
	if status >= 300 {
		return status, nil, fmt.Errorf("%v", resp.Status)
	}

	obj, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return status, nil, err
	}
	return status, obj, nil
}

func closing(c io.Closer) {
	err := c.Close()
	if err != nil {
		logp.Warn("Close failed with: %v", err)
	}
}
