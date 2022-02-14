// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gofrs/uuid"
	protobuf "github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/authority"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	// InitialCheckinTimeout is the maximum amount of wait time from initial check-in stream to
	// getting the first check-in observed state.
	InitialCheckinTimeout = 5 * time.Second
	// CheckinMinimumTimeoutGracePeriod is additional time added to the client.CheckinMinimumTimeout
	// to ensure the application is checking in correctly.
	CheckinMinimumTimeoutGracePeriod = 30 * time.Second
	// WatchdogCheckLoop is the amount of time that the watchdog will wait between checking for
	// applications that have not checked in the correct amount of time.
	WatchdogCheckLoop = 5 * time.Second
)

var (
	// ErrApplicationAlreadyRegistered returned when trying to register an application more than once.
	ErrApplicationAlreadyRegistered = errors.New("application already registered", errors.TypeApplication)
	// ErrApplicationStopping returned when trying to update an application config but it is stopping.
	ErrApplicationStopping = errors.New("application stopping", errors.TypeApplication)
	// ErrApplicationStopTimedOut returned when calling Stop and the application timed out stopping.
	ErrApplicationStopTimedOut = errors.New("application stopping timed out", errors.TypeApplication)
	// ErrActionTimedOut returned on PerformAction when the action timed out.
	ErrActionTimedOut = errors.New("application action timed out", errors.TypeApplication)
	// ErrActionCancelled returned on PerformAction when an action is cancelled, normally due to the application
	// being stopped or removed from the server.
	ErrActionCancelled = errors.New("application action cancelled", errors.TypeApplication)
)

// ApplicationState represents the applications state according to the server.
type ApplicationState struct {
	srv *Server
	app interface{}

	srvName string
	token   string
	cert    *authority.Pair

	pendingExpected   chan *proto.StateExpected
	expected          proto.StateExpected_State
	expectedConfigIdx uint64
	expectedConfig    string
	status            proto.StateObserved_Status
	statusMessage     string
	statusPayload     map[string]interface{}
	statusPayloadStr  string
	statusConfigIdx   uint64
	statusTime        time.Time
	checkinConn       bool
	checkinDone       chan bool
	checkinLock       sync.RWMutex

	pendingActions chan *pendingAction
	sentActions    map[string]*sentAction
	actionsConn    bool
	actionsDone    chan bool
	actionsLock    sync.RWMutex

	inputTypes map[string]struct{}
}

// Handler is the used by the server to inform of status changes.
type Handler interface {
	// OnStatusChange called when a registered application observed status is changed.
	OnStatusChange(*ApplicationState, proto.StateObserved_Status, string, map[string]interface{})
}

// Server is the GRPC server that the launched applications connect back to.
type Server struct {
	logger     *logger.Logger
	ca         *authority.CertificateAuthority
	listenAddr string
	handler    Handler
	tracer     *apm.Tracer

	listener     net.Listener
	server       *grpc.Server
	watchdogDone chan bool
	watchdogWG   sync.WaitGroup

	apps sync.Map

	// overridden in tests
	watchdogCheckInterval time.Duration
	checkInMinTimeout     time.Duration
}

// New creates a new GRPC server for clients to connect to.
func New(logger *logger.Logger, listenAddr string, handler Handler, tracer *apm.Tracer) (*Server, error) {
	ca, err := authority.NewCA()
	if err != nil {
		return nil, err
	}
	return &Server{
		logger:                logger,
		ca:                    ca,
		listenAddr:            listenAddr,
		handler:               handler,
		watchdogCheckInterval: WatchdogCheckLoop,
		checkInMinTimeout:     client.CheckinMinimumTimeout + CheckinMinimumTimeoutGracePeriod,
		tracer:                tracer,
	}, nil
}

// Start starts the GRPC endpoint and accepts new connections.
func (s *Server) Start() error {
	if s.server != nil {
		// already started
		return nil
	}

	lis, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	s.listener = lis
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(s.ca.Crt()); !ok {
		return errors.New("failed to append root CA", errors.TypeSecurity)
	}
	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:     tls.RequireAndVerifyClientCert,
		ClientCAs:      certPool,
		GetCertificate: s.getCertificate,
	})
	if s.tracer != nil {
		apmInterceptor := apmgrpc.NewUnaryServerInterceptor(apmgrpc.WithRecovery(), apmgrpc.WithTracer(s.tracer))
		s.server = grpc.NewServer(
			grpc.UnaryInterceptor(apmInterceptor),
			grpc.Creds(creds),
		)
	} else {
		s.server = grpc.NewServer(grpc.Creds(creds))
	}
	proto.RegisterElasticAgentServer(s.server, s)

	// start serving GRPC connections
	go func() {
		err := s.server.Serve(lis)
		if err != nil {
			s.logger.Errorf("error listening for GRPC: %s", err)
		}
	}()

	// start the watchdog
	s.watchdogDone = make(chan bool)
	s.watchdogWG.Add(1)
	go s.watchdog()

	return nil
}

// Stop stops the GRPC endpoint.
func (s *Server) Stop() {
	if s.server != nil {
		close(s.watchdogDone)
		s.server.Stop()
		s.server = nil
		s.listener = nil
		s.watchdogWG.Wait()
	}
}

// Get returns the application state from the server for the passed application.
func (s *Server) Get(app interface{}) (*ApplicationState, bool) {
	var foundState *ApplicationState
	s.apps.Range(func(_ interface{}, val interface{}) bool {
		as := val.(*ApplicationState)
		if as.app == app {
			foundState = as
			return false
		}
		return true
	})
	return foundState, foundState != nil
}

// FindByInputType application by input type
func (s *Server) FindByInputType(inputType string) (*ApplicationState, bool) {
	var foundState *ApplicationState
	s.apps.Range(func(_ interface{}, val interface{}) bool {
		as := val.(*ApplicationState)
		if as.inputTypes == nil {
			return true
		}

		if _, ok := as.inputTypes[inputType]; ok {
			foundState = as
			return false
		}
		return true
	})
	return foundState, foundState != nil
}

// Register registers a new application to connect to the server.
func (s *Server) Register(app interface{}, config string) (*ApplicationState, error) {
	if _, ok := s.Get(app); ok {
		return nil, ErrApplicationAlreadyRegistered
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	srvName, err := genServerName()
	if err != nil {
		return nil, err
	}
	pair, err := s.ca.GeneratePairWithName(srvName)
	if err != nil {
		return nil, err
	}
	appState := &ApplicationState{
		srv:               s,
		app:               app,
		srvName:           srvName,
		token:             id.String(),
		cert:              pair,
		pendingExpected:   make(chan *proto.StateExpected),
		expected:          proto.StateExpected_RUNNING,
		expectedConfigIdx: 1,
		expectedConfig:    config,
		checkinConn:       true,
		status:            proto.StateObserved_STARTING,
		statusConfigIdx:   client.InitialConfigIdx,
		statusTime:        time.Now().UTC(),
		pendingActions:    make(chan *pendingAction, 100),
		sentActions:       make(map[string]*sentAction),
		actionsConn:       true,
	}
	s.apps.Store(appState.token, appState)
	return appState, nil
}

// Checkin implements the GRPC bi-direction stream connection for check-ins.
func (s *Server) Checkin(server proto.ElasticAgent_CheckinServer) error {
	firstCheckinChan := make(chan *proto.StateObserved)
	go func() {
		// go func will not be leaked, because when the main function
		// returns it will close the connection. that will cause this
		// function to return.
		observed, err := server.Recv()
		if err != nil {
			close(firstCheckinChan)
			return
		}
		firstCheckinChan <- observed
	}()

	var ok bool
	var observedConfigStateIdx uint64
	var firstCheckin *proto.StateObserved
	select {
	case firstCheckin, ok = <-firstCheckinChan:
		if firstCheckin != nil {
			observedConfigStateIdx = firstCheckin.ConfigStateIdx
		}
		break
	case <-time.After(InitialCheckinTimeout):
		// close connection
		s.logger.Debug("check-in stream never sent initial observed message; closing connection")
		return nil
	}
	if !ok {
		// close connection
		return nil
	}
	appState, ok := s.getByToken(firstCheckin.Token)
	if !ok {
		// no application with token; close connection
		s.logger.Debug("check-in stream sent an invalid token; closing connection")
		return status.Error(codes.PermissionDenied, "invalid token")
	}
	appState.checkinLock.Lock()
	if appState.checkinDone != nil {
		// application is already connected (cannot have multiple); close connection
		appState.checkinLock.Unlock()
		s.logger.Debug("check-in stream already exists for application; closing connection")
		return status.Error(codes.AlreadyExists, "application already connected")
	}
	if !appState.checkinConn {
		// application is being destroyed cannot reconnect; close connection
		appState.checkinLock.Unlock()
		s.logger.Debug("check-in stream cannot connect, application is being destroyed; closing connection")
		return status.Error(codes.Unavailable, "application cannot connect being destroyed")
	}

	// application is running as a service and counter is already counting
	// force config reload
	if observedConfigStateIdx > 0 {
		appState.expectedConfigIdx = observedConfigStateIdx + 1
	}

	checkinDone := make(chan bool)
	appState.checkinDone = checkinDone
	appState.checkinLock.Unlock()

	defer func() {
		appState.checkinLock.Lock()
		appState.checkinDone = nil
		appState.checkinLock.Unlock()
	}()

	// send the config and expected state changes to the applications when
	// pushed on the channel
	recvDone := make(chan bool)
	sendDone := make(chan bool)
	go func() {
		defer func() {
			close(sendDone)
		}()
		for {
			var expected *proto.StateExpected
			select {
			case <-checkinDone:
				return
			case <-recvDone:
				return
			case expected = <-appState.pendingExpected:
			}

			err := server.Send(expected)
			if err != nil {
				if reportableErr(err) {
					s.logger.Debugf("check-in stream failed to send expected state: %s", err)
				}
				return
			}
		}
	}()

	// update status after the pendingExpected channel has a reader
	appState.updateStatus(firstCheckin, true)

	// read incoming state observations from the application and act based on
	// the servers expected state of the application
	go func() {
		for {
			checkin, err := server.Recv()
			if err != nil {
				if reportableErr(err) {
					s.logger.Debugf("check-in stream failed to receive data: %s", err)
				}
				close(recvDone)
				return
			}
			appState.updateStatus(checkin, false)
		}
	}()

	<-sendDone
	return nil
}

// Actions implements the GRPC bi-direction stream connection for actions.
func (s *Server) Actions(server proto.ElasticAgent_ActionsServer) error {
	firstRespChan := make(chan *proto.ActionResponse)
	go func() {
		// go func will not be leaked, because when the main function
		// returns it will close the connection. that will cause this
		// function to return.
		observed, err := server.Recv()
		if err != nil {
			close(firstRespChan)
			return
		}
		firstRespChan <- observed
	}()

	var ok bool
	var firstResp *proto.ActionResponse
	select {
	case firstResp, ok = <-firstRespChan:
		break
	case <-time.After(InitialCheckinTimeout):
		// close connection
		s.logger.Debug("actions stream never sent initial response message; closing connection")
		return nil
	}
	if !ok {
		// close connection
		return nil
	}
	if firstResp.Id != client.ActionResponseInitID {
		// close connection
		s.logger.Debug("actions stream first response message must be an init message; closing connection")
		return status.Error(codes.InvalidArgument, "initial response must be an init message")
	}
	appState, ok := s.getByToken(firstResp.Token)
	if !ok {
		// no application with token; close connection
		s.logger.Debug("actions stream sent an invalid token; closing connection")
		return status.Error(codes.PermissionDenied, "invalid token")
	}
	appState.actionsLock.Lock()
	if appState.actionsDone != nil {
		// application is already connected (cannot have multiple); close connection
		appState.actionsLock.Unlock()
		s.logger.Debug("actions stream already exists for application; closing connection")
		return status.Error(codes.AlreadyExists, "application already connected")
	}
	if !appState.actionsConn {
		// application is being destroyed cannot reconnect; close connection
		appState.actionsLock.Unlock()
		s.logger.Debug("actions stream cannot connect, application is being destroyed; closing connection")
		return status.Error(codes.Unavailable, "application cannot connect being destroyed")
	}
	actionsDone := make(chan bool)
	appState.actionsDone = actionsDone
	appState.actionsLock.Unlock()

	defer func() {
		appState.actionsLock.Lock()
		appState.actionsDone = nil
		appState.actionsLock.Unlock()
	}()

	// send the pending actions that need to be performed
	recvDone := make(chan bool)
	sendDone := make(chan bool)
	go func() {
		defer func() { close(sendDone) }()
		for {
			var pending *pendingAction
			select {
			case <-actionsDone:
				return
			case <-recvDone:
				return
			case pending = <-appState.pendingActions:
			}

			if pending.expiresOn.Sub(time.Now().UTC()) <= 0 {
				// to late action already expired
				pending.callback(nil, ErrActionTimedOut)
				continue
			}

			appState.actionsLock.Lock()
			err := server.Send(&proto.ActionRequest{
				Id:     pending.id,
				Name:   pending.name,
				Params: pending.params,
			})
			if err != nil {
				// failed to send action; add back to channel to retry on re-connect from the client
				appState.actionsLock.Unlock()
				appState.pendingActions <- pending
				if reportableErr(err) {
					s.logger.Debugf("failed to send pending action %s (will retry, after re-connect): %s", pending.id, err)
				}
				return
			}
			appState.sentActions[pending.id] = &sentAction{
				callback:  pending.callback,
				expiresOn: pending.expiresOn,
			}
			appState.actionsLock.Unlock()
		}
	}()

	// receive the finished actions
	go func() {
		for {
			response, err := server.Recv()
			if err != nil {
				if reportableErr(err) {
					s.logger.Debugf("actions stream failed to receive data: %s", err)
				}
				close(recvDone)
				return
			}
			appState.actionsLock.Lock()
			action, ok := appState.sentActions[response.Id]
			if !ok {
				// nothing to do, unknown action request
				s.logger.Debugf("actions stream received an unknown action: %s", response.Id)
				appState.actionsLock.Unlock()
				continue
			}
			delete(appState.sentActions, response.Id)
			appState.actionsLock.Unlock()

			var result map[string]interface{}
			err = json.Unmarshal(response.Result, &result)
			if err != nil {
				action.callback(nil, err)
			} else if response.Status == proto.ActionResponse_FAILED {
				errStr, ok := result["error"]
				if ok {
					err = fmt.Errorf("%s", errStr)
				} else {
					err = fmt.Errorf("unknown error")
				}
				action.callback(nil, err)
			} else {
				action.callback(result, nil)
			}
		}
	}()

	<-sendDone
	return nil
}

// WriteConnInfo writes the connection information for the application into the writer.
//
// Note: If the writer implements io.Closer the writer is also closed.
func (as *ApplicationState) WriteConnInfo(w io.Writer) error {
	connInfo := &proto.ConnInfo{
		Addr:       as.srv.getListenAddr(),
		ServerName: as.srvName,
		Token:      as.token,
		CaCert:     as.srv.ca.Crt(),
		PeerCert:   as.cert.Crt,
		PeerKey:    as.cert.Key,
	}
	infoBytes, err := protobuf.Marshal(connInfo)
	if err != nil {
		return errors.New(err, "failed to marshal connection information", errors.TypeApplication)
	}
	_, err = w.Write(infoBytes)
	if err != nil {
		return errors.New(err, "failed to write connection information", errors.TypeApplication)
	}
	closer, ok := w.(io.Closer)
	if ok {
		_ = closer.Close()
	}
	return nil
}

// Stop instructs the application to stop gracefully within the timeout.
//
// Once the application is stopped or the timeout is reached the application is destroyed. Even in the case
// the application times out during stop and ErrApplication
func (as *ApplicationState) Stop(timeout time.Duration) error {
	as.checkinLock.Lock()
	wasConn := as.checkinDone != nil
	cfgIdx := as.statusConfigIdx
	as.expected = proto.StateExpected_STOPPING
	as.checkinLock.Unlock()

	// send it to the client if its connected, otherwise it will be sent once it connects.
	as.sendExpectedState(&proto.StateExpected{
		State:          proto.StateExpected_STOPPING,
		ConfigStateIdx: cfgIdx,
		Config:         "",
	}, false)

	started := time.Now().UTC()
	for {
		if time.Now().UTC().Sub(started) > timeout {
			as.Destroy()
			return ErrApplicationStopTimedOut
		}

		as.checkinLock.RLock()
		s := as.status
		doneChan := as.checkinDone
		as.checkinLock.RUnlock()
		if (wasConn && doneChan == nil) || (!wasConn && s == proto.StateObserved_STOPPING && doneChan == nil) {
			// either occurred:
			// * client was connected then disconnected on stop
			// * client was not connected; connected; received stopping; then disconnected
			as.Destroy()
			return nil
		}

		<-time.After(500 * time.Millisecond)
	}
}

// Destroy completely removes the application from the server without sending any stop command to the application.
//
// The ApplicationState at this point cannot be used.
func (as *ApplicationState) Destroy() {
	as.destroyActionsStream()
	as.destroyCheckinStream()
	as.srv.apps.Delete(as.token)
}

// UpdateConfig pushes an updated configuration to the connected application.
func (as *ApplicationState) UpdateConfig(config string) error {
	as.checkinLock.RLock()
	expected := as.expected
	currentCfg := as.expectedConfig
	as.checkinLock.RUnlock()
	if expected == proto.StateExpected_STOPPING {
		return ErrApplicationStopping
	}
	if config == currentCfg {
		// already at that expected config
		return nil
	}

	as.checkinLock.Lock()
	idx := as.expectedConfigIdx + 1
	as.expectedConfigIdx = idx
	as.expectedConfig = config
	as.checkinLock.Unlock()

	// send it to the client if its connected, otherwise it will be sent once it connects.
	as.sendExpectedState(&proto.StateExpected{
		State:          expected,
		ConfigStateIdx: idx,
		Config:         config,
	}, false)
	return nil
}

// PerformAction synchronously performs an action on the application.
func (as *ApplicationState) PerformAction(name string, params map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	if !as.actionsConn {
		// actions stream destroyed, action cancelled
		return nil, ErrActionCancelled
	}

	resChan := make(chan actionResult)
	as.pendingActions <- &pendingAction{
		id:     id.String(),
		name:   name,
		params: paramBytes,
		callback: func(m map[string]interface{}, err error) {
			resChan <- actionResult{
				result: m,
				err:    err,
			}
		},
		expiresOn: time.Now().UTC().Add(timeout),
	}
	res := <-resChan
	return res.result, res.err
}

// App returns the registered app for the state.
func (as *ApplicationState) App() interface{} {
	return as.app
}

// Expected returns the expected state of the process.
func (as *ApplicationState) Expected() proto.StateExpected_State {
	as.checkinLock.RLock()
	defer as.checkinLock.RUnlock()
	return as.expected
}

// Config returns the expected config of the process.
func (as *ApplicationState) Config() string {
	as.checkinLock.RLock()
	defer as.checkinLock.RUnlock()
	return as.expectedConfig
}

// Status returns the current observed status.
func (as *ApplicationState) Status() (proto.StateObserved_Status, string, map[string]interface{}) {
	as.checkinLock.RLock()
	defer as.checkinLock.RUnlock()
	return as.status, as.statusMessage, as.statusPayload
}

// SetStatus allows the status to be overwritten by the agent.
//
// This status will be overwritten by the client if it reconnects and updates it status.
func (as *ApplicationState) SetStatus(status proto.StateObserved_Status, msg string, payload map[string]interface{}) error {
	payloadStr, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	as.checkinLock.RLock()
	as.status = status
	as.statusMessage = msg
	as.statusPayload = payload
	as.statusPayloadStr = string(payloadStr)
	as.checkinLock.RUnlock()
	return nil
}

// SetInputTypes sets the allowed action input types for this application
func (as *ApplicationState) SetInputTypes(inputTypes []string) {
	as.checkinLock.Lock()
	as.inputTypes = make(map[string]struct{})
	for _, inputType := range inputTypes {
		as.inputTypes[inputType] = struct{}{}
	}
	as.checkinLock.Unlock()
}

// updateStatus updates the current observed status from the application, sends the expected state back to the
// application if the server expects it to be different then its observed state, and alerts the handler on the
// server when the application status has changed.
func (as *ApplicationState) updateStatus(checkin *proto.StateObserved, waitForReader bool) {
	// convert payload from string to JSON
	var payload map[string]interface{}
	if checkin.Payload != "" {
		// ignore the error, if client is sending bad JSON, then payload will just be nil
		_ = json.Unmarshal([]byte(checkin.Payload), &payload)
	}

	as.checkinLock.Lock()
	expectedStatus := as.expected
	expectedConfigIdx := as.expectedConfigIdx
	expectedConfig := as.expectedConfig
	prevStatus := as.status
	prevMessage := as.statusMessage
	prevPayloadStr := as.statusPayloadStr
	as.status = checkin.Status
	as.statusMessage = checkin.Message
	as.statusPayloadStr = checkin.Payload
	as.statusPayload = payload
	as.statusConfigIdx = checkin.ConfigStateIdx
	as.statusTime = time.Now().UTC()
	as.checkinLock.Unlock()

	var expected *proto.StateExpected
	if expectedStatus == proto.StateExpected_STOPPING && checkin.Status != proto.StateObserved_STOPPING {
		expected = &proto.StateExpected{
			State:          expectedStatus,
			ConfigStateIdx: checkin.ConfigStateIdx, // stopping always inform that the config it has is correct
			Config:         "",
		}
	} else if checkin.ConfigStateIdx != expectedConfigIdx {
		expected = &proto.StateExpected{
			State:          expectedStatus,
			ConfigStateIdx: expectedConfigIdx,
			Config:         expectedConfig,
		}
	}
	if expected != nil {
		as.sendExpectedState(expected, waitForReader)
	}

	// alert the service handler that status has changed for the application
	if prevStatus != checkin.Status || prevMessage != checkin.Message || prevPayloadStr != checkin.Payload {
		as.srv.handler.OnStatusChange(as, checkin.Status, checkin.Message, payload)
	}
}

// sendExpectedState sends the expected status over the pendingExpected channel if the other side is
// waiting for a message.
func (as *ApplicationState) sendExpectedState(expected *proto.StateExpected, waitForReader bool) {
	if waitForReader {
		as.pendingExpected <- expected
		return
	}

	select {
	case as.pendingExpected <- expected:
	default:
	}
}

// destroyActionsStream disconnects the actions stream (prevent reconnect), cancel all pending actions
func (as *ApplicationState) destroyActionsStream() {
	as.actionsLock.Lock()
	as.actionsConn = false
	if as.actionsDone != nil {
		close(as.actionsDone)
		as.actionsDone = nil
	}
	as.actionsLock.Unlock()
	as.cancelActions()
}

// flushExpiredActions flushes any expired actions from the pending channel or current processing.
func (as *ApplicationState) flushExpiredActions() {
	now := time.Now().UTC()
	pendingActions := make([]*pendingAction, 0, len(as.pendingActions))
	for {
		done := false
		select {
		case pending := <-as.pendingActions:
			pendingActions = append(pendingActions, pending)
		default:
			done = true
		}
		if done {
			break
		}
	}
	for _, pending := range pendingActions {
		if pending.expiresOn.Sub(now) <= 0 {
			pending.callback(nil, ErrActionTimedOut)
		} else {
			as.pendingActions <- pending
		}
	}
	as.actionsLock.Lock()
	for id, pendingResp := range as.sentActions {
		if pendingResp.expiresOn.Sub(now) <= 0 {
			delete(as.sentActions, id)
			pendingResp.callback(nil, ErrActionTimedOut)
		}
	}
	as.actionsLock.Unlock()
}

// cancelActions cancels all pending or currently processing actions.
func (as *ApplicationState) cancelActions() {
	for {
		done := false
		select {
		case pending := <-as.pendingActions:
			pending.callback(nil, ErrActionCancelled)
		default:
			done = true
		}
		if done {
			break
		}
	}
	as.actionsLock.Lock()
	for id, pendingResp := range as.sentActions {
		delete(as.sentActions, id)
		pendingResp.callback(nil, ErrActionCancelled)
	}
	as.actionsLock.Unlock()
}

// destroyCheckinStream disconnects the check stream (prevent reconnect).
func (as *ApplicationState) destroyCheckinStream() {
	as.checkinLock.Lock()
	as.checkinConn = false
	if as.checkinDone != nil {
		close(as.checkinDone)
		as.checkinDone = nil
	}
	as.checkinLock.Unlock()
}

// watchdog ensures that the current applications are checking in during the correct intervals of time.
func (s *Server) watchdog() {
	defer s.watchdogWG.Done()
	for {
		t := time.NewTimer(s.watchdogCheckInterval)
		select {
		case <-s.watchdogDone:
			t.Stop()
			return
		case <-t.C:
		}

		now := time.Now().UTC()
		s.apps.Range(func(_ interface{}, val interface{}) bool {
			serverApp := val.(*ApplicationState)
			serverApp.checkinLock.RLock()
			statusTime := serverApp.statusTime
			serverApp.checkinLock.RUnlock()
			if now.Sub(statusTime) > s.checkInMinTimeout {
				serverApp.checkinLock.Lock()
				prevStatus := serverApp.status
				s := prevStatus
				prevMessage := serverApp.statusMessage
				message := prevMessage
				if serverApp.status == proto.StateObserved_DEGRADED {
					s = proto.StateObserved_FAILED
					message = "Missed two check-ins"
					serverApp.status = s
					serverApp.statusMessage = message
					serverApp.statusPayload = nil
					serverApp.statusPayloadStr = ""
					serverApp.statusTime = now
				} else if serverApp.status != proto.StateObserved_FAILED {
					s = proto.StateObserved_DEGRADED
					message = "Missed last check-in"
					serverApp.status = s
					serverApp.statusMessage = message
					serverApp.statusPayload = nil
					serverApp.statusPayloadStr = ""
					serverApp.statusTime = now
				}
				serverApp.checkinLock.Unlock()
				if prevStatus != s || prevMessage != message {
					serverApp.srv.handler.OnStatusChange(serverApp, s, message, nil)
				}
			}
			serverApp.flushExpiredActions()
			return true
		})
	}
}

// getByToken returns an application state by its token.
func (s *Server) getByToken(token string) (*ApplicationState, bool) {
	val, ok := s.apps.Load(token)
	if ok {
		return val.(*ApplicationState), true
	}
	return nil, false
}

// getCertificate returns the TLS certificate based on the clientHello or errors if not found.
func (s *Server) getCertificate(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	var cert *tls.Certificate
	s.apps.Range(func(_ interface{}, val interface{}) bool {
		sa := val.(*ApplicationState)
		if sa.srvName == chi.ServerName {
			cert = sa.cert.Certificate
			return false
		}
		return true
	})
	if cert != nil {
		return cert, nil
	}
	return nil, errors.New("no supported TLS certificate", errors.TypeSecurity)
}

// getListenAddr returns the listening address of the server.
func (s *Server) getListenAddr() string {
	addr := strings.SplitN(s.listenAddr, ":", 2)
	if len(addr) == 2 && addr[1] == "0" {
		port := s.listener.Addr().(*net.TCPAddr).Port
		return fmt.Sprintf("%s:%d", addr[0], port)
	}
	return s.listenAddr
}

type pendingAction struct {
	id        string
	name      string
	params    []byte
	callback  func(map[string]interface{}, error)
	expiresOn time.Time
}

type sentAction struct {
	callback  func(map[string]interface{}, error)
	expiresOn time.Time
}

type actionResult struct {
	result map[string]interface{}
	err    error
}

func reportableErr(err error) bool {
	if err == io.EOF {
		return false
	}
	s, ok := status.FromError(err)
	if !ok {
		return true
	}
	if s.Code() == codes.Canceled {
		return false
	}
	return true
}

func genServerName() (string, error) {
	u, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return strings.Replace(u.String(), "-", "", -1), nil
}
