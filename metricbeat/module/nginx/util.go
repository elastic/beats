package nginx

import (
	"fmt"
	"net/url"
	"strings"
)

// Ftoi returns a copy of input map where float values are casted to int.
// The conversion is applied to nested maps and arrays as well.
func Ftoi(in map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}

	for k, v := range in {
		switch v.(type) {
		case float64:
			vt := v.(float64)
			out[k] = int(vt)
		case map[string]interface{}:
			vt := v.(map[string]interface{})
			out[k] = Ftoi(vt)
		case []interface{}:
			vt := v.([]interface{})
			l := len(vt)
			a := make([]interface{}, l)
			for i := 0; i < l; i++ {
				e := vt[i]
				switch e.(type) {
				case float64:
					et := e.(float64)
					a[i] = int(et)
				case map[string]interface{}:
					et := e.(map[string]interface{})
					a[i] = Ftoi(et)
				default:
					a[i] = e
				}
			}
			out[k] = a
		default:
			out[k] = v
		}
	}

	return out
}

// GetURL constructs a URL from the rawHost value and path if one was not set in the rawHost value.
func GetURL(statusPath, rawHost string) (*url.URL, error) {
	u, err := url.Parse(rawHost)
	if err != nil {
		return nil, fmt.Errorf("error parsing nginx host: %v", err)
	}

	if u.Scheme == "" {
		// Add scheme and re-parse.
		u, err = url.Parse(fmt.Sprintf("%s://%s", "http", rawHost))
		if err != nil {
			return nil, fmt.Errorf("error parsing nginx host: %v", err)
		}
	}

	if u.Host == "" {
		return nil, fmt.Errorf("error parsing nginx host: empty host")
	}

	if u.Path == "" {
		// The path given in the host config takes precedence over the
		// server_status_path config value.
		path := statusPath
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		u.Path = path
	}

	return u, nil
}
