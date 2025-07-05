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

// --- Mocks untuk Dependensi ---

// Mock untuk QueuePublisher (menggantikan MockNotificationClient)
type MockQueuePublisher struct {
	mock.Mock
}

// Implementasi interface QueuePublisher
func (m *MockQueuePublisher) Enqueue(ctx context.Context, payload client.NotificationPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func (m *MockQueuePublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Pastikan mock memenuhi interface.
var _ client.QueuePublisher = (*MockQueuePublisher)(nil)

// MockTokenGenerator tetap sama.
type MockTokenGenerator struct {
	TokenToReturn string
}

func (m *MockTokenGenerator) Generate() string {
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
	fixedToken := "this-is-a-fixed-token-for-testing"

	t.Run("Success", func(t *testing.T) {
		// Arrange
		redisClient, mockRedis := redismock.NewClientMock()
		mockPublisher := new(MockQueuePublisher) // Menggunakan mock publisher baru
		mockTokenGen := &MockTokenGenerator{TokenToReturn: fixedToken}
		// DIUBAH: Inject mock publisher ke service
		svc := NewInvitationService(redisClient, mockPublisher, mockTokenGen, ttlHours)

		hash := sha256.Sum256([]byte(fixedToken))
		tokenHash := base64.StdEncoding.EncodeToString(hash[:])
		expectedRedisKey := fmt.Sprintf("invitation:%s", tokenHash)
		expectedData := InvitationData{Email: email, Role: role}
		expectedPayload, _ := json.Marshal(expectedData)

		mockRedis.ExpectSet(expectedRedisKey, expectedPayload, ttlDuration).SetVal("OK")
		// DIUBAH: Ekspektasi sekarang adalah pemanggilan Enqueue dengan payload yang benar.
		mockPublisher.On("Enqueue", ctx, mock.AnythingOfType("client.NotificationPayload")).Return(nil).Once()

		// Act
		token, err := svc.CreateInvitation(ctx, email, role)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, fixedToken, token)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
		mockPublisher.AssertExpectations(t)
	})

	t.Run("Redis Failure", func(t *testing.T) {
		// Arrange
		redisClient, mockRedis := redismock.NewClientMock()
		mockPublisher := new(MockQueuePublisher) // Gunakan mock publisher
		mockTokenGen := &MockTokenGenerator{TokenToReturn: fixedToken}
		svc := NewInvitationService(redisClient, mockPublisher, mockTokenGen, ttlHours)
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
		// Verifikasi bahwa Enqueue tidak pernah dipanggil jika penyimpanan Redis gagal.
		mockPublisher.AssertNotCalled(t, "Enqueue", mock.Anything, mock.Anything)
	})
}

// TestValidateInvitation tidak perlu diubah karena tidak berinteraksi dengan publisher.
func TestInvitationService_ValidateInvitation(t *testing.T) {
	ctx := context.Background()
	token := "valid-token-string"

	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.StdEncoding.EncodeToString(hash[:])
	expectedRedisKey := fmt.Sprintf("invitation:%s", tokenHash)

	t.Run("Success - Valid Token", func(t *testing.T) {
		redisClient, mockRedis := redismock.NewClientMock()
		// Kirim nil untuk publisher karena tidak digunakan di sini.
		svc := NewInvitationService(redisClient, nil, &MockTokenGenerator{}, 1)

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
		svc := NewInvitationService(redisClient, nil, &MockTokenGenerator{}, 1)
		mockRedis.ExpectGet(expectedRedisKey).RedisNil()

		data, err := svc.ValidateInvitation(ctx, "valid-token-string")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "undangan tidak valid atau sudah kedaluwarsa")
		assert.Nil(t, data)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})
}
