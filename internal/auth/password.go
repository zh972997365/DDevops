package auth

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 生成 bcrypt 摘要，用于持久化存储。
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword 校验密码。第二个返回值表示该凭证仍是历史明文格式，
// 调用方应在校验通过后立即升级为 bcrypt 摘要（平滑迁移，用户无感）。
func VerifyPassword(stored, plain string) (ok bool, legacy bool) {
	if !isBcryptHash(stored) {
		return stored == plain, true
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil, false
}

func isBcryptHash(s string) bool {
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}
