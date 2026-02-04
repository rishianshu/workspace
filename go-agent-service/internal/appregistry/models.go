package appregistry

import "time"

// AppInstance represents a shared app configuration identity (non-secret).
type AppInstance struct {
	ID          string
	TemplateID  string
	InstanceKey string
	DisplayName string
	Config      map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UserApp binds a user to an app instance and credential reference.
type UserApp struct {
	ID            string
	UserID        string
	AppInstanceID string
	CredentialRef string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ProjectApp links a user app to a project and Nucleus endpoint.
type ProjectApp struct {
	ID         string
	ProjectID  string
	UserAppID  string
	EndpointID string
	Alias      string
	IsDefault  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
