// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package vault provides a go-ucfg config resolver that expands references of
// the form ${vault/<path>#<field>} to secret values fetched at runtime from a
// HashiCorp Vault server.
//
// The resolver is wired into the process-global config options next to the
// keystore resolver, so any ${vault/...} reference in any monitor, stream or
// top-level config value is transparently resolved during config unpack. The
// plaintext secret only ever lives in the Heartbeat process memory; it is never
// written to disk or shipped anywhere.
//
// This is a proof-of-concept. Placement (oss vs x-pack), token lease renewal,
// additional auth methods and TTL-based cache expiry are follow-ups.
package vault

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	vaultapi "github.com/hashicorp/vault/api"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/parse"
)

// registry caches one resolver (Vault client + secret cache) per unique
// connection for the lifetime of the process. Many monitors share the same
// Vault connection, so the client is built once and the resolved-secret cache
// is shared across every monitor — a monitor run never re-fetches a secret that
// another monitor already resolved.
var (
	registry   = map[string]Resolver{}
	registryMu sync.Mutex
)

func connKey(c Config) string {
	h := sha256.Sum256([]byte(strings.Join([]string{
		c.Type, c.Name, c.Address, c.Namespace, c.AuthMethod, c.Token, c.RoleID, c.SecretID, c.KVMount,
		c.SecretRefreshInterval, c.Version,
	}, "\x00")))
	return hex.EncodeToString(h[:])
}

// GetResolver returns the shared resolver for a connection, building (and
// caching) the Vault client on first use and reusing it for every subsequent
// monitor that uses the same connection.
func GetResolver(c Config, log *logp.Logger) (Resolver, error) {
	key := connKey(c)

	registryMu.Lock()
	defer registryMu.Unlock()

	if r, ok := registry[key]; ok {
		return r, nil
	}
	r, err := newResolver(c, log)
	if err != nil {
		return nil, err
	}
	registry[key] = r
	return r, nil
}

// refPrefix is the sentinel that routes a config reference to this resolver.
// A reference looks like ${vault/<secret-path>#<field>}, e.g.
// ${vault/myapp/creds#password}. We deliberately use '/' and '#' (rather than
// ':') because ':' is the go-ucfg default-value operator and would break
// parsing, and because '/' and '#' are outside Kibana's param-substitution
// character class, so Kibana passes the token through untouched.
const refPrefix = "vault/"

const defaultKVMount = "secret"

// ProviderHashiCorpVault is the only implemented secret-provider backend. New
// backends add their own constant and a case in newResolver.
const ProviderHashiCorpVault = "hashicorp_vault"

// Config is a single vault connection. Multiple connections may be configured
// (as an array); each is addressed by Name in references of the form
// ${vault/<name>@<path>#<field>}. A reference without "<name>@" uses the default
// connection. Config may be provided as a structured block (standalone
// heartbeat.yml) or, under Fleet, as a base64-encoded JSON value (a single
// object or an array) so it can be one $co.elastic.secret{...} value. The json
// tags match the base64/JSON form Kibana produces.
type Config struct {
	Enabled bool `config:"enabled" json:"enabled"`

	// Type selects the secret-provider backend. Only "hashicorp_vault" is
	// implemented; empty defaults to it. Adding a provider (CyberArk, Azure Key
	// Vault, AWS Secrets Manager) means a new Resolver impl plus a case in
	// newResolver — references, caching and dispatch are provider-agnostic.
	Type string `config:"type" json:"type"`

	// Name addresses this connection in references (${vault/<name>@...}). Empty
	// means the default connection.
	Name string `config:"name" json:"name"`

	Address   string `config:"address" json:"address"`
	Namespace string `config:"namespace" json:"namespace"`

	// AuthMethod is "token" or "approle" (default "approle" when role_id is set,
	// otherwise "token").
	AuthMethod string `config:"auth_method" json:"auth_method"`
	Token      string `config:"token" json:"token"`
	RoleID     string `config:"role_id" json:"role_id"`
	SecretID   string `config:"secret_id" json:"secret_id"`

	// KVMount is the KV v2 secrets-engine mount path (default "secret").
	KVMount string `config:"kv_mount" json:"kv_mount"`

	// TLSSkipVerify disables TLS verification. Intended for local/dev only.
	TLSSkipVerify bool `config:"tls_skip_verify" json:"tls_skip_verify"`

	// SecretRefreshInterval is how long a resolved secret is cached before it is
	// re-read from Vault — the cache-invalidation window that lets rotated
	// secrets be picked up without restarting. A Go duration string, e.g. "5m".
	// Defaults to defaultRefreshInterval.
	SecretRefreshInterval string `config:"secret_refresh_interval" json:"secret_refresh_interval"`

	// Version is an opaque value bumped by the control plane (e.g. when a user
	// clicks "Refresh secrets"). It is part of the connection identity, so a new
	// version yields a fresh resolver with an empty cache — the next read
	// re-fetches from Vault. This is what makes a manual refresh force an update.
	Version string `config:"version" json:"version"`
}

const defaultRefreshInterval = 5 * time.Minute

func (c Config) refreshInterval() time.Duration {
	if c.SecretRefreshInterval == "" {
		return defaultRefreshInterval
	}
	d, err := time.ParseDuration(c.SecretRefreshInterval)
	if err != nil || d <= 0 {
		return defaultRefreshInterval
	}
	return d
}

// decodeConfigs decodes a base64-encoded JSON vault connection. This is the form
// delivered by Fleet, where the connection(s) are stored as a single Fleet
// secret and injected as one opaque string — base64 avoids any YAML formatting
// issues in the agent policy. The JSON may be a single object or an array of
// connections.
func decodeConfigs(s string) ([]Config, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("vault: config is not valid base64: %w", err)
	}
	// Array of connections first, then a single object.
	var arr []Config
	if err := json.Unmarshal(decoded, &arr); err == nil {
		for i := range arr {
			arr[i].Enabled = true // a delivered connection is enabled
		}
		return arr, nil
	}
	var one Config
	if err := json.Unmarshal(decoded, &one); err != nil {
		return nil, fmt.Errorf("vault: decoded config is not valid JSON: %w", err)
	}
	one.Enabled = true
	return []Config{one}, nil
}

// parseConfigs reads the `vault` field as one or more connections. It accepts a
// base64 string (Fleet form), a single object, or an array of objects
// (standalone heartbeat.yml).
func parseConfigs(cfg *config.C) ([]Config, error) {
	if s, err := cfg.String("vault", -1); err == nil && s != "" {
		return decodeConfigs(s)
	}
	var arrWrap struct {
		Vault []Config `config:"vault"`
	}
	if err := cfg.Unpack(&arrWrap); err == nil && len(arrWrap.Vault) > 0 {
		return arrWrap.Vault, nil
	}
	sub, err := cfg.Child("vault", -1)
	if err != nil {
		return nil, fmt.Errorf("vault: reading config block: %w", err)
	}
	var one Config
	if err := sub.Unpack(&one); err != nil {
		return nil, fmt.Errorf("vault: invalid config: %w", err)
	}
	return []Config{one}, nil
}

// dispatcher routes ${vault/[<name>@]<path>#<field>} references to the resolver
// of the named connection (empty name = default).
type dispatcher struct {
	byName map[string]Resolver
}

// buildDispatcher builds (or reuses cached) resolvers for the enabled
// connections, indexed by name. Returns nil when nothing is enabled.
func buildDispatcher(configs []Config, log *logp.Logger) (*dispatcher, error) {
	d := &dispatcher{byName: map[string]Resolver{}}
	var enabled []Config
	for _, c := range configs {
		if !c.Enabled {
			continue
		}
		enabled = append(enabled, c)
		r, err := GetResolver(c, log)
		if err != nil {
			return nil, err
		}
		d.byName[c.Name] = r
	}
	if len(enabled) == 0 {
		return nil, nil
	}
	// A single connection is also the default, so unqualified references work.
	if _, ok := d.byName[""]; !ok && len(enabled) == 1 {
		d.byName[""] = d.byName[enabled[0].Name]
	}
	return d, nil
}

func (d *dispatcher) resolve(key string) (string, parse.Config, error) {
	noop := parse.NoopConfig
	if !strings.HasPrefix(key, refPrefix) {
		return "", noop, ucfg.ErrMissing
	}
	spec := strings.TrimPrefix(key, refPrefix)

	name := ""
	if at := strings.Index(spec, "@"); at >= 0 {
		name = spec[:at]
		spec = spec[at+1:]
	}

	hash := strings.LastIndex(spec, "#")
	if hash < 0 {
		return "", noop, fmt.Errorf(
			"vault: reference %q must be vault/[<connection>@]<path>#<field>", key)
	}
	path := strings.Trim(spec[:hash], "/")
	field := spec[hash+1:]
	if path == "" || field == "" {
		return "", noop, fmt.Errorf("vault: reference %q has an empty path or field", key)
	}

	r, ok := d.byName[name]
	if !ok {
		return "", noop, fmt.Errorf("vault: no connection named %q for reference %q", name, key)
	}
	val, err := r.read(path, field)
	if err != nil {
		return "", noop, err
	}
	return val, noop, nil
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// resolver reads secrets from Vault and caches them for secret_refresh_interval.
type resolver struct {
	client          *vaultapi.Client
	cfg             Config // retained so the token can be re-established on expiry
	mount           string
	refreshInterval time.Duration
	log             *logp.Logger

	mu    sync.Mutex
	cache map[string]cacheEntry
}

// NewResolverOption reads the `vault:` config (one or more connections) and,
// when present and enabled, returns a go-ucfg resolver option plus ok=true. When
// absent or nothing is enabled it returns ok=false and the caller should leave
// the config options unchanged.
func NewResolverOption(cfg *config.C, log *logp.Logger) (ucfg.Option, bool, error) {
	if cfg == nil || !cfg.HasField("vault") {
		return nil, false, nil
	}

	configs, err := parseConfigs(cfg)
	if err != nil {
		return nil, false, err
	}
	d, err := buildDispatcher(configs, log)
	if err != nil {
		return nil, false, err
	}
	if d == nil {
		return nil, false, nil
	}
	return ucfg.Resolve(d.resolve), true, nil
}

// ResolveConfig handles the per-monitor delivery form: under Fleet, each monitor
// carries its own `vault` connection(s) (as a base64 string, single or array).
// The shared resolver for each connection is looked up (built once, cached), the
// monitor's ${vault/[<name>@]<path>#<field>} references are resolved in place,
// and the `vault` field is stripped so the monitor plugin never sees it.
// Monitors without a `vault` field are returned unchanged.
func ResolveConfig(raw *config.C, log *logp.Logger) (*config.C, error) {
	if raw == nil || !raw.HasField("vault") {
		return raw, nil
	}

	configs, err := parseConfigs(raw)
	if err != nil {
		return nil, err
	}
	d, err := buildDispatcher(configs, log)
	if err != nil {
		return nil, err
	}

	// Resolve the monitor's ${vault/...} references using the shared (cached)
	// resolvers. config.C.Unpack always uses the process-global options, so drop
	// to the underlying ucfg config to add the per-monitor vault resolver.
	var m map[string]interface{}
	if d != nil {
		opts := []ucfg.Option{
			ucfg.PathSep("."),
			ucfg.Resolve(d.resolve),
			ucfg.ResolveEnv,
			ucfg.VarExp,
		}
		if err := (*ucfg.Config)(raw).Unpack(&m, opts...); err != nil {
			return nil, fmt.Errorf("vault: resolving monitor references: %w", err)
		}
	} else if err := raw.Unpack(&m); err != nil {
		return nil, err
	}

	// The connection is consumed here; the monitor plugin must not see it.
	delete(m, "vault")

	resolved, err := config.NewConfigFrom(m)
	if err != nil {
		return nil, fmt.Errorf("vault: rebuilding monitor config: %w", err)
	}
	return resolved, nil
}

// Resolver reads a secret field for one connection. Each provider backend
// implements it; the registry, dispatcher and cache treat every provider
// uniformly, so only newResolver knows about concrete backends.
type Resolver interface {
	read(path, field string) (string, error)
}

// providerTypeOf returns the connection's backend, defaulting to HashiCorp Vault
// when unset (older blobs, or a single-provider deployment).
func providerTypeOf(c Config) string {
	if c.Type == "" {
		return ProviderHashiCorpVault
	}
	return c.Type
}

// newResolver builds the resolver for a connection's provider type. This switch is
// the single extension point for new backends.
func newResolver(c Config, log *logp.Logger) (Resolver, error) {
	switch providerTypeOf(c) {
	case ProviderHashiCorpVault:
		return newVaultResolver(c, log)
	default:
		return nil, fmt.Errorf("vault: unknown provider type %q", c.Type)
	}
}

// newVaultResolver builds a HashiCorp Vault resolver: a Vault API client
// authenticated per the connection, with a TTL secret cache.
func newVaultResolver(c Config, log *logp.Logger) (*resolver, error) {
	if log == nil {
		log = logp.NewLogger("vault")
	} else {
		log = log.Named("vault")
	}

	if c.Address == "" {
		return nil, fmt.Errorf("vault: address is required")
	}

	apiCfg := vaultapi.DefaultConfig()
	apiCfg.Address = c.Address
	if c.TLSSkipVerify {
		if err := apiCfg.ConfigureTLS(&vaultapi.TLSConfig{Insecure: true}); err != nil {
			return nil, fmt.Errorf("vault: configuring TLS: %w", err)
		}
	}

	client, err := vaultapi.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("vault: creating client: %w", err)
	}
	if c.Namespace != "" {
		client.SetNamespace(c.Namespace)
	}

	if err := authenticate(client, c); err != nil {
		return nil, err
	}

	mount := c.KVMount
	if mount == "" {
		mount = defaultKVMount
	}

	r := &resolver{
		client:          client,
		cfg:             c,
		mount:           mount,
		refreshInterval: c.refreshInterval(),
		log:             log,
		cache:           map[string]cacheEntry{},
	}
	log.Infof("initialized HashiCorp Vault resolver (address=%s, kv_mount=%s, auth=%s, refresh=%s)",
		c.Address, mount, authMethodOf(c), r.refreshInterval)
	return r, nil
}

// authMethodOf returns the effective auth method (defaults to approle when a
// role_id is present, otherwise token).
func authMethodOf(c Config) string {
	if c.AuthMethod != "" {
		return c.AuthMethod
	}
	if c.RoleID != "" {
		return "approle"
	}
	return "token"
}

// authenticate establishes the client token from the configured auth method. It
// is used both on initial connect and to re-authenticate when the token expires.
func authenticate(client *vaultapi.Client, c Config) error {
	switch authMethodOf(c) {
	case "token":
		if c.Token == "" {
			return fmt.Errorf("vault: token auth requires a token")
		}
		client.SetToken(c.Token)
	case "approle":
		if c.RoleID == "" || c.SecretID == "" {
			return fmt.Errorf("vault: approle auth requires role_id and secret_id")
		}
		secret, err := client.Logical().WriteWithContext(context.Background(), "auth/approle/login", map[string]interface{}{
			"role_id":   c.RoleID,
			"secret_id": c.SecretID,
		})
		if err != nil {
			return fmt.Errorf("vault: approle login failed: %w", err)
		}
		if secret == nil || secret.Auth == nil || secret.Auth.ClientToken == "" {
			return fmt.Errorf("vault: approle login returned no client token")
		}
		client.SetToken(secret.Auth.ClientToken)
	default:
		return fmt.Errorf("vault: unknown auth_method %q", c.AuthMethod)
	}
	return nil
}

func (r *resolver) read(path, field string) (string, error) {
	cacheKey := path + "#" + field

	r.mu.Lock()
	defer r.mu.Unlock()

	// Serve from cache until secret_refresh_interval elapses; after that the
	// secret is re-read so a rotated value is picked up without a restart.
	if e, ok := r.cache[cacheKey]; ok && time.Now().Before(e.expiresAt) {
		return e.value, nil
	}

	val, err := r.fetch(path, field)
	if err != nil {
		return "", err
	}

	r.cache[cacheKey] = cacheEntry{value: val, expiresAt: time.Now().Add(r.refreshInterval)}
	// Never log the secret value itself.
	r.log.Debugf("resolved vault reference %s/%s#%s (cached %s)", r.mount, path, field, r.refreshInterval)
	return val, nil
}

// fetch reads a single field from Vault, re-authenticating once if the token has
// expired (403), so a long-lived Heartbeat survives token/lease expiry.
func (r *resolver) fetch(path, field string) (string, error) {
	secret, err := r.client.KVv2(r.mount).Get(context.Background(), path)
	if err != nil && isPermissionError(err) {
		r.log.Debugf("vault token rejected, re-authenticating")
		if aerr := authenticate(r.client, r.cfg); aerr != nil {
			return "", fmt.Errorf("vault: re-authentication failed: %w", aerr)
		}
		secret, err = r.client.KVv2(r.mount).Get(context.Background(), path)
	}
	if err != nil {
		return "", fmt.Errorf("vault: reading %s/%s: %w", r.mount, path, err)
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("vault: no data at %s/%s", r.mount, path)
	}
	raw, ok := secret.Data[field]
	if !ok {
		return "", fmt.Errorf("vault: field %q not found at %s/%s", field, r.mount, path)
	}
	return fmt.Sprintf("%v", raw), nil
}

// isPermissionError reports whether err is a Vault 401/403 (expired/invalid token).
func isPermissionError(err error) bool {
	var respErr *vaultapi.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == 401 || respErr.StatusCode == 403
	}
	return false
}
