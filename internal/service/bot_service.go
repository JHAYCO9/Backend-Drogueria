package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrBotDisabled = errors.New("el bot se encuentra deshabilitado actualmente")
var ErrBotNotAuthorized = errors.New("no tienes acceso al bot de consultas rápidas")
var ErrUnknownCommand = errors.New("comando no reconocido, escribe 'ayuda' para ver los comandos disponibles")

type BotService interface {
	ExecuteCommand(userID uuid.UUID, command string) (string, error) // RF-074/075/076/077 + RF-078

	// RF-070
	GrantAccess(userID uuid.UUID) error
	RevokeAccess(id uuid.UUID) error
	ListAuthorized() ([]domain.BotAuthorizedUser, error)

	// RF-078
	ListQueryLogs() ([]domain.BotQueryLog, error)

	// RF-082
	GetConfig() (*domain.BotConfig, error)
	UpdateConfig(isEnabled bool, welcomeMessage string) error

	// RF-071/072/073: alertas mostradas al abrir el bot
	GetPendingAlerts() ([]string, error)
}

type botService struct {
	botRepo      repository.BotRepository
	medicineRepo repository.MedicineRepository
	batchRepo    repository.BatchRepository
	saleRepo     repository.SaleRepository
	userRepo     repository.UserRepository
	cashRepo     repository.CashRegisterRepository
	reporteZ     ReporteZService
}

func NewBotService(
	botRepo repository.BotRepository, medicineRepo repository.MedicineRepository,
	batchRepo repository.BatchRepository, saleRepo repository.SaleRepository,
	userRepo repository.UserRepository, cashRepo repository.CashRegisterRepository,
	reporteZ ReporteZService,
) BotService {
	return &botService{
		botRepo: botRepo, medicineRepo: medicineRepo, batchRepo: batchRepo,
		saleRepo: saleRepo, userRepo: userRepo, cashRepo: cashRepo, reporteZ: reporteZ,
	}
}

// ExecuteCommand cubre RF-074/075/076/077 (interpretación de comandos)
// + RNF-019 (control de acceso) + RF-078 (registro del historial).
func (s *botService) ExecuteCommand(userID uuid.UUID, command string) (string, error) {
	cfg, err := s.botRepo.GetConfig()
	if err != nil || !cfg.IsEnabled {
		return "", ErrBotDisabled
	}

	authorized, err := s.botRepo.IsAuthorized(userID)
	if err != nil || !authorized {
		return "", ErrBotNotAuthorized
	}

	result, err := s.interpret(strings.TrimSpace(command))
	if err != nil {
		result = err.Error()
	}

	// RF-078: se registra siempre, incluso si el comando no fue reconocido.
	_ = s.botRepo.LogQuery(&domain.BotQueryLog{
		ID: uuid.New(), UserID: userID, Command: command, Result: result, CreatedAt: time.Now(),
	})

	return result, nil
}

func (s *botService) interpret(command string) (string, error) {
	lower := strings.ToLower(command)

	switch {
	case lower == "ayuda": // RF-077
		return s.helpText(), nil

	case strings.HasPrefix(lower, "stock "): // RF-074
		query := strings.TrimSpace(command[6:])
		return s.queryStock(query)

	case strings.HasPrefix(lower, "ventas "): // RF-075
		dateStr := strings.TrimSpace(command[7:])
		return s.queryVentas(dateStr)

	case lower == "productos bajos": // RF-076
		return s.queryProductosBajos()

	default:
		return "", ErrUnknownCommand
	}
}

// RF-077: lista de comandos disponibles.
func (s *botService) helpText() string {
	return "Comandos disponibles:\n\n" +
		"• stock [medicamento] — consulta el stock de un producto\n" +
		"• ventas [dd/mm/aaaa] — consulta las ventas de una fecha\n" +
		"• productos bajos — lista productos con stock bajo\n" +
		"• ayuda — muestra este mensaje"
}

// RF-074: consulta de stock por nombre.
func (s *botService) queryStock(name string) (string, error) {
	if name == "" {
		return "", errors.New("debes indicar el nombre del medicamento. Ej: stock acetaminofén")
	}

	medicines, err := s.medicineRepo.Search(name, "", "", nil, nil, nil)
	if err != nil || len(medicines) == 0 {
		return fmt.Sprintf("No se encontró ningún medicamento que coincida con \"%s\"", name), nil
	}

	m := medicines[0]
	stock, _ := s.batchRepo.SumStockByMedicine(m.ID)
	nextExpiration := "N/A"
	if batches, err := s.batchRepo.FindByMedicine(m.ID); err == nil && len(batches) > 0 {
		nextExpiration = batches[0].ExpirationDate.Format("2006-01-02")
	}

	return fmt.Sprintf(
		"📦 *%s*\n\nStock disponible: %d\nStock mínimo: %d\nPróximo vencimiento: %s",
		m.Name, stock, m.MinStock, nextExpiration,
	), nil
}

// RF-075: consulta de ventas por fecha (formato dd/mm/aaaa).
func (s *botService) queryVentas(dateStr string) (string, error) {
	date, err := time.Parse("02/01/2006", dateStr)
	if err != nil {
		return "", errors.New("formato de fecha inválido, usa dd/mm/aaaa. Ej: ventas 15/08/2026")
	}

	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)

	sales, err := s.saleRepo.FindByDateRange(start, end)
	if err != nil {
		return "", errors.New("no se pudo consultar las ventas")
	}

	count := 0
	total := 0.0
	for _, sale := range sales {
		if sale.Status != "finalizada" {
			continue
		}
		count++
		total += sale.TotalAmount
	}

	return fmt.Sprintf(
		"📊 *Ventas del %s*\n\nTotal vendido: $%.0f\nNúmero de ventas: %d",
		dateStr, total, count,
	), nil
}

// RF-076: productos con stock bajo.
func (s *botService) queryProductosBajos() (string, error) {
	medicines, err := s.medicineRepo.FindAll()
	if err != nil {
		return "", errors.New("no se pudo consultar el inventario")
	}

	var lines []string
	for _, m := range medicines {
		stock, err := s.batchRepo.SumStockByMedicine(m.ID)
		if err != nil || stock > m.MinStock {
			continue
		}
		lines = append(lines, fmt.Sprintf("• %s — Stock: %d (mín. %d)", m.Name, stock, m.MinStock))
	}

	if len(lines) == 0 {
		return "✅ No hay productos con stock bajo actualmente.", nil
	}

	return "⚠️ *Productos con stock bajo:*\n\n" + strings.Join(lines, "\n"), nil
}

// GetPendingAlerts cubre RF-071/072/073: se ejecuta al abrir el bot,
// mostrando alertas relevantes como si el bot las "empujara" al usuario.
func (s *botService) GetPendingAlerts() ([]string, error) {
	var alerts []string

	medicines, err := s.medicineRepo.FindAll()
	if err == nil {
		for _, m := range medicines {
			stock, err := s.batchRepo.SumStockByMedicine(m.ID)
			if err != nil || stock > m.MinStock {
				continue
			}
			alerts = append(alerts, fmt.Sprintf(
				"⚠️ *Stock bajo*: %s — %d unidades (mínimo %d)", m.Name, stock, m.MinStock,
			))
		}
	}

	if batches, err := s.batchRepo.FindExpiringSoon(30); err == nil {
		for _, b := range batches {
			medicineName := ""
			if m, err := s.medicineRepo.FindByID(b.MedicineID); err == nil {
				medicineName = m.Name
			}
			days := int(time.Until(b.ExpirationDate).Hours() / 24)
			alerts = append(alerts, fmt.Sprintf(
				"⏳ *Próximo a vencer*: %s (Lote %s) — %d días restantes", medicineName, b.BatchCode, days,
			))
		}
	}

	if cr, err := s.cashRepo.FindOpen(); err == nil && cr.ClosedAt != nil {
		if report, err := s.reporteZ.Generate(cr.ID); err == nil {
			alerts = append(alerts, fmt.Sprintf(
				"📊 *Reporte Z del último cierre*: $%.0f vendidos en %d ventas, cerrado a las %s",
				report.TotalVendido, report.NumeroVentas, cr.ClosedAt.Format("15:04"),
			))
		}
	}

	return alerts, nil
}

func (s *botService) GrantAccess(userID uuid.UUID) error { return s.botRepo.GrantAccess(userID) }
func (s *botService) RevokeAccess(id uuid.UUID) error    { return s.botRepo.RevokeAccess(id) }
func (s *botService) ListAuthorized() ([]domain.BotAuthorizedUser, error) {
	return s.botRepo.ListAuthorized()
}
func (s *botService) ListQueryLogs() ([]domain.BotQueryLog, error) {
	return s.botRepo.ListQueryLogs(50)
}
func (s *botService) GetConfig() (*domain.BotConfig, error) { return s.botRepo.GetConfig() }
func (s *botService) UpdateConfig(isEnabled bool, welcomeMessage string) error {
	cfg, err := s.botRepo.GetConfig()
	if err != nil {
		return err
	}
	cfg.IsEnabled = isEnabled
	cfg.WelcomeMessage = welcomeMessage
	return s.botRepo.UpdateConfig(cfg)
}
