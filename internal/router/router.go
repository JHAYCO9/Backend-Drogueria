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

func Setup(r *gin.Engine, db *gorm.DB, cfg *config.Config) *service.AlertService {
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	medicineRepo := repository.NewMedicineRepository(db)
	batchRepo := repository.NewBatchRepository(db)
	movementRepo := repository.NewInventoryMovementRepository(db)
	adjustmentRepo := repository.NewInventoryAdjustmentRepository(db)

	authService := service.NewAuthService(userRepo, roleRepo, auditRepo, cfg.JWTSecret)
	userService := service.NewUserService(userRepo, roleRepo, auditRepo)
	auditService := service.NewAuditService(auditRepo, userRepo)

	medicineService := service.NewMedicineService(medicineRepo, batchRepo, auditRepo)
	batchService := service.NewBatchService(batchRepo, medicineRepo)
	inventoryService := service.NewInventoryService(batchRepo, movementRepo, adjustmentRepo, medicineRepo, userRepo)
	alertService := service.NewAlertService(medicineRepo, batchRepo)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleRepo)
	auditHandler := handler.NewAuditHandler(auditService)

	medicineHandler := handler.NewMedicineHandler(medicineService, batchRepo, alertService)
	batchHandler := handler.NewBatchHandler(batchService, medicineService, medicineRepo)
	inventoryHandler := handler.NewInventoryHandler(inventoryService, alertService)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)

		protected := api.Group("/")
		protected.Use(middleware.RequireAuth(cfg.JWTSecret, db))
		{

			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/roles", roleHandler.ListRoles)

			// (cualquier usuario autenticado puede consultar/buscar):
			protected.GET("/medicines/barcode/:barcode", medicineHandler.FindByBarcode) // RF-027
			protected.GET("/medicines", medicineHandler.ListMedicines)
			protected.GET("/medicines/:medicineId/batches", batchHandler.ListByMedicine) // RF-018
			protected.GET("/batches/expiring", batchHandler.ListExpiringSoon)            // RF-023
			protected.GET("/batches/expired", batchHandler.ListExpired)                  // RF-024
			protected.GET("/medicines/critical", medicineHandler.ListCritical)           // RF-021

			// RF-014: busqueda avanzada de medicamentos (cajeros y administradores)
			protected.GET("/medicines/search", medicineHandler.SearchMedicines)

			// RF-006: cualquier usuario autenticado puede cambiar su propia contrasena
			protected.PUT("/users/change-password", userHandler.ChangePassword)

			admin := protected.Group("/")
			admin.Use(middleware.RequireRole("Administrador"))
			{
				// Sprint 1 - endpoints existentes
				admin.POST("/medicines", medicineHandler.RegisterMedicine)              // RF-011
				admin.PATCH("/medicines/:id/min-stock", medicineHandler.UpdateMinStock) // RF-020
				admin.POST("/users", userHandler.RegisterUser)
				admin.PUT("/users/:id", userHandler.UpdateUser)
				admin.GET("/users", userHandler.ListUsers)
				admin.PATCH("/users/:id/deactivate", userHandler.DeactivateUser)          // RF-005
				admin.PATCH("/users/:id/activate", userHandler.ActivateUser)              // Activar usuario
				admin.GET("/audit-logs", auditHandler.GetAuditLogs)                       // RF-010
				admin.POST("/batches", batchHandler.RegisterBatch)                        // RF-015
				admin.POST("/inventory/receipt", inventoryHandler.RegisterIngreso)        // RF-016
				admin.PATCH("/medicines/:id/alert-days", medicineHandler.UpdateAlertDays) // RF-022

				// Sprint 2 - Bloque 1: Mantenimiento de Medicamentos
				admin.PUT("/medicines/:id", medicineHandler.UpdateMedicine)              // RF-012
				admin.PATCH("/medicines/:id/deactivate", medicineHandler.DeactivateMedicine) // RF-013
				admin.PATCH("/medicines/:id/activate", medicineHandler.ActivateMedicine)     // Activar medicamento

				// Sprint 2 - Bloque 2: Movimientos y Ajustes de Inventario
				admin.POST("/inventory/adjustments", inventoryHandler.RegisterAdjustment)          // RF-017
				admin.GET("/inventory/movements", inventoryHandler.ListMovements)             // RF-019
				admin.GET("/inventory/stock-history", inventoryHandler.GetStockHistory) // RF-030

				// Sprint 2 - Bloque 3: Consultas Avanzadas
				admin.GET("/inventory/general", inventoryHandler.GetGeneralInventory)   // RF-028
				admin.GET("/inventory/critical", inventoryHandler.GetCriticalInventory) // RF-029
			}
		}
	}

	return alertService
}
