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
	Connection
	index string
}

type Connection struct {
	URL      string
	Username string
	Password string

	http      *http.Client
	connected bool
}

func NewClient(
	url, index string, tls *tls.Config,
	username, password string,
) *Client {
	client := &Client{
		Connection{
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

func (client *Client) Clone() *Client {
	newClient := &Client{
		Connection{
			URL:      client.URL,
			Username: client.Username,
			Password: client.Password,
			http: &http.Client{
				Transport: client.http.Transport,
			},
			connected: false,
		},
		client.index,
	}
	return newClient
}

func (client *Client) PublishEvents(
	events []common.MapStr,
) (n int, err error) {
	if !client.connected {
		return 0, ErrNotConnected
	}

	request, err := client.startBulkRequest("", "", nil)
	if err != nil {
		logp.Err(
			"Failed to perform many index operations in a single API call: %s",
			err)
		return 0, err
	}

	for _, event := range events {
		ts := time.Time(event["@timestamp"].(common.Time))
		index := fmt.Sprintf("%s-%d.%02d.%02d",
			client.index, ts.Year(), ts.Month(), ts.Day())
		meta := bulkMeta{
			Index: bulkMetaIndex{
				Index:   index,
				DocType: event["type"].(string),
			},
		}
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

func (client *Client) PublishEvent(event common.MapStr) error {
	if !client.connected {
		return ErrNotConnected
	}

	ts := time.Time(event["@timestamp"].(common.Time))
	index := fmt.Sprintf("%s-%d.%02d.%02d",
		client.index, ts.Year(), ts.Month(), ts.Day())
	logp.Debug("output_elasticsearch", "Publish event: %s", event)

	// insert the events one by one
	_, err := client.Index(index, event["type"].(string), "", nil, event)
	if err != nil {
		logp.Warn("Fail to insert a single event: %s", err)
	}
	return nil
}

func (conn *Connection) Connect(timeout time.Duration) error {
	var err error
	conn.connected, err = conn.Ping()
	if err != nil {
		return err
	}
	if !conn.connected {
		return ErrNotConnected
	}
	return nil
}

func (conn *Connection) Ping() (bool, error) {
	conn.http.Timeout = defaultEsOpenTimeout
	resp, err := conn.http.Head(conn.URL)
	if err != nil {
		return false, err
	}
	defer closing(resp.Body)

	status := resp.StatusCode
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
) ([]byte, error) {
	url := makeURL(conn.URL, path, params)
	logp.Debug("elasticsearch", "%s %s %s", method, url, body)

	var obj []byte
	if body != nil {
		var err error
		obj, err = json.Marshal(body)
		if err != nil {
			return nil, ErrJSONEncodeFailed
		}
	}

	return conn.execRequest(method, url, bytes.NewReader(obj))
}

func (conn *Connection) execRequest(
	method, url string,
	body io.Reader,
) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		logp.Warn("Failed to create request", err)
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	if conn.Username != "" || conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	resp, err := conn.http.Do(req)
	if err != nil {
		conn.connected = false
		return nil, err
	}
	defer closing(resp.Body)

	status := resp.StatusCode
	if status >= 300 {
		conn.connected = false
		return nil, fmt.Errorf("%v", resp.Status)
	}

	obj, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		conn.connected = false
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
