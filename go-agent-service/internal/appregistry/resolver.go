package appregistry

import (
	"context"
	"fmt"

	"github.com/antigravity/go-agent-service/internal/keystore"
	"github.com/antigravity/go-agent-service/internal/nucleus"
)

// ResolvedApp is the resolved execution context for a user app.
type ResolvedApp struct {
	AppID            string
	UserID           string
	ProjectID        string
	EndpointID       string
	TemplateID       string
	CredentialRef    string
	AppInstance      *AppInstance
	Endpoint         *nucleus.MetadataEndpoint
	DelegatedEnabled bool
}

// Resolver resolves app registry entries into endpoint + credential context.
type Resolver struct {
	Registry Store
	Nucleus  *nucleus.Client
	KeyStore keystore.Store
}

// ResolveApp resolves a user app within a project into endpoint context.
func (r *Resolver) ResolveApp(ctx context.Context, userID, projectID, appID string) (*ResolvedApp, error) {
	if r.Registry == nil {
		return nil, fmt.Errorf("app registry unavailable")
	}

	userApp, err := r.Registry.GetUserApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	if userApp.UserID != userID {
		return nil, ErrForbidden
	}

	projectApp, err := r.Registry.FindProjectApp(ctx, projectID, userApp.ID)
	if err != nil {
		return nil, err
	}

	instance, err := r.Registry.GetAppInstance(ctx, userApp.AppInstanceID)
	if err != nil {
		return nil, err
	}

	resolved := &ResolvedApp{
		AppID:         userApp.ID,
		UserID:        userID,
		ProjectID:     projectID,
		EndpointID:    projectApp.EndpointID,
		TemplateID:    instance.TemplateID,
		CredentialRef: userApp.CredentialRef,
		AppInstance:   instance,
	}

	if r.Nucleus != nil && projectApp.EndpointID != "" {
		endpoint, err := r.Nucleus.GetEndpoint(ctx, projectApp.EndpointID)
		if err != nil {
			return nil, err
		}
		resolved.Endpoint = endpoint
		if endpoint != nil {
			if endpoint.TemplateID != "" {
				resolved.TemplateID = endpoint.TemplateID
			}
			resolved.DelegatedEnabled = endpoint.DelegatedConnected
		}
	}

	return resolved, nil
}

// ResolveProjectApps resolves all app bindings for a user in a project.
func (r *Resolver) ResolveProjectApps(ctx context.Context, userID, projectID string) ([]*ResolvedApp, error) {
	if r.Registry == nil {
		return nil, fmt.Errorf("app registry unavailable")
	}
	projectApps, err := r.Registry.ListProjectAppsForUser(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	resolved := make([]*ResolvedApp, 0, len(projectApps))
	for _, projectApp := range projectApps {
		userApp, err := r.Registry.GetUserApp(ctx, projectApp.UserAppID)
		if err != nil {
			return nil, err
		}
		if userApp.UserID != userID {
			return nil, ErrForbidden
		}
		instance, err := r.Registry.GetAppInstance(ctx, userApp.AppInstanceID)
		if err != nil {
			return nil, err
		}
		entry := &ResolvedApp{
			AppID:         userApp.ID,
			UserID:        userID,
			ProjectID:     projectID,
			EndpointID:    projectApp.EndpointID,
			TemplateID:    instance.TemplateID,
			CredentialRef: userApp.CredentialRef,
			AppInstance:   instance,
		}
		if r.Nucleus != nil && projectApp.EndpointID != "" {
			endpoint, err := r.Nucleus.GetEndpoint(ctx, projectApp.EndpointID)
			if err != nil {
				return nil, err
			}
			entry.Endpoint = endpoint
			if endpoint != nil {
				if endpoint.TemplateID != "" {
					entry.TemplateID = endpoint.TemplateID
				}
				entry.DelegatedEnabled = endpoint.DelegatedConnected
			}
		}
		resolved = append(resolved, entry)
	}

	return resolved, nil
}
