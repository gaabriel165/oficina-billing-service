package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/configs"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newTestServer() *api.Server {
	gin.SetMode(gin.TestMode)
	return api.NewServer(&configs.Config{ServerPort: "0", ServiceName: "oficina-billing-service"}, nil)
}

func TestHealth_ShouldReturnOK(t *testing.T) {
	recorder := httptest.NewRecorder()

	newTestServer().Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
}

func TestServer_ShouldAnswerWithRequestID(t *testing.T) {
	recorder := httptest.NewRecorder()

	newTestServer().Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.NotEmpty(t, recorder.Header().Get("X-Request-ID"))
}

func TestServer_ShouldRecoverFromPanicWithInternalError(t *testing.T) {
	recorder := httptest.NewRecorder()

	newTestServer().Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}
