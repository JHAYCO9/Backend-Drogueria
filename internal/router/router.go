package router

import (
	"net/http"

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

	syncRepo := repository.NewSyncRepository(db)
	syncHandler := handler.NewSyncHandler(syncRepo)

	cashRepo := repository.NewCashRegisterRepository(db)
	saleRepo := repository.NewSaleRepository(db)
	saleService := service.NewSaleService(saleRepo, cashRepo, medicineRepo, batchRepo, movementRepo, userRepo)
	saleHandler := handler.NewSaleHandler(saleService, medicineRepo, userRepo)

	api := r.Group("/api/v1")
	{
		// Sin autenticación: el SyncService lo usa solo para saber si hay
		// conexión al servidor, antes de intentar autenticar nada.
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

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
			protected.GET("/inventory/general", inventoryHandler.GetGeneralInventory)    // RF-028 — consulta, no modifica

			// RF-014: busqueda avanzada de medicamentos (cajeros y administradores)
			protected.GET("/medicines/search", medicineHandler.SearchMedicines)

			// RF-006: cualquier usuario autenticado puede cambiar su propia contrasena
			protected.PUT("/users/change-password", userHandler.ChangePassword)

			protected.POST("/sync/events", syncHandler.PushEvent)

			protected.POST("/sales", saleHandler.StartSale) // RF-031 + RF-032
			protected.GET("/sales/:id", saleHandler.GetSale)
			protected.POST("/sales/:id/items", saleHandler.AddItem)                // RF-033
			protected.DELETE("/sales/:id/items/:itemId", saleHandler.RemoveItem)   // RF-035
			protected.POST("/sales/:id/cancel", saleHandler.CancelSale)            // RF-036
			protected.POST("/sales/:id/finalize/cash", saleHandler.FinalizeCash)   // RF-039 + RF-041
			protected.POST("/sales/:id/finalize/nequi", saleHandler.FinalizeNequi) // RF-040 + RF-041

			protected.GET("/sales/:id/receipt", saleHandler.GetReceipt)

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
				admin.PUT("/medicines/:id", medicineHandler.UpdateMedicine)                  // RF-012
				admin.PATCH("/medicines/:id/deactivate", medicineHandler.DeactivateMedicine) // RF-013
				admin.PATCH("/medicines/:id/activate", medicineHandler.ActivateMedicine)     // Activar medicamento

				// Sprint 2 - Bloque 2: Movimientos y Ajustes de Inventario
				admin.POST("/inventory/adjustments", inventoryHandler.RegisterAdjustment) // RF-017
				admin.GET("/inventory/movements", inventoryHandler.ListMovements)         // RF-019
				admin.GET("/inventory/stock-history", inventoryHandler.GetStockHistory)   // RF-030

				// Sprint 2 - Bloque 3: Consultas Avanzadas
				admin.GET("/inventory/critical", inventoryHandler.GetCriticalInventory) // RF-029

				admin.GET("/sales", saleHandler.ListSales)                 // RF-044
				admin.GET("/sales/range", saleHandler.GetSalesByDateRange) // RF-047
			}
		}
	}

	return alertService
}
