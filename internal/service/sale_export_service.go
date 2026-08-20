package service

import (
	"time"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

type SaleExportService interface {
	ExportSales(from, to time.Time) (*dto.SalesExportResponse, error) // RF-087 + RF-088
}

type saleExportService struct {
	saleRepo     repository.SaleRepository
	medicineRepo repository.MedicineRepository
	userRepo     repository.UserRepository
}

func NewSaleExportService(saleRepo repository.SaleRepository, medicineRepo repository.MedicineRepository, userRepo repository.UserRepository) SaleExportService {
	return &saleExportService{saleRepo: saleRepo, medicineRepo: medicineRepo, userRepo: userRepo}
}

// ExportSales cubre RF-087 (estructura compatible con facturación
// electrónica) + RF-088 (exportación en formato JSON estructurado).
func (s *saleExportService) ExportSales(from, to time.Time) (*dto.SalesExportResponse, error) {
	sales, err := s.saleRepo.FindByDateRange(from, to)
	if err != nil {
		return nil, err
	}

	exports := make([]dto.SaleExport, 0, len(sales))
	for _, sale := range sales {
		if sale.Status != "finalizada" {
			continue
		}

		items, err := s.saleRepo.FindItems(sale.ID)
		if err != nil {
			continue
		}

		exportItems := make([]dto.SaleExportItem, 0, len(items))
		totalTax := 0.0
		for _, item := range items {
			medicineName := ""
			if m, err := s.medicineRepo.FindByID(item.MedicineID); err == nil {
				medicineName = m.Name
			}
			base := item.Subtotal - item.TaxAmount
			totalTax += item.TaxAmount

			exportItems = append(exportItems, dto.SaleExportItem{
				MedicineName: medicineName, Quantity: item.Quantity, SaleUnit: item.SaleUnit,
				UnitPrice: item.UnitPrice, BaseAmount: base, TaxAmount: item.TaxAmount, Subtotal: item.Subtotal,
			})
		}

		cashierName := ""
		if u, err := s.userRepo.FindByID(sale.UserID); err == nil {
			cashierName = u.FullName
		}

		method := ""
		if sale.PaymentMethod != nil {
			method = *sale.PaymentMethod
		}

		exports = append(exports, dto.SaleExport{
			Consecutive: sale.SaleNumber, Date: sale.CreatedAt.Format(time.RFC3339),
			Items: exportItems, TotalTax: totalTax, PaymentMethod: method,
			TotalAmount: sale.TotalAmount, CashierName: cashierName,
		})
	}

	return &dto.SalesExportResponse{
		ExportedAt: FormatBogota(time.Now()),
		FromDate:   from.Format("2006-01-02"), ToDate: to.Format("2006-01-02"),
		Count: len(exports), Sales: exports,
	}, nil
}
