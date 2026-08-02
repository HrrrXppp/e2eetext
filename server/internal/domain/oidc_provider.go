package domain

import "net/http"

// ResponseModeFormPost is the OIDC response_mode value that makes an IdP
// deliver its callback via a POST'd form instead of a GET redirect (required
// by Apple whenever name/email scopes are requested). It's the single
// condition under which a provider's callback endpoint accepts POST — see
// AllowedCallbackMethods.
const ResponseModeFormPost = "form_post"

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

// AllowedCallbackMethods returns the HTTP methods this provider's callback
// endpoint accepts. GET is always allowed (the default 302-redirect
// callback); POST is only allowed for providers configured with
// response_mode=form_post, since that's the only way an IdP legitimately
// delivers a callback via POST (OpenID Connect Core §3.1.3.7 / OAuth 2.0
// Multiple Response Type Encoding Practices §2.1).
//
// The router itself can't enforce this per-provider — it registers both
// methods on the shared /api/v1/auth/{provider}/callback path since it
// doesn't know a given provider's response_mode (stored in the DB) until
// the path is parsed at request time. Callers that do know the provider
// (e.g. AuthService.CompleteLogin) should check the actual request method
// against this list themselves.
func (p OIDCProvider) AllowedCallbackMethods() []string {
	methods := []string{http.MethodGet}
	if p.ResponseMode == ResponseModeFormPost {
		methods = append(methods, http.MethodPost)
	}
	return methods
}

// AllowsCallbackMethod reports whether method is one of
// AllowedCallbackMethods for this provider.
func (p OIDCProvider) AllowsCallbackMethod(method string) bool {
	for _, m := range p.AllowedCallbackMethods() {
		if m == method {
			return true
		}
	}
	return false
}
