// Package keystore provides credential storage and management
package keystore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrCredentialNotFound = errors.New("credential not found")
	ErrCredentialExpired  = errors.New("credential has expired")
)

// Credentials holds OAuth/API credentials
type Credentials struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	APIKey       string `json:"api_key,omitempty"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	ExtraFields  map[string]string `json:"extra,omitempty"`
}

// StoredCredential represents a credential entry in the store
type StoredCredential struct {
	KeyToken       string
	OwnerType      string // "user" | "org" | "service"
	OwnerID        string
	EndpointID     string
	Credentials    Credentials
	CredentialType string // "oauth2" | "api_key" | "basic"
	Scopes         []string
	ExpiresAt      *time.Time
	RefreshedAt    *time.Time
	CreatedAt      time.Time
}

// Store interface for credential management
type Store interface {
	// Store saves credentials and returns a key token
	Store(ctx context.Context, cred *StoredCredential) (string, error)
	// Get retrieves credentials by key token
	Get(ctx context.Context, keyToken string) (*StoredCredential, error)
	// Delete removes credentials by key token
	Delete(ctx context.Context, keyToken string) error
	// Refresh updates the access token for OAuth credentials
	Refresh(ctx context.Context, keyToken string, newAccessToken string, expiresAt *time.Time) error
}

// PostgresStore implements Store using PostgreSQL
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed key store
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// generateKeyToken creates a unique token
func generateKeyToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "kt_" + base64.URLEncoding.EncodeToString(bytes), nil
}

// Store saves credentials and returns a key token
func (s *PostgresStore) Store(ctx context.Context, cred *StoredCredential) (string, error) {
	token, err := generateKeyToken()
	if err != nil {
		return "", err
	}

	credJSON, err := json.Marshal(cred.Credentials)
	if err != nil {
		return "", err
	}

	query := `
		INSERT INTO credential_store (
			key_token, owner_type, owner_id, endpoint_id,
			credentials, credential_type, scopes, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = s.db.ExecContext(ctx, query,
		token,
		cred.OwnerType,
		cred.OwnerID,
		cred.EndpointID,
		credJSON,
		cred.CredentialType,
		cred.Scopes,
		cred.ExpiresAt,
	)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Get retrieves credentials by key token
func (s *PostgresStore) Get(ctx context.Context, keyToken string) (*StoredCredential, error) {
	query := `
		SELECT key_token, owner_type, owner_id, endpoint_id,
			   credentials, credential_type, scopes, expires_at, refreshed_at, created_at
		FROM credential_store
		WHERE key_token = $1
	`

	var cred StoredCredential
	var credJSON []byte
	
	err := s.db.QueryRowContext(ctx, query, keyToken).Scan(
		&cred.KeyToken,
		&cred.OwnerType,
		&cred.OwnerID,
		&cred.EndpointID,
		&credJSON,
		&cred.CredentialType,
		&cred.Scopes,
		&cred.ExpiresAt,
		&cred.RefreshedAt,
		&cred.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCredentialNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(credJSON, &cred.Credentials); err != nil {
		return nil, err
	}

	// Check expiration
	if cred.ExpiresAt != nil && time.Now().After(*cred.ExpiresAt) {
		return nil, ErrCredentialExpired
	}

	return &cred, nil
}

// Delete removes credentials by key token
func (s *PostgresStore) Delete(ctx context.Context, keyToken string) error {
	query := `DELETE FROM credential_store WHERE key_token = $1`
	result, err := s.db.ExecContext(ctx, query, keyToken)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

// Refresh updates the access token for OAuth credentials
func (s *PostgresStore) Refresh(ctx context.Context, keyToken string, newAccessToken string, expiresAt *time.Time) error {
	// Get existing credentials
	cred, err := s.Get(ctx, keyToken)
	if err != nil {
		return err
	}

	cred.Credentials.AccessToken = newAccessToken
	credJSON, err := json.Marshal(cred.Credentials)
	if err != nil {
		return err
	}

	query := `
		UPDATE credential_store
		SET credentials = $1, expires_at = $2, refreshed_at = NOW()
		WHERE key_token = $3
	`
	_, err = s.db.ExecContext(ctx, query, credJSON, expiresAt, keyToken)
	return err
}
