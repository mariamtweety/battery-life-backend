package handler

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"battery-life/internal/app/ds"
	"battery-life/internal/app/singleton"

	"github.com/gin-gonic/gin"
)

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

func (h *Handler) LogoutUserAPI(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ds.AuthResponse{
		Message: "Деавторизация успешна (заглушка для ЛР4)",
	})
}
