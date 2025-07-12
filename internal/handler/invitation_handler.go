package handler

import (
	"net/http"

	"github.com/Lumina-Enterprise-Solutions/prism-invitation-service/internal/service"
	"github.com/gin-gonic/gin"
)

type InvitationHandler struct {
	service service.InvitationService
}

func NewInvitationHandler(svc service.InvitationService) *InvitationHandler {
	return &InvitationHandler{service: svc}
}

func (h *InvitationHandler) CreateInvitation(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ekstrak tenantID dan inviterID dari klaim token JWT.
	// Middleware auth harus sudah memvalidasi token dan menempatkan klaim di context.
	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenantID tidak ditemukan di dalam token"})
		return
	}

	inviterID, exists := c.Get("userID") // Atau `sub`, tergantung implementasi token.
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "inviterID tidak ditemukan di dalam token"})
		return
	}

	_, err := h.service.CreateInvitation(c.Request.Context(), req.Email, req.Role, tenantID.(string), inviterID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat undangan"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "undangan berhasil dikirim"})
}

func (h *InvitationHandler) ValidateInvitation(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token wajib diisi"})
		return
	}

	data, err := h.service.ValidateInvitation(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}
