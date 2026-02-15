package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Zhou-JK/hls-streamer/internal/middleware"
	"github.com/Zhou-JK/hls-streamer/internal/service"
	"github.com/Zhou-JK/hls-streamer/pkg/response"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, err := h.authSvc.Login(input.Username, input.Password)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.OK(c, tokens)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		RoleID   uint   `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if input.RoleID == 0 {
		input.RoleID = 3 // default: viewer
	}

	user, err := h.authSvc.Register(input.Username, input.Email, input.Password, input.RoleID)
	if err != nil {
		response.Conflict(c, err.Error())
		return
	}

	response.Created(c, user)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, err := h.authSvc.RefreshToken(input.RefreshToken)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.OK(c, tokens)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	_ = h.authSvc.Logout(input.RefreshToken)
	response.NoContent(c)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextKeyUserID)
	user, err := h.authSvc.GetUser(userID.(uint))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, user)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var input struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID, _ := c.Get(middleware.ContextKeyUserID)
	if err := h.authSvc.ChangePassword(userID.(uint), input.OldPassword, input.NewPassword); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "password changed"})
}

// User management (admin)

func (h *AuthHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	users, total, err := h.authSvc.ListUsers(page, perPage)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OKWithMeta(c, users, &response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	user, err := h.authSvc.GetUser(uint(id))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.OK(c, user)
}

func (h *AuthHandler) CreateUser(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		RoleID   uint   `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.authSvc.Register(input.Username, input.Email, input.Password, input.RoleID)
	if err != nil {
		response.Conflict(c, err.Error())
		return
	}

	response.Created(c, user)
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var input struct {
		Username *string `json:"username"`
		Email    *string `json:"email"`
		RoleID   *uint   `json:"role_id"`
		IsActive *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.authSvc.UpdateUser(uint(id), input.Username, input.Email, input.RoleID, input.IsActive)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, user)
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.authSvc.DeleteUser(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.NoContent(c)
}

func (h *AuthHandler) ListRoles(c *gin.Context) {
	roles, err := h.authSvc.ListRoles()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, roles)
}
