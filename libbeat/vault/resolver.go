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
	"fmt"
	"strings"
	"sync"

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
	registry   = map[string]*resolver{}
	registryMu sync.Mutex
)

func connKey(c Config) string {
	h := sha256.Sum256([]byte(strings.Join([]string{
		c.Address, c.Namespace, c.AuthMethod, c.Token, c.RoleID, c.SecretID, c.KVMount,
	}, "\x00")))
	return hex.EncodeToString(h[:])
}

// GetResolver returns the shared resolver for a connection, building (and
// caching) the Vault client on first use and reusing it for every subsequent
// monitor that uses the same connection.
func GetResolver(c Config, log *logp.Logger) (*resolver, error) {
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

// Config is the vault connection. It can be provided either as a structured
// block (standalone heartbeat.yml) or, under Fleet, as a single base64-encoded
// JSON string so the whole connection is one $co.elastic.secret{...} value
// (see NewResolverOption). The json tags match the base64/JSON form Kibana
// produces.
type Config struct {
	Enabled bool `config:"enabled" json:"enabled"`

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
}

// decodeBase64Config decodes a base64-encoded JSON vault connection. This is
// the form delivered by Fleet, where the whole connection is stored as a single
// Fleet secret and injected as one opaque string — base64 avoids any YAML
// formatting issues in the agent policy.
func decodeBase64Config(s string) (Config, error) {
	var c Config
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return c, fmt.Errorf("vault: config is not valid base64: %w", err)
	}
	if err := json.Unmarshal(decoded, &c); err != nil {
		return c, fmt.Errorf("vault: decoded config is not valid JSON: %w", err)
	}
	// Presence of a delivered connection implies it is enabled.
	c.Enabled = true
	return c, nil
}

// resolver reads secrets from Vault and memoizes them for the process lifetime.
type resolver struct {
	client *vaultapi.Client
	mount  string
	log    *logp.Logger

	mu    sync.Mutex
	cache map[string]string
}

// NewResolverOption reads the `vault:` block from the given config and, when it
// is present and enabled, returns a go-ucfg resolver option plus ok=true. When
// the block is absent or disabled it returns ok=false and the caller should
// leave the config options unchanged.
func NewResolverOption(cfg *config.C, log *logp.Logger) (ucfg.Option, bool, error) {
	if cfg == nil || !cfg.HasField("vault") {
		return nil, false, nil
	}

	var c Config
	// Fleet delivers the whole connection as a single base64-encoded JSON string
	// (a $co.elastic.secret{...} value). Standalone heartbeat.yml can instead use
	// a structured block. Detect the string form first.
	if s, serr := cfg.String("vault", -1); serr == nil && s != "" {
		decoded, derr := decodeBase64Config(s)
		if derr != nil {
			return nil, false, derr
		}
		c = decoded
	} else {
		sub, cerr := cfg.Child("vault", -1)
		if cerr != nil {
			return nil, false, fmt.Errorf("vault: reading config block: %w", cerr)
		}
		if uerr := sub.Unpack(&c); uerr != nil {
			return nil, false, fmt.Errorf("vault: invalid config: %w", uerr)
		}
	}
	if !c.Enabled {
		return nil, false, nil
	}

	r, err := GetResolver(c, log)
	if err != nil {
		return nil, false, err
	}
	return ucfg.Resolve(r.resolve), true, nil
}

// ResolveConfig handles the per-monitor delivery form: under Fleet, each monitor
// carries its own `vault` connection (as a base64 string). When present, the
// shared resolver for that connection is looked up (built once, cached), the
// monitor's ${vault/<path>#<field>} references are resolved in place, and the
// `vault` field is stripped so the monitor plugin never sees it. Monitors
// without a `vault` field are returned unchanged.
func ResolveConfig(raw *config.C, log *logp.Logger) (*config.C, error) {
	if raw == nil || !raw.HasField("vault") {
		return raw, nil
	}

	var c Config
	if s, err := raw.String("vault", -1); err == nil && s != "" {
		decoded, derr := decodeBase64Config(s)
		if derr != nil {
			return nil, derr
		}
		c = decoded
	} else {
		sub, cerr := raw.Child("vault", -1)
		if cerr != nil {
			return nil, fmt.Errorf("vault: reading monitor connection: %w", cerr)
		}
		if uerr := sub.Unpack(&c); uerr != nil {
			return nil, fmt.Errorf("vault: invalid monitor connection: %w", uerr)
		}
	}

	// Resolve the monitor's ${vault/...} references using the shared (cached)
	// resolver. config.C.Unpack always uses the process-global options, so drop
	// to the underlying ucfg config to add the per-monitor vault resolver.
	var m map[string]interface{}
	if c.Enabled {
		r, err := GetResolver(c, log)
		if err != nil {
			return nil, err
		}
		opts := []ucfg.Option{
			ucfg.PathSep("."),
			ucfg.Resolve(r.resolve),
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

func newResolver(c Config, log *logp.Logger) (*resolver, error) {
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

	authMethod := c.AuthMethod
	if authMethod == "" {
		if c.RoleID != "" {
			authMethod = "approle"
		} else {
			authMethod = "token"
		}
	}

	switch authMethod {
	case "token":
		if c.Token == "" {
			return nil, fmt.Errorf("vault: token auth requires a token")
		}
		client.SetToken(c.Token)
	case "approle":
		if c.RoleID == "" || c.SecretID == "" {
			return nil, fmt.Errorf("vault: approle auth requires role_id and secret_id")
		}
		secret, err := client.Logical().WriteWithContext(context.Background(), "auth/approle/login", map[string]interface{}{
			"role_id":   c.RoleID,
			"secret_id": c.SecretID,
		})
		if err != nil {
			return nil, fmt.Errorf("vault: approle login failed: %w", err)
		}
		if secret == nil || secret.Auth == nil || secret.Auth.ClientToken == "" {
			return nil, fmt.Errorf("vault: approle login returned no client token")
		}
		client.SetToken(secret.Auth.ClientToken)
	default:
		return nil, fmt.Errorf("vault: unknown auth_method %q", authMethod)
	}

	mount := c.KVMount
	if mount == "" {
		mount = defaultKVMount
	}

	log.Infof("initialized HashiCorp Vault resolver (address=%s, kv_mount=%s, auth=%s)", c.Address, mount, authMethod)

	return &resolver{
		client: client,
		mount:  mount,
		log:    log,
		cache:  map[string]string{},
	}, nil
}

// resolve is the go-ucfg resolver callback. It only handles keys prefixed with
// "vault/"; anything else returns ucfg.ErrMissing so the next resolver (keystore
// / env) gets a chance.
func (r *resolver) resolve(key string) (string, parse.Config, error) {
	// parse.NoopConfig prevents the resolved secret from being re-interpreted by
	// the ucfg parser (e.g. a value that happens to contain ${...} or commas).
	noop := parse.NoopConfig

	if !strings.HasPrefix(key, refPrefix) {
		return "", noop, ucfg.ErrMissing
	}

	spec := strings.TrimPrefix(key, refPrefix)
	hash := strings.LastIndex(spec, "#")
	if hash < 0 {
		return "", noop, fmt.Errorf("vault: reference %q must be of the form vault/<path>#<field>", key)
	}
	path := strings.Trim(spec[:hash], "/")
	field := spec[hash+1:]
	if path == "" || field == "" {
		return "", noop, fmt.Errorf("vault: reference %q has an empty path or field", key)
	}

	val, err := r.read(path, field)
	if err != nil {
		return "", noop, err
	}
	return val, noop, nil
}

func (r *resolver) read(path, field string) (string, error) {
	cacheKey := path + "#" + field

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok := r.cache[cacheKey]; ok {
		return v, nil
	}

	secret, err := r.client.KVv2(r.mount).Get(context.Background(), path)
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

	val := fmt.Sprintf("%v", raw)
	r.cache[cacheKey] = val
	// Never log the secret value itself.
	r.log.Debugf("resolved vault reference %s/%s#%s", r.mount, path, field)
	return val, nil
}
