package service

import (
	"sort"
	"time"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
	"github.com/google/uuid"
)

type DashboardService interface {
	GetSummary() (*dto.DashboardSummaryResponse, error)
}

type dashboardService struct {
	saleRepo         repository.SaleRepository
	medicineRepo     repository.MedicineRepository
	batchRepo        repository.BatchRepository
	cashRepo         repository.CashRegisterRepository
	cashMovementRepo repository.CashMovementRepository
	movementRepo     repository.InventoryMovementRepository
	userRepo         repository.UserRepository
}

func NewDashboardService(
	saleRepo repository.SaleRepository,
	medicineRepo repository.MedicineRepository,
	batchRepo repository.BatchRepository,
	cashRepo repository.CashRegisterRepository,
	cashMovementRepo repository.CashMovementRepository,
	movementRepo repository.InventoryMovementRepository,
	userRepo repository.UserRepository,
) DashboardService {
	return &dashboardService{
		saleRepo: saleRepo, medicineRepo: medicineRepo, batchRepo: batchRepo,
		cashRepo: cashRepo, cashMovementRepo: cashMovementRepo,
		movementRepo: movementRepo, userRepo: userRepo,
	}
}

func (s *dashboardService) GetSummary() (*dto.DashboardSummaryResponse, error) {
	resp := &dto.DashboardSummaryResponse{}

	// RF-058: ventas de hoy
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		loc = time.UTC // fallback, no debería pasar
	}
	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	todaySales, err := s.saleRepo.FindTodayFinalized(startOfDay, endOfDay)
	if err == nil {
		resp.TodaySalesCount = len(todaySales)
		paymentCounts := map[string]int{}
		for _, sale := range todaySales {
			resp.TodaySalesTotal += sale.TotalAmount
			if sale.PaymentMethod != nil {
				paymentCounts[*sale.PaymentMethod]++
			}
		}
		if resp.TodaySalesCount > 0 {
			resp.AverageTicket = resp.TodaySalesTotal / float64(resp.TodaySalesCount)
		}
		resp.MostUsedPaymentMethod = mostFrequent(paymentCounts)
	}

	// RF-059: inventario
	medicines, err := s.medicineRepo.FindAll()
	if err == nil {
		for _, m := range medicines {
			if !m.IsActive {
				continue
			}
			stock, _ := s.batchRepo.SumStockByMedicine(m.ID)
			if stock == 0 {
				resp.OutOfStockCount++
			} else if stock <= m.MinStock {
				resp.LowStockCount++
			}
		}
	}
	if expiring, err := s.batchRepo.FindExpiringSoon(30); err == nil {
		resp.ExpiringSoonCount = len(expiring)
	}
	if expired, err := s.batchRepo.FindExpired(); err == nil {
		resp.ExpiredCount = len(expired)
	}

	// RF-057: estado de la caja
	if cr, err := s.cashRepo.FindOpen(); err == nil {
		resp.CashRegisterOpen = true
		expectedCash := cr.OpeningAmount
		if movements, err := s.cashMovementRepo.FindByCashRegister(cr.ID); err == nil {
			for _, m := range movements {
				if m.MovementType == "venta_efectivo" {
					expectedCash += m.Amount
				}
			}
		}
		resp.CashBalance = expectedCash
	}

	// RF-060: actividad reciente combinada
	resp.RecentActivity = s.buildRecentActivity()

	return resp, nil
}

func (s *dashboardService) buildRecentActivity() []dto.ActivityItem {
	var items []dto.ActivityItem

	if sales, err := s.saleRepo.FindAll(); err == nil {
		limit := 5
		for i, sale := range sales {
			if i >= limit || sale.Status != "finalizada" {
				continue
			}
			username := s.resolveUsername(sale.UserID)
			items = append(items, dto.ActivityItem{
				Type: "venta", Description: "Venta finalizada", Amount: sale.TotalAmount,
				Username: username, CreatedAt: sale.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	if adjustments, err := s.movementRepo.FindRecentByType("ajuste", 5); err == nil {
		for _, adj := range adjustments {
			username := s.resolveUsername(adj.UserID)
			items = append(items, dto.ActivityItem{
				Type: "ajuste", Description: "Ajuste de inventario: " + adj.Reason,
				Username: username, CreatedAt: adj.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	if cashEvents, err := s.cashMovementRepo.FindRecentByTypes([]string{"apertura", "cierre"}, 5); err == nil {
		for _, ce := range cashEvents {
			username := s.resolveUsername(ce.UserID)
			activityType := "apertura_caja"
			desc := "Apertura de caja"
			if ce.MovementType == "cierre" {
				activityType = "cierre_caja"
				desc = "Cierre de caja"
			}
			items = append(items, dto.ActivityItem{
				Type: activityType, Description: desc, Amount: ce.Amount,
				Username: username, CreatedAt: ce.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})

	if len(items) > 10 {
		items = items[:10]
	}
	return items
}

func (s *dashboardService) resolveUsername(userID uuid.UUID) string {
	if u, err := s.userRepo.FindByID(userID); err == nil {
		return u.FullName
	}
	return "Usuario"
}

func mostFrequent(counts map[string]int) string {
	best := ""
	max := 0
	for k, v := range counts {
		if v > max {
			max = v
			best = k
		}
	}
	return best
}
