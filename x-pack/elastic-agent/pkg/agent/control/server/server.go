// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/reexec"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/proto"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	monitoring "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/beats"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/socket"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"
)

// Server is the daemon side of the control protocol.
type Server struct {
	proto.UnimplementedElasticAgentControlServer

	logger     *logger.Logger
	rex        reexec.ExecManager
	statusCtrl status.Controller
	up         *upgrade.Upgrader
	routeFn    func() *sorted.Set
	listener   net.Listener
	server     *grpc.Server
	lock       sync.RWMutex
}

type specer interface {
	Specs() map[string]program.Spec
}

// New creates a new control protocol server.
func New(log *logger.Logger, rex reexec.ExecManager, statusCtrl status.Controller, up *upgrade.Upgrader) *Server {
	return &Server{
		logger:     log,
		rex:        rex,
		statusCtrl: statusCtrl,
		up:         up,
	}
}

// SetUpgrader changes the upgrader.
func (s *Server) SetUpgrader(up *upgrade.Upgrader) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.up = up
}

// SetRouteFn changes the route retrieval function.
func (s *Server) SetRouteFn(routesFetchFn func() *sorted.Set) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.routeFn = routesFetchFn
}

// Start starts the GRPC endpoint and accepts new connections.
func (s *Server) Start() error {
	if s.server != nil {
		// already started
		return nil
	}

	lis, err := createListener(s.logger)
	if err != nil {
		s.logger.Errorf("unable to create listener: %s", err)
		return err
	}
	s.listener = lis
	s.server = grpc.NewServer()
	proto.RegisterElasticAgentControlServer(s.server, s)

	// start serving GRPC connections
	go func() {
		err := s.server.Serve(lis)
		if err != nil {
			s.logger.Errorf("error listening for GRPC: %s", err)
		}
	}()

	return nil
}

// Stop stops the GRPC endpoint.
func (s *Server) Stop() {
	if s.server != nil {
		s.server.Stop()
		s.server = nil
		s.listener = nil
		cleanupListener(s.logger)
	}
}

// Version returns the currently running version.
func (s *Server) Version(_ context.Context, _ *proto.Empty) (*proto.VersionResponse, error) {
	return &proto.VersionResponse{
		Version:   release.Version(),
		Commit:    release.Commit(),
		BuildTime: release.BuildTime().Format(control.TimeFormat()),
		Snapshot:  release.Snapshot(),
	}, nil
}

// Status returns the overall status of the agent.
func (s *Server) Status(_ context.Context, _ *proto.Empty) (*proto.StatusResponse, error) {
	status := s.statusCtrl.Status()
	return &proto.StatusResponse{
		Status:       agentStatusToProto(status.Status),
		Message:      status.Message,
		Applications: agentAppStatusToProto(status.Applications),
	}, nil
}

// Restart performs re-exec.
func (s *Server) Restart(_ context.Context, _ *proto.Empty) (*proto.RestartResponse, error) {
	s.rex.ReExec(nil)
	return &proto.RestartResponse{
		Status: proto.ActionStatus_SUCCESS,
	}, nil
}

// Upgrade performs the upgrade operation.
func (s *Server) Upgrade(ctx context.Context, request *proto.UpgradeRequest) (*proto.UpgradeResponse, error) {
	s.lock.RLock()
	u := s.up
	s.lock.RUnlock()
	if u == nil {
		// not running with upgrader (must be controlled by Fleet)
		return &proto.UpgradeResponse{
			Status: proto.ActionStatus_FAILURE,
			Error:  "cannot be upgraded; perform upgrading using Fleet",
		}, nil
	}
	cb, err := u.Upgrade(ctx, &upgradeRequest{request}, false, request.SkipVerify, request.PgpBytes...)
	if err != nil {
		return &proto.UpgradeResponse{
			Status: proto.ActionStatus_FAILURE,
			Error:  err.Error(),
		}, nil
	}
	// perform the re-exec after a 1 second delay
	// this ensures that the upgrade response over GRPC is returned
	go func() {
		<-time.After(time.Second)
		s.rex.ReExec(cb)
	}()
	return &proto.UpgradeResponse{
		Status:  proto.ActionStatus_SUCCESS,
		Version: request.Version,
	}, nil
}

// BeatInfo is the metadata response a beat will provide when the root ("/") is queried.
type BeatInfo struct {
	Beat            string `json:"beat"`
	Name            string `json:"name"`
	Hostname        string `json:"hostname"`
	ID              string `json:"uuid"`
	EphemeralID     string `json:"ephemeral_id"`
	Version         string `json:"version"`
	Commit          string `json:"build_commit"`
	Time            string `json:"build_time"`
	Username        string `json:"username"`
	UserID          string `json:"uid"`
	GroupID         string `json:"gid"`
	BinaryArch      string `json:"binary_arch"`
	ElasticLicensed bool   `json:"elastic_licensed"`
}

// ProcMeta returns version and beat inforation for all running processes.
func (s *Server) ProcMeta(ctx context.Context, _ *proto.Empty) (*proto.ProcMetaResponse, error) {
	if s.routeFn == nil {
		return nil, errors.New("route function is nil")
	}

	resp := &proto.ProcMetaResponse{
		Procs: []*proto.ProcMeta{},
	}

	routes := s.routeFn()
	for _, rk := range routes.Keys() {
		programs, ok := routes.Get(rk)
		if !ok {
			s.logger.With("route_key", rk).Warn("Unable to retrieve route.")
			continue
		}

		sp, ok := programs.(specer)
		if !ok {
			s.logger.With("route_key", rk, "route", programs).Warn("Unable to cast route as specer.")
			continue
		}
		specs := sp.Specs()

		for n, spec := range specs {
			procMeta := &proto.ProcMeta{
				Name:     n,
				RouteKey: rk,
			}

			client := http.Client{
				Timeout: time.Second * 5,
			}
			endpoint := monitoring.MonitoringEndpoint(spec, runtime.GOOS, rk)
			if strings.HasPrefix(endpoint, "unix://") {
				client.Transport = &http.Transport{
					Proxy:       nil,
					DialContext: socket.DialContext(strings.TrimPrefix(endpoint, "unix://")),
				}
				endpoint = "unix"
			} else if strings.HasPrefix(endpoint, "npipe://") {
				client.Transport = &http.Transport{
					Proxy:       nil,
					DialContext: socket.DialContext(strings.TrimPrefix(endpoint, "npipe:///")),
				}
				endpoint = "npipe"
			}

			res, err := client.Get("http://" + endpoint + "/")
			if err != nil {
				procMeta.Error = err.Error()
				resp.Procs = append(resp.Procs, procMeta)
				continue
			}
			if res.StatusCode != 200 {
				procMeta.Error = "response status is: " + res.Status
				resp.Procs = append(resp.Procs, procMeta)
				continue
			}

			bi := &BeatInfo{}
			dec := json.NewDecoder(res.Body)
			if err := dec.Decode(bi); err != nil {
				res.Body.Close()
				procMeta.Error = err.Error()
				resp.Procs = append(resp.Procs, procMeta)
				continue
			}
			res.Body.Close()

			procMeta.Process = bi.Beat
			procMeta.Hostname = bi.Hostname
			procMeta.Id = bi.ID
			procMeta.EphemeralId = bi.EphemeralID
			procMeta.Version = bi.Version
			procMeta.BuildCommit = bi.Commit
			procMeta.BuildTime = bi.Time
			procMeta.Username = bi.Username
			procMeta.UserId = bi.UserID
			procMeta.UserGid = bi.GroupID
			procMeta.Architecture = bi.BinaryArch
			procMeta.ElasticLicensed = bi.ElasticLicensed

			resp.Procs = append(resp.Procs, procMeta)
		}
	}
	return resp, nil
}

type upgradeRequest struct {
	*proto.UpgradeRequest
}

func (r *upgradeRequest) Version() string {
	return r.GetVersion()
}

func (r *upgradeRequest) SourceURI() string {
	return r.GetSourceURI()
}

func (r *upgradeRequest) FleetAction() *fleetapi.ActionUpgrade {
	// upgrade request not from Fleet
	return nil
}

func agentStatusToProto(code status.AgentStatusCode) proto.Status {
	if code == status.Degraded {
		return proto.Status_DEGRADED
	}
	if code == status.Failed {
		return proto.Status_FAILED
	}
	return proto.Status_HEALTHY
}

func agentAppStatusToProto(apps []status.AgentApplicationStatus) []*proto.ApplicationStatus {
	s := make([]*proto.ApplicationStatus, len(apps))
	for i, a := range apps {
		var payload []byte
		if a.Payload != nil {
			payload, _ = json.Marshal(a.Payload)
		}
		s[i] = &proto.ApplicationStatus{
			Id:      a.ID,
			Name:    a.Name,
			Status:  proto.Status(a.Status.ToProto()),
			Message: a.Message,
			Payload: string(payload),
		}
	}
	return s
}
