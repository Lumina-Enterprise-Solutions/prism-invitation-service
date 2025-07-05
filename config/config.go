package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Lumina-Enterprise-Solutions/prism-common-libs/config"
	"github.com/rs/zerolog/log"
)

// Config menampung semua konfigurasi untuk invitation-service.
type Config struct {
	Port           int
	ServiceName    string
	JaegerEndpoint string
	RedisAddr      string
	VaultAddr      string
	VaultToken     string
	InvitationTTL  int
	// BARU: Menambahkan URL RabbitMQ untuk koneksi ke message broker.
	RabbitMQURL string
}

// Load memuat konfigurasi dari environment variables dan Consul.
func Load() *Config {
	loader, err := config.NewLoader()
	if err != nil {
		log.Fatal().Err(err).Msg("Gagal membuat config loader")
	}

	serviceName := "prism-invitation-service"
	pathPrefix := fmt.Sprintf("config/%s", serviceName)

	invitationTTL, _ := strconv.Atoi(loader.Get(fmt.Sprintf("%s/invitation_ttl_hours", pathPrefix), "168")) // Default 7 hari

	return &Config{
		Port:           loader.GetInt(fmt.Sprintf("%s/port", pathPrefix), 8080),
		ServiceName:    serviceName,
		JaegerEndpoint: loader.Get("config/global/jaeger_endpoint", "jaeger:4317"),
		RedisAddr:      loader.Get("config/global/redis_addr", "cache-redis:6379"),
		InvitationTTL:  invitationTTL,
		VaultAddr:      os.Getenv("VAULT_ADDR"),
		VaultToken:     os.Getenv("VAULT_TOKEN"),
		// BARU: Memuat URL RabbitMQ dari environment variable. Ini adalah praktik umum
		// karena URL koneksi sering kali berisi kredensial.
		RabbitMQURL: os.Getenv("RABBITMQ_URL"),
	}
}
