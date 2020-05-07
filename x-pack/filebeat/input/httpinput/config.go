// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpinput

// Config contains information about httpjson configuration
type config struct {
	UseSSL          bool   `config:"ssl"`
	SSLCertificate  string `config:"ssl_certificate"`
	SSLKey          string `config:"ssl_key"`
	SSLCA           string `config:"ssl_certificate_authorities"`
	BasicAuth       bool   `config:"basic_auth"`
	Username        string `config:"username"`
	Password        string `config:"password"`
	ResponseCode    int    `config:"response_code"`
	ResponseBody    string `config:"response_body"`
	ResponseHeaders string `config:"response_headers"`
	ListenAddress   string `config:"listen_address"`
	ListenPort      string `config:"listen_port"`
	URL             string `config:"url"`
	Prefix          string `config:"prefix"`
}

func defaultConfig() config {
	var c config
	c.UseSSL = false
	c.SSLCertificate = ""
	c.SSLKey = ""
	c.SSLCA = ""
	c.BasicAuth = false
	c.Username = ""
	c.Password = ""
	c.ResponseCode = 200
	c.ResponseBody = `{"message": "success"}`
	c.ResponseHeaders = `{"Content-Type": "application/json"}`
	c.ListenAddress = ""
	c.ListenPort = "8000"
	c.URL = "/"
	c.Prefix = "json"
	return c
}
