package service

import (
	"log"
	"time"

	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

// AlertService centraliza la verificacion de alertas de inventario
// (RNF-006). Se ejecuta al inicio del sistema como goroutine y se
// invoca despues de cada modificacion de stock.
type AlertService struct {
	medicineRepo repository.MedicineRepository
	batchRepo    repository.BatchRepository
}

func NewAlertService(medicineRepo repository.MedicineRepository, batchRepo repository.BatchRepository) *AlertService {
	return &AlertService{medicineRepo: medicineRepo, batchRepo: batchRepo}
}

// CheckInventoryAlerts evalua todos los medicamentos activos y reporta:
//   - Medicamentos con stock bajo (stock actual <= stock minimo)
//   - Lotes proximos a vencer (dentro del rango de dias configurado)
//   - Lotes completamente vencidos
//
// Las alertas se imprimen en el log del servidor. En el futuro, este
// mismo punto se puede conectar al envio de WhatsApp (Epica 7).
func (s *AlertService) CheckInventoryAlerts() {
	log.Println("[ALERTAS] Iniciando verificacion de alertas de inventario...")

	medicines, err := s.medicineRepo.FindAll()
	if err != nil {
		log.Printf("[ALERTAS] Error obteniendo medicamentos: %v", err)
		return
	}

	now := time.Now()
	stockBajoCount := 0
	proximoVencerCount := 0
	vencidoCount := 0

	for _, m := range medicines {
		// Verificar stock bajo
		stock, err := s.batchRepo.SumStockByMedicine(m.ID)
		if err != nil {
			continue
		}
		if stock <= m.MinStock {
			log.Printf("[ALERTA - STOCK BAJO] %s (barcode: %s) - Stock actual: %d, Stock minimo: %d",
				m.Name, m.Barcode, stock, m.MinStock)
			stockBajoCount++
		}

		// Verificar lotes proximos a vencer y vencidos
		batches, err := s.batchRepo.FindByMedicine(m.ID)
		if err != nil {
			continue
		}

		alertLimit := now.AddDate(0, 0, m.ExpirationAlertDays)

		for _, b := range batches {
			if b.Quantity <= 0 {
				continue
			}

			if b.ExpirationDate.Before(now) || b.ExpirationDate.Equal(now) {
				log.Printf("[ALERTA - VENCIDO] %s - Lote: %s - Vencio: %s - Cantidad: %d",
					m.Name, b.BatchCode, b.ExpirationDate.Format("2006-01-02"), b.Quantity)
				vencidoCount++
			} else if b.ExpirationDate.Before(alertLimit) || b.ExpirationDate.Equal(alertLimit) {
				daysLeft := int(time.Until(b.ExpirationDate).Hours() / 24)
				log.Printf("[ALERTA - PROXIMO A VENCER] %s - Lote: %s - Vence: %s - Dias restantes: %d - Cantidad: %d",
					m.Name, b.BatchCode, b.ExpirationDate.Format("2006-01-02"), daysLeft, b.Quantity)
				proximoVencerCount++
			}
		}
	}

	log.Printf("[ALERTAS] Verificacion completada - Stock bajo: %d, Proximos a vencer: %d, Vencidos: %d",
		stockBajoCount, proximoVencerCount, vencidoCount)
}
