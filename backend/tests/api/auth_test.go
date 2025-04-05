package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/garrettallen/aiboards/backend/internal/handlers"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *utils.TestEnv) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test environment
	env := utils.NewTestEnv(t)

	// Create router
	router := gin.Default()

	// Create auth handler
	authHandler := handlers.NewAuthHandler(env.AuthService)

	// Setup routes
	router.POST("/api/auth/register", authHandler.Register)
	router.POST("/api/auth/login", authHandler.Login)
	router.POST("/api/auth/refresh", authHandler.RefreshToken)

	return router, env
}

func TestRegisterEndpoint(t *testing.T) {
	router, env := setupTestRouter(t)
	defer env.Cleanup()

	// Create a test beta code
	betaCode := createTestBetaCode(t, env)

	// Test data
	requestBody := map[string]string{
		"email":     "test@example.com",
		"password":  "password123",
		"name":      "Test User",
		"beta_code": betaCode,
	}
	jsonData, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response data
	assert.Contains(t, response, "user")
	assert.Contains(t, response, "token")

	user := response["user"].(map[string]interface{})
	token := response["token"].(map[string]interface{})

	assert.Equal(t, "test@example.com", user["email"])
	assert.Equal(t, "Test User", user["name"])
	assert.NotEmpty(t, token["access_token"])
}

func TestLoginEndpoint(t *testing.T) {
	router, env := setupTestRouter(t)
	defer env.Cleanup()

	// Create a test user
	email := "login-test@example.com"
	password := "password123"
	createTestUser(t, env, email, password)

	// Test data
	requestBody := map[string]string{
		"email":    email,
		"password": password,
	}
	jsonData, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response data
	assert.Contains(t, response, "user")
	assert.Contains(t, response, "token")

	user := response["user"].(map[string]interface{})
	token := response["token"].(map[string]interface{})

	assert.Equal(t, email, user["email"])
	assert.NotEmpty(t, token["access_token"])
}

// Helper functions
func createTestBetaCode(t *testing.T, env *utils.TestEnv) string {
	// Generate a code that's 16 characters or less to fit the VARCHAR(16) constraint
	code := "T" + time.Now().Format("0102150405")
	betaCode := &models.BetaCode{
		ID:        uuid.New(),
		Code:      code,
		IsUsed:    false,
		CreatedAt: time.Now(),
	}

	err := env.BetaCodeRepository.Create(env.Ctx, betaCode)
	assert.NoError(t, err)

	return code
}

func createTestUser(t *testing.T, env *utils.TestEnv, email, password string) *models.User {
	user, err := models.NewUser(email, password, "Test User")
	assert.NoError(t, err)

	err = env.UserRepository.Create(env.Ctx, user)
	assert.NoError(t, err)

	return user
}
