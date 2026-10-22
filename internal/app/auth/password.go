package auth

import (
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptPrefix = "$2"

// HashPassword возвращает bcrypt-хэш пароля.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword сравнивает пароль с сохранённым значением.
// Второй результат — true, если пароль хранится в старом (небезопасном)
// виде и его следует немедленно перехэшировать.
func CheckPassword(storedHash, password string) (ok bool, needsRehash bool) {
	if strings.HasPrefix(storedHash, bcryptPrefix) {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil, false
	}

	// Совместимость с пользователями, созданными в ЛР3 (пароль хранился открыто).
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(password)) == 1 {
		return true, true
	}
	return false, false
}
