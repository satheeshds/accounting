package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubNexusServer builds a minimal httptest.Server that simulates the nexus-control
// /api/v1/register and /api/v1/admin/query endpoints.
//
//   - nexusErr: when non-empty the register endpoint returns 500 with this error.
//   - nexusConflict: when true the register endpoint returns 409 (email already registered).
//   - adminQueryStatus: the HTTP status code the admin/query endpoint should return.
func stubNexusServer(t *testing.T, nexusErr string, nexusConflict bool, adminQueryStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if nexusConflict {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"email already registered"}`))
			return
		}
		if nexusErr != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"` + nexusErr + `"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"tenant_id":"acme_12345678"}`))
	})

	mux.HandleFunc("/api/v1/admin/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(adminQueryStatus)
		if adminQueryStatus == http.StatusOK {
			_, _ = w.Write([]byte(`{"results":[]}`))
		} else {
			_, _ = w.Write([]byte(`{"error":"admin query failed"}`))
		}
	})

	return httptest.NewServer(mux)
}

func postRegister(t *testing.T, url string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	http.HandlerFunc(Register).ServeHTTP(rec, req)
	return rec
}

func TestRegister_Success(t *testing.T) {
	nexus := stubNexusServer(t, "", false, http.StatusOK)
	defer nexus.Close()

	t.Setenv("NEXUS_CONTROL_URL", nexus.URL)
	t.Setenv("ADMIN_API_KEY", "test-admin-key")

	rec := postRegister(t, "/api/v1/register", map[string]string{
		"org_name": "Acme Corp",
		"email":    "admin@acme.com",
		"password": "secret123",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			TenantID string `json:"tenant_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.TenantID == "" {
		t.Error("expected non-empty tenant_id in response")
	}
}

func TestRegister_MissingFields(t *testing.T) {
	cases := []map[string]string{
		{"email": "a@b.com", "password": "pw"},           // missing org_name
		{"org_name": "Acme", "password": "pw"},            // missing email
		{"org_name": "Acme", "email": "a@b.com"},          // missing password
		{},                                                // all missing
	}

	for _, body := range cases {
		rec := postRegister(t, "/api/v1/register", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %v: expected 400, got %d", body, rec.Code)
		}
	}
}

func TestRegister_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	http.HandlerFunc(Register).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRegister_NexusConflict(t *testing.T) {
	nexus := stubNexusServer(t, "", true, http.StatusOK)
	defer nexus.Close()

	t.Setenv("NEXUS_CONTROL_URL", nexus.URL)
	t.Setenv("ADMIN_API_KEY", "test-admin-key")

	rec := postRegister(t, "/api/v1/register", map[string]string{
		"org_name": "Acme Corp",
		"email":    "existing@acme.com",
		"password": "secret123",
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegister_NexusError(t *testing.T) {
	nexus := stubNexusServer(t, "provisioning failed", false, http.StatusOK)
	defer nexus.Close()

	t.Setenv("NEXUS_CONTROL_URL", nexus.URL)
	t.Setenv("ADMIN_API_KEY", "test-admin-key")

	rec := postRegister(t, "/api/v1/register", map[string]string{
		"org_name": "Acme Corp",
		"email":    "admin@acme.com",
		"password": "secret123",
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegister_MigrationError(t *testing.T) {
	nexus := stubNexusServer(t, "", false, http.StatusInternalServerError)
	defer nexus.Close()

	t.Setenv("NEXUS_CONTROL_URL", nexus.URL)
	t.Setenv("ADMIN_API_KEY", "test-admin-key")

	rec := postRegister(t, "/api/v1/register", map[string]string{
		"org_name": "Acme Corp",
		"email":    "admin@acme.com",
		"password": "secret123",
	})

	// Tenant was created but migration failed — we return 500 to the client.
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when migration fails, got %d: %s", rec.Code, rec.Body.String())
	}
}
