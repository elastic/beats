package http

import (
	"io"
	"io/ioutil"
	"net/http"
	"unicode/utf8"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/heartbeat/reason"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
)

func handleRespBody(event *beat.Event, resp *http.Response, responseConfig responseConfig, errReason reason.Reason) error {
	defer resp.Body.Close()

	if responseConfig.IncludeBody == "always" || (errReason != nil && responseConfig.IncludeBody == "on_error") {
		err := addRespBodyFields(event, resp, responseConfig, errReason)
		if err != nil {
			return err
		}
	} else {
		// Even if we don't need the body we read it in its entirety to ensure we download the full thing.
		// This is important in terms of timing the full request length
		_, err := io.Copy(ioutil.Discard, resp.Body)
		return err
	}

	return nil
}

func addRespBodyFields(event *beat.Event, resp *http.Response, responseConfig responseConfig, errReason reason.Reason) error {
	maxBodyBytes := responseConfig.IncludeBodyMaxBytes
	if responseConfig.IncludeBody == "never" {
		// Don't return the body if the config says not to
		maxBodyBytes = 0
	} else if errReason == nil && responseConfig.IncludeBody == "on_error" {
		// If configured to only return the body on error, don't return it on success
		maxBodyBytes = 0
	}
	respBody, respSize := readBodyPrefix(resp, maxBodyBytes)

	if resp != nil {
		if respSize > -1 {
			eventext.MergeEventFields(event, common.MapStr{"http": common.MapStr{
				"response": common.MapStr{
					"body.content": respBody,
					"body.bytes":   respSize,
				},
			}})
		}
	}

	return nil
}

// readBodyPrefix reads the first sampleSize bytes from the httpResponse,
// then closes the body (which closes the connection). It doesn't return any errors
// but does log them. During an error case the return values will be (nil, -1).
// The maxBytes params controls how many bytes will be returned in a string, not how many will be read.
// We always read the full response here since we want to time downloading the full thing.
// This may return a nil body if the response is not valid UTF-8
func readBodyPrefix(resp *http.Response, maxBytes int) (bodySample *string, bodySize int64) {
	if resp == nil {
		return nil, -1
	}

	// Function to lazily get the body of the response
	buf := make([]byte, maxBytes)
	respSize := int64(-1)
	if resp != nil {
		startSize, readErr := resp.Body.Read(buf)
		if startSize > 0 {
			buf = buf[0:startSize]
			// Read the entirety of the body. Otherwise, the stats for the check
			// don't include download time.
			restSize, _ := io.Copy(ioutil.Discard, resp.Body)
			respSize = int64(startSize) + restSize
		} else if readErr != nil {
			logp.Warn("could not read HTTP response body after ping: %s", readErr)
			buf = buf[:0]
		}
	}

	if utf8.Valid(buf) {
		bodyStr := string(buf)
		return &bodyStr, respSize
	} else {
		return nil, respSize
	}
}
