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
	reporteZService := service.NewReporteZService(cashRepo, saleRepo, medicineRepo)

	// Sprint 4: servicio de caja con DB e Inyección de auditRepo para auditoría y transacciones atómicas (RNF-012).
	cashService := service.NewCashRegisterService(
		db, cashRepo, cashMovementRepo, saleRepo, userRepo, auditRepo, reporteZService,
	)
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

	reporteZHandler := handler.NewReporteZHandler(reporteZService)

	reportService := service.NewReportService(saleRepo)
	reportHandler := handler.NewReportHandler(reportService)

	botRepo := repository.NewBotRepository(db)
	botService := service.NewBotService(botRepo, medicineRepo, batchRepo, saleRepo, userRepo, cashRepo, reporteZService)
	botHandler := handler.NewBotHandler(botService, userRepo)

	systemSettingsRepo := repository.NewSystemSettingsRepository(db)
	systemSettingsService := service.NewSystemSettingsService(systemSettingsRepo)
	backupService := service.NewBackupService(db)
	systemSettingsHandler := handler.NewSystemSettingsHandler(systemSettingsService, backupService)

	saleExportService := service.NewSaleExportService(saleRepo, medicineRepo, userRepo)
	saleExportHandler := handler.NewSaleExportHandler(saleExportService)

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

			// RF-056: Historial de cierres de caja (filtros: ?from=YYYY-MM-DD&to=YYYY-MM-DD&user_id=UUID)
			protected.GET("/cash-register/history", cashHandler.ListClosedRegisters)

			protected.GET("/cash-register/:id/reporte-z", reporteZHandler.GetReporteZ)

			// ────────────────────────────────────────────────────────────────────
			// Módulo de Ventas e Inventario
			// RNF-010: el sistema no permite realizar ventas cuando la caja está cerrada.
			// Se aplica middleware RequireCashOpen a todos los endpoints de creación/finalización.
			// ────────────────────────────────────────────────────────────────────
			protected.POST("/sales", middleware.RequireCashOpen(cashRepo), saleHandler.StartSale)                        // RF-031 + RNF-010
			protected.GET("/sales/:id", saleHandler.GetSale)                                                             // RF-045
			protected.POST("/sales/:id/items", middleware.RequireCashOpen(cashRepo), saleHandler.AddItem)                // RF-033 + RNF-010
			protected.PUT("/sales/:id/items/:itemId", middleware.RequireCashOpen(cashRepo), saleHandler.UpdateItem)      // RF-034 + RNF-010
			protected.DELETE("/sales/:id/items/:itemId", saleHandler.RemoveItem)                                         // RF-035
			protected.POST("/sales/:id/cancel", saleHandler.CancelSale)                                                  // RF-036
			protected.POST("/sales/:id/finalize/cash", middleware.RequireCashOpen(cashRepo), saleHandler.FinalizeCash)   // RF-039 + RNF-010
			protected.POST("/sales/:id/finalize/nequi", middleware.RequireCashOpen(cashRepo), saleHandler.FinalizeNequi) // RF-040 + RNF-010
			protected.GET("/sales/:id/receipt", saleHandler.GetReceipt)

			// Venta atómica (RNF-010: también requiere caja abierta)
			protected.POST("/sales/atomic", middleware.RequireCashOpen(cashRepo), saleHandler.CreateAtomicSale)

			// RF-093: Consultar historial de devoluciones (RNF-027 permanencia)
			protected.GET("/returns", returnHandler.ListReturns)

			protected.POST("/bot/query", botHandler.ExecuteCommand)   // RF-074/075/076/077
			protected.GET("/bot/alerts", botHandler.GetPendingAlerts) // RF-071/072/073

			admin := protected.Group("/")
			admin.Use(middleware.RequireRole("Administrador"))
			{
				// RNF-029: Solo Administrador puede registrar, modificar o anular devoluciones
				// RF-089/091/095 + RNF-009 + RNF-029: registrar devolución
				admin.POST("/sales/:id/returns", middleware.RequireCashOpen(cashRepo), returnHandler.RegisterReturn)
				// RF-094 + RNF-027 + RNF-029: anular devolución y revertir inventario
				admin.POST("/returns/:id/void", returnHandler.VoidReturn)
				admin.POST("/returns/:id/anular", returnHandler.VoidReturn)

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

				admin.GET("/reports/daily-sales", reportHandler.GetDailySalesReport)     // RF-061
				admin.GET("/reports/monthly-sales", reportHandler.GetMonthlySalesReport) // RF-062

				admin.POST("/bot/access", botHandler.GrantAccess)        // RF-070
				admin.DELETE("/bot/access/:id", botHandler.RevokeAccess) // RF-070
				admin.GET("/bot/access", botHandler.ListAuthorized)      // RF-070
				admin.GET("/bot/logs", botHandler.ListQueryLogs)         // RF-078
				admin.GET("/bot/config", botHandler.GetConfig)           // RF-082
				admin.PUT("/bot/config", botHandler.UpdateConfig)        // RF-082

				admin.GET("/settings", systemSettingsHandler.Get)                                       // RF-079/080/081/083
				admin.PUT("/settings/business", systemSettingsHandler.UpdateBusinessInfo)               // RF-079
				admin.PUT("/settings/expiration", systemSettingsHandler.UpdateExpirationSettings)       // RF-080
				admin.PUT("/settings/stock", systemSettingsHandler.UpdateStockSettings)                 // RF-081
				admin.PUT("/settings/reporte-z-schedule", systemSettingsHandler.UpdateReporteZSchedule) // RF-083
				admin.POST("/settings/backup", systemSettingsHandler.CreateBackup)                      // RF-084

				admin.PUT("/settings/tax", systemSettingsHandler.UpdateTaxSettings) // RF-086
				admin.GET("/sales/export", saleExportHandler.ExportSales)           // RF-087 + RF-088

			}
		}
	}

	return alertService
}
