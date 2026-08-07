package service

import (
	"sort"
	"time"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
	"github.com/google/uuid"
)

type ReportService interface {
	GetDailySalesReport(date time.Time) (*dto.DailySalesReportResponse, error)                          // RF-061
	GetMonthlySalesReport(year int, month int) (*dto.MonthlySalesReportResponse, error)                // RF-062
	GetTopSellingProducts(from, to time.Time, limit int) (*dto.TopSellingProductsResponse, error)       // RF-063
	GetLowRotationProducts(from, to time.Time, limit int) (*dto.LowRotationProductsResponse, error)    // RF-064
}

type reportService struct {
	saleRepo     repository.SaleRepository
	medicineRepo repository.MedicineRepository
}

func NewReportService(saleRepo repository.SaleRepository, medicineRepo repository.MedicineRepository) ReportService {
	return &reportService{saleRepo: saleRepo, medicineRepo: medicineRepo}
}

// GetDailySalesReport cubre RF-061: fecha consultada, número de ventas,
// total vendido, total en efectivo, total mediante Nequi, ticket promedio.
func (s *reportService) GetDailySalesReport(date time.Time) (*dto.DailySalesReportResponse, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)

	sales, err := s.saleRepo.FindByDateRange(start, end)
	if err != nil {
		return nil, err
	}

	count := 0
	total := 0.0
	efectivo := 0.0
	nequi := 0.0

	for _, sale := range sales {
		if sale.Status != "finalizada" {
			continue
		}
		count++
		total += sale.TotalAmount
		if sale.PaymentMethod != nil {
			if *sale.PaymentMethod == "nequi" {
				nequi += sale.TotalAmount
			} else {
				efectivo += sale.TotalAmount
			}
		}
	}

	average := 0.0
	if count > 0 {
		average = total / float64(count)
	}

	return &dto.DailySalesReportResponse{
		Date: date.Format("2006-01-02"), SalesCount: count, TotalSold: total,
		TotalEfectivo: efectivo, TotalNequi: nequi, AverageTicket: average,
	}, nil
}

// GetMonthlySalesReport cubre RF-062: mes consultado, número de ventas,
// total vendido, promedio diario, día con mayores ventas.
func (s *reportService) GetMonthlySalesReport(year int, month int) (*dto.MonthlySalesReportResponse, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)

	sales, err := s.saleRepo.FindByDateRange(start, end)
	if err != nil {
		return nil, err
	}

	count := 0
	total := 0.0
	byDay := map[string]float64{}

	for _, sale := range sales {
		if sale.Status != "finalizada" {
			continue
		}
		count++
		total += sale.TotalAmount
		dayKey := sale.CreatedAt.Format("2006-01-02")
		byDay[dayKey] += sale.TotalAmount
	}

	daysInMonth := end.Sub(start).Hours() / 24
	averageDaily := 0.0
	if daysInMonth > 0 {
		averageDaily = total / daysInMonth
	}

	bestDay := ""
	bestDayTotal := 0.0
	for day, amount := range byDay {
		if amount > bestDayTotal {
			bestDayTotal = amount
			bestDay = day
		}
	}

	return &dto.MonthlySalesReportResponse{
		Month: start.Format("2006-01"), SalesCount: count, TotalSold: total,
		AverageDaily: averageDaily, BestDay: bestDay, BestDayTotal: bestDayTotal,
	}, nil
}

// productAgg es una estructura auxiliar para agrupar ventas por medicamento.
type productAgg struct {
	medicineID uuid.UUID
	qtySold    int
	totalGen   float64
}

// aggregateByProduct recorre todas las ventas finalizadas en el rango dado,
// consulta sus ítems y agrupa por medicamento. Es la base común de RF-063 y RF-064.
// RNF-017: lee directamente de la BD sin caché.
func (s *reportService) aggregateByProduct(from, to time.Time) (map[uuid.UUID]*productAgg, error) {
	sales, err := s.saleRepo.FindByDateRange(from, to)
	if err != nil {
		return nil, err
	}

	agg := make(map[uuid.UUID]*productAgg)

	for _, sale := range sales {
		if sale.Status != "finalizada" {
			continue
		}
		items, err := s.saleRepo.FindItems(sale.ID)
		if err != nil {
			continue
		}
		for _, item := range items {
			entry, ok := agg[item.MedicineID]
			if !ok {
				entry = &productAgg{medicineID: item.MedicineID}
				agg[item.MedicineID] = entry
			}
			entry.qtySold += item.Quantity
			entry.totalGen += item.Subtotal
		}
	}

	return agg, nil
}

// GetTopSellingProducts cubre RF-063: productos más vendidos ordenados
// de mayor a menor. Muestra Nombre, Cantidad vendida, Total generado.
func (s *reportService) GetTopSellingProducts(from, to time.Time, limit int) (*dto.TopSellingProductsResponse, error) {
	agg, err := s.aggregateByProduct(from, to)
	if err != nil {
		return nil, err
	}

	// Convertir a slice para ordenar
	list := make([]productAgg, 0, len(agg))
	for _, v := range agg {
		list = append(list, *v)
	}

	// RF-063: ordenar de mayor a menor por cantidad vendida
	sort.Slice(list, func(i, j int) bool {
		if list[i].qtySold == list[j].qtySold {
			return list[i].totalGen > list[j].totalGen // desempate por total generado
		}
		return list[i].qtySold > list[j].qtySold
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}

	products := make([]dto.TopSellingProductItem, 0, len(list))
	for _, p := range list {
		name := "Medicamento eliminado"
		if m, err := s.medicineRepo.FindByID(p.medicineID); err == nil {
			name = m.Name
		}
		products = append(products, dto.TopSellingProductItem{
			MedicineID:     p.medicineID.String(),
			Name:           name,
			QuantitySold:   p.qtySold,
			TotalGenerated: p.totalGen,
		})
	}

	return &dto.TopSellingProductsResponse{
		GeneratedAt: time.Now().Format(time.RFC3339),
		From:        from.Format("2006-01-02"),
		To:          to.Format("2006-01-02"),
		Products:    products,
	}, nil
}

// GetLowRotationProducts cubre RF-064: productos con menor rotación
// ordenados de menor a mayor. Muestra Nombre, Cantidad vendida.
func (s *reportService) GetLowRotationProducts(from, to time.Time, limit int) (*dto.LowRotationProductsResponse, error) {
	agg, err := s.aggregateByProduct(from, to)
	if err != nil {
		return nil, err
	}

	// Convertir a slice para ordenar
	list := make([]productAgg, 0, len(agg))
	for _, v := range agg {
		list = append(list, *v)
	}

	// RF-064: ordenar de menor a mayor por cantidad vendida
	sort.Slice(list, func(i, j int) bool {
		return list[i].qtySold < list[j].qtySold
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}

	products := make([]dto.LowRotationProductItem, 0, len(list))
	for _, p := range list {
		name := "Medicamento eliminado"
		if m, err := s.medicineRepo.FindByID(p.medicineID); err == nil {
			name = m.Name
		}
		products = append(products, dto.LowRotationProductItem{
			MedicineID:   p.medicineID.String(),
			Name:         name,
			QuantitySold: p.qtySold,
		})
	}

	return &dto.LowRotationProductsResponse{
		GeneratedAt: time.Now().Format(time.RFC3339),
		From:        from.Format("2006-01-02"),
		To:          to.Format("2006-01-02"),
		Products:    products,
	}, nil
}
