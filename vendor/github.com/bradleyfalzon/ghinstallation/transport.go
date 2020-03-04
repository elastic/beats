package ghinstallation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/v28/github"
)

const (
	// acceptHeader is the GitHub Apps Preview Accept header.
	acceptHeader = "application/vnd.github.machine-man-preview+json"
	apiBaseURL   = "https://api.github.com"
)

// Transport provides a http.RoundTripper by wrapping an existing
// http.RoundTripper and provides GitHub Apps authentication as an
// installation.
//
// Client can also be overwritten, and is useful to change to one which
// provides retry logic if you do experience retryable errors.
//
// See https://developer.github.com/apps/building-integrations/setting-up-and-registering-github-apps/about-authentication-options-for-github-apps/
type Transport struct {
	BaseURL                  string                           // BaseURL is the scheme and host for GitHub API, defaults to https://api.github.com
	Client                   Client                           // Client to use to refresh tokens, defaults to http.Client with provided transport
	tr                       http.RoundTripper                // tr is the underlying roundtripper being wrapped
	appID                    int64                            // appID is the GitHub App's ID
	installationID           int64                            // installationID is the GitHub App Installation ID
	InstallationTokenOptions *github.InstallationTokenOptions // parameters restrict a token's access
	appsTransport            *AppsTransport

	mu    *sync.Mutex  // mu protects token
	token *accessToken // token is the installation's access token
}

// accessToken is an installation access token response from GitHub
type accessToken struct {
	Token        string                         `json:"token"`
	ExpiresAt    time.Time                      `json:"expires_at"`
	Permissions  github.InstallationPermissions `json:"permissions,omitempty"`
	Repositories []github.Repository            `json:"repositories,omitempty"`
}

var _ http.RoundTripper = &Transport{}

// NewKeyFromFile returns a Transport using a private key from file.
func NewKeyFromFile(tr http.RoundTripper, appID, installationID int64, privateKeyFile string) (*Transport, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("could not read private key: %s", err)
	}
	return New(tr, appID, installationID, privateKey)
}

// Client is a HTTP client which sends a http.Request and returns a http.Response
// or an error.
type Client interface {
	Do(*http.Request) (*http.Response, error)
}

// New returns an Transport using private key. The key is parsed
// and if any errors occur the error is non-nil.
//
// The provided tr http.RoundTripper should be shared between multiple
// installations to ensure reuse of underlying TCP connections.
//
// The returned Transport's RoundTrip method is safe to be used concurrently.
func New(tr http.RoundTripper, appID, installationID int64, privateKey []byte) (*Transport, error) {
	atr, err := NewAppsTransport(tr, appID, privateKey)
	if err != nil {
		return nil, err
	}

	return NewFromAppsTransport(atr, installationID), nil
}

// NewFromAppsTransport returns a Transport using an existing *AppsTransport.
func NewFromAppsTransport(atr *AppsTransport, installationID int64) *Transport {
	return &Transport{
		BaseURL:        apiBaseURL,
		Client:         &http.Client{Transport: atr.tr},
		tr:             atr.tr,
		appID:          atr.appID,
		installationID: installationID,
		appsTransport:  atr,
		mu:             &sync.Mutex{},
	}
}

// RoundTrip implements http.RoundTripper interface.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.Token(req.Context())
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Add("Accept", acceptHeader) // We add to "Accept" header to avoid overwriting existing req headers.
	resp, err := t.tr.RoundTrip(req)
	return resp, err
}

// Token checks the active token expiration and renews if necessary. Token returns
// a valid access token. If renewal fails an error is returned.
func (t *Transport) Token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token == nil || t.token.ExpiresAt.Add(-time.Minute).Before(time.Now()) {
		// Token is not set or expired/nearly expired, so refresh
		if err := t.refreshToken(ctx); err != nil {
			return "", fmt.Errorf("could not refresh installation id %v's token: %s", t.installationID, err)
		}
	}

	return t.token.Token, nil
}

// Permissions returns a transport token's GitHub installation permissions.
func (t *Transport) Permissions() (github.InstallationPermissions, error) {
	if t.token == nil {
		return github.InstallationPermissions{}, fmt.Errorf("Permissions() = nil, err: nil token")
	}
	return t.token.Permissions, nil
}

// Repositories returns a transport token's GitHub repositories.
func (t *Transport) Repositories() ([]github.Repository, error) {
	if t.token == nil {
		return nil, fmt.Errorf("Repositories() = nil, err: nil token")
	}
	return t.token.Repositories, nil
}

func (t *Transport) refreshToken(ctx context.Context) error {
	// Convert InstallationTokenOptions into a ReadWriter to pass as an argument to http.NewRequest.
	body, err := GetReadWriter(t.InstallationTokenOptions)
	if err != nil {
		return fmt.Errorf("could not convert installation token parameters into json: %s", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/app/installations/%v/access_tokens", t.BaseURL, t.installationID), body)
	if err != nil {
		return fmt.Errorf("could not create request: %s", err)
	}

	// Set Content and Accept headers.
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", acceptHeader)

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	t.appsTransport.BaseURL = t.BaseURL
	t.appsTransport.Client = t.Client
	resp, err := t.appsTransport.RoundTrip(req)
	if err != nil {
		return fmt.Errorf("could not get access_tokens from GitHub API for installation ID %v: %v", t.installationID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("request %+v received non 2xx response status %q with body %+v and TLS %+v", resp.Request, resp.Body, resp.Request, resp.TLS)
	}

	return json.NewDecoder(resp.Body).Decode(&t.token)
}

// GetReadWriter converts a body interface into an io.ReadWriter object.
func GetReadWriter(i interface{}) (io.ReadWriter, error) {
	var buf io.ReadWriter
	if i != nil {
		buf = new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		err := enc.Encode(i)
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}
