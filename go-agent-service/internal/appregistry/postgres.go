package appregistry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed app registry store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) UpsertAppInstance(ctx context.Context, instance AppInstance) (*AppInstance, error) {
	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}
	configBytes, err := marshalConfig(instance.Config)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO app_instances (
			id, template_id, instance_key, display_name, config
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (template_id, instance_key) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			config = EXCLUDED.config,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	if err := s.db.QueryRowContext(ctx, query,
		instance.ID,
		instance.TemplateID,
		instance.InstanceKey,
		instance.DisplayName,
		configBytes,
	).Scan(&instance.ID, &instance.CreatedAt, &instance.UpdatedAt); err != nil {
		return nil, err
	}

	return &instance, nil
}

func (s *PostgresStore) GetAppInstance(ctx context.Context, id string) (*AppInstance, error) {
	query := `
		SELECT id, template_id, instance_key, display_name, config, created_at, updated_at
		FROM app_instances WHERE id = $1
	`
	return s.scanAppInstance(ctx, query, id)
}

func (s *PostgresStore) FindAppInstance(ctx context.Context, templateID, instanceKey string) (*AppInstance, error) {
	query := `
		SELECT id, template_id, instance_key, display_name, config, created_at, updated_at
		FROM app_instances WHERE template_id = $1 AND instance_key = $2
	`
	return s.scanAppInstance(ctx, query, templateID, instanceKey)
}

func (s *PostgresStore) scanAppInstance(ctx context.Context, query string, args ...any) (*AppInstance, error) {
	var instance AppInstance
	var configBytes []byte
	var createdAt time.Time
	var updatedAt time.Time
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&instance.ID,
		&instance.TemplateID,
		&instance.InstanceKey,
		&instance.DisplayName,
		&configBytes,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(configBytes) > 0 {
		_ = json.Unmarshal(configBytes, &instance.Config)
	}
	instance.CreatedAt = createdAt
	instance.UpdatedAt = updatedAt
	return &instance, nil
}

func (s *PostgresStore) UpsertUserApp(ctx context.Context, userApp UserApp) (*UserApp, error) {
	if userApp.ID == "" {
		userApp.ID = uuid.New().String()
	}
	query := `
		INSERT INTO user_apps (
			id, user_id, app_instance_id, credential_ref
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, app_instance_id) DO UPDATE SET
			credential_ref = EXCLUDED.credential_ref,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	if err := s.db.QueryRowContext(ctx, query,
		userApp.ID,
		userApp.UserID,
		userApp.AppInstanceID,
		userApp.CredentialRef,
	).Scan(&userApp.ID, &userApp.CreatedAt, &userApp.UpdatedAt); err != nil {
		return nil, err
	}
	return &userApp, nil
}

func (s *PostgresStore) GetUserApp(ctx context.Context, id string) (*UserApp, error) {
	query := `
		SELECT id, user_id, app_instance_id, credential_ref, created_at, updated_at
		FROM user_apps WHERE id = $1
	`
	return s.scanUserApp(ctx, query, id)
}

func (s *PostgresStore) FindUserApp(ctx context.Context, userID, appInstanceID string) (*UserApp, error) {
	query := `
		SELECT id, user_id, app_instance_id, credential_ref, created_at, updated_at
		FROM user_apps WHERE user_id = $1 AND app_instance_id = $2
	`
	return s.scanUserApp(ctx, query, userID, appInstanceID)
}

func (s *PostgresStore) ListUserApps(ctx context.Context, userID string) ([]*UserApp, error) {
	query := `
		SELECT id, user_id, app_instance_id, credential_ref, created_at, updated_at
		FROM user_apps WHERE user_id = $1 ORDER BY created_at
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*UserApp
	for rows.Next() {
		var app UserApp
		if err := rows.Scan(
			&app.ID,
			&app.UserID,
			&app.AppInstanceID,
			&app.CredentialRef,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, &app)
	}
	return results, rows.Err()
}

func (s *PostgresStore) scanUserApp(ctx context.Context, query string, args ...any) (*UserApp, error) {
	var app UserApp
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&app.ID,
		&app.UserID,
		&app.AppInstanceID,
		&app.CredentialRef,
		&app.CreatedAt,
		&app.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (s *PostgresStore) UpsertProjectApp(ctx context.Context, projectApp ProjectApp) (*ProjectApp, error) {
	if projectApp.ID == "" {
		projectApp.ID = uuid.New().String()
	}
	query := `
		INSERT INTO project_apps (
			id, project_id, user_app_id, endpoint_id, alias, is_default
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (project_id, user_app_id) DO UPDATE SET
			endpoint_id = EXCLUDED.endpoint_id,
			alias = EXCLUDED.alias,
			is_default = EXCLUDED.is_default,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	if err := s.db.QueryRowContext(ctx, query,
		projectApp.ID,
		projectApp.ProjectID,
		projectApp.UserAppID,
		projectApp.EndpointID,
		projectApp.Alias,
		projectApp.IsDefault,
	).Scan(&projectApp.ID, &projectApp.CreatedAt, &projectApp.UpdatedAt); err != nil {
		return nil, err
	}
	return &projectApp, nil
}

func (s *PostgresStore) GetProjectApp(ctx context.Context, id string) (*ProjectApp, error) {
	query := `
		SELECT id, project_id, user_app_id, endpoint_id, alias, is_default, created_at, updated_at
		FROM project_apps WHERE id = $1
	`
	return s.scanProjectApp(ctx, query, id)
}

func (s *PostgresStore) FindProjectApp(ctx context.Context, projectID, userAppID string) (*ProjectApp, error) {
	query := `
		SELECT id, project_id, user_app_id, endpoint_id, alias, is_default, created_at, updated_at
		FROM project_apps WHERE project_id = $1 AND user_app_id = $2
	`
	return s.scanProjectApp(ctx, query, projectID, userAppID)
}

func (s *PostgresStore) ListProjectApps(ctx context.Context, projectID string) ([]*ProjectApp, error) {
	query := `
		SELECT id, project_id, user_app_id, endpoint_id, alias, is_default, created_at, updated_at
		FROM project_apps WHERE project_id = $1 ORDER BY created_at
	`
	return s.scanProjectApps(ctx, query, projectID)
}

func (s *PostgresStore) ListProjectAppsForUser(ctx context.Context, projectID, userID string) ([]*ProjectApp, error) {
	query := `
		SELECT pa.id, pa.project_id, pa.user_app_id, pa.endpoint_id, pa.alias, pa.is_default, pa.created_at, pa.updated_at
		FROM project_apps pa
		JOIN user_apps ua ON ua.id = pa.user_app_id
		WHERE pa.project_id = $1 AND ua.user_id = $2
		ORDER BY pa.created_at
	`
	return s.scanProjectApps(ctx, query, projectID, userID)
}

func (s *PostgresStore) scanProjectApp(ctx context.Context, query string, args ...any) (*ProjectApp, error) {
	var app ProjectApp
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&app.ID,
		&app.ProjectID,
		&app.UserAppID,
		&app.EndpointID,
		&app.Alias,
		&app.IsDefault,
		&app.CreatedAt,
		&app.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (s *PostgresStore) scanProjectApps(ctx context.Context, query string, args ...any) ([]*ProjectApp, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*ProjectApp
	for rows.Next() {
		var app ProjectApp
		if err := rows.Scan(
			&app.ID,
			&app.ProjectID,
			&app.UserAppID,
			&app.EndpointID,
			&app.Alias,
			&app.IsDefault,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		apps = append(apps, &app)
	}
	return apps, rows.Err()
}

func marshalConfig(config map[string]any) ([]byte, error) {
	if config == nil {
		return []byte("null"), nil
	}
	return json.Marshal(config)
}
