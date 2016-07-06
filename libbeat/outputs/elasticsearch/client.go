package elasticsearch

import (
	"bytes"
	"crypto/tls"
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
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type Client struct {
	Connection
	index  string
	params map[string]string

	// buffered bulk requests
	bulkRequ *bulkRequest

	// buffered json response reader
	json jsonReader

	// additional configs
	compressionLevel int
	proxyURL         *url.URL
}

type connectCallback func(client *Client) error

type Connection struct {
	URL      string
	Username string
	Password string

	http              *http.Client
	connected         bool
	onConnectCallback func() error

	encoder bodyEncoder
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
	esURL, index string, proxyURL *url.URL, tls *tls.Config,
	username, password string,
	params map[string]string,
	timeout time.Duration,
	compression int,
	onConnectCallback connectCallback,
) (*Client, error) {
	proxy := http.ProxyFromEnvironment
	if proxyURL != nil {
		proxy = http.ProxyURL(proxyURL)
	}

	logp.Info("Elasticsearch url: %s", esURL)

	dialer := transport.NetDialer(timeout)
	dialer = transport.StatsDialer(dialer, &transport.IOStats{
		Read:        statReadBytes,
		Write:       statWriteBytes,
		ReadErrors:  statReadErrors,
		WriteErrors: statWriteErrors,
	})

	bulkRequ, err := newBulkRequest(esURL, "", "", params, nil)
	if err != nil {
		return nil, err
	}

	var encoder bodyEncoder
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
			URL:      esURL,
			Username: username,
			Password: password,
			http: &http.Client{
				Transport: &http.Transport{
					Dial:            dialer.Dial,
					TLSClientConfig: tls,
					Proxy:           proxy,
				},
				Timeout: timeout,
			},
			encoder: encoder,
		},
		index:  index,
		params: params,

		bulkRequ: bulkRequ,

		compressionLevel: compression,
		proxyURL:         proxyURL,
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

	transport := client.http.Transport.(*http.Transport)
	c, _ := NewClient(
		client.URL,
		client.index,
		client.proxyURL,
		transport.TLSClientConfig,
		client.Username,
		client.Password,
		nil, // XXX: do not pass params?
		client.http.Timeout,
		client.compressionLevel,
		nil, // XXX: do not pass connection callback?
	)
	return c
}

// PublishEvents sends all events to elasticsearch. On error a slice with all
// events not published or confirmed to be processed by elasticsearch will be
// returned. The input slice backing memory will be reused by return the value.
func (client *Client) PublishEvents(
	events []common.MapStr,
) ([]common.MapStr, error) {
	begin := time.Now()
	publishEventsCallCount.Add(1)

	if len(events) == 0 {
		return nil, nil
	}

	if !client.connected {
		return events, ErrNotConnected
	}

	body := client.encoder
	body.Reset()

	// encode events into bulk request buffer, dropping failed elements from
	// events slice
	events = bulkEncodePublishRequest(body, client.index, events)
	if len(events) == 0 {
		return nil, nil
	}

	requ := client.bulkRequ
	requ.Reset(body)
	status, result, sendErr := client.sendBulkRequest(requ)
	if sendErr != nil {
		logp.Err("Failed to perform any bulk index operations: %s", sendErr)
		return events, sendErr
	}

	debugf("PublishEvents: %d metrics have been  published to elasticsearch in %v.",
		len(events),
		time.Now().Sub(begin))

	// check response for transient errors
	var failedEvents []common.MapStr
	if status != 200 {
		failedEvents = events
	} else {
		client.json.init(result.raw)
		failedEvents = bulkCollectPublishFails(&client.json, events)
	}

	ackedEvents.Add(int64(len(events) - len(failedEvents)))
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
	index string,
	events []common.MapStr,
) []common.MapStr {
	okEvents := events[:0]
	for _, event := range events {
		meta := eventBulkMeta(index, event)
		err := body.Add(meta, event)
		if err != nil {
			logp.Err("Failed to encode event: %s", err)
			continue
		}

		okEvents = append(okEvents, event)
	}
	return okEvents
}

func eventBulkMeta(index string, event common.MapStr) bulkMeta {
	index = getIndex(event, index)
	meta := bulkMeta{
		Index: bulkMetaIndex{
			Index:   index,
			DocType: event["type"].(string),
		},
	}
	return meta
}

// getIndex returns the full index name
// Index is either defined in the config as part of the output
// or can be overload by the event through setting index
func getIndex(event common.MapStr, index string) string {

	ts := time.Time(event["@timestamp"].(common.Time)).UTC()

	// Check for dynamic index
	if _, ok := event["beat"]; ok {
		beatMeta, ok := event["beat"].(common.MapStr)
		if ok {
			// Check if index is set dynamically
			if dynamicIndex, ok := beatMeta["index"]; ok {
				dynamicIndexValue, ok := dynamicIndex.(string)
				if ok {
					index = dynamicIndexValue
				}
			}
		}
	}

	// Append timestamp to index
	index = fmt.Sprintf("%s-%d.%02d.%02d", index,
		ts.Year(), ts.Month(), ts.Day())

	return index
}

// bulkCollectPublishFails checks per item errors returning all events
// to be tried again due to error code returned for that items. If indexing an
// event failed due to some error in the event itself (e.g. does not respect mapping),
// the event will be dropped.
func bulkCollectPublishFails(
	reader *jsonReader,
	events []common.MapStr,
) []common.MapStr {
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

	count := len(events)
	failed := events[:0]
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
		failed = append(failed, events[i])
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
			reader.ignoreNext()
		}
	}

	if status < 0 {
		return 0, nil, errExpectedStatusCode
	}
	return status, msg, nil
}

func (client *Client) PublishEvent(event common.MapStr) error {
	if !client.connected {
		return ErrNotConnected
	}

	index := getIndex(event, client.index)
	debugf("Publish event: %s", event)

	// insert the events one by one
	status, _, err := client.Index(
		index, event["type"].(string), "", client.params, event)
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
	status, _, err := client.request("PUT", path, nil, template)

	if err != nil {
		return fmt.Errorf("Template could not be loaded. Error: %s", err)
	}
	if status != 200 {
		return fmt.Errorf("Template could not be loaded. Status: %v", status)
	}

	logp.Info("Elasticsearch template with name '%s' loaded", templateName)

	return nil
}

// CheckTemplate checks if a given template already exist. It returns true if
// and only if Elasticsearch returns with HTTP status code 200.
func (client *Client) CheckTemplate(templateName string) bool {

	status, _, _ := client.request("HEAD", "/_template/"+templateName, nil, nil)

	if status != 200 {
		return false
	}

	return true
}

func (conn *Connection) Connect(timeout time.Duration) error {
	var err error
	conn.connected, err = conn.Ping(timeout)
	if err != nil {
		return err
	}
	if !conn.connected {
		return ErrNotConnected
	}

	err = conn.onConnectCallback()
	if err != nil {
		return fmt.Errorf("Connection marked as failed because the onConnect callback failed: %v", err)
	}
	return nil
}

func (conn *Connection) Ping(timeout time.Duration) (bool, error) {
	debugf("ES Ping(url=%v, timeout=%v)", conn.URL, timeout)

	conn.http.Timeout = timeout
	status, _, err := conn.execRequest("HEAD", conn.URL, nil)
	if err != nil {
		debugf("Ping request failed with: %v", err)
		return false, err
	}

	debugf("Ping status code: %v", status)
	return status < 300, nil
}

func (conn *Connection) IsConnected() bool {
	return conn.connected
}

func (conn *Connection) Close() error {
	conn.connected = false
	return nil
}

func (conn *Connection) request(
	method, path string,
	params map[string]string,
	body interface{},
) (int, []byte, error) {
	url := makeURL(conn.URL, path, params)
	debugf("%s %s %v", method, url, body)

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
		conn.connected = false
		return 0, nil, err
	}
	defer closing(resp.Body)

	status := resp.StatusCode
	if status >= 300 {
		conn.connected = false
		return status, nil, fmt.Errorf("%v", resp.Status)
	}

	obj, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		conn.connected = false
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
