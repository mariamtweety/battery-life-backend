package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"battery-life/internal/app/auth"
	"battery-life/internal/app/ds"
	"battery-life/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RegisterUserAPI godoc
//
//	@Summary		Регистрация пользователя
//	@Description	Создаёт пользователя. Роль назначается на бэкенде (user), is_moderator с клиента не принимается.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		ds.UserRegisterRequest	true	"Логин и пароль"
//	@Success		201		{object}	ds.BatteryAppUser
//	@Failure		400		{object}	ds.ErrorResponse
//	@Failure		409		{object}	ds.ErrorResponse
//	@Router			/users/register [post]
func (h *Handler) RegisterUserAPI(ctx *gin.Context) {
	var req ds.UserRegisterRequest

	if ctx.ContentType() == "application/json" {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apiError(ctx, http.StatusBadRequest, "Неверный формат JSON: "+err.Error())
			return
		}
	} else {
		req.Username = ctx.PostForm("username")
		req.Password = ctx.PostForm("password")
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
	if len(req.Password) > 72 {
		// Ограничение алгоритма bcrypt.
		apiError(ctx, http.StatusBadRequest, "Пароль не должен превышать 72 байта")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, "Не удалось захешировать пароль")
		return
	}

	user, err := h.Repository.RegisterUser(req.Username, passwordHash, false)
	if err != nil {
		writeRepoError(ctx, err, "Не удалось зарегистрировать пользователя")
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

// LoginUserAPI godoc
//
//	@Summary		Аутентификация по логину и паролю
//	@Description	Проверяет пароль и выдаёт JWT. Токен также сохраняется в cookie access_token.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		ds.UserLoginRequest	true	"Логин и пароль"
//	@Success		200			{object}	ds.AuthResponse
//	@Failure		400			{object}	ds.ErrorResponse
//	@Failure		401			{object}	ds.ErrorResponse
//	@Router			/users/login [post]
func (h *Handler) LoginUserAPI(ctx *gin.Context) {
	var req ds.UserLoginRequest

	if ctx.ContentType() == "application/json" {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apiError(ctx, http.StatusBadRequest, "Неверный формат JSON: "+err.Error())
			return
		}
	} else {
		req.Username = ctx.PostForm("username")
		req.Password = ctx.PostForm("password")
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		apiError(ctx, http.StatusBadRequest, "Логин и пароль обязательны")
		return
	}

	user, err := h.Repository.GetUserByUsername(req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Не раскрываем, существует ли пользователь.
			apiError(ctx, http.StatusUnauthorized, "Неверный логин или пароль")
			return
		}
		apiError(ctx, http.StatusInternalServerError, "Ошибка аутентификации: "+err.Error())
		return
	}

	ok, needsRehash := auth.CheckPassword(user.PasswordHash, req.Password)
	if !ok {
		apiError(ctx, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}

	// Прозрачно обновляем пароли, сохранённые в открытом виде в ЛР3.
	if needsRehash {
		if newHash, hashErr := auth.HashPassword(req.Password); hashErr == nil {
			if updateErr := h.Repository.UpdateUserPasswordHash(user.ID, newHash); updateErr != nil {
				logrus.Warnf("не удалось обновить устаревший хэш пароля пользователя %d: %v", user.ID, updateErr)
			} else {
				user.PasswordHash = newHash
			}
		}
	}

	tokenStr, claims, err := h.Auth.Generate(user)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, "Не удалось выпустить токен: "+err.Error())
		return
	}

	if err := h.Auth.SaveSession(ctx.Request.Context(), claims); err != nil {
		apiError(ctx, http.StatusInternalServerError, "Не удалось сохранить сессию в Redis: "+err.Error())
		return
	}

	h.setAuthCookie(ctx, tokenStr, h.Config.JWT.ExpiresIn)

	ctx.JSON(http.StatusOK, ds.AuthResponse{
		AccessToken: tokenStr,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.Config.JWT.ExpiresIn.Seconds()),
		User:        user,
	})
}

// CurrentUserAPI godoc
//
//	@Summary		Текущий пользователь
//	@Description	Возвращает пользователя, которому принадлежит переданный JWT.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	ds.BatteryAppUser
//	@Failure		401	{object}	ds.ErrorResponse
//	@Failure		404	{object}	ds.ErrorResponse
//	@Router			/users/me [get]
func (h *Handler) CurrentUserAPI(ctx *gin.Context) {
	user, err := h.Repository.GetUserByID(currentUserID(ctx))
	if err != nil {
		writeRepoError(ctx, err, "Не удалось получить пользователя")
		return
	}
	ctx.JSON(http.StatusOK, user)
}

// LogoutUserAPI godoc
//
//	@Summary		Деавторизация
//	@Description	Добавляет JWT в blacklist Redis до истечения токена и удаляет сессию.
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	ds.MessageResponse
//	@Failure		401	{object}	ds.ErrorResponse
//	@Router			/users/logout [post]
func (h *Handler) LogoutUserAPI(ctx *gin.Context) {
	claims := currentClaims(ctx)
	tokenStr := h.tokenFromRequest(ctx)

	if claims == nil || tokenStr == "" {
		apiError(ctx, http.StatusUnauthorized, "Требуется авторизация")
		return
	}

	if err := h.Auth.Logout(ctx.Request.Context(), tokenStr, claims); err != nil {
		apiError(ctx, http.StatusInternalServerError, "Не удалось выполнить деавторизацию: "+err.Error())
		return
	}

	h.clearAuthCookie(ctx)

	ctx.JSON(http.StatusOK, ds.MessageResponse{Message: "деавторизация выполнена, токен добавлен в blacklist"})
}

func (h *Handler) setAuthCookie(ctx *gin.Context, token string, ttl time.Duration) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(
		auth.AccessTokenCookie,
		token,
		int(ttl.Seconds()),
		"/",
		"",
		ctx.Request.TLS != nil,
		true,
	)
}

func (h *Handler) clearAuthCookie(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(auth.AccessTokenCookie, "", -1, "/", "", ctx.Request.TLS != nil, true)
}
