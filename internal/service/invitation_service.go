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
	"github.com/rs/zerolog/log"
)

// InvitationData tetap sama.
type InvitationData struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	TenantID string `json:"tenantID"`
}

// InvitationService interface tetap sama.
type InvitationService interface {
	CreateInvitation(ctx context.Context, email, role, tenantID, inviterID string) (string, error)
	ValidateInvitation(ctx context.Context, token string) (*InvitationData, error)
}

// Implementasi service sekarang bergantung pada QueuePublisher.
type invitationService struct {
	redisClient    *redis.Client
	queuePublisher client.QueuePublisher // DIUBAH: Bergantung pada interface publisher.
	tokenGenerator TokenGenerator
	ttl            time.Duration
}

// NewInvitationService sekarang menerima QueuePublisher, bukan NotificationClient.
func NewInvitationService(redisClient *redis.Client, publisher client.QueuePublisher, tokenGen TokenGenerator, ttlHours int) InvitationService {
	return &invitationService{
		redisClient:    redisClient,
		queuePublisher: publisher, // DIUBAH: Menyimpan publisher.
		tokenGenerator: tokenGen,
		ttl:            time.Hour * time.Duration(ttlHours),
	}
}

// CreateInvitation sekarang menerbitkan event, bukan memanggil HTTP client.
func (s *invitationService) CreateInvitation(ctx context.Context, email, role, tenantID, inviterID string) (string, error) {
	token := s.tokenGenerator.Generate()
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.StdEncoding.EncodeToString(hash[:])

	redisKey := fmt.Sprintf("invitation:%s", tokenHash)
	invitationData := InvitationData{Email: email, Role: role, TenantID: tenantID}
	payload, err := json.Marshal(invitationData)
	if err != nil {
		return "", err
	}

	if err := s.redisClient.Set(ctx, redisKey, payload, s.ttl).Err(); err != nil {
		return "", err
	}

	// Buat payload untuk notifikasi.
	invitationLink := fmt.Sprintf("https://app.prismerp.com/accept-invitation?token=%s", token)
	notificationPayload := client.NotificationPayload{
		Recipient:    email,
		Subject:      "Anda Diundang untuk Bergabung dengan Prism ERP",
		TemplateName: "invitation.html",
		TemplateData: map[string]interface{}{
			"InvitationLink": invitationLink,
			"RecipientEmail": email,
		},
	}

	// Terbitkan event ke RabbitMQ.
	if err := s.queuePublisher.Enqueue(ctx, notificationPayload); err != nil {
		// Jika pengiriman ke queue gagal, kita harus mempertimbangkan untuk rollback penyimpanan Redis.
		// Namun, untuk ketahanan, lebih baik log error ini dan biarkan sistem lain (monitoring/alerting) menanganinya.
		// Menghapus token dari Redis akan membuat undangan menjadi tidak valid.
		log.Error().Err(err).Str("email", email).Msg("Gagal menerbitkan event undangan, undangan mungkin tidak terkirim.")
		// Kita tetap mengembalikan token, dengan asumsi pengiriman bisa dicoba lagi secara manual.
	}

	return token, nil
}

// ValidateInvitation tidak berubah.
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
		log.Warn().Err(err).Msg("PERINGATAN: gagal menghapus token undangan bekas pakai")
	}

	return &data, nil
}
