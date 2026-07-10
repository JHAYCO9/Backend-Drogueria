package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

func RequireAuth(jwtSecret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token no proporcionado"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido o expirado"})
			return
		}

		roleID, _ := claims["role_id"].(string)
		var role domain.Role
		if err := db.First(&role, "id = ?", roleID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "rol no identificado"})
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("role_id", roleID)
		c.Set("role_name", role.Name)
		c.Next()
	}
}

// RequireRole bloquea la ruta si el rol del usuario autenticado no está
// en la lista de roles permitidos (RF-008: "impedir el acceso directo
// mediante rutas internas" — esto es lo que lo garantiza de verdad,
// no solo ocultar el botón en la UI).
func RequireRole(allowedRoleNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleName, _ := c.Get("role_name")
		for _, allowed := range allowedRoleNames {
			if roleName == allowed {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no tienes permisos para esta acción"})
	}
}
