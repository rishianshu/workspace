// Package keystore provides remote client implementations.
package keystore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// RemoteStore implements Store over HTTP.
type RemoteStore struct {
	baseURL string
	http    *http.Client
	logger  *zap.SugaredLogger
}

// NewRemoteStore creates a new remote keystore client.
func NewRemoteStore(baseURL string, logger *zap.SugaredLogger) *RemoteStore {
	if baseURL == "" {
		baseURL = "http://localhost:9200"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &RemoteStore{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// Store saves credentials and returns a key token.
func (r *RemoteStore) Store(ctx context.Context, cred *StoredCredential) (string, error) {
	payload := storeRequest{
		OwnerType:      cred.OwnerType,
		OwnerID:        cred.OwnerID,
		EndpointID:     cred.EndpointID,
		Credentials:    cred.Credentials,
		CredentialType: cred.CredentialType,
		Scopes:         cred.Scopes,
		ExpiresAt:      cred.ExpiresAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/v1/credentials", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("keystore store failed: %s", resp.Status)
	}

	var result storeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.KeyToken, nil
}

// Get retrieves credentials by key token.
func (r *RemoteStore) Get(ctx context.Context, keyToken string) (*StoredCredential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/v1/credentials/"+keyToken, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrCredentialNotFound
	}
	if resp.StatusCode == http.StatusGone {
		return nil, ErrCredentialExpired
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("keystore get failed: %s", resp.Status)
	}

	var result credentialResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.toStoredCredential(), nil
}

// Delete removes credentials by key token.
func (r *RemoteStore) Delete(ctx context.Context, keyToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.baseURL+"/v1/credentials/"+keyToken, nil)
	if err != nil {
		return err
	}

	resp, err := r.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrCredentialNotFound
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("keystore delete failed: %s", resp.Status)
	}
	return nil
}

// Refresh updates the access token for OAuth credentials.
func (r *RemoteStore) Refresh(ctx context.Context, keyToken string, newAccessToken string, expiresAt *time.Time) error {
	payload := refreshRequest{
		AccessToken: newAccessToken,
		ExpiresAt:   expiresAt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/v1/credentials/"+keyToken+"/refresh", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrCredentialNotFound
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("keystore refresh failed: %s", resp.Status)
	}
	return nil
}

// request/response types live in types.go
