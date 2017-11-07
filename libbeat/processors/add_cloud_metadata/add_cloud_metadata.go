package add_cloud_metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

const (
	// metadataHost is the IP that each of the cloud providers supported here
	// use for their metadata service.
	metadataHost = "169.254.169.254"

	// Default config
	defaultTimeOut = 3 * time.Second
)

var debugf = logp.MakeDebug("filters")

// init registers the add_cloud_metadata processor.
func init() {
	processors.RegisterPlugin("add_cloud_metadata", newCloudMetadata)
}

type schemaConv func(m map[string]interface{}) common.MapStr

// responseHandler is the callback function that used to write something
// to the result according the HTTP response.
type responseHandler func(all []byte, res *result) error

type metadataFetcher struct {
	provider         string
	headers          map[string]string
	responseHandlers map[string]responseHandler
	conv             schemaConv
}

// fetchRaw queries raw metadata from a hosting provider's metadata service.
func (f *metadataFetcher) fetchRaw(
	ctx context.Context,
	client http.Client,
	url string,
	responseHandler responseHandler,
	result *result,
) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.err = errors.Wrapf(err, "failed to create http request for %v", f.provider)
		return
	}
	for k, v := range f.headers {
		req.Header.Add(k, v)
	}
	req = req.WithContext(ctx)

	rsp, err := client.Do(req)
	if err != nil {
		result.err = errors.Wrapf(err, "failed requesting %v metadata", f.provider)
		return
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		result.err = errors.Errorf("failed with http status code %v", rsp.StatusCode)
		return
	}

	all, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		result.err = errors.Wrapf(err, "failed requesting %v metadata", f.provider)
		return
	}

	// Decode JSON.
	err = responseHandler(all, result)
	if err != nil {
		result.err = err
		return
	}

	return
}

// fetchMetadata queries metadata from a hosting provider's metadata service.
// Some providers require multiple HTTP requests to gather the whole metadata,
// len(f.responseHandlers)  > 1 indicates that multiple requests are needed.
func (f *metadataFetcher) fetchMetadata(ctx context.Context, client http.Client) result {
	res := result{provider: f.provider, metadata: common.MapStr{}}
	for url, responseHandler := range f.responseHandlers {
		f.fetchRaw(ctx, client, url, responseHandler, &res)
		if res.err != nil {
			return res
		}
	}

	// Apply schema.
	res.metadata = f.conv(res.metadata)
	res.metadata["provider"] = f.provider

	return res
}

// result is the result of a query for a specific hosting provider's metadata.
type result struct {
	provider string        // Hosting provider type.
	err      error         // Error that occurred while fetching (if any).
	metadata common.MapStr // A specific subset of the metadata received from the hosting provider.
}

func (r result) String() string {
	return fmt.Sprintf("result=[provider:%v, error=%v, metadata=%v]",
		r.provider, r.err, r.metadata)
}

// writeResult blocks until it can write the result r to the channel c or until
// the context times out.
func writeResult(ctx context.Context, c chan result, r result) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c <- r:
		return nil
	}
}

// fetchMetadata attempts to fetch metadata in parallel from each of the
// hosting providers supported by this processor. It wait for the results to
// be returned or for a timeout to occur then returns the results that
// completed in time.
func fetchMetadata(metadataFetchers []*metadataFetcher, timeout time.Duration) *result {
	debugf("add_cloud_metadata: starting to fetch metadata, timeout=%v", timeout)
	start := time.Now()
	defer func() {
		debugf("add_cloud_metadata: fetchMetadata ran for %v", time.Since(start))
	}()

	// Create HTTP client with our timeouts and keep-alive disabled.
	client := http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 0,
			}).DialContext,
		},
	}

	// Create context to enable explicit cancellation of the http requests.
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	c := make(chan result)
	for _, fetcher := range metadataFetchers {
		go func(fetcher *metadataFetcher) {
			writeResult(ctx, c, fetcher.fetchMetadata(ctx, client))
		}(fetcher)
	}

	for i := 0; i < len(metadataFetchers); i++ {
		select {
		case result := <-c:
			debugf("add_cloud_metadata: received disposition for %v after %v. %v",
				result.provider, time.Since(start), result)
			// Bail out on first success.
			if result.err == nil && result.metadata != nil {
				return &result
			}
		case <-ctx.Done():
			debugf("add_cloud_metadata: timed-out waiting for all responses")
			return nil
		}
	}

	return nil
}

// getMetadataURLs loads config and generates the metadata URLs.
func getMetadataURLs(c *common.Config, defaultHost string, metadataURIs []string) ([]string, error) {
	var urls []string
	config := struct {
		MetadataHostAndPort string `config:"host"` // Specifies the host and port of the metadata service (for testing purposes only).
	}{
		MetadataHostAndPort: defaultHost,
	}
	err := c.Unpack(&config)
	if err != nil {
		return urls, errors.Wrap(err, "failed to unpack add_cloud_metadata config")
	}
	for _, uri := range metadataURIs {
		urls = append(urls, "http://"+config.MetadataHostAndPort+uri)
	}
	return urls, nil
}

// makeJSONPicker returns a responseHandler function that unmarshals JSON
// from a hosting provider's HTTP response and writes it to the result.
func makeJSONPicker(provider string) responseHandler {
	return func(all []byte, res *result) error {
		dec := json.NewDecoder(bytes.NewReader(all))
		dec.UseNumber()
		err := dec.Decode(&res.metadata)
		if err != nil {
			err = errors.Wrapf(err, "failed to unmarshal %v JSON of '%v'", provider, string(all))
			return err
		}
		return nil
	}
}

// newMetadataFetcher return metadataFetcher with one pass JSON responseHandler.
func newMetadataFetcher(
	c *common.Config,
	provider string,
	headers map[string]string,
	host string,
	conv schemaConv,
	uri string,
) (*metadataFetcher, error) {
	urls, err := getMetadataURLs(c, host, []string{uri})
	if err != nil {
		return nil, err
	}
	responseHandlers := map[string]responseHandler{urls[0]: makeJSONPicker(provider)}
	fetcher := &metadataFetcher{provider, headers, responseHandlers, conv}
	return fetcher, nil
}

func setupFetchers(c *common.Config) ([]*metadataFetcher, error) {
	var fetchers []*metadataFetcher
	doFetcher, err := newDoMetadataFetcher(c)
	if err != nil {
		return fetchers, err
	}
	ec2Fetcher, err := newEc2MetadataFetcher(c)
	if err != nil {
		return fetchers, err
	}
	gceFetcher, err := newGceMetadataFetcher(c)
	if err != nil {
		return fetchers, err
	}
	qcloudFetcher, err := newQcloudMetadataFetcher(c)
	if err != nil {
		return fetchers, err
	}
	ecsFetcher, err := newAlibabaCloudMetadataFetcher(c)
	if err != nil {
		return fetchers, err
	}
	azFetcher, err := newAzureVmMetadataFetcher(c)
	if err != nil {
		return fetchers, err
	}

	fetchers = []*metadataFetcher{
		doFetcher,
		ec2Fetcher,
		gceFetcher,
		qcloudFetcher,
		ecsFetcher,
		azFetcher,
	}
	return fetchers, nil
}

func newCloudMetadata(c *common.Config) (processors.Processor, error) {
	config := struct {
		Timeout time.Duration `config:"timeout"` // Amount of time to wait for responses from the metadata services.
	}{
		Timeout: defaultTimeOut,
	}
	err := c.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack add_cloud_metadata config")
	}

	fetchers, err := setupFetchers(c)
	if err != nil {
		return nil, err
	}

	result := fetchMetadata(fetchers, config.Timeout)
	if result == nil {
		logp.Info("add_cloud_metadata: hosting provider type not detected.")
		return &addCloudMetadata{}, nil
	}

	logp.Info("add_cloud_metadata: hosting provider type detected as %v, metadata=%v",
		result.provider, result.metadata.String())

	return &addCloudMetadata{metadata: result.metadata}, nil
}

type addCloudMetadata struct {
	metadata common.MapStr
}

func (p addCloudMetadata) Run(event *beat.Event) (*beat.Event, error) {
	if len(p.metadata) == 0 {
		return event, nil
	}

	// This overwrites the meta.cloud if it exists. But the cloud key should be
	// reserved for this processor so this should happen.
	_, err := event.PutValue("meta.cloud", p.metadata)

	return event, err
}

func (p addCloudMetadata) String() string {
	return "add_cloud_metadata=" + p.metadata.String()
}
