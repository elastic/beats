package checksum

import (
	"crypto/md5"
	"encoding/base64"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
)

const contentMD5Header = "Content-Md5"

// AddBodyContentMD5Handler computes and sets the HTTP Content-MD5 header for requests that
// require it.
func AddBodyContentMD5Handler(r *aws.Request) {
	// if Content-MD5 header is already present, return
	if v := r.HTTPRequest.Header.Get(contentMD5Header); len(v) != 0 {
		return
	}

	h := md5.New()

	// hash the body.  seek back to the first position after reading to reset
	// the body for transmission.  copy errors may be assumed to be from the
	// body.
	_, err := io.Copy(h, r.Body)
	if err != nil {
		r.Error = awserr.New("ContentMD5", "failed to read body", err)
		return
	}
	_, err = r.Body.Seek(0, io.SeekStart)
	if err != nil {
		r.Error = awserr.New("ContentMD5", "failed to seek body", err)
		return
	}

	// encode the md5 checksum in base64 and set the request header.
	sum := h.Sum(nil)
	sum64 := make([]byte, base64.StdEncoding.EncodedLen(len(sum)))
	base64.StdEncoding.Encode(sum64, sum)
	r.HTTPRequest.Header.Set("Content-MD5", string(sum64))
}
