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
	// AWS EC2 Metadata Service
	ec2MetadataHost        = "169.254.169.254"
	ec2InstanceIdentityURI = "/2014-02-25/dynamic/instance-identity/document"

	// DigitalOcean Metadata Service
	doMetadataHost = "169.254.169.254"
	doMetadataURI  = "/metadata/v1.json"

	// Google GCE Metadata Service
	gceMetadataHost = "169.254.169.254"
	gceMetadataURI  = "/computeMetadata/v1/?recursive=true&alt=json"
)

var debugf = logp.MakeDebug("filters")

var (
	ec2URL    = "http://" + ec2MetadataHost + ec2InstanceIdentityURI
	ec2Schema = s.Schema{
		"instance_id":       c.Str("instanceId"),
		"machine_type":      c.Str("instanceType"),
		"region":            c.Str("region"),
		"availability_zone": c.Str("availabilityZone"),
	}.Apply

	doURL    = "http://" + doMetadataHost + doMetadataURI
	doSchema = s.Schema{
		"instance_id": c.StrFromNum("droplet_id"),
		"region":      c.Str("region"),
	}.Apply

	gceHeaders = map[string]string{"Metadata-Flavor": "Google"}
	gceURL     = "http://" + gceMetadataHost + gceMetadataURI
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
)

// init registers the add_cloud_metadata processor.
func init() {
	processors.RegisterPlugin("add_cloud_metadata", newCloudMetadata)
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

// fetchJSON query metadata from a hosting provider's metadata service.
func fetchJSON(
	provider string,
	headers map[string]string,
	url string,
	conv func(map[string]interface{}) common.MapStr,
	client http.Client,
	ctx context.Context,
) result {
	result := result{provider: provider}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.err = errors.Wrapf(err, "failed to create http request for %v", provider)
		return result
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	req = req.WithContext(ctx)

	rsp, err := client.Do(req)
	if err != nil {
		result.err = errors.Wrapf(err, "failed requesting %v metadata", provider)
		return result
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		result.err = errors.Errorf("failed with http status code %v", rsp.StatusCode)
		return result
	}

	all, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		result.err = errors.Wrapf(err, "failed requesting %v metadata", provider)
		return result
	}

	// Decode JSON.
	dec := json.NewDecoder(bytes.NewReader(all))
	dec.UseNumber()
	err = dec.Decode(&result.metadata)
	if err != nil {
		result.err = errors.Wrapf(err, "failed to unmarshal %v JSON of '%v'", provider, string(all))
		return result
	}

	// Apply schema.
	result.metadata = conv(result.metadata)
	result.metadata["provider"] = provider

	return result
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
func fetchMetadata(timeout time.Duration) *result {
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
	go func() { writeResult(ctx, c, fetchJSON("digitalocean", nil, doURL, doSchema, client, ctx)) }()
	go func() { writeResult(ctx, c, fetchJSON("ec2", nil, ec2URL, ec2Schema, client, ctx)) }()
	go func() { writeResult(ctx, c, fetchJSON("gce", gceHeaders, gceURL, gceSchema, client, ctx)) }()

	for i := 0; i < 3; i++ {
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

type addCloudMetadata struct {
	metadata common.MapStr
}

func newCloudMetadata(c common.Config) (processors.Processor, error) {
	config := struct {
		Timeout time.Duration `config:"timeout"` // Amount of time to wait for responses from the metadata services.
	}{
		Timeout: 3 * time.Second,
	}
	err := c.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpack add_cloud_metadata config")
	}

	result := fetchMetadata(config.Timeout)
	if result == nil {
		logp.Info("add_cloud_metadata: hosting provider type not detected.")
		return addCloudMetadata{}, nil
	}

	logp.Info("add_cloud_metadata: hosting provider type detected as %v, metadata=%v",
		result.provider, result.metadata.String())

	return addCloudMetadata{metadata: result.metadata}, nil
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
