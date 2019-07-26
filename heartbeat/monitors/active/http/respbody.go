package http

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"unicode/utf8"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/heartbeat/reason"
	"github.com/elastic/beats/libbeat/beat"
)

func handleRespBody(event *beat.Event, resp *http.Response, responseConfig responseConfig, errReason reason.Reason) error {
	defer resp.Body.Close()

	sampleMaxBytes := responseConfig.IncludeBodyMaxBytes

	includeSample := responseConfig.IncludeBody == "always" || (responseConfig.IncludeBody == "on_error" && errReason != nil)

	// No need to return any actual body bytes if we'll discard them anyway. This should save on allocation
	if !includeSample {
		sampleMaxBytes = 0
	}

	sampleStr, bodyBytes, bodyHash := readBody(resp, sampleMaxBytes)

	if includeSample {
		addRespBodyFields(event, sampleStr, bodyBytes, bodyHash)
	}

	return nil
}

func addRespBodyFields(event *beat.Event, sampleStr string, bodyBytes int64, bodyHash string) {
	body := common.MapStr{"bytes": bodyBytes}
	if sampleStr != "" {
		body["content"] = sampleStr
	}
	if bodyHash != "" {
		body["hash"] = bodyHash
	}

	eventext.MergeEventFields(event, common.MapStr{"http": common.MapStr{
		"response": common.MapStr{
			"body": body,
		},
	}})
}

// readBody reads the first sampleSize bytes from the httpResponse,
// then closes the body (which closes the connection). It doesn't return any errors
// but does log them. During an error case the return values will be (nil, -1).
// The maxBytes params controls how many bytes will be returned in a string, not how many will be read.
// We always read the full response here since we want to time downloading the full thing.
// This may return a nil body if the response is not valid UTF-8
func readBody(resp *http.Response, maxSampleBytes int) (bodySample string, bodySize int64, hashStr string) {
	if resp == nil {
		return "", -1, ""
	}

	hash := sha256.New()

	// Function to lazily get the body of the response
	rawBuf := make([]byte, 1024)
	sampleBuf := make([]byte, maxSampleBytes)
	sampleRemainingBytes := maxSampleBytes
	sampleWriteOffset := 0
	respSize := int64(0)
	for {
		readSize, readErr := resp.Body.Read(rawBuf)

		// Create a truncated buffer to the number of bytes that were read
		truncBuf := rawBuf[:readSize]

		respSize += int64(readSize)
		hash.Write(truncBuf)

		if sampleRemainingBytes > 0 {
			if readSize >= sampleRemainingBytes {
				copy(sampleBuf[sampleWriteOffset:maxSampleBytes-1], truncBuf[:sampleRemainingBytes])
			} else {
				copy(sampleBuf[sampleWriteOffset:sampleWriteOffset+readSize], truncBuf)
			}
			sampleRemainingBytes -= readSize
			sampleWriteOffset += readSize
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			logp.Warn("could not read HTTP response body: %s", readErr)
			break
		}
	}

	if utf8.Valid(sampleBuf[:sampleWriteOffset]) {
		bodySample = string(sampleBuf[:sampleWriteOffset])
	}

	return bodySample, respSize, hex.EncodeToString(hash.Sum(nil))
}
