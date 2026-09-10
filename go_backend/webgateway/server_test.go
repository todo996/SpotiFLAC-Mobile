package webgateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeBackend struct {
	searchPayload   json.RawMessage
	resolvePayload  json.RawMessage
	providerPayload json.RawMessage
	searchErr       error
	resolveErr      error
	providerErr     error
	providerCount   int
	lastQuery       string
	lastLimit       int
	lastProvider    string
	lastTrack       string
	lastQuality     string
}

func (f *fakeBackend) Search(query string, limit int) (json.RawMessage, error) {
	f.lastQuery, f.lastLimit = query, limit
	return f.searchPayload, f.searchErr
}

func (f *fakeBackend) ResolveStream(providerID, trackID, quality string, _ map[string]any) (json.RawMessage, error) {
	f.lastProvider, f.lastTrack, f.lastQuality = providerID, trackID, quality
	return f.resolvePayload, f.resolveErr
}

func (f *fakeBackend) Providers() (json.RawMessage, error) {
	return f.providerPayload, f.providerErr
}

func (f *fakeBackend) ProviderCount() int { return f.providerCount }

func TestHealthRequiresConfiguredBearerToken(t *testing.T) {
	backend := &fakeBackend{providerCount: 3}
	handler := NewHandler(backend, "secret")

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/health", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthorized.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var payload struct {
		Ready         bool   `json:"ready"`
		Readiness     string `json:"readiness"`
		ProviderCount int    `json:"providerCount"`
		Auth          struct {
			Required      bool `json:"required"`
			Authenticated bool `json:"authenticated"`
		} `json:"auth"`
		Capabilities struct {
			Providers bool `json:"providers"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if !payload.Ready || payload.Readiness != "ready" || payload.ProviderCount != 3 {
		t.Fatalf("unexpected readiness response: %s", response.Body.String())
	}
	if !payload.Auth.Required || !payload.Auth.Authenticated || !payload.Capabilities.Providers {
		t.Fatalf("unexpected auth/capability response: %s", response.Body.String())
	}
}

func TestHealthReportsNoProvidersWithoutExposingCredentials(t *testing.T) {
	backend := &fakeBackend{providerCount: 0}
	handler := NewHandler(backend, "")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"readiness":"no_providers"`) || !strings.Contains(body, `"ready":false`) {
		t.Fatalf("unexpected no-provider health response: %s", body)
	}
	if !strings.Contains(body, `"required":false`) || strings.Contains(body, "secret") || strings.Contains(body, "token") {
		t.Fatalf("health response exposed or misreported auth state: %s", body)
	}
}

func TestProvidersReturnsSanitizedInventoryPayload(t *testing.T) {
	backend := &fakeBackend{providerPayload: json.RawMessage(`{"providers":[{"id":"qobuz-web","displayName":"Qobuz Web","status":"ready","configuration":{"settings":[{"key":"token","secret":true,"configured":true}]}}],"count":1}`)}
	handler := NewHandler(backend, "")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/providers", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"qobuz-web"`) || !strings.Contains(body, `"configured":true`) {
		t.Fatalf("unexpected provider response: %s", body)
	}
	if strings.Contains(body, "access_token") || strings.Contains(body, "refresh_token") || strings.Contains(body, "secret-value") {
		t.Fatalf("provider response exposed credentials: %s", body)
	}
}

func TestSearchValidatesAndCapsLimit(t *testing.T) {
	backend := &fakeBackend{searchPayload: json.RawMessage(`{"tracks":[]}`)}
	handler := NewHandler(backend, "")

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/search", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing query, got %d", missing.Code)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/search?q=Daft+Punk&limit=500", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if backend.lastQuery != "Daft Punk" || backend.lastLimit != 50 {
		t.Fatalf("unexpected search forwarding: query=%q limit=%d", backend.lastQuery, backend.lastLimit)
	}
}

func TestResolveForwardsProviderTrackAndQuality(t *testing.T) {
	backend := &fakeBackend{resolvePayload: json.RawMessage(`{"success":true,"url":"https://cdn.example/song.flac"}`)}
	handler := NewHandler(backend, "")
	request := httptest.NewRequest(http.MethodPost, "/resolve", strings.NewReader(`{"providerId":"tidal","trackId":"42","quality":"LOSSLESS"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if backend.lastProvider != "tidal" || backend.lastTrack != "42" || backend.lastQuality != "LOSSLESS" {
		t.Fatalf("resolve request was not forwarded correctly")
	}
}

func TestBackendErrorsBecomeBadGateway(t *testing.T) {
	backend := &fakeBackend{searchErr: errors.New("provider unavailable")}
	handler := NewHandler(backend, "")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/search?q=test", nil))
	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", response.Code)
	}

	providerBackend := &fakeBackend{providerErr: errors.New("inventory unavailable")}
	providerHandler := NewHandler(providerBackend, "")
	providerResponse := httptest.NewRecorder()
	providerHandler.ServeHTTP(providerResponse, httptest.NewRequest(http.MethodGet, "/providers", nil))
	if providerResponse.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for provider inventory, got %d", providerResponse.Code)
	}
}
