package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/client"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock untuk Dependensi ---

// Mock untuk NotificationClient
type MockNotificationClient struct {
	mock.Mock
}

func (m *MockNotificationClient) SendInvitationEmail(ctx context.Context, email, invitationLink string) {
	m.Called(ctx, email, invitationLink)
}

func (m *MockNotificationClient) SendWelcomeEmail(ctx context.Context, userID, email, firstName string) {
	m.Called(ctx, userID, email, firstName)
}

var _ client.NotificationClient = (*MockNotificationClient)(nil)

// PERBAIKAN: Tambahkan MockTokenGenerator yang hilang
type MockTokenGenerator struct {
	TokenToReturn string
}

func (m *MockTokenGenerator) Generate() string {
	// Jika token diset, kembalikan itu. Jika tidak, kembalikan string default.
	if m.TokenToReturn != "" {
		return m.TokenToReturn
	}
	return "fixed-mock-token"
}

var _ TokenGenerator = (*MockTokenGenerator)(nil)

func TestInvitationService_CreateInvitation(t *testing.T) {
	ctx := context.Background()
	email := "new.user@example.com"
	role := "viewer"
	ttlDuration := time.Hour * 24 * 7
	ttlHours := 7 * 24

	// Gunakan token yang sudah kita tentukan
	fixedToken := "this-is-a-fixed-token-for-testing"

	t.Run("Success", func(t *testing.T) {
		// Arrange
		redisClient, mockRedis := redismock.NewClientMock()
		mockNotifClient := new(MockNotificationClient)
		// Inject MOCK generator
		mockTokenGen := &MockTokenGenerator{TokenToReturn: fixedToken}
		svc := NewInvitationService(redisClient, mockNotifClient, mockTokenGen, ttlHours)

		// Replikasi logika hashing untuk mendapatkan redisKey yang pasti
		hash := sha256.Sum256([]byte(fixedToken))
		tokenHash := base64.StdEncoding.EncodeToString(hash[:])
		expectedRedisKey := fmt.Sprintf("invitation:%s", tokenHash)

		expectedData := InvitationData{Email: email, Role: role}
		expectedPayload, _ := json.Marshal(expectedData)

		// Set ekspektasi dengan nilai yang PASTI
		mockRedis.ExpectSet(expectedRedisKey, expectedPayload, ttlDuration).SetVal("OK")
		mockNotifClient.On("SendInvitationEmail", ctx, email, mock.AnythingOfType("string")).Return()

		// Act
		token, err := svc.CreateInvitation(ctx, email, role)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, fixedToken, token) // Verifikasi token yang dikembalikan
		assert.NoError(t, mockRedis.ExpectationsWereMet())
		mockNotifClient.AssertExpectations(t)
	})

	t.Run("Redis Failure", func(t *testing.T) {
		// Arrange
		redisClient, mockRedis := redismock.NewClientMock()
		mockNotifClient := new(MockNotificationClient)
		mockTokenGen := &MockTokenGenerator{TokenToReturn: fixedToken}
		svc := NewInvitationService(redisClient, mockNotifClient, mockTokenGen, ttlHours)
		expectedError := errors.New("redis connection failed")

		hash := sha256.Sum256([]byte(fixedToken))
		tokenHash := base64.StdEncoding.EncodeToString(hash[:])
		expectedRedisKey := fmt.Sprintf("invitation:%s", tokenHash)
		expectedData := InvitationData{Email: email, Role: role}
		expectedPayload, _ := json.Marshal(expectedData)

		mockRedis.ExpectSet(expectedRedisKey, expectedPayload, ttlDuration).SetErr(expectedError)

		// Act
		token, err := svc.CreateInvitation(ctx, email, role)

		// Assert
		require.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.Empty(t, token)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
		mockNotifClient.AssertNotCalled(t, "SendInvitationEmail", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestInvitationService_ValidateInvitation(t *testing.T) {
	ctx := context.Background()
	token := "valid-token-string"

	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.StdEncoding.EncodeToString(hash[:])
	expectedRedisKey := fmt.Sprintf("invitation:%s", tokenHash)

	t.Run("Success - Valid Token", func(t *testing.T) {
		redisClient, mockRedis := redismock.NewClientMock()
		svc := NewInvitationService(redisClient, &MockNotificationClient{}, &MockTokenGenerator{}, 1)

		expectedData := InvitationData{Email: "valid.user@example.com", Role: "editor"}
		payload, _ := json.Marshal(expectedData)

		mockRedis.ExpectGet(expectedRedisKey).SetVal(string(payload))
		mockRedis.ExpectDel(expectedRedisKey).SetVal(1)

		data, err := svc.ValidateInvitation(ctx, token)

		require.NoError(t, err)
		require.NotNil(t, data)
		assert.Equal(t, "valid.user@example.com", data.Email)
		assert.Equal(t, "editor", data.Role)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("Failure - Token Not Found", func(t *testing.T) {
		redisClient, mockRedis := redismock.NewClientMock()
		svc := NewInvitationService(redisClient, &MockNotificationClient{}, &MockTokenGenerator{}, 1)
		mockRedis.ExpectGet(expectedRedisKey).RedisNil()

		data, err := svc.ValidateInvitation(ctx, "valid-token-string")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "undangan tidak valid atau sudah kedaluwarsa")
		assert.Nil(t, data)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})
}
