//+build !windows

package helper

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/outputs/transport"
)

func makeDialer(t time.Duration, uri string) (transport.Dialer, string, error) {
	if strings.Contains(uri, "unix://") {
		return nil, fmt.Errorf(
			"cannot use %s as the URI, unix sockets are not supported on Windows, use npipe instead",
			uri,
		)
	}

	if strings.HasPrefix(uri, "http+npipe://") || strings.HasPrefix(uri, "npipe://") {
		s := strings.TrimPrefix(uri, "http+npipe://")
		s = strings.TrimPrefix(s, "npipe://")

		parts := strings.SplitN(s, "/", 2)

		sockFile, err := url.PathUnescape(parts[0])
		if err != nil {
			return nil, "", errors.Wrap(err, "could no decode path to the socket")
		}

		if len(parts) == 1 {
			return npipe.DialContext(npipe.TransformString(p)), "http://npipe/", nil
		}

		return npipe.DialContext(npipe.TransformString(p)), "http://npipe/" + parts[1], nil
	}

	return transport.NetDialer(t), uri, nil
}
