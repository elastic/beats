package mongodb

import (
	"fmt"
	"net/url"
	"strings"

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
