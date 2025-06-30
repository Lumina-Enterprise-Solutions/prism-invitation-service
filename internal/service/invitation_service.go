package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/client"
	"github.com/redis/go-redis/v9"
)

// ... (struct InvitationData tetap sama) ...
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
	tokenGenerator     TokenGenerator // <-- TAMBAHKAN INI
	ttl                time.Duration
}

// PERBAIKAN: Konstruktor sekarang menerima TokenGenerator.
func NewInvitationService(redisClient *redis.Client, notificationClient client.NotificationClient, tokenGen TokenGenerator, ttlHours int) InvitationService {
	return &invitationService{
		redisClient:        redisClient,
		notificationClient: notificationClient,
		tokenGenerator:     tokenGen, // <-- SIMPAN INI
		ttl:                time.Hour * time.Duration(ttlHours),
	}
}

func (s *invitationService) CreateInvitation(ctx context.Context, email, role string) (string, error) {
	token := s.tokenGenerator.Generate() // <-- Gunakan generator
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

	invitationLink := fmt.Sprintf("https://app.prismerp.com/accept-invitation?token=%s", token)
	s.notificationClient.SendInvitationEmail(ctx, email, invitationLink)

	return token, nil
}

// ... (ValidateInvitation tetap sama) ...
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

	if err := s.redisClient.Del(ctx, redisKey).Err(); err != nil {
		fmt.Printf("PERINGATAN: gagal menghapus token undangan bekas pakai: %v\n", err)
	}

	return &data, nil
}
