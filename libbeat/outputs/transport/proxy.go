package transport

import (
	"net"
	"net/url"

	"github.com/elastic/beats/libbeat/logp"
	"golang.org/x/net/proxy"
)

// ProxyConfig holds the configuration information required to proxy
// connections through a SOCKS5 proxy server.
type ProxyConfig struct {
	// URL of the SOCKS proxy. Scheme must be socks5. Username and password can be
	// embedded in the URL.
	URL string `config:"proxy_url"`

	// Resolve names locally instead of on the SOCKS server.
	LocalResolve bool `config:"proxy_use_local_resolver"`
}

func (c *ProxyConfig) Validate() error {
	if c.URL == "" {
		return nil
	}

	url, err := url.Parse(c.URL)
	if err != nil {
		return err
	}
	if _, err := proxy.FromURL(url, nil); err != nil {
		return err
	}

	return nil
}

func ProxyDialer(config *ProxyConfig, forward Dialer) (Dialer, error) {
	if config == nil || config.URL == "" {
		return forward, nil
	}

	url, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}

	if _, err := proxy.FromURL(url, nil); err != nil {
		return nil, err
	}

	logp.Info("proxy host: '%s'", url.Host)
	return DialerFunc(func(network, address string) (net.Conn, error) {
		var err error
		var addresses []string

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		if config.LocalResolve {
			addresses, err = net.LookupHost(host)
			if err != nil {
				logp.Warn(`DNS lookup failure "%s": %v`, host, err)
				return nil, err
			}
		} else {
			// Do not resolve the address locally. It will be resolved on the
			// SOCKS server. The beat will have no control over the randomization
			// of the IP used when multiple IPs are returned by DNS.
			addresses = []string{host}
		}

		dialer, err := proxy.FromURL(url, forward)
		if err != nil {
			return nil, err
		}
		return dialWith(dialer, network, host, addresses, port)
	}), nil
}
