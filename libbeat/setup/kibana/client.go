package kibana

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type Connection struct {
	URL      string
	Username string
	Password string
	Headers  map[string]string

	http    *http.Client
	version string
}

type Client struct {
	Connection
}

func addPath(_url, _path string) (string, error) {

	u, err := url.Parse(_url)
	if err != nil {
		return "", fmt.Errorf("fail to parse URL %s: %v", _url, err)
	}
	u.Path = path.Join(u.Path, _path)
	return u.String(), nil
}

func NewKibanaClient(cfg *common.Config) (*Client, error) {

	config := defaultKibanaConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	kibanaURL, err := common.MakeURL(config.Protocol, config.Path, config.Host, 5601)
	if err != nil {
		return nil, fmt.Errorf("invalid Kibana host: %v", err)
	}

	u, err := url.Parse(kibanaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the Kibana URL: %v", err)
	}

	username := config.Username
	password := config.Password

	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
		u.User = nil

		// Re-write URL without credentials.
		kibanaURL = u.String()
	}

	logp.Info("Kibana url: %s", kibanaURL)

	var dialer, tlsDialer transport.Dialer

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, fmt.Errorf("fail to load the TLS config: %v", err)
	}

	dialer = transport.NetDialer(config.Timeout)
	tlsDialer, err = transport.TLSDialer(dialer, tlsConfig, config.Timeout)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Connection: Connection{
			URL:      kibanaURL,
			Username: username,
			Password: password,
			http: &http.Client{
				Transport: &http.Transport{
					Dial:    dialer.Dial,
					DialTLS: tlsDialer.Dial,
				},
				Timeout: config.Timeout,
			},
		},
	}

	if err = client.SetVersion(); err != nil {
		return nil, fmt.Errorf("fail to get the Kibana version:%v", err)
	}

	return client, nil
}

func (conn *Connection) Request(method, extraPath string, params url.Values, body io.Reader) (int, []byte, error) {

	reqURL, err := addPath(conn.URL, extraPath)
	if err != nil {
		return 0, nil, err
	}

	logp.Debug("kibana", "HTTP request URL: %s", reqURL)

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return 0, nil, fmt.Errorf("fail to create the HTTP %s request: %v", method, err)
	}

	if conn.Username != "" || conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	if method != "GET" {
		req.Header.Set("kbn-version", conn.version)
	}

	resp, err := conn.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("fail to execute the HTTP %s request: %v", method, err)
	}

	defer resp.Body.Close()

	var retError error
	if resp.StatusCode >= 300 {
		retError = fmt.Errorf("%v", resp.Status)
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("fail to read response %s", err)
	}

	return resp.StatusCode, result, retError
}

func (client *Client) SetVersion() error {

	type kibanaVersionResponse struct {
		Name    string `json:"name"`
		Version struct {
			Number   string `json:"number"`
			Snapshot bool   `json:"build_snapshot"`
		} `json:"version"`
	}

	_, result, err := client.Connection.Request("GET", "/api/status", nil, nil)
	if err != nil {
		return fmt.Errorf("HTTP GET request to /api/status fails: %v. Returns: %s.", err, result)
	}

	var kibanaVersion kibanaVersionResponse
	err = json.Unmarshal(result, &kibanaVersion)
	if err != nil {
		return fmt.Errorf("fail to unmarshal the response from GET %s/api/status: %v", client.Connection.URL, err)
	}

	client.version = kibanaVersion.Version.Number

	if kibanaVersion.Version.Snapshot {
		// needed for the tests
		client.version = client.version + "-SNAPSHOT"
	}

	return nil
}

func (client *Client) GetVersion() string {

	return client.version
}

func (client *Client) ImportJSON(url string, params url.Values, body io.Reader) error {

	statusCode, response, err := client.Connection.Request("POST", url, params, body)
	if err != nil {
		return fmt.Errorf("%v. Response: %s", err, response)
	}
	if statusCode >= 300 {
		return fmt.Errorf("returned %d to import file: %v. Response: %s", statusCode, err, response)
	}
	return nil
}

func (client *Client) Close() error {
	return nil
}
