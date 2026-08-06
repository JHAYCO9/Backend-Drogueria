package service

import (
	"time"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

type ReportService interface {
	GetDailySalesReport(date time.Time) (*dto.DailySalesReportResponse, error)          // RF-061
	GetMonthlySalesReport(year int, month int) (*dto.MonthlySalesReportResponse, error) // RF-062
}

type reportService struct {
	saleRepo repository.SaleRepository
}

func NewReportService(saleRepo repository.SaleRepository) ReportService {
	return &reportService{saleRepo: saleRepo}
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
