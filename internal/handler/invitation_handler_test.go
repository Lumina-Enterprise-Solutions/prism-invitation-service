package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock untuk InvitationService
type MockInvitationService struct {
	mock.Mock
}

func (m *MockInvitationService) CreateInvitation(ctx context.Context, email, role string) (string, error) {
	args := m.Called(ctx, email, role)
	return args.String(0), args.Error(1)
}

func (m *MockInvitationService) ValidateInvitation(ctx context.Context, token string) (*service.InvitationData, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.InvitationData), args.Error(1)
}

func setupTestRouter(handler *InvitationHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	group := router.Group("/invitations")
	group.POST("", handler.CreateInvitation)
	group.POST("/validate", handler.ValidateInvitation)
	return router
}

func TestInvitationHandler_CreateInvitation(t *testing.T) {
	mockService := new(MockInvitationService)
	handler := NewInvitationHandler(mockService)
	router := setupTestRouter(handler)

	t.Run("Success", func(t *testing.T) {
		mockService.On("CreateInvitation", mock.Anything, "test@example.com", "admin").Return("new-token", nil).Once()

		payload := `{"email": "test@example.com", "role": "admin"}`
		req, _ := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.JSONEq(t, `{"message": "undangan berhasil dikirim"}`, rr.Body.String())
		mockService.AssertExpectations(t)
	})

	t.Run("Bad Request - Missing Email", func(t *testing.T) {
		payload := `{"role": "admin"}` // Email tidak ada
		req, _ := http.NewRequest(http.MethodPost, "/invitations", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestInvitationHandler_ValidateInvitation(t *testing.T) {
	mockService := new(MockInvitationService)
	handler := NewInvitationHandler(mockService)
	router := setupTestRouter(handler)

	t.Run("Success", func(t *testing.T) {
		expectedData := &service.InvitationData{Email: "user@example.com", Role: "editor"}
		mockService.On("ValidateInvitation", mock.Anything, "valid-token").Return(expectedData, nil).Once()

		payload := `{"token": "valid-token"}`
		req, _ := http.NewRequest(http.MethodPost, "/invitations/validate", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		expectedJSON, _ := json.Marshal(expectedData)
		assert.JSONEq(t, string(expectedJSON), rr.Body.String())
		mockService.AssertExpectations(t)
	})

	t.Run("Token Not Found", func(t *testing.T) {
		mockService.On("ValidateInvitation", mock.Anything, "invalid-token").Return(nil, errors.New("undangan tidak valid")).Once()

		payload := `{"token": "invalid-token"}`
		req, _ := http.NewRequest(http.MethodPost, "/invitations/validate", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockService.AssertExpectations(t)
	})
}
