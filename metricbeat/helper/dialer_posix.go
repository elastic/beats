//+build !windows

package helper

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/api/npipe"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

func makeDialer(t time.Duration, uri string) (transport.Dialer, string, error) {
	if npipe.IsNPipe(uri) {
		return nil, "", fmt.Errorf(
			"cannot use %s as the URI, named pipes are only supported on Windows",
			uri,
		)
	}

	if strings.HasPrefix(uri, "http+unix://") || strings.HasPrefix(uri, "unix://") {
		u, err := url.Parse(uri)
		if err != nil {
			return nil, "", errors.Wrap(err, "fail to parse URI")
		}

		sockFile := u.Path

		q := u.Query()
		path := q.Get("__path")
		if path != "" {
			path, err = url.PathUnescape(path)
			if err != nil {
				return nil, "", fmt.Errorf("could not unescape resource path %s", path)
			}
		}
		q.Del("__path")

		var qStr string
		if encoded := q.Encode(); encoded != "" {
			qStr = "?" + encoded
		}

		return transport.UnixDialer(t, sockFile), "http://unix/" + path + qStr, nil
	}

	return transport.NetDialer(t), uri, nil
}
