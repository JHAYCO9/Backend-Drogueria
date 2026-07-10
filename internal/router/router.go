package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/config"
	"github.com/drogueria/drogueria-micky-backend/internal/handler"
	"github.com/drogueria/drogueria-micky-backend/internal/middleware"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

func Setup(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	authService := service.NewAuthService(userRepo, roleRepo, auditRepo, cfg.JWTSecret)
	userService := service.NewUserService(userRepo, roleRepo, auditRepo)
	auditService := service.NewAuditService(auditRepo, userRepo)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleRepo)
	auditHandler := handler.NewAuditHandler(auditService)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)

		protected := api.Group("/")
		protected.Use(middleware.RequireAuth(cfg.JWTSecret, db))
		{
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/roles", roleHandler.ListRoles) // cualquier usuario autenticado puede ver los roles (para el combo del form)

			// RF-006: cualquier usuario autenticado puede cambiar su propia contraseña
			protected.PUT("/users/change-password", userHandler.ChangePassword)

			admin := protected.Group("/")
			admin.Use(middleware.RequireRole("Administrador"))
			{
				admin.POST("/users", userHandler.RegisterUser)
				admin.PUT("/users/:id", userHandler.UpdateUser)
				admin.GET("/users", userHandler.ListUsers)
				admin.PATCH("/users/:id/deactivate", userHandler.DeactivateUser) // RF-005
				admin.PATCH("/users/:id/activate", userHandler.ActivateUser)     // Activar usuario
				admin.GET("/audit-logs", auditHandler.GetAuditLogs)             // RF-010
			}
		}
	}
}

