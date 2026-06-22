package domain

type OIDCProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Link    string `json:"link"`
	Picture []byte `json:"picture,omitempty"`
}
