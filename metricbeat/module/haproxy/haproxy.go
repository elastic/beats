package haproxy

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/mitchellh/mapstructure"
	"io"
	"io/ioutil"
	"net"
	"strings"
)

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
	Name                       string
	Version                    string
	ReleaseDate                string
	Nbproc                     string
	ProcessNum                 string
	Pid                        string
	Uptime                     string
	UptimeSec                  string
	MemMaxMB                   string
	UlimitN                    string
	Maxsock                    string
	Maxconn                    string
	HardMaxconn                string
	CurrConns                  string
	CumConns                   string
	CumReq                     string
	MaxSslConns                string
	CurrSslConns               string
	CumSslConns                string
	Maxpipes                   string
	PipesUsed                  string
	PipesFree                  string
	ConnRate                   string
	ConnRateLimit              string
	MaxConnRate                string
	SessRate                   string
	SessRateLimit              string
	MaxSessRate                string
	SslRate                    string
	SslRateLimit               string
	MaxSslRate                 string
	SslFrontendKeyRate         string
	SslFrontendMaxKeyRate      string
	SslFrontendSessionReusePct string
	SslBackendKeyRate          string
	SslBackendMaxKeyRate       string
	SslCacheLookups            string
	SslCacheMisses             string
	CompressBpsIn              string
	CompressBpsOut             string
	CompressBpsRateLim         string
	ZlibMemUsage               string
	MaxZlibMemUsage            string
	Tasks                      string
	RunQueue                   string
	IdlePct                    string
	Node                       string
	Description                string
}

// Client is an instance of the HAProxy client
type Client struct {
	Address     string
	ProtoScheme string
}

// NewHaproxyClient returns a new instance of HaproxyClient
func NewHaproxyClient(address string) (*Client, error) {
	parts := strings.Split(address, "://")
	if len(parts) != 2 {
		return nil, errors.New("Must have protocol scheme and address!")
	}

	if parts[0] != "tcp" && parts[0] != "unix" {
		return nil, errors.New("Invalid Protocol Scheme!")
	}

	return &Client{
		Address:     parts[1],
		ProtoScheme: parts[0],
	}, nil
}

// Run sends a designated command to the haproxy stats socket
func (c *Client) run(cmd string) (*bytes.Buffer, error) {
	var conn net.Conn
	response := bytes.NewBuffer(nil)

	conn, err := net.Dial(c.ProtoScheme, c.Address)
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
		return response, fmt.Errorf("Unknown command: %s", cmd)
	}

	return response, nil
}

// GetStat returns the result from the 'show stat' command
func (c *Client) GetStat() ([]*Stat, error) {

	runResult, err := c.run("show stat")
	if err != nil {
		return nil, err
	}

	var statRes []*Stat
	csvReader := csv.NewReader(runResult)
	csvReader.TrailingComma = true

	err = gocsv.UnmarshalCSV(csvReader, &statRes)
	if err != nil {
		return nil, fmt.Errorf("Error parsing CSV: %s", err)
	}

	return statRes, nil

}

// GetInfo returns the result from the 'show stat' command
func (c *Client) GetInfo() (*Info, error) {

	res, err := c.run("show info")
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
		err := mapstructure.Decode(resultMap, &result)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, err

}
