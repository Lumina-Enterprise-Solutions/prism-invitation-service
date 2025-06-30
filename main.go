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
	notifclient "github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/client"
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
	enhanced_logger.LogStartup(cfg.ServiceName, cfg.Port, nil)

	tp, err := telemetry.InitTracerProvider(cfg.ServiceName, cfg.JaegerEndpoint)
	if err != nil {
		serviceLogger.Fatal().Err(err).Msg("Gagal menginisialisasi tracer")
	}
	defer tp.Shutdown(context.Background())

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	notificationClient := notifclient.NewNotificationClient()

	invitationService := service.NewInvitationService(redisClient, notificationClient, cfg.InvitationTTL)
	invitationHandler := handler.NewInvitationHandler(invitationService)

	router := gin.Default()
	router.Use(otelgin.Middleware(cfg.ServiceName))

	p := ginprometheus.NewPrometheus("gin")
	p.Use(router)

	// --- Routes ---
	group := router.Group("/invitations")
	group.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy"}) })
	// Endpoint publik untuk membuat undangan (akan diproteksi oleh auth di fase berikutnya)
	group.POST("", invitationHandler.CreateInvitation)
	// Endpoint internal untuk validasi token oleh auth-service
	group.POST("/validate", invitationHandler.ValidateInvitation)

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
