package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gin-examples/project/utils"
)

func Auth(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 获取 Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Error(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		// 提取 Token（Bearer <token>）
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Error(c, http.StatusUnauthorized, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证 Token
		claims, err := utils.ParseToken(tokenString, jwtSecret)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		// 将用户信息存储到 Context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("userRole", claims.Role)

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("userRole")
		if !exists || userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
