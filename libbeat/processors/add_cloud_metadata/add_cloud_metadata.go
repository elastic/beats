package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstriface"
	"github.com/pkg/errors"
)

const (
	// metadataHost is the IP that each of the cloud providers supported here
	// use for their metadata service.
	metadataHost = "169.254.169.254"

	// AWS EC2 Metadata Service
	ec2InstanceIdentityURI = "/2014-02-25/dynamic/instance-identity/document"

	// DigitalOcean Metadata Service
	doMetadataURI = "/metadata/v1.json"

	// Google GCE Metadata Service
	gceMetadataURI = "/computeMetadata/v1/?recursive=true&alt=json"

	// Tencent Clound Metadata Service
	qcloudMetadataHost          = "metadata.tencentyun.com"
	qcloudMetadataInstanceIdURI = "/meta-data/instance-id"
	qcloudMetadataRegionURI     = "/meta-data/placement/region"
	qcloudMetadataZoneURI       = "/meta-data/placement/zone"
)

var debugf = logp.MakeDebug("filters")

// metadata schemas for all prividers
var (
	ec2Schema = func(m map[string]interface{}) common.MapStr {
		out, _ := s.Schema{
			"instance_id":       c.Str("instanceId"),
			"machine_type":      c.Str("instanceType"),
			"region":            c.Str("region"),
			"availability_zone": c.Str("availabilityZone"),
		}.Apply(m)
		return out
	}

	doSchema = func(m map[string]interface{}) common.MapStr {
		out, _ := s.Schema{
			"instance_id": c.StrFromNum("droplet_id"),
			"region":      c.Str("region"),
		}.Apply(m)
		return out
	}

	gceHeaders = map[string]string{"Metadata-Flavor": "Google"}
	gceSchema  = func(m map[string]interface{}) common.MapStr {
		out := common.MapStr{}

		if instance, ok := m["instance"].(map[string]interface{}); ok {
			s.Schema{
				"instance_id":       c.StrFromNum("id"),
				"machine_type":      c.Str("machineType"),
				"availability_zone": c.Str("zone"),
			}.ApplyTo(out, instance)
		}

		if project, ok := m["project"].(map[string]interface{}); ok {
			s.Schema{
				"project_id": c.Str("projectId"),
			}.ApplyTo(out, project)
		}

		return out
	}

	qcloudSchema = func(m map[string]interface{}) common.MapStr {
		out, _ := s.Schema{
			"instance_id":       c.Str("instance_id"),
			"region":            c.Str("region"),
			"availability_zone": c.Str("zone"),
		}.Apply(m)
		return out
	}
)

// init registers the add_cloud_metadata processor.
func init() {
	processors.RegisterPlugin("add_cloud_metadata", newCloudMetadata)
}

type schemaConv func(m map[string]interface{}) common.MapStr

type pick func([]byte, *result) error

type metadataFetcher struct {
	provider string
	headers  map[string]string
	pickers  map[string]pick
	conv     schemaConv
}

// fetchRaw query raw metadata from a hosting provider's metadata service.
func (f *metadataFetcher) fetchRaw(
	ctx context.Context,
	client http.Client,
	url string,
	pick pick,
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
	err = pick(all, result)
	if err != nil {
		result.err = err
		return
	}

	return
}

// fetchMetadata query metadata from a hosting provider's metadata service.
// some providers require multiple HTTP request to gather the whole metadata
// len(f.pickers)  > 1 indicates that multiple requesting is needed
func (f *metadataFetcher) fetchMetadata(ctx context.Context, client http.Client) result {
	res := result{provider: f.provider, metadata: common.MapStr{}}
	for url, pick := range f.pickers {
		f.fetchRaw(ctx, client, url, pick, &res)
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

// getMetadataURL loads config and generate metadata url
func getMetadataURL(c common.Config, defaultHost string, timeout time.Duration, metadataURIs []string) ([]string, error) {
	var urls []string
	config := struct {
		MetadataHostAndPort string        `config:"host"`    // Specifies the host and port of the metadata service (for testing purposes only).
		Timeout             time.Duration `config:"timeout"` // Amount of time to wait for responses from the metadata services.
	}{
		MetadataHostAndPort: defaultHost,
		Timeout:             timeout,
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

// makeCommomJSONPicker generate fetch function which query json metadata from a hosting provider's HTTP response
func makeCommomJSONPicker(provider string) pick {
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

// newMetadataFetcher return metadataFetcher with one pass json picker
func newMetadataFetcher(
	c common.Config,
	timeout time.Duration,
	provider string,
	headers map[string]string,
	host string,
	conv schemaConv,
	uris []string,
) (*metadataFetcher, error) {
	urls, err := getMetadataURL(c, host, timeout, uris)
	if err != nil {
		return nil, err
	}
	picker := map[string]pick{urls[0]: makeCommomJSONPicker(provider)}
	fetcher := &metadataFetcher{provider, headers, picker, conv}
	return fetcher, nil
}

func newDoMetadataFetcher(c common.Config, timeout time.Duration) (*metadataFetcher, error) {
	fetcher, err := newMetadataFetcher(c, timeout, "digitalocean", nil, metadataHost, doSchema, []string{
		doMetadataURI,
	})
	return fetcher, err
}

func newEc2MetadataFetcher(c common.Config, timeout time.Duration) (*metadataFetcher, error) {
	fetcher, err := newMetadataFetcher(c, timeout, "ec2", nil, metadataHost, ec2Schema, []string{
		ec2InstanceIdentityURI,
	})
	return fetcher, err
}

func newGceMetadataFetcher(c common.Config, timeout time.Duration) (*metadataFetcher, error) {
	fetcher, err := newMetadataFetcher(c, timeout, "gce", gceHeaders, metadataHost, gceSchema, []string{
		gceMetadataURI,
	})
	return fetcher, err
}

// newQcloudMetadataFetcher return the concrete metadata fetcher for qcloud provider
// which requires more than one way to assemble the metadata
func newQcloudMetadataFetcher(c common.Config, timeout time.Duration) (*metadataFetcher, error) {
	urls, err := getMetadataURL(c, qcloudMetadataHost, timeout, []string{
		qcloudMetadataInstanceIdURI,
		qcloudMetadataRegionURI,
		qcloudMetadataZoneURI,
	})
	if err != nil {
		return nil, err
	}
	picker := map[string]pick{
		urls[0]: func(all []byte, result *result) error {
			result.metadata["instance_id"] = string(all)
			return nil
		},
		urls[1]: func(all []byte, result *result) error {
			result.metadata["region"] = string(all)
			return nil
		},
		urls[2]: func(all []byte, result *result) error {
			result.metadata["zone"] = string(all)
			return nil
		},
	}
	fetcher := &metadataFetcher{"qcloud", nil, picker, qcloudSchema}
	return fetcher, nil
}

func newCloudMetadata(c common.Config) (processors.Processor, error) {
	timeout := 3 * time.Second

	doFetcher, err := newDoMetadataFetcher(c, timeout)
	if err != nil {
		return nil, err
	}
	ec2Fetcher, err := newEc2MetadataFetcher(c, timeout)
	if err != nil {
		return nil, err
	}
	gceFetcher, err := newGceMetadataFetcher(c, timeout)
	if err != nil {
		return nil, err
	}
	qcloudFetcher, err := newQcloudMetadataFetcher(c, timeout)
	if err != nil {
		return nil, err
	}

	var fetchers = []*metadataFetcher{
		doFetcher,
		ec2Fetcher,
		gceFetcher,
		qcloudFetcher,
	}

	result := fetchMetadata(fetchers, timeout)
	if result == nil {
		logp.Info("add_cloud_metadata: hosting provider type not detected.")
		return addCloudMetadata{}, nil
	}

	logp.Info("add_cloud_metadata: hosting provider type detected as %v, metadata=%v",
		result.provider, result.metadata.String())

	return addCloudMetadata{metadata: result.metadata}, nil
}

type addCloudMetadata struct {
	metadata common.MapStr
}

func (p addCloudMetadata) Run(event common.MapStr) (common.MapStr, error) {
	if len(p.metadata) == 0 {
		return event, nil
	}

	// This overwrites the meta.cloud if it exists. But the cloud key should be
	// reserved for this processor so this should happen.
	_, err := event.Put("meta.cloud", p.metadata)

	return event, err
}

func (p addCloudMetadata) String() string {
	return "add_cloud_metadata=" + p.metadata.String()
}
