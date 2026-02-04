// Package keystore provides shared request/response types.
package keystore

import "time"

type storeRequest struct {
	OwnerType      string       `json:"owner_type"`
	OwnerID        string       `json:"owner_id"`
	EndpointID     string       `json:"endpoint_id"`
	Credentials    Credentials  `json:"credentials"`
	CredentialType string       `json:"credential_type"`
	Scopes         []string     `json:"scopes,omitempty"`
	ExpiresAt      *time.Time   `json:"expires_at,omitempty"`
}

type storeResponse struct {
	KeyToken string `json:"key_token"`
}

type refreshRequest struct {
	AccessToken string     `json:"access_token"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type credentialResponse struct {
	KeyToken       string      `json:"key_token"`
	OwnerType      string      `json:"owner_type"`
	OwnerID        string      `json:"owner_id"`
	EndpointID     string      `json:"endpoint_id"`
	Credentials    Credentials `json:"credentials"`
	CredentialType string      `json:"credential_type"`
	Scopes         []string    `json:"scopes,omitempty"`
	ExpiresAt      *time.Time  `json:"expires_at,omitempty"`
	RefreshedAt    *time.Time  `json:"refreshed_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

func (c credentialResponse) toStoredCredential() *StoredCredential {
	return &StoredCredential{
		KeyToken:       c.KeyToken,
		OwnerType:      c.OwnerType,
		OwnerID:        c.OwnerID,
		EndpointID:     c.EndpointID,
		Credentials:    c.Credentials,
		CredentialType: c.CredentialType,
		Scopes:         c.Scopes,
		ExpiresAt:      c.ExpiresAt,
		RefreshedAt:    c.RefreshedAt,
		CreatedAt:      c.CreatedAt,
	}
}
