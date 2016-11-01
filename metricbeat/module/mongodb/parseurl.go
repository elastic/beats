package mongodb

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	mgo "gopkg.in/mgo.v2"
)

/*
 * Functions copied from the mgo driver to help with parsing the URL.
 *
 * http://bazaar.launchpad.net/+branch/mgo/v2/view/head:/session.go#L382
 */

type urlInfo struct {
	addrs   []string
	user    string
	pass    string
	db      string
	options map[string]string
}

func parseURL(s string) (*urlInfo, error) {
	if strings.HasPrefix(s, "mongodb://") {
		s = s[10:]
	}
	info := &urlInfo{options: make(map[string]string)}
	if c := strings.Index(s, "?"); c != -1 {
		for _, pair := range strings.FieldsFunc(s[c+1:], isOptSep) {
			l := strings.SplitN(pair, "=", 2)
			if len(l) != 2 || l[0] == "" || l[1] == "" {
				return nil, errors.New("connection option must be key=value: " + pair)
			}
			info.options[l[0]] = l[1]
		}
		s = s[:c]
	}
	if c := strings.Index(s, "@"); c != -1 {
		pair := strings.SplitN(s[:c], ":", 2)
		if len(pair) > 2 || pair[0] == "" {
			return nil, errors.New("credentials must be provided as user:pass@host")
		}
		var err error
		info.user, err = url.QueryUnescape(pair[0])
		if err != nil {
			return nil, fmt.Errorf("cannot unescape username in URL: %q", pair[0])
		}
		if len(pair) > 1 {
			info.pass, err = url.QueryUnescape(pair[1])
			if err != nil {
				return nil, fmt.Errorf("cannot unescape password in URL")
			}
		}
		s = s[c+1:]
	}
	if c := strings.Index(s, "/"); c != -1 {
		info.db = s[c+1:]
		s = s[:c]
	}
	info.addrs = strings.Split(s, ",")
	return info, nil
}

func isOptSep(c rune) bool {
	return c == ';' || c == '&'
}

// ParseURL parses the given URL and returns a DialInfo structure ready
// to be passed to DialWithInfo
func ParseURL(host, username, pass string) (*mgo.DialInfo, error) {
	uinfo, err := parseURL(host)
	if err != nil {
		return nil, err
	}
	direct := false
	mechanism := ""
	service := ""
	source := ""
	for k, v := range uinfo.options {
		switch k {
		case "authSource":
			source = v
		case "authMechanism":
			mechanism = v
		case "gssapiServiceName":
			service = v
		case "connect":
			if v == "direct" {
				direct = true
				break
			}
			if v == "replicaSet" {
				break
			}
			fallthrough
		default:
			return nil, errors.New("unsupported connection URL option: " + k + "=" + v)
		}
	}

	info := &mgo.DialInfo{
		Addrs:     uinfo.addrs,
		Direct:    direct,
		Database:  uinfo.db,
		Username:  uinfo.user,
		Password:  uinfo.pass,
		Mechanism: mechanism,
		Service:   service,
		Source:    source,
	}

	if len(username) > 0 {
		info.Username = username
	}
	if len(pass) > 0 {
		info.Password = pass
	}

	return info, nil
}
