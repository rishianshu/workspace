package nucleus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestListProjectsUsesBearerToken(t *testing.T) {
	logger := zap.NewNop().Sugar()
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"metadataProjects":[{"id":"p1","slug":"s1","displayName":"P1","description":"d"}]}}`))
	}))
	defer srv.Close()

	client := NewClientWithConfig(ClientConfig{
		APIURL:      srv.URL,
		BearerToken: "test-bearer",
	}, logger)

	_, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects error: %v", err)
	}
	if gotAuth != "Bearer test-bearer" {
		t.Fatalf("expected bearer auth, got %q", gotAuth)
	}
}

func TestListProjectsUsesKeycloakTokenWhenNoBearer(t *testing.T) {
	logger := zap.NewNop().Sugar()
	var graphqlAuth string
	var tokenCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/nucleus/protocol/openid-connect/token":
			tokenCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "kc-token",
				"expires_in":   120,
			})
		case "/graphql":
			graphqlAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"metadataProjects":[{"id":"p1","slug":"s1","displayName":"P1","description":"d"}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClientWithConfig(ClientConfig{
		APIURL:           srv.URL,
		KeycloakURL:      srv.URL,
		KeycloakRealm:    "nucleus",
		KeycloakClientID: "client",
		KeycloakUsername: "user",
		KeycloakPassword: "pass",
	}, logger)

	_, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects error: %v", err)
	}
	if tokenCalls == 0 {
		t.Fatalf("expected keycloak token fetch")
	}
	if graphqlAuth != "Bearer kc-token" {
		t.Fatalf("expected bearer auth from keycloak, got %q", graphqlAuth)
	}
}

func TestListProjectsFallsBackToBasicAuth(t *testing.T) {
	logger := zap.NewNop().Sugar()
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"metadataProjects":[{"id":"p1","slug":"s1","displayName":"P1","description":"d"}]}}`))
	}))
	defer srv.Close()

	client := NewClientWithConfig(ClientConfig{
		APIURL:   srv.URL,
		Username: "user",
		Password: "pass",
	}, logger)

	_, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects error: %v", err)
	}

	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
	if gotAuth != expected {
		t.Fatalf("expected basic auth %q, got %q", expected, gotAuth)
	}
}
