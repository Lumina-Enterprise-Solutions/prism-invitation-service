package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/client"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type InvitationData struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type InvitationService interface {
	CreateInvitation(ctx context.Context, email, role string) (string, error)
	ValidateInvitation(ctx context.Context, token string) (*InvitationData, error)
}

type invitationService struct {
	redisClient        *redis.Client
	notificationClient client.NotificationClient
	ttl                time.Duration
}

func NewInvitationService(redisClient *redis.Client, notificationClient client.NotificationClient, ttlHours int) InvitationService {
	return &invitationService{
		redisClient:        redisClient,
		notificationClient: notificationClient,
		ttl:                time.Hour * time.Duration(ttlHours),
	}
}

func (s *invitationService) CreateInvitation(ctx context.Context, email, role string) (string, error) {
	token := uuid.NewString() // Token yang akan dikirim ke user
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.StdEncoding.EncodeToString(hash[:])

	redisKey := fmt.Sprintf("invitation:%s", tokenHash)
	invitationData := InvitationData{Email: email, Role: role}
	payload, err := json.Marshal(invitationData)
	if err != nil {
		return "", err
	}

	if err := s.redisClient.Set(ctx, redisKey, payload, s.ttl).Err(); err != nil {
		return "", err
	}

	// Kirim email undangan
	invitationLink := fmt.Sprintf("https://app.prismerp.com/accept-invitation?token=%s", token)
	s.notificationClient.SendInvitationEmail(ctx, email, invitationLink)

	return token, nil
}

func (s *invitationService) ValidateInvitation(ctx context.Context, token string) (*InvitationData, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.StdEncoding.EncodeToString(hash[:])

	redisKey := fmt.Sprintf("invitation:%s", tokenHash)

	payload, err := s.redisClient.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("undangan tidak valid atau sudah kedaluwarsa")
	} else if err != nil {
		return nil, err
	}

	var data InvitationData
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, fmt.Errorf("gagal unmarshal data undangan: %w", err)
	}

	// Hapus token setelah berhasil divalidasi
	if err := s.redisClient.Del(ctx, redisKey).Err(); err != nil {
		// Log error ini, tapi jangan gagalkan proses utama
		fmt.Printf("PERINGATAN: gagal menghapus token undangan bekas pakai: %v\n", err)
	}

	return &data, nil
}
