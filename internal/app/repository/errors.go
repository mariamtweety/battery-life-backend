package repository

import "errors"

// Сентинел-ошибки репозитория: по ним handler выбирает HTTP-код ответа.
var (
	// ErrNotFound — запись не существует (404).
	ErrNotFound = errors.New("запись не найдена")
	// ErrForbidden — действие запрещено текущему пользователю (403).
	ErrForbidden = errors.New("действие запрещено")
	// ErrConflict — нарушение бизнес-правила или уникальности (409).
	ErrConflict = errors.New("конфликт состояния")
	// ErrValidation — некорректные входные данные (400).
	ErrValidation = errors.New("некорректные данные")
)
