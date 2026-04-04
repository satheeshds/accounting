package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/satheeshds/portal/db"
)

type registerRequest struct {
	OrgName  string `json:"org_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	TenantID string `json:"tenant_id"`
}

// nexusRegisterResponse is the JSON response from the nexus registration endpoint.
type nexusRegisterResponse struct {
	TenantID string `json:"tenant_id"`
	Error    string `json:"error,omitempty"`
}

// Register handles POST /api/v1/register.
// It provisions a new tenant via nexus-control and then runs portal database
// migrations for all tenants so that the new tenant's DuckDB schema is ready.
//
// @Summary     Register a new tenant
// @Description Provision a new tenant via nexus and initialise the portal schema for it.
// @Tags        tenants
// @Accept      json
// @Produce     json
// @Param       body body registerRequest true "Registration data"
// @Success     201 {object} registerResponse
// @Failure     400 {object} Response
// @Failure     409 {object} Response
// @Failure     500 {object} Response
// @Router      /api/v1/register [post]
func Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OrgName == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "org_name, email, and password are required")
		return
	}

	nexusURL := nexusControlURL()

	// Step 1: Create the tenant in nexus-control.
	tenantID, statusCode, err := createTenantViaNexus(r.Context(), nexusURL, req)
	if err != nil {
		slog.Error("failed to create tenant via nexus", "error", err)
		if statusCode == http.StatusConflict {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "tenant provisioning failed")
		return
	}

	// Step 2: Run portal migrations for all tenants (idempotent; new tenant is included).
	adminKey := os.Getenv("ADMIN_API_KEY")
	if err := db.MigrateViaAPI(r.Context(), nexusURL, adminKey); err != nil {
		// The tenant was created successfully; log the migration failure but do
		// not roll back the registration — migrations can be retried later.
		slog.Error("portal migrations failed after tenant creation", "tenant_id", tenantID, "error", err)
		writeError(w, http.StatusInternalServerError, "tenant created but database initialisation failed")
		return
	}

	slog.Info("tenant registered and portal schema initialised", "tenant_id", tenantID)
	writeJSON(w, http.StatusCreated, registerResponse{TenantID: tenantID})
}

// createTenantViaNexus calls the nexus-control registration endpoint and returns
// the new tenant ID. It also returns the HTTP status code from nexus so that
// callers can map it to the appropriate response code.
func createTenantViaNexus(ctx context.Context, nexusURL string, req registerRequest) (tenantID string, statusCode int, err error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", 0, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, nexusURL+"/api/v1/register", bytes.NewReader(payload))
	if err != nil {
		return "", 0, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("nexus register: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		var nexusResp nexusRegisterResponse
		if err := json.Unmarshal(body, &nexusResp); err == nil && nexusResp.Error != "" {
			return "", resp.StatusCode, fmt.Errorf("nexus: %s", nexusResp.Error)
		}
		return "", resp.StatusCode, fmt.Errorf("nexus returned status %d", resp.StatusCode)
	}

	var nexusResp nexusRegisterResponse
	if err := json.Unmarshal(body, &nexusResp); err != nil {
		return "", resp.StatusCode, fmt.Errorf("decode nexus response: %w", err)
	}
	return nexusResp.TenantID, resp.StatusCode, nil
}

// nexusControlURL returns the nexus-control base URL from the NEXUS_CONTROL_URL
// environment variable, defaulting to http://nexus-control:8080.
func nexusControlURL() string {
	if u := os.Getenv("NEXUS_CONTROL_URL"); u != "" {
		return u
	}
	return "http://nexus-control:8080"
}
