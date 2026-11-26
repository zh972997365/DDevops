package middleware

import (
	"ddevops/internal/auth"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 校验 Authorization header 中的登录 token，
// 校验通过后将 username / name / role 写入上下文供后续 handler 使用。
func AuthMiddleware(authSvc *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "未登录"})
			return
		}
		session, ok := authSvc.Validate(token)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "登录已过期"})
			return
		}
		c.Set("username", session.Username)
		c.Set("name", session.Name)
		c.Set("role", session.Role)
		c.Next()
	}
}

// AdminMiddleware 校验当前会话是否为管理员。
// 必须挂在 AuthMiddleware 之后，直接复用其写入上下文的角色，避免每个请求重复查会话。
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if role, _ := c.Get("role"); role != "admin" {
			c.AbortWithStatusJSON(403, gin.H{"error": "权限不足"})
			return
		}
		c.Next()
	}
}
