package server

import "context"

type contextKey string

const (
	contextUserIDKey    contextKey = "userId"
	contextProjectIDKey contextKey = "projectId"
)

func withUserProject(ctx context.Context, userID, projectID string) context.Context {
	if userID != "" {
		ctx = context.WithValue(ctx, contextUserIDKey, userID)
	}
	if projectID != "" {
		ctx = context.WithValue(ctx, contextProjectIDKey, projectID)
	}
	return ctx
}

func getUserProject(ctx context.Context) (string, string) {
	var userID string
	var projectID string
	if v := ctx.Value(contextUserIDKey); v != nil {
		if s, ok := v.(string); ok {
			userID = s
		}
	}
	if v := ctx.Value(contextProjectIDKey); v != nil {
		if s, ok := v.(string); ok {
			projectID = s
		}
	}
	return userID, projectID
}
