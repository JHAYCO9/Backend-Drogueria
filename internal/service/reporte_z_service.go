package service

import (
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

type ReporteZService interface {
	Generate(cashRegisterID uuid.UUID) (*dto.ReporteZResponse, error)
}

type reporteZService struct {
	cashRepo     repository.CashRegisterRepository
	saleRepo     repository.SaleRepository
	medicineRepo repository.MedicineRepository
}

func NewReporteZService(cashRepo repository.CashRegisterRepository, saleRepo repository.SaleRepository, medicineRepo repository.MedicineRepository) ReporteZService {
	return &reporteZService{cashRepo: cashRepo, saleRepo: saleRepo, medicineRepo: medicineRepo}
}

// Generate cubre RF-055: reporte Z consolidado de una jornada de caja
// ya cerrada. El total vendido (y por lo tanto CAJA No) suma TODAS las
// ventas finalizadas sin importar el método de pago — efectivo y Nequi
// se tratan como un único total, tal como lo haría la registradora física.
func (s *reporteZService) Generate(cashRegisterID uuid.UUID) (*dto.ReporteZResponse, error) {
	cr, err := s.cashRepo.FindByID(cashRegisterID)
	if err != nil {
		return nil, err
	}
	if cr.Status != "cerrada" {
		return nil, ErrCashNotOpen
	}

	sales, err := s.saleRepo.FindByCashRegister(cashRegisterID)
	if err != nil {
		return nil, err
	}

	registroInicial, registroFinal := 0, 0
	numeroVentas := 0
	totalVendido := 0.0
	totalExentos := 0.0
	totalGravados := 0.0
	totalIva := 0.0
	exemptCount := 0
	taxedCount := 0

	first := true
	for _, sale := range sales {
		if sale.Status != "finalizada" {
			continue
		}
		numeroVentas++
		totalVendido += sale.TotalAmount // suma efectivo Y nequi por igual

		if first || sale.SaleNumber < registroInicial {
			registroInicial = sale.SaleNumber
		}
		if sale.SaleNumber > registroFinal {
			registroFinal = sale.SaleNumber
		}
		first = false

		items, err := s.saleRepo.FindItems(sale.ID)
		if err != nil {
			continue
		}
		for _, item := range items {
			if item.TaxAmount > 0 {
				base := item.Subtotal - item.TaxAmount
				totalGravados += base
				totalIva += item.TaxAmount
				taxedCount++
			} else {
				totalExentos += item.Subtotal
				exemptCount++
			}
		}
	}

	zNumero := 1
	if closedAt := cr.ClosedAt; closedAt != nil {
		if count, err := s.cashRepo.CountClosedUpTo(*closedAt); err == nil {
			zNumero = int(count)
		}
	}

	dateTime := NowBogota()
	if cr.ClosedAt != nil {
		dateTime = *cr.ClosedAt
	}
	return &dto.ReporteZResponse{
		BusinessName: "DROGUERÍA MICKEY", OwnerName: "GERARDO ANTONIO MUÑOZ",
		Nit: "16704858-2 REG/COMUN", Address: "CRA 4#3-18 ESQUINA P.PRI SOLITA CAQUETÁ",
		DateTime: dateTime.Format("2006-01-02 15:04"), RegistradoraNo: "9940",
		RegistroInicial: registroInicial, RegistroFinal: registroFinal, ZNumero: zNumero,
		TotalExentos: totalExentos, ExemptCount: exemptCount,
		TotalGravados19: totalGravados, TotalIva19: totalIva, TaxedCount19: taxedCount,
		NumeroVentas: numeroVentas, TotalVendido: totalVendido,
	}, nil
}
