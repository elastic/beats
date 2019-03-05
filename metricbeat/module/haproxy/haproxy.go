// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package haproxy

import (
	"bytes"
	"encoding/csv"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/gocarina/gocsv"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// HostParser is used for parsing the configured HAProxy hosts.
var HostParser = parse.URLHostParserBuilder{DefaultScheme: "tcp"}.Build()

// Stat is an instance of the HAProxy stat information
type Stat struct {
	PxName        string `csv:"# pxname"`
	SvName        string `csv:"svname"`
	Qcur          string `csv:"qcur"`
	Qmax          string `csv:"qmax"`
	Scur          string `csv:"scur"`
	Smax          string `csv:"smax"`
	Slim          string `csv:"slim"`
	Stot          string `csv:"stot"`
	Bin           string `csv:"bin"`
	Bout          string `csv:"bout"`
	Dreq          string `csv:"dreq"`
	Dresp         string `csv:"dresp"`
	Ereq          string `csv:"ereq"`
	Econ          string `csv:"econ"`
	Eresp         string `csv:"eresp"`
	Wretr         string `csv:"wretr"`
	Wredis        string `csv:"wredis"`
	Status        string `csv:"status"`
	Weight        string `csv:"weight"`
	Act           string `csv:"act"`
	Bck           string `csv:"bck"`
	ChkFail       string `csv:"chkfail"`
	ChkDown       string `csv:"chkdown"`
	Lastchg       string `csv:"lastchg"`
	Downtime      string `csv:"downtime"`
	Qlimit        string `csv:"qlimit"`
	Pid           string `csv:"pid"`
	Iid           string `csv:"iid"`
	Sid           string `csv:"sid"`
	Throttle      string `csv:"throttle"`
	Lbtot         string `csv:"lbtot"`
	Tracked       string `csv:"tracked"`
	Type          string `csv:"type"`
	Rate          string `csv:"rate"`
	RateLim       string `csv:"rate_lim"`
	RateMax       string `csv:"rate_max"`
	CheckStatus   string `csv:"check_status"`
	CheckCode     string `csv:"check_code"`
	CheckDuration string `csv:"check_duration"`
	Hrsp1xx       string `csv:"hrsp_1xx"`
	Hrsp2xx       string `csv:"hrsp_2xx"`
	Hrsp3xx       string `csv:"hrsp_3xx"`
	Hrsp4xx       string `csv:"hrsp_4xx"`
	Hrsp5xx       string `csv:"hrsp_5xx"`
	HrspOther     string `csv:"hrsp_other"`
	Hanafail      string `csv:"hanafail"`
	ReqRate       string `csv:"req_rate"`
	ReqRateMax    string `csv:"req_rate_max"`
	ReqTot        string `csv:"req_tot"`
	CliAbrt       string `csv:"cli_abrt"`
	SrvAbrt       string `csv:"srv_abrt"`
	CompIn        string `csv:"comp_in"`
	CompOut       string `csv:"comp_out"`
	CompByp       string `csv:"comp_byp"`
	CompRsp       string `csv:"comp_rsp"`
	LastSess      string `csv:"lastsess"`
	LastChk       string `csv:"last_chk"`
	LastAgt       string `csv:"last_agt"`
	Qtime         string `csv:"qtime"`
	Ctime         string `csv:"ctime"`
	Rtime         string `csv:"rtime"`
	Ttime         string `csv:"ttime"`
}

type Info struct {
	Name                       string `mapstructure:"Name"`
	Version                    string `mapstructure:"Version"`
	ReleaseDate                string `mapstructure:"Release_date"`
	Nbproc                     string `mapstructure:"Nbproc"`
	ProcessNum                 string `mapstructure:"Process_num"`
	Pid                        string `mapstructure:"Pid"`
	Uptime                     string `mapstructure:"Uptime"`
	UptimeSec                  string `mapstructure:"Uptime_sec"`
	MemMax                     string `mapstructure:"Memmax_MB"`
	UlimitN                    string `mapstructure:"Ulimit-n"`
	Maxsock                    string `mapstructure:"Maxsock"`
	Maxconn                    string `mapstructure:"Maxconn"`
	HardMaxconn                string `mapstructure:"Hard_maxconn"`
	CurrConns                  string `mapstructure:"CurrConns"`
	CumConns                   string `mapstructure:"CumConns"`
	CumReq                     string `mapstructure:"CumReq"`
	MaxSslConns                string `mapstructure:"MaxSslConns"`
	CurrSslConns               string `mapstructure:"CurrSslConns"`
	CumSslConns                string `mapstructure:"CumSslConns"`
	Maxpipes                   string `mapstructure:"Maxpipes"`
	PipesUsed                  string `mapstructure:"PipesUsed"`
	PipesFree                  string `mapstructure:"PipesFree"`
	ConnRate                   string `mapstructure:"ConnRate"`
	ConnRateLimit              string `mapstructure:"ConnRateLimit"`
	MaxConnRate                string `mapstructure:"MaxConnRate"`
	SessRate                   string `mapstructure:"SessRate"`
	SessRateLimit              string `mapstructure:"SessRateLimit"`
	MaxSessRate                string `mapstructure:"MaxSessRate"`
	SslRate                    string `mapstructure:"SslRate"`
	SslRateLimit               string `mapstructure:"SslRateLimit"`
	MaxSslRate                 string `mapstructure:"MaxSslRate"`
	SslFrontendKeyRate         string `mapstructure:"SslFrontendKeyRate"`
	SslFrontendMaxKeyRate      string `mapstructure:"SslFrontendMaxKeyRate"`
	SslFrontendSessionReusePct string `mapstructure:"SslFrontendSessionReuse_pct"`
	SslBackendKeyRate          string `mapstructure:"SslBackendKeyRate"`
	SslBackendMaxKeyRate       string `mapstructure:"SslBackendMaxKeyRate"`
	SslCacheLookups            string `mapstructure:"SslCacheLookups"`
	SslCacheMisses             string `mapstructure:"SslCacheMisses"`
	CompressBpsIn              string `mapstructure:"CompressBpsIn"`
	CompressBpsOut             string `mapstructure:"CompressBpsOut"`
	CompressBpsRateLim         string `mapstructure:"CompressBpsRateLim"`
	ZlibMemUsage               string `mapstructure:"ZlibMemUsage"`
	MaxZlibMemUsage            string `mapstructure:"MaxZlibMemUsage"`
	Tasks                      string `mapstructure:"Tasks"`
	RunQueue                   string `mapstructure:"Run_queue"`
	IdlePct                    string `mapstructure:"Idle_pct"`
	Node                       string `mapstructure:"Node"`
	Description                string `mapstructure:"Description"`
}

// Client is an instance of the HAProxy client
type clientProto interface {
	Stat() (*bytes.Buffer, error)
	Info() (*bytes.Buffer, error)
}

type Client struct {
	proto clientProto
}

// NewHaproxyClient returns a new instance of HaproxyClient
func NewHaproxyClient(address string) (*Client, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, errors.Wrap(err, "invalid url")
	}

	switch u.Scheme {
	case "tcp":
		return &Client{&unixProto{Network: u.Scheme, Address: u.Host}}, nil
	case "unix":
		return &Client{&unixProto{Network: u.Scheme, Address: u.Path}}, nil
	case "http", "https":
		return &Client{&httpProto{URL: u}}, nil
	default:
		return nil, errors.Errorf("invalid protocol scheme: %s", u.Scheme)
	}
}

// GetStat returns the result from the 'show stat' command
func (c *Client) GetStat() ([]*Stat, error) {
	runResult, err := c.proto.Stat()
	if err != nil {
		return nil, err
	}

	var statRes []*Stat
	csvReader := csv.NewReader(runResult)
	csvReader.TrailingComma = true

	err = gocsv.UnmarshalCSV(csvReader, &statRes)
	if err != nil {
		return nil, errors.Errorf("error parsing CSV: %s", err)
	}

	return statRes, nil
}

// GetInfo returns the result from the 'show stat' command
func (c *Client) GetInfo() (*Info, error) {
	res, err := c.proto.Info()
	if err != nil {
		return nil, err
	}

	if b, err := ioutil.ReadAll(res); err == nil {

		resultMap := map[string]interface{}{}

		for _, ln := range strings.Split(string(b), "\n") {

			ln := strings.TrimSpace(ln)
			if ln == "" {
				continue
			}

			parts := strings.Split(ln, ":")
			if len(parts) != 2 {
				continue
			}

			resultMap[parts[0]] = strings.TrimSpace(parts[1])
		}

		var result *Info

		if err := mapstructure.Decode(resultMap, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, err
}

type unixProto struct {
	Network string
	Address string
}

// Run sends a designated command to the haproxy stats socket
func (p *unixProto) run(cmd string) (*bytes.Buffer, error) {
	var conn net.Conn
	response := bytes.NewBuffer(nil)

	conn, err := net.Dial(p.Network, p.Address)
	if err != nil {
		return response, err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(cmd + "\n"))
	if err != nil {
		return response, err
	}

	_, err = io.Copy(response, conn)
	if err != nil {
		return response, err
	}

	if strings.HasPrefix(response.String(), "Unknown command") {
		return response, errors.Errorf("unknown command: %s", cmd)
	}

	return response, nil
}

func (p *unixProto) Stat() (*bytes.Buffer, error) {
	return p.run("show stat")
}

func (p *unixProto) Info() (*bytes.Buffer, error) {
	return p.run("show info")
}

type httpProto struct {
	URL *url.URL
}

func (p *httpProto) Stat() (*bytes.Buffer, error) {
	url := p.URL.String()
	// Force csv format
	if !strings.HasSuffix(url, ";csv") {
		url += ";csv"
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if p.URL.User != nil {
		password, _ := p.URL.User.Password()
		req.SetBasicAuth(p.URL.User.Username(), password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("couldn't connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("invalid response: %s", resp.Status)
	}

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("couldn't read response body: %v", err)
	}
	return bytes.NewBuffer(d), nil
}

func (p *httpProto) Info() (*bytes.Buffer, error) {
	return nil, errors.New("not supported")
}
