package kerberos

import (
	"fmt"
	"net/http"
	"net/url"

	krbclient "gopkg.in/jcmturner/gokrb5.v7/client"
	krbconfig "gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/keytab"
	"gopkg.in/jcmturner/gokrb5.v7/spnego"
)

type Client struct {
	spClient *spnego.Client
}

func NewClient(config *Config, httpClient *http.Client, esurl string) (*Client, error) {
	var krbClient *krbclient.Client
	krbConf, err := krbconfig.Load(config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error creating Kerberos client: %+v", err)
	}

	switch config.AuthType {
	case AUTH_KEYTAB:
		kTab, err := keytab.Load(config.KeyTabPath)
		if err != nil {
			return nil, fmt.Errorf("cannot load keytab file %s: %+v", config.KeyTabPath, err)
		}
		krbClient = krbclient.NewClientWithKeytab(config.Username, config.Realm, kTab, krbConf)
	case AUTH_PASSWORD:
		krbClient = krbclient.NewClientWithPassword(config.Username, config.Realm, config.Password, krbConf)
	default:
		return nil, InvalidAuthType
	}

	parsedURL, err := url.Parse(esurl)
	if err != nil {
		return nil, fmt.Errorf("cannot parse elasticsearch URL %s: %v", esurl, err)
	}
	spn := fmt.Sprintf("HTTP/%s@%s", parsedURL.Hostname(), config.Realm)
	return &Client{
		spClient: spnego.NewClient(krbClient, httpClient, spn),
	}, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.spClient.Do(req)
}

func (c *Client) CloseIdleConnections() {
	c.spClient.CloseIdleConnections()
}
