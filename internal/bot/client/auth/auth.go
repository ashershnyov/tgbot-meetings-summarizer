package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	authEndpoint = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"

	refreshAhead = 30 * time.Second
)

// Client defines Sber auth client suitable for fetching SaluteSpeech and GigaChat access tokens.
type Client struct {
	http        *http.Client
	mu          *sync.Mutex
	authKey     string
	scope       string
	accessToken string
	expiry      time.Time
}

// NewClient returns a newly created Sber auth client.
func NewClient(authKey, scope string) (*Client, error) {
	if authKey == "" {
		return nil, fmt.Errorf("no auth key was passed to auth client")
	}
	if scope == "" {
		return nil, fmt.Errorf("no scope was passed to auth client")
	}

	c := &Client{
		authKey: authKey,
		scope:   scope,
		http:    &http.Client{Timeout: 10 * time.Second},
		mu:      &sync.Mutex{},
	}

	if err := c.refresh(); err != nil {
		return nil, fmt.Errorf("error refreshing token for scope %s: %w", scope, err)
	}
	return c, nil
}

// Token returns an access token. Refreshes token upon its expiration. Can be called concurrently.
func (c *Client) Token() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Now().Add(refreshAhead).Before(c.expiry) {
		return c.accessToken, nil
	}
	if err := c.refresh(); err != nil {
		return "", err
	}
	return c.accessToken, nil
}

func (c *Client) refresh() error {
	form := url.Values{"scope": {c.scope}}

	req, err := http.NewRequest(http.MethodPost, authEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+c.authKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rqUID := uuid.New()
	req.Header.Set("RqUID", rqUID.String())

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error executing http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error getting token with HTTP code %d: %s", resp.StatusCode, string(raw))
	}

	var tr tokenResponse
	if err = json.Unmarshal(raw, &tr); err != nil {
		return fmt.Errorf("error parsing token: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("empty access_token was returned")
	}

	c.accessToken = tr.AccessToken
	c.expiry = time.UnixMilli(tr.ExpiresAt)
	return nil
}
