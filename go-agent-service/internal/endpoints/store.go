// Package endpoints handles endpoint replication and user bindings
package endpoints

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEndpointNotFound = errors.New("endpoint not found")
	ErrBindingNotFound  = errors.New("binding not found")
	ErrBindingExists    = errors.New("binding already exists")
)

// Endpoint represents a replicated endpoint from Nucleus
type Endpoint struct {
	ID               string
	NucleusEndpointID string
	ProjectID        *string
	TemplateID       string
	DisplayName      string
	SourceSystem     string
	Capabilities     []string
	Config           map[string]interface{}
	SyncedAt         time.Time
	CreatedAt        time.Time
}

// UserBinding links a user to an endpoint via credentials
type UserBinding struct {
	ID         string
	UserID     string
	EndpointID string
	KeyToken   string
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Store interface for endpoint operations
type Store interface {
	// Endpoints
	UpsertEndpoint(ctx context.Context, ep *Endpoint) error
	GetEndpoint(ctx context.Context, id string) (*Endpoint, error)
	GetEndpointByNucleusID(ctx context.Context, nucleusID string) (*Endpoint, error)
	ListEndpoints(ctx context.Context, projectID *string) ([]*Endpoint, error)
	
	// Bindings
	CreateBinding(ctx context.Context, binding *UserBinding) error
	GetBinding(ctx context.Context, userID, endpointID string) (*UserBinding, error)
	ListUserBindings(ctx context.Context, userID string) ([]*UserBinding, error)
	DeleteBinding(ctx context.Context, userID, endpointID string) error
}

// PostgresStore implements Store using PostgreSQL
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed endpoint store
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// UpsertEndpoint inserts or updates an endpoint
func (s *PostgresStore) UpsertEndpoint(ctx context.Context, ep *Endpoint) error {
	query := `
		INSERT INTO endpoints (
			id, nucleus_endpoint_id, project_id, template_id, 
			display_name, source_system, capabilities, config, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (nucleus_endpoint_id) DO UPDATE SET
			project_id = EXCLUDED.project_id,
			template_id = EXCLUDED.template_id,
			display_name = EXCLUDED.display_name,
			source_system = EXCLUDED.source_system,
			capabilities = EXCLUDED.capabilities,
			config = EXCLUDED.config,
			synced_at = NOW()
	`

	if ep.ID == "" {
		ep.ID = uuid.New().String()
	}

	_, err := s.db.ExecContext(ctx, query,
		ep.ID,
		ep.NucleusEndpointID,
		ep.ProjectID,
		ep.TemplateID,
		ep.DisplayName,
		ep.SourceSystem,
		ep.Capabilities,
		ep.Config,
	)
	return err
}

// GetEndpoint retrieves an endpoint by ID
func (s *PostgresStore) GetEndpoint(ctx context.Context, id string) (*Endpoint, error) {
	query := `
		SELECT id, nucleus_endpoint_id, project_id, template_id,
			   display_name, source_system, capabilities, config, synced_at, created_at
		FROM endpoints WHERE id = $1
	`
	return s.scanEndpoint(ctx, query, id)
}

// GetEndpointByNucleusID retrieves an endpoint by Nucleus endpoint ID
func (s *PostgresStore) GetEndpointByNucleusID(ctx context.Context, nucleusID string) (*Endpoint, error) {
	query := `
		SELECT id, nucleus_endpoint_id, project_id, template_id,
			   display_name, source_system, capabilities, config, synced_at, created_at
		FROM endpoints WHERE nucleus_endpoint_id = $1
	`
	return s.scanEndpoint(ctx, query, nucleusID)
}

func (s *PostgresStore) scanEndpoint(ctx context.Context, query, arg string) (*Endpoint, error) {
	var ep Endpoint
	err := s.db.QueryRowContext(ctx, query, arg).Scan(
		&ep.ID,
		&ep.NucleusEndpointID,
		&ep.ProjectID,
		&ep.TemplateID,
		&ep.DisplayName,
		&ep.SourceSystem,
		&ep.Capabilities,
		&ep.Config,
		&ep.SyncedAt,
		&ep.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEndpointNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ep, nil
}

// ListEndpoints returns all endpoints, optionally filtered by project
func (s *PostgresStore) ListEndpoints(ctx context.Context, projectID *string) ([]*Endpoint, error) {
	var query string
	var args []interface{}

	if projectID != nil {
		query = `
			SELECT id, nucleus_endpoint_id, project_id, template_id,
				   display_name, source_system, capabilities, config, synced_at, created_at
			FROM endpoints WHERE project_id = $1 ORDER BY display_name
		`
		args = []interface{}{*projectID}
	} else {
		query = `
			SELECT id, nucleus_endpoint_id, project_id, template_id,
				   display_name, source_system, capabilities, config, synced_at, created_at
			FROM endpoints ORDER BY display_name
		`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []*Endpoint
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(
			&ep.ID,
			&ep.NucleusEndpointID,
			&ep.ProjectID,
			&ep.TemplateID,
			&ep.DisplayName,
			&ep.SourceSystem,
			&ep.Capabilities,
			&ep.Config,
			&ep.SyncedAt,
			&ep.CreatedAt,
		); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, &ep)
	}
	return endpoints, rows.Err()
}

// CreateBinding creates a new user-endpoint binding
func (s *PostgresStore) CreateBinding(ctx context.Context, binding *UserBinding) error {
	if binding.ID == "" {
		binding.ID = uuid.New().String()
	}

	query := `
		INSERT INTO user_endpoint_bindings (id, user_id, endpoint_id, key_token, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.db.ExecContext(ctx, query,
		binding.ID,
		binding.UserID,
		binding.EndpointID,
		binding.KeyToken,
		binding.IsActive,
	)
	return err
}

// GetBinding retrieves a binding for a user-endpoint pair
func (s *PostgresStore) GetBinding(ctx context.Context, userID, endpointID string) (*UserBinding, error) {
	query := `
		SELECT id, user_id, endpoint_id, key_token, is_active, created_at, updated_at
		FROM user_endpoint_bindings
		WHERE user_id = $1 AND endpoint_id = $2
	`

	var b UserBinding
	err := s.db.QueryRowContext(ctx, query, userID, endpointID).Scan(
		&b.ID,
		&b.UserID,
		&b.EndpointID,
		&b.KeyToken,
		&b.IsActive,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBindingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// ListUserBindings returns all active bindings for a user
func (s *PostgresStore) ListUserBindings(ctx context.Context, userID string) ([]*UserBinding, error) {
	query := `
		SELECT id, user_id, endpoint_id, key_token, is_active, created_at, updated_at
		FROM user_endpoint_bindings
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []*UserBinding
	for rows.Next() {
		var b UserBinding
		if err := rows.Scan(
			&b.ID,
			&b.UserID,
			&b.EndpointID,
			&b.KeyToken,
			&b.IsActive,
			&b.CreatedAt,
			&b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bindings = append(bindings, &b)
	}
	return bindings, rows.Err()
}

// DeleteBinding soft-deletes a binding (sets is_active = false)
func (s *PostgresStore) DeleteBinding(ctx context.Context, userID, endpointID string) error {
	query := `
		UPDATE user_endpoint_bindings
		SET is_active = FALSE, updated_at = NOW()
		WHERE user_id = $1 AND endpoint_id = $2
	`
	result, err := s.db.ExecContext(ctx, query, userID, endpointID)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrBindingNotFound
	}
	return nil
}
