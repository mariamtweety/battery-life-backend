package handler

import (
	"errors"
	"net/http"

	"battery-life/internal/app/repository"

	"github.com/gin-gonic/gin"
)

// writeRepoError переводит сентинел-ошибки репозитория в HTTP-коды.
func writeRepoError(ctx *gin.Context, err error, fallbackMessage string) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		apiError(ctx, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrForbidden):
		apiError(ctx, http.StatusForbidden, err.Error())
	case errors.Is(err, repository.ErrConflict):
		apiError(ctx, http.StatusConflict, err.Error())
	case errors.Is(err, repository.ErrValidation):
		apiError(ctx, http.StatusBadRequest, err.Error())
	default:
		apiError(ctx, http.StatusInternalServerError, fallbackMessage+": "+err.Error())
	}
}
