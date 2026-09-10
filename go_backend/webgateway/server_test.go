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
	searchPayload  json.RawMessage
	resolvePayload json.RawMessage
	searchErr      error
	resolveErr     error
	providerCount  int
	lastQuery      string
	lastLimit      int
	lastProvider   string
	lastTrack      string
	lastQuality    string
}

func (f *fakeBackend) Search(query string, limit int) (json.RawMessage, error) {
	f.lastQuery, f.lastLimit = query, limit
	return f.searchPayload, f.searchErr
}

func (f *fakeBackend) ResolveStream(providerID, trackID, quality string, _ map[string]any) (json.RawMessage, error) {
	f.lastProvider, f.lastTrack, f.lastQuality = providerID, trackID, quality
	return f.resolvePayload, f.resolveErr
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
	if !strings.Contains(response.Body.String(), `"providerCount":3`) {
		t.Fatalf("unexpected health response: %s", response.Body.String())
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
}
