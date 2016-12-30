package mongodb

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	mgo "gopkg.in/mgo.v2"
)

func init() {
	// Register the ModuleFactory function for the "mongodb" module.
	if err := mb.Registry.AddModule("mongodb", NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new mb.Module instance and validates that at least one host has been
// specified
func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}

// ParseURL parses valid MongoDB URL strings into an mb.HostData instance
func ParseURL(module mb.Module, host string) (mb.HostData, error) {
	c := struct {
		Username string `config:"username"`
		Password string `config:"password"`
	}{}
	if err := module.UnpackConfig(&c); err != nil {
		return mb.HostData{}, err
	}

	if parts := strings.SplitN(host, "://", 2); len(parts) != 2 {
		// Add scheme.
		host = fmt.Sprintf("mongodb://%s", host)
	}

	// This doesn't use URLHostParserBuilder because MongoDB URLs can contain
	// multiple hosts separated by commas (mongodb://host1,host2,host3?options).
	u, err := url.Parse(host)
	if err != nil {
		return mb.HostData{}, fmt.Errorf("error parsing URL: %v", err)
	}

	parse.SetURLUser(u, c.Username, c.Password)

	// https://docs.mongodb.com/manual/reference/connection-string/
	_, err = mgo.ParseURL(u.String())
	if err != nil {
		return mb.HostData{}, err
	}

	return parse.NewHostDataFromURL(u), nil
}

// NewSession returns a connection to MongoDB (*mgo.Session) by dialing the mongo
// instance specified in settings. If a connection cannot be established, a Critical error is
// thrown and the program exits
func NewSession(dialInfo *mgo.DialInfo) *mgo.Session {
	mongo, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		logp.Critical("Failed to establish connection to MongDB at %s", dialInfo.Addrs)
	}
	return mongo
}

// NewDirectSessions estbalishes direct connections with a list of hosts. It uses the supplied
// dialInfo parameter as a template for establishing more direct connections
func NewDirectSessions(urls []string, dialInfo *mgo.DialInfo) ([]*mgo.Session, error) {

	var nodes []*mgo.Session

	logp.Info("%d MongoDB nodes configured for monitoring", len(urls))
	fmt.Printf("%d MongoDB nodes configured for monitoring", len(urls))

	for _, url := range urls {

		// make a copy
		nodeDialInfo := *dialInfo
		nodeDialInfo.Addrs = []string{
			url,
		}
		nodeDialInfo.Direct = true
		nodeDialInfo.FailFast = true

		logp.Info("Connecting to MongoDB node at %s", url)
		fmt.Printf("Connecting to MongoDB node at %s", url)
		session, err := mgo.DialWithInfo(&nodeDialInfo)
		if err != nil {
			logp.Err("Error establishing direct connection to mongo node at %s. Error output: %s", url, err.Error())
			// set i back a value so we don't skip an index when adding successful connections
			continue
		}
		nodes = append(nodes, session)
	}

	if len(nodes) == 0 {
		msg := "Error establishing connection to any mongo nodes"
		logp.Err(msg)
		return []*mgo.Session{}, errors.New(msg)
	}

	return nodes, nil
}
