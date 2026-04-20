package auth

// tokenResponse is a response from SberDevices' auth server.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}
