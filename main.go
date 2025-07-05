package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Lumina-Enterprise-Solutions/prism-common-libs/client"
	"github.com/Lumina-Enterprise-Solutions/prism-common-libs/enhanced_logger"
	"github.com/Lumina-Enterprise-Solutions/prism-common-libs/telemetry"
	"github.com/Lumina-Enterprise-Solutions/prism-invitation-service/config"

	// DIUBAH: Menggunakan package client yang telah dimodifikasi.
	invitationclient "github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/client"
	"github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/handler"
	"github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	enhanced_logger.Init()
	cfg := config.Load()
	serviceLogger := enhanced_logger.WithService(cfg.ServiceName)
	enhanced_logger.LogStartup(cfg.ServiceName, cfg.Port, map[string]interface{}{"rabbitmq_url": cfg.RabbitMQURL})

	// Setup Telemetry
	tp, err := telemetry.InitTracerProvider(cfg.ServiceName, cfg.JaegerEndpoint)
	if err != nil {
		serviceLogger.Fatal().Err(err).Msg("Gagal menginisialisasi tracer")
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			serviceLogger.Error().Err(err).Msg("Gagal mematikan tracer provider dengan benar")
		}
	}()

	// Setup Redis Client
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer func() {
		if err := redisClient.Close(); err != nil {
			serviceLogger.Error().Err(err).Msg("Gagal menutup koneksi Redis dengan benar")
		}
	}()

	// BARU: Setup RabbitMQ Publisher
	queuePublisher, err := invitationclient.NewQueuePublisher(cfg.RabbitMQURL)
	if err != nil {
		serviceLogger.Fatal().Err(err).Msg("Gagal terhubung ke RabbitMQ")
	}
	defer func() {
		if err := queuePublisher.Close(); err != nil {
			serviceLogger.Error().Err(err).Msg("Gagal menutup koneksi RabbitMQ dengan benar")
		}
	}()

	// Inisialisasi service dan handler dengan publisher baru.
	realTokenGenerator := &service.UUIDTokenGenerator{}
	invitationService := service.NewInvitationService(redisClient, queuePublisher, realTokenGenerator, cfg.InvitationTTL)
	invitationHandler := handler.NewInvitationHandler(invitationService)

	// Setup Gin Router
	router := gin.Default()
	router.Use(otelgin.Middleware(cfg.ServiceName))
	p := ginprometheus.NewPrometheus("gin")
	p.Use(router)

	// --- Routes ---
	group := router.Group("/invitations")
	group.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy"}) })
	group.POST("", invitationHandler.CreateInvitation)
	group.POST("/validate", invitationHandler.ValidateInvitation)

	// Setup Consul Service Discovery
	regInfo := client.ServiceRegistrationInfo{
		ServiceName:    cfg.ServiceName,
		ServiceID:      fmt.Sprintf("%s-%d", cfg.ServiceName, cfg.Port),
		Port:           cfg.Port,
		HealthCheckURL: fmt.Sprintf("http://%s:%d/invitations/health", cfg.ServiceName, cfg.Port),
	}
	consulClient, err := client.RegisterService(regInfo)
	if err != nil {
		serviceLogger.Fatal().Err(err).Msg("Gagal mendaftar ke Consul")
	}
	defer client.DeregisterService(consulClient, regInfo.ServiceID)

	// Start server & handle graceful shutdown
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Port),
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serviceLogger.Fatal().Err(err).Msg("Gagal menjalankan server HTTP")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	serviceLogger.Info().Msg("Memulai graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		serviceLogger.Fatal().Err(err).Msg("Server terpaksa dimatikan")
	}
	enhanced_logger.LogShutdown(cfg.ServiceName)
}
