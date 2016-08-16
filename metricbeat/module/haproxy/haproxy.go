package haproxy

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/gocarina/gocsv"
	"io"
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

/*
type Stat struct {
	PxName        string `csv:"# pxname"`
	SvName        string `csv:"svname"`
	Qcur          uint64 `csv:"qcur"`
	Qmax          uint64 `csv:"qmax"`
	Scur          uint64 `csv:"scur"`
	Smax          uint64 `csv:"smax"`
	Slim          uint64 `csv:"slim"`
	Stot          uint64 `csv:"stot"`
	Bin           uint64 `csv:"bin"`
	Bout          uint64 `csv:"bout"`
	Dreq          uint64 `csv:"dreq"`
	Dresp         uint64 `csv:"dresp"`
	Ereq          uint64 `csv:"ereq"`
	Econ          uint64 `csv:"econ"`
	Eresp         uint64 `csv:"eresp"`
	Wretr         uint64 `csv:"wretr"`
	Wredis        uint64 `csv:"wredis"`
	Status        string `csv:"status"`
	Weight        uint64 `csv:"weight"`
	Act           uint64 `csv:"act"`
	Bck           uint64 `csv:"bck"`
	ChkFail       uint64 `csv:"chkfail"`
	ChkDown       uint64 `csv:"chkdown"`
	Lastchg       uint64 `csv:"lastchg"`
	Downtime      uint64 `csv:"downtime"`
	Qlimit        uint64 `csv:"qlimit"`
	Pid           uint64 `csv:"pid"`
	Iid           uint64 `csv:"iid"`
	Sid           uint64 `csv:"sid"`
	Throttle      uint64 `csv:"throttle"`
	Lbtot         uint64 `csv:"lbtot"`
	Tracked       uint64 `csv:"tracked"`
	Type          uint64 `csv:"type"`
	Rate          uint64 `csv:"rate"`
	RateLim       uint64 `csv:"rate_lim"`
	RateMax       uint64 `csv:"rate_max"`
	CheckStatus   string `csv:"check_status"`
	CheckCode     uint64 `csv:"check_code"`
	CheckDuration uint64 `csv:"check_duration"`
	Hrsp1xx       uint64 `csv:"hrsp_1xx"`
	Hrsp2xx       uint64 `csv:"hrsp_2xx"`
	Hrsp3xx       uint64 `csv:"hrsp_3xx"`
	Hrsp4xx       uint64 `csv:"hrsp_4xx"`
	Hrsp5xx       uint64 `csv:"hrsp_5xx"`
	HrspOther     uint64 `csv:"hrsp_other"`
	Hanafail      uint64 `csv:"hanafail"`
	ReqRate       uint64 `csv:"req_rate"`
	ReqRateMax    uint64 `csv:"req_rate_max"`
	ReqTot        uint64 `csv:"req_tot"`
	CliAbrt       uint64 `csv:"cli_abrt"`
	SrvAbrt       uint64 `csv:"srv_abrt"`
	CompIn        uint64 `csv:"comp_in"`
	CompOut       uint64 `csv:"comp_out"`
	CompByp       uint64 `csv:"comp_byp"`
	CompRsp       uint64 `csv:"comp_rsp"`
	LastSess      int64  `csv:"lastsess"`
	LastChk       string `csv:"last_chk"`
	LastAgt       uint64 `csv:"last_agt"`
	Qtime         uint64 `csv:"qtime"`
	Ctime         uint64 `csv:"ctime"`
	Rtime         uint64 `csv:"rtime"`
	Ttime         uint64 `csv:"ttime"`
}
*/

// Client is an instance of the HAProxy client
type Client struct {
	connection  net.Conn
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
	c.connection = conn

	defer c.connection.Close()

	_, err = c.connection.Write([]byte(cmd + "\n"))
	if err != nil {
		return response, err
	}

	_, err = io.Copy(response, c.connection)
	if err != nil {
		return response, err
	}

	if strings.HasPrefix(response.String(), "Unknown command") {
		return response, fmt.Errorf("Unknown command: %s", cmd)
	}

	return response, nil
}

// ShowStat returns the result from the 'show stat' command
func (c *Client) GetStat() (statRes []*Stat, err error) {

	runResult, err := c.run("show stat")
	if err != nil {
		return nil, err
	}

	csvReader := csv.NewReader(runResult)
	csvReader.TrailingComma = true

	err = gocsv.UnmarshalCSV(csvReader, &statRes)
	if err != nil {
		return nil, fmt.Errorf("Error parsing CSV: %s", err)
	}

	return statRes, nil

}
