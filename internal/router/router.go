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
	cashMovementRepo := repository.NewCashMovementRepository(db)

	// Sprint 4: servicio de caja con DB inyectada para transacciones atómicas (RNF-012).
	cashService := service.NewCashRegisterServiceWithDB(db, cashRepo, cashMovementRepo, saleRepo, userRepo)
	cashHandler := handler.NewCashRegisterHandler(cashService, userRepo)

	saleService := service.NewSaleService(db, saleRepo, cashRepo, medicineRepo, batchRepo, movementRepo, userRepo, auditRepo, cashMovementRepo)

	returnRepo := repository.NewReturnRepository(db)
	// Sprint 4: se agrega cashRepo al returnService para RNF-009.
	returnService := service.NewReturnService(db, returnRepo, saleRepo, batchRepo, movementRepo, medicineRepo, userRepo, auditRepo, cashRepo)
	returnHandler := handler.NewReturnHandler(returnService)

	// SaleService actualizado con db y auditRepo (RF-034, RF-037, RF-038, RF-042, RF-044, RF-045, RF-046)
	saleHandler := handler.NewSaleHandler(saleService, medicineRepo, returnRepo, userRepo)

	dashboardService := service.NewDashboardService(saleRepo, medicineRepo, batchRepo, cashRepo, cashMovementRepo, movementRepo, userRepo)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)

	reporteZService := service.NewReporteZService(cashRepo, saleRepo, medicineRepo)
	reporteZHandler := handler.NewReporteZHandler(reporteZService)

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

			protected.GET("/dashboard/summary", dashboardHandler.GetSummary)

			// ────────────────────────────────────────────────────────────────────
			// Módulo de Caja — Sprint 4 (Integrado sobre las rutas existentes)
			// ────────────────────────────────────────────────────────────────────
			protected.POST("/cash-register/open", cashHandler.OpenCashRegister)   // RF-016 / RNF-008 / RNF-012
			protected.POST("/cash-register/close", cashHandler.CloseCashRegister) // RF-017 / RNF-011 / RNF-012
			
			// RF-018: Registrar movimiento manual (POST)
			protected.POST("/cash-register/movements", cashHandler.RegisterManualMovement)
			// RF-051: Listar movimientos (GET - existente)
			protected.GET("/cash-register/movements", cashHandler.GetMovements)
			
			// RF-019: Estado detallado del arqueo en tiempo real
			protected.GET("/cash-register/status", cashHandler.GetDetailedStatus)
			
			protected.GET("/cash-register/:id/reporte-z", reporteZHandler.GetReporteZ)

			// ────────────────────────────────────────────────────────────────────
			// Módulo de Ventas e Inventario (RF-034, RF-037, RF-038, RF-042, RF-044, RF-045, RF-046)
			// RNF-009: validación de caja abierta aplicada a inicio de venta y devoluciones.
			// Nota: StartSale y CreateAtomicSale ya tienen la validación en el SaleService.
			// ────────────────────────────────────────────────────────────────────
			protected.POST("/sales", saleHandler.StartSale)                        // RF-031
			protected.GET("/sales/:id", saleHandler.GetSale)                       // RF-045: Consultar detalle de una venta
			protected.POST("/sales/:id/items", saleHandler.AddItem)                // RF-033: Agregar ítem
			protected.PUT("/sales/:id/items/:itemId", saleHandler.UpdateItem)      // RF-034: Modificar cantidad de un medicamento
			protected.DELETE("/sales/:id/items/:itemId", saleHandler.RemoveItem)   // RF-035: Eliminar ítem
			protected.POST("/sales/:id/cancel", saleHandler.CancelSale)            // RF-036
			protected.POST("/sales/:id/finalize/cash", saleHandler.FinalizeCash)   // RF-039 + RF-041 + RF-042
			protected.POST("/sales/:id/finalize/nequi", saleHandler.FinalizeNequi) // RF-040 + RF-041 + RF-042
			protected.GET("/sales/:id/receipt", saleHandler.GetReceipt)

			// Venta atómica (una sola llamada HTTP - RF-037, RF-038, RF-042, RNF-008)
			protected.POST("/sales/atomic", saleHandler.CreateAtomicSale)

			// RF-089/091/095 + RNF-009 (devoluciones): middleware RequireCashOpen aplicado aquí
			protected.POST("/sales/:id/returns", middleware.RequireCashOpen(cashRepo), returnHandler.RegisterReturn)
			protected.GET("/returns", returnHandler.ListReturns) // RF-093

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

				// RF-044: Consultar historial de ventas (con filtros y paginación)
				admin.GET("/sales", saleHandler.ListSalesFiltered)
				admin.GET("/sales/range", saleHandler.GetSalesByDateRange)

				// RF-046: Anular una venta (con reintegro atómico de stock)
				admin.POST("/sales/:id/void", saleHandler.VoidCompletedSale)
				admin.POST("/sales/:id/anular", saleHandler.VoidCompletedSale)
				admin.PUT("/sales/:id/anular", saleHandler.VoidCompletedSale)
			}
		}
	}

	return alertService
}
