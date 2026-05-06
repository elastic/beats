// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.elastic.co/apm/module/apmelasticsearch/v2"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"

	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	krbclient "github.com/elastic/gokrb5/v8/client"
	krbconfig "github.com/elastic/gokrb5/v8/config"
	"github.com/elastic/gokrb5/v8/keytab"
	"github.com/elastic/gokrb5/v8/spnego"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

var (
	_                  extensionauth.HTTPClient = (*authenticator)(nil)
	_                  extensionauth.GRPCClient = (*authenticator)(nil)
	_                  extension.Extension      = (*authenticator)(nil)
	ErrInvalidAuthType                          = errors.New("invalid authentication type")
)

// roundTripperProvider is an interface that provides a RoundTripper
type roundTripperProvider interface {
	RoundTripper() http.RoundTripper
}

type authenticator struct {
	telemetry  component.TelemetrySettings
	logger     *logp.Logger
	cfg        *Config
	rtProvider roundTripperProvider
}

func newAuthenticator(cfg *Config, telemetry component.TelemetrySettings) (*authenticator, error) {
	// logp.NewZapLogger essentially never returns an error; look within the implementation
	logger, err := logp.NewZapLogger(telemetry.Logger.WithOptions(zap.AddCallerSkip(1)))
	if err != nil {
		return nil, err
	}

	auth := &authenticator{
		cfg:       cfg,
		telemetry: telemetry,
		logger:    logger,
	}
	return auth, nil
}

func (a *authenticator) Start(_ context.Context, host component.Host) error {
	switch {
	case a == nil:
		return fmt.Errorf("authenticator is nil")
	case a.cfg == nil:
		return fmt.Errorf("authenticator config is nil")
	}

	var provider roundTripperProvider
	prov, err := getHttpClient(a)
	if err != nil {
		componentstatus.ReportStatus(host, componentstatus.NewPermanentErrorEvent(err))
		err = fmt.Errorf("failed creating http client: %w", err)
		if !a.cfg.ContinueOnError {
			return err
		}
		a.logger.Warnf("%s", err.Error())
		provider = &errorRoundTripperProvider{err: err}
	} else {
		componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusOK))
		provider = prov
	}

	a.rtProvider = provider
	return nil
}

func (a *authenticator) Shutdown(_ context.Context) error {
	return nil
}

func (a *authenticator) RoundTripper(_ http.RoundTripper) (http.RoundTripper, error) {
	if a.rtProvider == nil {
		return nil, fmt.Errorf("authenticator not started")
	}
	return a.rtProvider.RoundTripper(), nil
}

// getHTTPOptions returns a list of http transport options
// these options are derived from beats codebase Ref: https://github.com/elastic/beats/blob/4dfef8b/libbeat/esleg/eslegclient/connection.go#L163-L171
// httpcommon.WithIOStats(s.Observer) is omitted as we do not have access to observer here
// httpcommon.WithHeaderRoundTripper with user-agent is also omitted as we continue to use ES exporter's user-agent
func (a *authenticator) getHTTPOptions(idleConnTimeout time.Duration) []httpcommon.TransportOption {
	return []httpcommon.TransportOption{
		httpcommon.WithLogger(a.logger),
		httpcommon.WithKeepaliveSettings{IdleConnTimeout: idleConnTimeout},
		httpcommon.WithModRoundtripper(func(rt http.RoundTripper) http.RoundTripper {
			return apmelasticsearch.WrapRoundTripper(rt)
		}),
	}
}

func (a *authenticator) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	// Elasticsearch doesn't support gRPC, this function won't be called
	return nil, nil
}

func getHttpClient(a *authenticator) (roundTripperProvider, error) {
	parsedCfg, err := config.NewConfigFrom(a.cfg.BeatAuthConfig)
	if err != nil {
		return nil, fmt.Errorf("failed creating config: %w", err)
	}

	beatAuthConfig := BeatsAuthConfig{}
	err = parsedCfg.Unpack(&beatAuthConfig)
	if err != nil {
		return nil, fmt.Errorf("failed unpacking config: %w", err)
	}

	reload := resolveCertificateReload(parsedCfg)

	httpOpts := a.getHTTPOptions(beatAuthConfig.Transport.IdleConnTimeout)

	reloaderOpt, err := certReloaderTransportOption(reload, beatAuthConfig.Transport.TLS, a.logger)
	if err != nil {
		return nil, err
	}
	if reloaderOpt != nil {
		httpOpts = append(httpOpts, reloaderOpt)
	}

	client, err := beatAuthConfig.Transport.Client(httpOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed creating http client: %w", err)
	}

	if beatAuthConfig.Kerberos.IsEnabled() {
		p, err := NewKerberosClientProvider(beatAuthConfig.Kerberos, client)
		if err != nil {
			return nil, fmt.Errorf("error creating kerberos client provider: %w", err)
		}
		return p, nil
	}

	return &httpClientProvider{client: client}, nil
}

// resolveCertificateReload reads ssl.certificate_reload (the primary key) and
// the legacy ssl.restart_on_cert_change alias, returning the effective reload
// settings.
//
// Semantics:
//   - When the ssl.certificate_reload block is present, Enabled defaults to
//     true; users opt out by setting enabled: false.
//   - When the block is absent, the legacy ssl.restart_on_cert_change.enabled
//     can still turn hot reload on (it never restarted the process here, but
//     the field is honored for backwards compatibility with existing Beats
//     configs).
//   - The legacy period field seeds reload_interval only if the new key did
//     not set it.
func resolveCertificateReload(cfg *config.C) CertificateReloadConfig {
	var result CertificateReloadConfig

	sslConfig, err := cfg.Child("ssl", -1)
	if err != nil {
		return result
	}

	if reloadCfg, err := sslConfig.Child("certificate_reload", -1); err == nil {
		_ = reloadCfg.Unpack(&result)
		// Block is present: enabled defaults to true unless explicitly disabled.
		if result.Enabled == nil {
			t := true
			result.Enabled = &t
		}
	}

	if rocc, err := sslConfig.Child("restart_on_cert_change", -1); err == nil {
		type restartOnCertChange struct {
			Enabled bool          `config:"enabled"`
			Period  time.Duration `config:"period"`
		}
		var alias restartOnCertChange
		if err := rocc.Unpack(&alias); err == nil {
			if result.Enabled == nil && alias.Enabled {
				t := true
				result.Enabled = &t
			}
			if alias.Period > 0 && result.ReloadInterval == 0 {
				result.ReloadInterval = alias.Period
			}
		}
	}

	return result
}

// certReloaderTransportOption returns an httpcommon.TransportOption that wires a
// CertReloader into the *http.Transport's TLSClientConfig when certificate hot
// reload is enabled and cert/key paths are configured.
// Returns nil (no option) when hot reload is not applicable.
// Encrypted keys (key_passphrase) are not supported by CertReloader; a warning
// is logged and nil is returned in that case.
func certReloaderTransportOption(reload CertificateReloadConfig, tlsCfg *tlscommon.Config, logger *logp.Logger) (httpcommon.TransportOption, error) {
	if reload.Enabled == nil || !*reload.Enabled {
		return nil, nil
	}

	if tlsCfg == nil || tlsCfg.Certificate.Certificate == "" || tlsCfg.Certificate.Key == "" {
		return nil, nil
	}

	if tlsCfg.Certificate.Passphrase != "" || tlsCfg.Certificate.PassphrasePath != "" {
		logger.Warn("ssl.certificate_reload is not supported with encrypted keys; hot reload disabled")
		return nil, nil
	}

	var opts []tlscommon.CertReloaderOption
	if reload.ReloadInterval > 0 {
		opts = append(opts, tlscommon.WithReloadInterval(reload.ReloadInterval))
	}

	reloader, err := tlscommon.NewCertReloader(tlsCfg.Certificate.Certificate, tlsCfg.Certificate.Key, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed creating cert reloader: %w", err)
	}

	opt := httpcommon.WithTransportFunc(func(t *http.Transport) {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{} //nolint:gosec // min version set by beats defaults
		}
		t.TLSClientConfig.GetClientCertificate = func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return reloader.GetCertificate(nil)
		}
		// Clear the static certificate list; the reloader's callback takes over.
		t.TLSClientConfig.Certificates = nil
	})

	return opt, nil
}

// httpClientProvider provides a RoundTripper from an http.Client
type httpClientProvider struct {
	client *http.Client
}

func (h *httpClientProvider) RoundTripper() http.RoundTripper {
	return h.client.Transport
}

// kerberosClientProvider provides a kerberos enabled roundtripper
type kerberosClientProvider struct {
	kerberosClient *krbclient.Client
	httpClient     *http.Client
}

func NewKerberosClientProvider(config *kerberos.Config, httpClient *http.Client) (*kerberosClientProvider, error) {
	var krbClient *krbclient.Client
	krbConf, err := krbconfig.Load(config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error creating Kerberos client: %w", err)
	}

	switch config.AuthType {
	// case 1 is password auth
	case 1:
		krbClient = krbclient.NewWithPassword(config.Username, config.Realm, config.Password, krbConf)
	// case 2 is keytab auth
	case 2:
		kTab, err := keytab.Load(config.KeyTabPath)
		if err != nil {
			return nil, fmt.Errorf("cannot load keytab file %s: %w", config.KeyTabPath, err)
		}
		krbClient = krbclient.NewWithKeytab(config.Username, config.Realm, kTab, krbConf)
	default:
		return nil, ErrInvalidAuthType
	}

	return &kerberosClientProvider{kerberosClient: krbClient, httpClient: httpClient}, nil
}

func (k *kerberosClientProvider) RoundTripper() http.RoundTripper {
	return k
}

func (k *kerberosClientProvider) RoundTrip(req *http.Request) (*http.Response, error) {
	// set appropriate headers on request
	err := spnego.SetSPNEGOHeader(k.kerberosClient, req, "")
	if err != nil {
		return nil, err
	}

	return k.httpClient.Transport.RoundTrip(req)
}

// errorRoundTripperProvider provides a RoundTripper that always returns an error
type errorRoundTripperProvider struct {
	err error
}

func (e *errorRoundTripperProvider) RoundTripper() http.RoundTripper {
	return &errorRoundTripper{err: e.err}
}

// errorRoundTripper is a RoundTripper that always returns an error
type errorRoundTripper struct {
	err error
}

func (e *errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, e.err
}
