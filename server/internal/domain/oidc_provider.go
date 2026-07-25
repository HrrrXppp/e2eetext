package domain

type OIDCProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Link    string `json:"link"`
	Picture []byte `json:"picture,omitempty"`

	// Scopes is the space-separated OAuth scopes requested for this
	// provider, parsed into individual values (e.g. Google:
	// ["openid","profile"]; Apple: ["openid","name"] — no "email" scope,
	// since we don't keep the user's email address).
	Scopes []string `json:"scopes,omitempty"`
	// ResponseMode is an optional OIDC response_mode auth param (e.g.
	// "form_post", required by Apple whenever name/email scopes are
	// requested). Empty means the param is omitted.
	ResponseMode string `json:"responseMode,omitempty"`
	// ClientSecretStrategy is "static" (read a configured client_secret) or
	// "private_key_jwt" (mint a signed client-authentication JWT per
	// request, RFC 7523 / OpenID Connect Core §9).
	ClientSecretStrategy string `json:"clientSecretStrategy,omitempty"`
}
