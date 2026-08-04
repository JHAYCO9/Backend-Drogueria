package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

// RequireCashOpen implementa RNF-009: bloquea las operaciones de venta y
// devolución si no hay ninguna caja abierta en el sistema.
// Si el usuario intenta registrar una venta o devolución sin caja abierta,
// responde 403 Forbidden con el mensaje estipulado en el requisito.
func RequireCashOpen(cashRepo repository.CashRegisterRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := cashRepo.FindOpen()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "No se pueden realizar operaciones de venta o devolución sin una caja abierta.",
			})
			return
		}
		c.Next()
	}
}
