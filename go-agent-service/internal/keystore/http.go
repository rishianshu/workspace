// Package keystore provides HTTP handlers for the keystore service.
package keystore

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// HTTPServer exposes keystore operations over HTTP.
type HTTPServer struct {
	store  Store
	logger *zap.SugaredLogger
}

// NewHTTPServer creates a new keystore HTTP server.
func NewHTTPServer(store Store, logger *zap.SugaredLogger) *HTTPServer {
	return &HTTPServer{store: store, logger: logger}
}

// Handler returns an http.Handler for keystore routes.
func (s *HTTPServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/credentials", s.handleCredentials)
	mux.HandleFunc("/v1/credentials/", s.handleCredentialByToken)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("OK"))
	})
	return mux
}

func (s *HTTPServer) handleCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req storeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	keyToken, err := s.store.Store(r.Context(), &StoredCredential{
		OwnerType:      req.OwnerType,
		OwnerID:        req.OwnerID,
		EndpointID:     req.EndpointID,
		Credentials:    req.Credentials,
		CredentialType: req.CredentialType,
		Scopes:         req.Scopes,
		ExpiresAt:      req.ExpiresAt,
	})
	if err != nil {
		s.logger.Warnw("Failed to store credentials", "error", err)
		http.Error(w, "Failed to store credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(storeResponse{KeyToken: keyToken})
}

func (s *HTTPServer) handleCredentialByToken(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/credentials/")
	if path == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if strings.HasSuffix(path, "/refresh") {
		s.handleRefresh(w, r, strings.TrimSuffix(path, "/refresh"))
		return
	}

	keyToken := path

	switch r.Method {
	case http.MethodGet:
		cred, err := s.store.Get(r.Context(), keyToken)
		if err == ErrCredentialNotFound {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if err == ErrCredentialExpired {
			http.Error(w, "Expired", http.StatusGone)
			return
		}
		if err != nil {
			http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
			return
		}

		resp := credentialResponse{
			KeyToken:       cred.KeyToken,
			OwnerType:      cred.OwnerType,
			OwnerID:        cred.OwnerID,
			EndpointID:     cred.EndpointID,
			Credentials:    cred.Credentials,
			CredentialType: cred.CredentialType,
			Scopes:         cred.Scopes,
			ExpiresAt:      cred.ExpiresAt,
			RefreshedAt:    cred.RefreshedAt,
			CreatedAt:      cred.CreatedAt,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	case http.MethodDelete:
		err := s.store.Delete(r.Context(), keyToken)
		if err == ErrCredentialNotFound {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to delete credentials", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleRefresh(w http.ResponseWriter, r *http.Request, keyToken string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.AccessToken == "" {
		http.Error(w, "Missing access_token", http.StatusBadRequest)
		return
	}

	if err := s.store.Refresh(r.Context(), keyToken, req.AccessToken, req.ExpiresAt); err != nil {
		if err == ErrCredentialNotFound {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to refresh credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
}

// request/response types live in types.go
