package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

// Mendefinisikan konstanta untuk RabbitMQ agar konsisten.
const (
	ExchangeName    = "prism_notifications_exchange"
	RoutingKey      = "email_notification"
	ContentTypeJSON = "application/json"
)

// NotificationPayload adalah struktur data yang akan dikirim sebagai pesan.
// Ini mendefinisikan kontrak antara invitation-service dan notification-service.
type NotificationPayload struct {
	Recipient    string                 `json:"recipient"`
	Subject      string                 `json:"subject"`
	TemplateName string                 `json:"template_name"`
	TemplateData map[string]interface{} `json:"template_data"`
}

// QueuePublisher adalah interface yang mendefinisikan cara mengirim pesan.
// Ini memungkinkan kita untuk menggunakan mock saat testing.
type QueuePublisher interface {
	Enqueue(ctx context.Context, payload NotificationPayload) error
	Close() error
}

// rabbitMQPublisher adalah implementasi nyata dari QueuePublisher.
type rabbitMQPublisher struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

// NewQueuePublisher membuat instance baru dari RabbitMQ publisher.
func NewQueuePublisher(amqpURL string) (QueuePublisher, error) {
	conn, err := amqp091.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close() // Pastikan koneksi ditutup jika channel gagal dibuat.
		return nil, fmt.Errorf("gagal membuka channel RabbitMQ: %w", err)
	}

	// Pastikan exchange yang akan kita gunakan sudah ada.
	// Ini membuat service lebih tangguh jika dijalankan sebelum infrastruktur RabbitMQ sepenuhnya siap.
	err = ch.ExchangeDeclare(
		ExchangeName, // name
		"direct",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("gagal mendeklarasikan exchange: %w", err)
	}

	return &rabbitMQPublisher{conn: conn, channel: ch}, nil
}

// Enqueue menerbitkan pesan ke RabbitMQ.
func (p *rabbitMQPublisher) Enqueue(ctx context.Context, payload NotificationPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal marshal payload notifikasi: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Info().Str("recipient", payload.Recipient).Str("subject", payload.Subject).Msg("Menerbitkan event notifikasi ke RabbitMQ")

	return p.channel.PublishWithContext(
		ctx,
		ExchangeName, // exchange
		RoutingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			ContentType:  ContentTypeJSON,
			DeliveryMode: amqp091.Persistent, // Pesan akan bertahan jika RabbitMQ restart.
			Body:         body,
		},
	)
}

// Close menutup channel dan koneksi RabbitMQ.
func (p *rabbitMQPublisher) Close() error {
	var firstErr error
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			firstErr = fmt.Errorf("gagal menutup channel: %w", err)
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("gagal menutup koneksi: %w", err)
			}
		}
	}
	return firstErr
}
