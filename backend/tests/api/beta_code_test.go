package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/garrettallen/aiboards/backend/internal/handlers"
	"github.com/garrettallen/aiboards/backend/internal/middleware"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupBetaCodeTestRouter(t *testing.T) (*gin.Engine, *utils.TestEnv) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test environment
	env := utils.NewTestEnv(t)

	// Create router
	router := gin.Default()

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(env.AuthService)

	// Create beta code handler
	betaCodeHandler := handlers.NewBetaCodeHandler(env.BetaCodeService)

	// Setup routes
	api := router.Group("/api/v1")
	betaCodeHandler.RegisterRoutes(api, authMiddleware)

	return router, env
}

func TestListBetaCodesEndpoint(t *testing.T) {
	router, env := setupBetaCodeTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token
	regularToken, _ := utils.CreateRegularUserAndGetToken(t, env)

	// Create some beta codes for testing
	for i := 0; i < 5; i++ {
		_, err := env.BetaCodeService.CreateBetaCode(env.Ctx)
		require.NoError(t, err)
	}

	t.Run("Admin user can list beta codes", func(t *testing.T) {
		// Create request
		req, _ := http.NewRequest("GET", "/api/v1/beta-codes", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response structure
		assert.Contains(t, response, "beta_codes")
		assert.Contains(t, response, "total_count")
		assert.Contains(t, response, "page")
		assert.Contains(t, response, "page_size")

		// Verify beta codes list
		betaCodes, ok := response["beta_codes"].([]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(betaCodes), 5)
	})

	t.Run("Regular user cannot list beta codes", func(t *testing.T) {
		// Create request
		req, _ := http.NewRequest("GET", "/api/v1/beta-codes", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", regularToken))

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Unauthenticated user cannot list beta codes", func(t *testing.T) {
		// Create request without token
		req, _ := http.NewRequest("GET", "/api/v1/beta-codes", nil)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Pagination works correctly", func(t *testing.T) {
		// Create request with pagination
		req, _ := http.NewRequest("GET", "/api/v1/beta-codes?page=1&page_size=3", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify pagination
		assert.Equal(t, float64(1), response["page"])
		assert.Equal(t, float64(3), response["page_size"])

		// Verify beta codes list length
		betaCodes, ok := response["beta_codes"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 3, len(betaCodes))
	})
}

func TestCreateBetaCodeEndpoint(t *testing.T) {
	router, env := setupBetaCodeTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token
	regularToken, _ := utils.CreateRegularUserAndGetToken(t, env)

	t.Run("Admin user can create a single beta code", func(t *testing.T) {
		// Create empty request (default to creating one code)
		req, _ := http.NewRequest("POST", "/api/v1/beta-codes", bytes.NewBuffer([]byte{}))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		req.Header.Set("Content-Type", "application/json")

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusCreated, w.Code)

		// Parse response
		var response []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response structure
		assert.Len(t, response, 1)
		assert.Contains(t, response[0], "id")
		assert.Contains(t, response[0], "code")
		assert.Contains(t, response[0], "created_at")
	})

	t.Run("Admin user can create multiple beta codes", func(t *testing.T) {
		// Create request for multiple codes
		requestBody := map[string]interface{}{
			"count": 5,
		}
		requestJSON, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/beta-codes", bytes.NewBuffer(requestJSON))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		req.Header.Set("Content-Type", "application/json")

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusCreated, w.Code)

		// Parse response
		var response []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response structure
		assert.Len(t, response, 5)
		for _, code := range response {
			assert.Contains(t, code, "id")
			assert.Contains(t, code, "code")
			assert.Contains(t, code, "created_at")
		}
	})

	t.Run("Regular user cannot create beta codes", func(t *testing.T) {
		// Create request
		req, _ := http.NewRequest("POST", "/api/v1/beta-codes", bytes.NewBuffer([]byte{}))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", regularToken))
		req.Header.Set("Content-Type", "application/json")

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Unauthenticated user cannot create beta codes", func(t *testing.T) {
		// Create request without token
		req, _ := http.NewRequest("POST", "/api/v1/beta-codes", bytes.NewBuffer([]byte{}))
		req.Header.Set("Content-Type", "application/json")

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid count returns bad request", func(t *testing.T) {
		// Create request with invalid count
		requestBody := map[string]interface{}{
			"count": 0, // Invalid: must be >= 1
		}
		requestJSON, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/beta-codes", bytes.NewBuffer(requestJSON))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		req.Header.Set("Content-Type", "application/json")

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteBetaCodeEndpoint(t *testing.T) {
	router, env := setupBetaCodeTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token
	regularToken, _ := utils.CreateRegularUserAndGetToken(t, env)

	// Create a beta code to delete
	betaCode, err := env.BetaCodeService.CreateBetaCode(env.Ctx)
	require.NoError(t, err)

	t.Run("Admin user can delete a beta code", func(t *testing.T) {
		// Create request
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/beta-codes/%s", betaCode.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify beta code is deleted
		deletedCode, err := env.BetaCodeService.GetBetaCodeByID(env.Ctx, betaCode.ID)
		assert.Error(t, err)
		assert.Nil(t, deletedCode)
	})

	// Create another beta code for the next tests
	anotherBetaCode, err := env.BetaCodeService.CreateBetaCode(env.Ctx)
	require.NoError(t, err)

	t.Run("Regular user cannot delete a beta code", func(t *testing.T) {
		// Create request
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/beta-codes/%s", anotherBetaCode.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", regularToken))

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)

		// Verify beta code still exists
		existingCode, err := env.BetaCodeService.GetBetaCodeByID(env.Ctx, anotherBetaCode.ID)
		assert.NoError(t, err)
		assert.NotNil(t, existingCode)
	})

	t.Run("Unauthenticated user cannot delete a beta code", func(t *testing.T) {
		// Create request without token
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/beta-codes/%s", anotherBetaCode.ID), nil)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Deleting non-existent beta code returns not found", func(t *testing.T) {
		// Create request with random UUID
		randomID := uuid.New()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/beta-codes/%s", randomID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be not found
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid UUID format returns bad request", func(t *testing.T) {
		// Create request with invalid UUID
		req, _ := http.NewRequest("DELETE", "/api/v1/beta-codes/invalid-uuid", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
