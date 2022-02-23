// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package client

import (
	"context"
	"encoding/json"
	"fmt"

	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/proto"
)

// Status is the status of the Elastic Agent
type Status = proto.Status

const (
	// Starting is when the it is still starting.
	Starting Status = proto.Status_STARTING
	// Configuring is when it is configuring.
	Configuring Status = proto.Status_CONFIGURING
	// Healthy is when it is healthy.
	Healthy Status = proto.Status_HEALTHY
	// Degraded is when it is degraded.
	Degraded Status = proto.Status_DEGRADED
	// Failed is when it is failed.
	Failed Status = proto.Status_FAILED
	// Stopping is when it is stopping.
	Stopping Status = proto.Status_STOPPING
	// Upgrading is when it is upgrading.
	Upgrading Status = proto.Status_UPGRADING
)

// Version is the current running version of the daemon.
type Version struct {
	Version   string
	Commit    string
	BuildTime time.Time
	Snapshot  bool
}

// ApplicationStatus is a status of an application inside of Elastic Agent.
type ApplicationStatus struct {
	ID      string
	Name    string
	Status  Status
	Message string
	Payload map[string]interface{}
}

// ProcMeta is the running version and ID information for a running process.
type ProcMeta struct {
	Process            string
	Name               string
	Hostname           string
	ID                 string
	EphemeralID        string
	Version            string
	BuildCommit        string
	BuildTime          time.Time
	Username           string
	UserID             string
	UserGID            string
	BinaryArchitecture string
	RouteKey           string
	ElasticLicensed    bool
	Error              string
}

// ProcPProf returns pprof data for a process.
type ProcPProf struct {
	Name     string
	RouteKey string
	Result   []byte
	Error    string
}

// AgentStatus is the current status of the Elastic Agent.
type AgentStatus struct {
	Status       Status
	Message      string
	Applications []*ApplicationStatus
}

// Client communicates to Elastic Agent through the control protocol.
type Client interface {
	// Connect connects to the running Elastic Agent.
	Connect(ctx context.Context) error
	// Disconnect disconnects from the running Elastic Agent.
	Disconnect()
	// Version returns the current version of the running agent.
	Version(ctx context.Context) (Version, error)
	// Status returns the current status of the running agent.
	Status(ctx context.Context) (*AgentStatus, error)
	// Restart triggers restarting the current running daemon.
	Restart(ctx context.Context) error
	// Upgrade triggers upgrade of the current running daemon.
	Upgrade(ctx context.Context, version string, sourceURI string) (string, error)
	// ProcMeta gathers running process meta-data.
	ProcMeta(ctx context.Context) ([]ProcMeta, error)
	// Pprof gathers data from the /debug/pprof/ endpoints specified.
	Pprof(ctx context.Context, d time.Duration, pprofTypes []proto.PprofOption, appName, routeKey string) (map[string][]ProcPProf, error)
}

// client manages the state and communication to the Elastic Agent.
type client struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	client proto.ElasticAgentControlClient
}

// New creates a client connection to Elastic Agent.
func New() Client {
	return &client{}
}

// Connect connects to the running Elastic Agent.
func (c *client) Connect(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	conn, err := dialContext(ctx)
	if err != nil {
		return err
	}
	c.client = proto.NewElasticAgentControlClient(conn)
	return nil
}

// Disconnect disconnects from the running Elastic Agent.
func (c *client) Disconnect() {
	if c.cancel != nil {
		c.cancel()
		c.wg.Wait()
		c.ctx = nil
		c.cancel = nil
	}
}

// Version returns the current version of the running agent.
func (c *client) Version(ctx context.Context) (Version, error) {
	res, err := c.client.Version(ctx, &proto.Empty{})
	if err != nil {
		return Version{}, err
	}
	bt, err := time.Parse(control.TimeFormat(), res.BuildTime)
	if err != nil {
		return Version{}, err
	}
	return Version{
		Version:   res.Version,
		Commit:    res.Commit,
		BuildTime: bt,
		Snapshot:  res.Snapshot,
	}, nil
}

// Status returns the current status of the running agent.
func (c *client) Status(ctx context.Context) (*AgentStatus, error) {
	res, err := c.client.Status(ctx, &proto.Empty{})
	if err != nil {
		return nil, err
	}
	s := &AgentStatus{
		Status:       res.Status,
		Message:      res.Message,
		Applications: make([]*ApplicationStatus, len(res.Applications)),
	}
	for i, appRes := range res.Applications {
		var payload map[string]interface{}
		if appRes.Payload != "" {
			err := json.Unmarshal([]byte(appRes.Payload), &payload)
			if err != nil {
				return nil, err
			}
		}
		s.Applications[i] = &ApplicationStatus{
			ID:      appRes.Id,
			Name:    appRes.Name,
			Status:  appRes.Status,
			Message: appRes.Message,
			Payload: payload,
		}
	}
	return s, nil
}

// Restart triggers restarting the current running daemon.
func (c *client) Restart(ctx context.Context) error {
	res, err := c.client.Restart(ctx, &proto.Empty{})
	if err != nil {
		return err
	}
	if res.Status == proto.ActionStatus_FAILURE {
		return fmt.Errorf(res.Error)
	}
	return nil
}

// Upgrade triggers upgrade of the current running daemon.
func (c *client) Upgrade(ctx context.Context, version string, sourceURI string) (string, error) {
	res, err := c.client.Upgrade(ctx, &proto.UpgradeRequest{
		Version:   version,
		SourceURI: sourceURI,
	})
	if err != nil {
		return "", err
	}
	if res.Status == proto.ActionStatus_FAILURE {
		return "", fmt.Errorf(res.Error)
	}
	return res.Version, nil
}

// ProcMeta gathers running beat metadata.
func (c *client) ProcMeta(ctx context.Context) ([]ProcMeta, error) {
	resp, err := c.client.ProcMeta(ctx, &proto.Empty{})
	if err != nil {
		return nil, err
	}
	procMeta := []ProcMeta{}

	for _, proc := range resp.Procs {
		meta := ProcMeta{
			Process:            proc.Process,
			Name:               proc.Name,
			Hostname:           proc.Hostname,
			ID:                 proc.Id,
			EphemeralID:        proc.EphemeralId,
			Version:            proc.Version,
			BuildCommit:        proc.BuildCommit,
			Username:           proc.Username,
			UserID:             proc.UserId,
			UserGID:            proc.UserGid,
			BinaryArchitecture: proc.Architecture,
			RouteKey:           proc.RouteKey,
			ElasticLicensed:    proc.ElasticLicensed,
			Error:              proc.Error,
		}
		if proc.BuildTime != "" {
			ts, err := time.Parse(time.RFC3339, proc.BuildTime)
			if err != nil {
				if meta.Error != "" {
					meta.Error += ", " + err.Error()
				} else {
					meta.Error = err.Error()
				}
			} else {
				meta.BuildTime = ts
			}
		}
		procMeta = append(procMeta, meta)
	}
	return procMeta, nil
}

// Pprof gathers /debug/pprof data and returns a map of pprof-type: ProcPProf data
func (c *client) Pprof(ctx context.Context, d time.Duration, pprofTypes []proto.PprofOption, appName, routeKey string) (map[string][]ProcPProf, error) {
	resp, err := c.client.Pprof(ctx, &proto.PprofRequest{
		PprofType:     pprofTypes,
		TraceDuration: d.String(),
		AppName:       appName,
		RouteKey:      routeKey,
	})
	if err != nil {
		return nil, err
	}
	res := map[string][]ProcPProf{}
	for _, pType := range pprofTypes {
		res[pType.String()] = make([]ProcPProf, 0)
	}
	for _, r := range resp.Results {
		res[r.PprofType.String()] = append(res[r.PprofType.String()], ProcPProf{
			Name:     r.AppName,
			RouteKey: r.RouteKey,
			Result:   r.Result,
			Error:    r.Error,
		})
	}
	return res, nil
}

// ProcMetrics gathers /buffer data and from the agent and each running process and returns the result.
func (c *client) ProcMetrics(ctx context.Context) (*proto.ProcMetricsResponse, error) {
	return c.client.ProcMetrics(ctx, &proto.Empty{})
}
