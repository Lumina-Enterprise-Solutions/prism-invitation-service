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

// NotificationClient adalah HTTP client untuk berkomunikasi dengan prism-notification-service.
type NotificationClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewNotificationClient membuat instance baru dari NotificationClient.
func NewNotificationClient() NotificationClient {
	// Service discovery name dari Docker Compose
	baseURL := "http://notification-service:8080"

	return NotificationClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

// NotificationPayload adalah struktur data yang dikirim ke notification-service.
type NotificationPayload struct {
	RecipientID  string                 `json:"recipient_id,omitempty"` // ID pengguna, jika ada
	Recipient    string                 `json:"recipient"`              // Alamat email
	Subject      string                 `json:"subject"`
	TemplateName string                 `json:"template_name"`
	TemplateData map[string]interface{} `json:"template_data"`
}

// sendNotification adalah fungsi internal untuk mengirim payload secara asinkron.
func (c *NotificationClient) sendNotification(ctx context.Context, payload NotificationPayload) {
	go func() {
		// Marshal payload ke JSON
		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[ERROR] NotificationClient: Gagal marshal payload: %v", err)
			return
		}

		// Buat request baru. Menggunakan context.Background() agar tidak dibatalkan oleh context parent.
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+"/notifications/send", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("[ERROR] NotificationClient: Gagal membuat request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		// Kirim request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("[ERROR] NotificationClient: Gagal mengirim request ke notification service: %v", err)
			return
		}
		defer resp.Body.Close()

		// Periksa status response
		if resp.StatusCode != http.StatusAccepted {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Printf("[ERROR] NotificationClient: Response tidak terduga dari notification service. Status: %s, Body: %s", resp.Status, string(bodyBytes))
		} else {
			log.Printf("[INFO] NotificationClient: Notifikasi '%s' untuk %s berhasil di-enqueue.", payload.Subject, payload.Recipient)
		}
	}()
}

// SendInvitationEmail adalah method spesifik untuk mengirim email undangan.
func (c *NotificationClient) SendInvitationEmail(ctx context.Context, email, invitationLink string) {
	// Template data yang dibutuhkan oleh `invitation.html`
	templateData := map[string]interface{}{
		"InvitationLink": invitationLink,
		"RecipientEmail": email,
	}

	payload := NotificationPayload{
		Recipient:    email,
		Subject:      "Anda Diundang untuk Bergabung dengan Prism ERP",
		TemplateName: "invitation.html", // Pastikan template ini ada di notification-service
		TemplateData: templateData,
	}

	c.sendNotification(ctx, payload)
}

// SendWelcomeEmail adalah method yang mungkin akan Anda tambahkan nanti
// setelah user menyelesaikan pendaftaran via undangan.
func (c *NotificationClient) SendWelcomeEmail(ctx context.Context, userID, email, firstName string) {
	payload := NotificationPayload{
		RecipientID:  userID,
		Recipient:    email,
		Subject:      "Selamat Datang di Prism ERP!",
		TemplateName: "welcome.html",
		TemplateData: map[string]interface{}{"FirstName": firstName},
	}
	c.sendNotification(ctx, payload)
}
