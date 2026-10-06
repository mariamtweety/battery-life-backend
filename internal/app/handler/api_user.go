package handler

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"battery-life/internal/app/ds"
	"battery-life/internal/app/singleton"

	"github.com/gin-gonic/gin"
)

// RegisterUserAPI handles POST /api/users/register
func (h *Handler) RegisterUserAPI(ctx *gin.Context) {
	var req ds.UserRegisterRequest

	if ctx.ContentType() == "application/json" {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apiError(ctx, http.StatusBadRequest, "Неверный формат JSON: "+err.Error())
			return
		}
	} else {
		req.Username = strings.TrimSpace(ctx.PostForm("username"))
		req.Password = strings.TrimSpace(ctx.PostForm("password"))
		req.IsModerator = ctx.PostForm("is_moderator") == "true" || ctx.PostForm("is_moderator") == "1"
	}

	req.Username = strings.TrimSpace(req.Username)
	if utf8.RuneCountInString(req.Username) < 3 || utf8.RuneCountInString(req.Username) > 50 {
		apiError(ctx, http.StatusBadRequest, "Имя пользователя должно содержать от 3 до 50 символов")
		return
	}

	if len(req.Password) < 4 {
		apiError(ctx, http.StatusBadRequest, "Пароль должен содержать не менее 4 символов")
		return
	}

	user, err := h.Repository.RegisterUser(req.Username, req.Password, req.IsModerator)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

// LoginUserAPI handles POST /api/users/login (stub for Lab 4)
func (h *Handler) LoginUserAPI(ctx *gin.Context) {
	var req ds.UserLoginRequest

	if ctx.ContentType() == "application/json" {
		_ = ctx.ShouldBindJSON(&req)
	} else {
		req.Username = strings.TrimSpace(ctx.PostForm("username"))
		req.Password = strings.TrimSpace(ctx.PostForm("password"))
	}

	currentUser := singleton.GetCurrentUser()

	ctx.JSON(http.StatusOK, ds.AuthResponse{
		Message: "Аутентификация успешна (заглушка для ЛР4)",
		Token:   "stub_battery_life_session_token_1",
		User:    currentUser,
	})
}

// LogoutUserAPI handles POST /api/users/logout (stub for Lab 4)
func (h *Handler) LogoutUserAPI(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ds.AuthResponse{
		Message: "Деавторизация успешна (заглушка для ЛР4)",
	})
}

// GetUserScenariosAPI handles GET /api/users/:id (nested serialization)
func (h *Handler) GetUserScenariosAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	idVal, err := strconv.Atoi(idStr)
	if err != nil || idVal <= 0 {
		apiError(ctx, http.StatusBadRequest, "Неверный ID пользователя")
		return
	}

	data, err := h.Repository.GetUserScenarios(uint(idVal))
	if err != nil {
		apiError(ctx, http.StatusNotFound, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, data)
}
