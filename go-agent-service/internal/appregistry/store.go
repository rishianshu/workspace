package appregistry

import (
	"context"
	"errors"
)

var (
	ErrNotFound  = errors.New("app registry: not found")
	ErrConflict  = errors.New("app registry: conflict")
	ErrForbidden = errors.New("app registry: forbidden")
)

// Store defines persistence operations for app registry data.
type Store interface {
	// App instances
	UpsertAppInstance(ctx context.Context, instance AppInstance) (*AppInstance, error)
	GetAppInstance(ctx context.Context, id string) (*AppInstance, error)
	FindAppInstance(ctx context.Context, templateID, instanceKey string) (*AppInstance, error)

	// User apps
	UpsertUserApp(ctx context.Context, userApp UserApp) (*UserApp, error)
	GetUserApp(ctx context.Context, id string) (*UserApp, error)
	FindUserApp(ctx context.Context, userID, appInstanceID string) (*UserApp, error)
	ListUserApps(ctx context.Context, userID string) ([]*UserApp, error)

	// Project apps
	UpsertProjectApp(ctx context.Context, projectApp ProjectApp) (*ProjectApp, error)
	GetProjectApp(ctx context.Context, id string) (*ProjectApp, error)
	FindProjectApp(ctx context.Context, projectID, userAppID string) (*ProjectApp, error)
	ListProjectApps(ctx context.Context, projectID string) ([]*ProjectApp, error)
	ListProjectAppsForUser(ctx context.Context, projectID, userID string) ([]*ProjectApp, error)
}
