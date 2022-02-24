// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/beats"
	monitoring "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/beats"
	monitoringCfg "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/socket"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"
)

// Server is the daemon side of the control protocol.
type Server struct {
	logger        *logger.Logger
	rex           reexec.ExecManager
	statusCtrl    status.Controller
	up            *upgrade.Upgrader
	routeFn       func() *sorted.Set
	monitoringCfg *monitoringCfg.MonitoringConfig
	listener      net.Listener
	server        *grpc.Server
	lock          sync.RWMutex
}

type specer interface {
	Specs() map[string]program.Spec
}

type specInfo struct {
	spec program.Spec
	app  string
	rk   string
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

// SetMonitoringCfg sets a reference to the monitoring config used by the running agent.
// the controller references this config to find out if pprof is enabled for the agent or not
func (s *Server) SetMonitoringCfg(cfg *monitoringCfg.MonitoringConfig) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.monitoringCfg = cfg
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
	cb, err := u.Upgrade(ctx, &upgradeRequest{request}, false)
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

	// gather spec data for all rk/apps running
	specs := s.getSpecInfo("", "")
	for _, si := range specs {
		endpoint := monitoring.MonitoringEndpoint(si.spec, runtime.GOOS, si.rk)
		client := newSocketRequester(si.app, si.rk, endpoint)

		procMeta := client.procMeta(ctx)
		resp.Procs = append(resp.Procs, procMeta)
	}

	return resp, nil
}

// Pprof returns /debug/pprof data for the requested applicaiont-route_key or all running applications.
func (s *Server) Pprof(ctx context.Context, req *proto.PprofRequest) (*proto.PprofResponse, error) {
	if s.monitoringCfg == nil || s.monitoringCfg.Pprof == nil || !s.monitoringCfg.Pprof.Enabled {
		return nil, fmt.Errorf("agent.monitoring.pprof disabled")
	}

	if s.routeFn == nil {
		return nil, errors.New("route function is nil")
	}

	dur, err := time.ParseDuration(req.TraceDuration)
	if err != nil {
		return nil, fmt.Errorf("unable to parse trace duration: %w", err)
	}

	resp := &proto.PprofResponse{
		Results: []*proto.PprofResult{},
	}

	var wg sync.WaitGroup
	ch := make(chan *proto.PprofResult, 1)

	// retrieve elastic-agent pprof data if requested or application is unspecified.
	if req.AppName == "" || req.AppName == "elastic-agent" {
		endpoint := beats.AgentMonitoringEndpoint(runtime.GOOS, s.monitoringCfg.HTTP)
		c := newSocketRequester("elastic-agent", "", endpoint)
		for _, opt := range req.PprofType {
			wg.Add(1)
			go func(opt proto.PprofOption) {
				res := c.getPprof(ctx, opt, dur)
				ch <- res
				wg.Done()
			}(opt)
		}
	}

	// get requested rk/appname spec or all specs
	var specs []specInfo
	if req.AppName != "elastic-agent" {
		specs = s.getSpecInfo(req.RouteKey, req.AppName)
	}
	for _, si := range specs {
		endpoint := monitoring.MonitoringEndpoint(si.spec, runtime.GOOS, si.rk)
		c := newSocketRequester(si.app, si.rk, endpoint)
		// Launch a concurrent goroutine to gather all pprof endpoints from a socket.
		for _, opt := range req.PprofType {
			wg.Add(1)
			go func(opt proto.PprofOption) {
				res := c.getPprof(ctx, opt, dur)
				ch <- res
				wg.Done()
			}(opt)
		}
	}

	// wait for the waitgroup to be done and close the channel
	go func() {
		wg.Wait()
		close(ch)
	}()

	// gather all results from channel until closed.
	for res := range ch {
		resp.Results = append(resp.Results, res)
	}
	return resp, nil
}

// ProcMetrics returns all buffered metrics data for the agent and running processes.
// If the agent.monitoring.http.buffer variable is not set, or set to false, a nil is returned
func (s *Server) ProcMetrics(ctx context.Context, _ *proto.Empty) (*proto.ProcMetricsResponse, error) {
	if s.monitoringCfg == nil || s.monitoringCfg.HTTP == nil || s.monitoringCfg.HTTP.Buffer == nil || !s.monitoringCfg.HTTP.Buffer.Enabled {
		return nil, nil
	}

	if s.routeFn == nil {
		return nil, errors.New("route function is nil")
	}

	// gather metrics buffer data from the elastic-agent
	endpoint := beats.AgentMonitoringEndpoint(runtime.GOOS, s.monitoringCfg.HTTP)
	c := newSocketRequester("elastic-agent", "", endpoint)
	metrics := c.procMetrics(ctx)

	resp := &proto.ProcMetricsResponse{
		Result: []*proto.MetricsResponse{metrics},
	}

	// gather metrics buffer data from all other processes
	specs := s.getSpecInfo("", "")
	for _, si := range specs {
		endpoint := monitoring.MonitoringEndpoint(si.spec, runtime.GOOS, si.rk)
		client := newSocketRequester(si.app, si.rk, endpoint)

		metrics := client.procMetrics(ctx)
		resp.Result = append(resp.Result, metrics)
	}
	return resp, nil
}

// getSpecs will return the specs for the program associated with the specified route key/app name, or all programs if no key(s) are specified.
// if matchRK or matchApp are empty all results will be returned.
func (s *Server) getSpecInfo(matchRK, matchApp string) []specInfo {
	routes := s.routeFn()

	// find specInfo for a specified rk/app
	if matchRK != "" && matchApp != "" {
		programs, ok := routes.Get(matchRK)
		if !ok {
			s.logger.With("route_key", matchRK).Debug("No matching route key found.")
			return []specInfo{}
		}
		sp, ok := programs.(specer)
		if !ok {
			s.logger.With("route_key", matchRK, "route", programs).Warn("Unable to cast route as specer.")
			return []specInfo{}
		}
		specs := sp.Specs()

		spec, ok := specs[matchApp]
		if !ok {
			s.logger.With("route_key", matchRK, "application_name", matchApp).Debug("No matching route key/application name found.")
			return []specInfo{}
		}
		return []specInfo{specInfo{spec: spec, app: matchApp, rk: matchRK}}
	}

	// gather specInfo for all rk/app values
	res := make([]specInfo, 0)
	for _, rk := range routes.Keys() {
		programs, ok := routes.Get(rk)
		if !ok {
			// we do not expect to ever hit this code path
			// if this log message occurs then the agent is unable to access one of the keys that is returned by the route function
			// might be a race condition if someone tries to update the policy to remove an output?
			s.logger.With("route_key", rk).Warn("Unable to retrieve route.")
			continue
		}
		sp, ok := programs.(specer)
		if !ok {
			s.logger.With("route_key", matchRK, "route", programs).Warn("Unable to cast route as specer.")
			continue
		}
		for n, spec := range sp.Specs() {
			res = append(res, specInfo{
				rk:   rk,
				app:  n,
				spec: spec,
			})
		}
	}
	return res
}

// socketRequester is a struct to gather (diagnostics) data from a socket opened by elastic-agent or one if it's processes
type socketRequester struct {
	c        http.Client
	endpoint string
	appName  string
	routeKey string
}

func newSocketRequester(appName, routeKey, endpoint string) *socketRequester {
	c := http.Client{}
	if strings.HasPrefix(endpoint, "unix://") {
		c.Transport = &http.Transport{
			Proxy:       nil,
			DialContext: socket.DialContext(strings.TrimPrefix(endpoint, "unix://")),
		}
		endpoint = "unix"
	} else if strings.HasPrefix(endpoint, "npipe://") {
		c.Transport = &http.Transport{
			Proxy:       nil,
			DialContext: socket.DialContext(strings.TrimPrefix(endpoint, "npipe:///")),
		}
		endpoint = "npipe"
	}
	return &socketRequester{
		c:        c,
		appName:  appName,
		routeKey: routeKey,
		endpoint: endpoint,
	}
}

// getPath creates a get request for the specified path.
// Will return an error if that status code is not 200.
func (r *socketRequester) getPath(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", "http://"+r.endpoint+path, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	res, err := r.c.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return nil, fmt.Errorf("response status is %d", res.StatusCode)
	}
	return res, nil

}

// procMeta will return process metadata by querying the "/" path.
func (r *socketRequester) procMeta(ctx context.Context) *proto.ProcMeta {
	pm := &proto.ProcMeta{
		Name:     r.appName,
		RouteKey: r.routeKey,
	}

	res, err := r.getPath(ctx, "/")
	if err != nil {
		pm.Error = err.Error()
		return pm
	}
	defer res.Body.Close()

	bi := &BeatInfo{}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(bi); err != nil {
		pm.Error = err.Error()
		return pm
	}

	pm.Process = bi.Beat
	pm.Hostname = bi.Hostname
	pm.Id = bi.ID
	pm.EphemeralId = bi.EphemeralID
	pm.Version = bi.Version
	pm.BuildCommit = bi.Commit
	pm.BuildTime = bi.Time
	pm.Username = bi.Username
	pm.UserId = bi.UserID
	pm.UserGid = bi.GroupID
	pm.Architecture = bi.BinaryArch
	pm.ElasticLicensed = bi.ElasticLicensed

	return pm
}

var pprofEndpoints = map[proto.PprofOption]string{
	proto.PprofOption_ALLOCS:       "/debug/pprof/allocs",
	proto.PprofOption_BLOCK:        "/debug/pprof/block",
	proto.PprofOption_CMDLINE:      "/debug/pprof/cmdline",
	proto.PprofOption_GOROUTINE:    "/debug/pprof/goroutine",
	proto.PprofOption_HEAP:         "/debug/pprof/heap",
	proto.PprofOption_MUTEX:        "/debug/pprof/mutex",
	proto.PprofOption_PROFILE:      "/debug/pprof/profile",
	proto.PprofOption_THREADCREATE: "/debug/pprof/threadcreate",
	proto.PprofOption_TRACE:        "/debug/pprof/trace",
}

// getProf will gather pprof data specified by the option.
func (r *socketRequester) getPprof(ctx context.Context, opt proto.PprofOption, dur time.Duration) *proto.PprofResult {
	res := &proto.PprofResult{
		AppName:   r.appName,
		RouteKey:  r.routeKey,
		PprofType: opt,
	}

	path, ok := pprofEndpoints[opt]
	if !ok {
		res.Error = "unknown path for option"
		return res
	}

	if opt == proto.PprofOption_PROFILE || opt == proto.PprofOption_TRACE {
		path += fmt.Sprintf("?seconds=%0.f", dur.Seconds())
	}

	resp, err := r.getPath(ctx, path)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()

	p, err := io.ReadAll(resp.Body)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Result = p
	return res
}

// procMetrics will gather metrics buffer data
func (r *socketRequester) procMetrics(ctx context.Context) *proto.MetricsResponse {
	res := &proto.MetricsResponse{
		AppName:  r.appName,
		RouteKey: r.routeKey,
	}

	resp, err := r.getPath(ctx, "/")
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()

	p, err := io.ReadAll(resp.Body)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	res.Result = p
	return res
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
