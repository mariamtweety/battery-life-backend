package handler

import (
	"errors"
	"net/http"
	"strings"

	"battery-life/internal/app/auth"
	"battery-life/internal/app/ds"
	"battery-life/internal/app/role"

	"github.com/gin-gonic/gin"
)

const claimsContextKey = "auth.claims"

// tokenFromRequest достаёт JWT из заголовка Authorization либо из cookie.
// Поддержка cookie нужна для сценария «из браузера» и Swagger UI.
func (h *Handler) tokenFromRequest(ctx *gin.Context) string {
	header := ctx.GetHeader("Authorization")
	if strings.HasPrefix(header, auth.BearerPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, auth.BearerPrefix))
	}
	if cookie, err := ctx.Cookie(auth.AccessTokenCookie); err == nil {
		return cookie
	}
	return ""
}

func (h *Handler) authenticate(ctx *gin.Context) (*ds.JWTClaims, string, error) {
	token := h.tokenFromRequest(ctx)
	claims, err := h.Auth.Authenticate(ctx.Request.Context(), token)
	return claims, token, err
}

// WithOptionalAuth не прерывает запрос: гость получает публичные данные,
// авторизованный пользователь — ещё и is_liked/is_creator.
func (h *Handler) WithOptionalAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, _, err := h.authenticate(ctx)
		if err == nil {
			ctx.Set(claimsContextKey, claims)
		}
		ctx.Next()
	}
}

// WithAuthCheck требует валидный токен и (опционально) одну из ролей.
func (h *Handler) WithAuthCheck(assignedRoles ...role.Role) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, _, err := h.authenticate(ctx)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrNoToken),
				errors.Is(err, auth.ErrInvalidToken),
				errors.Is(err, auth.ErrBlacklisted):
				apiError(ctx, http.StatusUnauthorized, "Требуется авторизация: "+err.Error())
			default:
				apiError(ctx, http.StatusInternalServerError, "Ошибка проверки авторизации: "+err.Error())
			}
			ctx.Abort()
			return
		}

		if len(assignedRoles) > 0 && !roleAllowed(claims.Role, assignedRoles) {
			apiError(ctx, http.StatusForbidden, "Недостаточно прав для выполнения операции")
			ctx.Abort()
			return
		}

		ctx.Set(claimsContextKey, claims)
		ctx.Next()
	}
}

func roleAllowed(current role.Role, allowed []role.Role) bool {
	for _, r := range allowed {
		if current == r {
			return true
		}
	}
	return false
}

// currentClaims возвращает claims авторизованного пользователя или nil.
func currentClaims(ctx *gin.Context) *ds.JWTClaims {
	value, exists := ctx.Get(claimsContextKey)
	if !exists {
		return nil
	}
	claims, ok := value.(*ds.JWTClaims)
	if !ok {
		return nil
	}
	return claims
}

// currentUserID возвращает 0 для гостя.
func currentUserID(ctx *gin.Context) uint {
	if claims := currentClaims(ctx); claims != nil {
		return claims.UserID
	}
	return 0
}

func currentRole(ctx *gin.Context) role.Role {
	if claims := currentClaims(ctx); claims != nil {
		return claims.Role
	}
	return role.User
}

func isModerator(ctx *gin.Context) bool {
	return currentRole(ctx) == role.Moderator
}
