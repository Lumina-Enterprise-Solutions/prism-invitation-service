package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// Definisikan Interface di sini
type NotificationClient interface {
	SendInvitationEmail(ctx context.Context, email, invitationLink string)
	SendWelcomeEmail(ctx context.Context, userID, email, firstName string)
}

// Struct sekarang private (huruf kecil)
type notificationClient struct {
	httpClient *http.Client
	baseURL    string
}

// Memastikan `notificationClient` memenuhi interface `NotificationClient`
var _ NotificationClient = (*notificationClient)(nil)

// NewNotificationClient sekarang mengembalikan interface, dan di dalamnya mengembalikan pointer ke struct
func NewNotificationClient() NotificationClient {
	baseURL := "http://notification-service:8080"
	// FIX: Kembalikan pointer ke struct `notificationClient`, bukan struct `NotificationClient` itu sendiri.
	return NotificationClient(&notificationClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	})
}

// (Sisa file tidak perlu diubah)

type NotificationPayload struct {
	RecipientID  string                 `json:"recipient_id,omitempty"`
	Recipient    string                 `json:"recipient"`
	Subject      string                 `json:"subject"`
	TemplateName string                 `json:"template_name"`
	TemplateData map[string]interface{} `json:"template_data"`
}

func (c *notificationClient) sendNotification(ctx context.Context, payload NotificationPayload) {
	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[ERROR] NotificationClient: Gagal marshal payload: %v", err)
			return
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+"/notifications/send", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("[ERROR] NotificationClient: Gagal membuat request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("[ERROR] NotificationClient: Gagal mengirim request ke notification service: %v", err)
			return
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("WARN: NotificationClient: Gagal menutup response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusAccepted {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Printf("[ERROR] NotificationClient: Response tidak terduga dari notification service. Status: %s, Body: %s", resp.Status, string(bodyBytes))
		} else {
			log.Printf("[INFO] NotificationClient: Notifikasi '%s' untuk %s berhasil di-enqueue.", payload.Subject, payload.Recipient)
		}
	}()
}

func (c *notificationClient) SendInvitationEmail(ctx context.Context, email, invitationLink string) {
	templateData := map[string]interface{}{
		"InvitationLink": invitationLink,
		"RecipientEmail": email,
	}

	payload := NotificationPayload{
		Recipient:    email,
		Subject:      "Anda Diundang untuk Bergabung dengan Prism ERP",
		TemplateName: "invitation.html",
		TemplateData: templateData,
	}
	c.sendNotification(ctx, payload)
}

func (c *notificationClient) SendWelcomeEmail(ctx context.Context, userID, email, firstName string) {
	payload := NotificationPayload{
		RecipientID:  userID,
		Recipient:    email,
		Subject:      "Selamat Datang di Prism ERP!",
		TemplateName: "welcome.html",
		TemplateData: map[string]interface{}{"FirstName": firstName},
	}
	c.sendNotification(ctx, payload)
}
