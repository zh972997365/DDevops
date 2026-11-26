package auth

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"ddevops/internal/repository"
)

// SessionTTL 登录有效期。
const SessionTTL = 30 * 24 * time.Hour

type Session struct {
	Username string
	Name     string
	Role     string
}

// AuthService 负责密码校验与登录会话。
// 会话持久化在 sessions 表，因此服务重启、多副本部署下登录态都保持一致。
type AuthService struct {
	permRepo    *repository.PermissionRepository
	sessionRepo *repository.SessionRepository
}

func NewAuthService(permRepo *repository.PermissionRepository, sessionRepo *repository.SessionRepository) *AuthService {
	return &AuthService{permRepo: permRepo, sessionRepo: sessionRepo}
}

func (a *AuthService) Login(username, password string) (string, string, bool) {
	perm, err := a.permRepo.GetByUsername(username)
	if err != nil {
		return "", "", false
	}

	ok, legacy := VerifyPassword(perm.Password, password)
	if !ok {
		return "", "", false
	}
	// 历史明文密码：登录成功后顺手升级为 bcrypt，无需用户改密。
	if legacy {
		if hashed, herr := HashPassword(password); herr == nil {
			if uerr := a.permRepo.UpdatePassword(perm.UserID, hashed); uerr != nil {
				log.Printf("升级用户 %s 的密码摘要失败: %v", perm.UserID, uerr)
			}
		}
	}

	role := perm.WebRole
	if role == "" {
		role = "user"
	}

	token := generateToken()
	if err := a.sessionRepo.Create(token, perm.Username, perm.UserName, role, time.Now().Add(SessionTTL)); err != nil {
		log.Printf("保存会话失败: %v", err)
		return "", "", false
	}
	return token, role, true
}

func (a *AuthService) Validate(token string) (Session, bool) {
	if token == "" {
		return Session{}, false
	}
	s, err := a.sessionRepo.Get(token)
	if err != nil {
		return Session{}, false
	}
	if time.Now().After(s.ExpiresAt) {
		_ = a.sessionRepo.Delete(token)
		return Session{}, false
	}
	return Session{Username: s.Username, Name: s.Name, Role: s.Role}, true
}

func (a *AuthService) Logout(token string) {
	if token == "" {
		return
	}
	_ = a.sessionRepo.Delete(token)
}

// LogoutUser 使某用户的全部会话失效（改密码 / 删除账号后调用）。
func (a *AuthService) LogoutUser(username string) {
	_ = a.sessionRepo.DeleteByUsername(username)
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败极其罕见，此处退化为时间戳仍不致命
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b)
}
