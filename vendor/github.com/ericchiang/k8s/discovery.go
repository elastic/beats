package k8s

import (
	"context"
	"path"

	"github.com/ericchiang/k8s/api/unversioned"
)

type Version struct {
	Major        string `json:"major"`
	Minor        string `json:"minor"`
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

func (c *Client) Discovery() *Discovery {
	return &Discovery{c}
}

// Discovery is a client used to determine the API version and supported
// resources of the server.
type Discovery struct {
	client *Client
}

func (d *Discovery) Version(ctx context.Context) (*Version, error) {
	var v Version
	if err := d.client.get(ctx, jsonCodec, d.client.urlForPath("version"), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (d *Discovery) APIGroups(ctx context.Context) (*unversioned.APIGroupList, error) {
	var groups unversioned.APIGroupList
	if err := d.client.get(ctx, pbCodec, d.client.urlForPath("apis"), &groups); err != nil {
		return nil, err
	}
	return &groups, nil
}

func (d *Discovery) APIGroup(ctx context.Context, name string) (*unversioned.APIGroup, error) {
	var group unversioned.APIGroup
	if err := d.client.get(ctx, pbCodec, d.client.urlForPath(path.Join("apis", name)), &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (d *Discovery) APIResources(ctx context.Context, groupName, groupVersion string) (*unversioned.APIResourceList, error) {
	var list unversioned.APIResourceList
	if err := d.client.get(ctx, pbCodec, d.client.urlForPath(path.Join("apis", groupName, groupVersion)), &list); err != nil {
		return nil, err
	}
	return &list, nil

}
