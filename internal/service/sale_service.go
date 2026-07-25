package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrNoCashOpen = errors.New("no hay una caja abierta, debe abrirse antes de iniciar una venta")
var ErrSaleNotFound = errors.New("venta no encontrada")
var ErrSaleNotInProgress = errors.New("la venta ya fue finalizada o cancelada")
var ErrNoStockAvailable = errors.New("no hay existencias disponibles de este medicamento")
var ErrInsufficientPayment = errors.New("el valor recibido es insuficiente para cubrir el total de la venta")
var ErrEmptySale = errors.New("la venta no tiene productos agregados")
var ErrItemNotFound = errors.New("el producto no pertenece a esta venta")

type SaleService interface {
	StartSale(userID uuid.UUID) (*domain.Sale, error)                                     // RF-031 + RF-032
	AddItem(saleID uuid.UUID, input AddSaleItemInput) (*domain.SaleItem, error)           // RF-033
	RemoveItem(saleID, itemID uuid.UUID) error                                            // RF-035
	CancelSale(saleID uuid.UUID) error                                                    // RF-036
	GetSaleDetail(saleID uuid.UUID) (*domain.Sale, []domain.SaleItem, error)              // usado por varios RF
	FinalizeCash(saleID uuid.UUID, amountReceived float64) (*domain.Sale, float64, error) // RF-039 + RF-041
	FinalizeNequi(saleID uuid.UUID, amountPaid float64) (*domain.Sale, error)             // RF-040 + RF-041
	GetReceipt(saleID uuid.UUID) (*ReceiptData, error)                                    // RF-043
	ListSales() ([]domain.Sale, error)                                                    // RF-044
	GetSalesByDateRange(from, to time.Time) (*SalesRangeResult, error)                    // RF-047

}

type AddSaleItemInput struct {
	MedicineID uuid.UUID
	Barcode    string
	Quantity   int
	SaleUnit   string
}

type saleService struct {
	saleRepo     repository.SaleRepository
	cashRepo     repository.CashRegisterRepository
	medicineRepo repository.MedicineRepository
	batchRepo    repository.BatchRepository
	movementRepo repository.InventoryMovementRepository
	userRepo     repository.UserRepository
}

func NewSaleService(
	saleRepo repository.SaleRepository,
	cashRepo repository.CashRegisterRepository,
	medicineRepo repository.MedicineRepository,
	batchRepo repository.BatchRepository,
	movementRepo repository.InventoryMovementRepository,
	userRepo repository.UserRepository,
) SaleService {
	return &saleService{
		saleRepo: saleRepo, cashRepo: cashRepo, medicineRepo: medicineRepo,
		batchRepo: batchRepo, movementRepo: movementRepo, userRepo: userRepo,
	}
}

// StartSale cubre RF-031 (crear venta vacía, consecutivo automático via
// autoIncrement de la BD, fecha/hora automática, usuario asociado) y
// RF-032 (valida que exista una caja abierta antes de permitir iniciar).
func (s *saleService) StartSale(userID uuid.UUID) (*domain.Sale, error) {
	openCash, err := s.cashRepo.FindOpen()
	if err != nil {
		return nil, ErrNoCashOpen
	}

	// Si ya hay una venta sin finalizar en esta caja, se retoma en vez
	// de crear una nueva — evita dejar ventas huérfanas "en_progreso"
	// cada vez que el cajero entra al módulo de Ventas.
	if existing, err := s.saleRepo.FindInProgress(openCash.ID); err == nil {
		return existing, nil
	}

	sale := &domain.Sale{
		ID:             uuid.New(),
		UserID:         userID,
		CashRegisterID: openCash.ID,
		Status:         "en_progreso",
		TotalAmount:    0,
		CreatedAt:      time.Now(),
	}

	if err := s.saleRepo.Create(sale); err != nil {
		return nil, err
	}
	return sale, nil
}

// AddItem cubre RF-033: busca el medicamento por ID o por código de barras
// (RF-027), asigna automáticamente el lote más próximo a vencer disponible
// (FEFO, ya resuelto por BatchRepository al reutilizar la misma consulta
// que usa el escaneo), calcula el subtotal, y NO toca el inventario todavía.
var ErrInvalidSaleUnit = errors.New("el medicamento no se vende en la unidad seleccionada")

func (s *saleService) AddItem(saleID uuid.UUID, input AddSaleItemInput) (*domain.SaleItem, error) {
	sale, err := s.saleRepo.FindByID(saleID)
	if err != nil {
		return nil, ErrSaleNotFound
	}
	if sale.Status != "en_progreso" {
		return nil, ErrSaleNotInProgress
	}

	var medicine *domain.Medicine
	if input.MedicineID != uuid.Nil {
		medicine, err = s.medicineRepo.FindByID(input.MedicineID)
	} else if input.Barcode != "" {
		medicine, err = s.medicineRepo.FindByBarcode(input.Barcode)
	} else {
		return nil, errors.New("debe indicar medicine_id o barcode")
	}
	if err != nil {
		return nil, ErrMedicineNotFound
	}

	// Valida que el medicamento efectivamente se venda en la unidad
	// elegida por el cajero, y calcula cuántas pastillas reales
	// representa esa cantidad (unitsConsumed) — es lo que se descuenta
	// del inventario, que siempre se cuenta en unidades sueltas.
	var unitPrice float64
	unitsPerItem := 1
	switch input.SaleUnit {
	case "pastilla":
		if !medicine.SellsByPill {
			return nil, ErrInvalidSaleUnit
		}
		unitPrice = medicine.PricePerPill
		unitsPerItem = 1
	case "sello":
		if !medicine.SellsBySeal {
			return nil, ErrInvalidSaleUnit
		}
		unitPrice = medicine.PricePerSeal
		unitsPerItem = medicine.UnitsPerSeal
	case "caja":
		if !medicine.SellsByBox {
			return nil, ErrInvalidSaleUnit
		}
		unitPrice = medicine.PricePerBox
		unitsPerItem = medicine.UnitsPerBox
	default:
		return nil, ErrInvalidSaleUnit
	}

	unitsConsumed := input.Quantity * unitsPerItem

	// FEFO, pero ahora buscando un lote con suficientes UNIDADES SUELTAS
	// disponibles (no "cantidad de cajas"), ya que el inventario siempre
	// se mide en pastillas.
	batches, err := s.batchRepo.FindByMedicine(medicine.ID)
	if err != nil {
		return nil, err
	}
	var selectedBatch *domain.Batch
	for i := range batches {
		if batches[i].Quantity >= unitsConsumed {
			selectedBatch = &batches[i]
			break
		}
	}
	if selectedBatch == nil {
		return nil, ErrNoStockAvailable
	}

	subtotal := unitPrice * float64(input.Quantity)

	var taxAmount float64
	if medicine.HasIva {
		taxAmount = subtotal - (subtotal / 1.19)
	}

	item := &domain.SaleItem{
		ID: uuid.New(), SaleID: saleID, MedicineID: medicine.ID, BatchID: selectedBatch.ID,
		Quantity: input.Quantity, SaleUnit: input.SaleUnit, UnitsConsumed: unitsConsumed,
		UnitPrice: unitPrice, Subtotal: subtotal, TaxAmount: taxAmount, CreatedAt: time.Now(),
	}

	if err := s.saleRepo.AddItem(item); err != nil {
		return nil, err
	}

	if err := s.recalculateTotal(sale); err != nil {
		return nil, err
	}

	return item, nil
}

type SalesRangeResult struct {
	Sales       []domain.Sale
	SalesCount  int
	TotalSold   float64
	AverageSale float64
}

// ListSales cubre RF-044: historial completo de ventas.
func (s *saleService) ListSales() ([]domain.Sale, error) {
	return s.saleRepo.FindAll()
}

// GetSalesByDateRange cubre RF-047: número de ventas, total vendido y
// promedio por venta entre dos fechas.
func (s *saleService) GetSalesByDateRange(from, to time.Time) (*SalesRangeResult, error) {
	sales, err := s.saleRepo.FindByDateRange(from, to)
	if err != nil {
		return nil, err
	}

	total := 0.0
	for _, sale := range sales {
		total += sale.TotalAmount
	}

	average := 0.0
	if len(sales) > 0 {
		average = total / float64(len(sales))
	}

	return &SalesRangeResult{
		Sales: sales, SalesCount: len(sales), TotalSold: total, AverageSale: average,
	}, nil
}

// RemoveItem cubre RF-035: quita la línea y recalcula el total; el
// inventario nunca se tocó al agregar, así que tampoco hay nada que
// revertir aquí.
func (s *saleService) RemoveItem(saleID, itemID uuid.UUID) error {
	sale, err := s.saleRepo.FindByID(saleID)
	if err != nil {
		return ErrSaleNotFound
	}
	if sale.Status != "en_progreso" {
		return ErrSaleNotInProgress
	}

	item, err := s.saleRepo.FindItemByID(itemID)
	if err != nil || item.SaleID != saleID {
		return ErrItemNotFound
	}

	if err := s.saleRepo.RemoveItem(itemID, saleID); err != nil {
		return err
	}

	return s.recalculateTotal(sale)
}

// CancelSale cubre RF-036: marca la venta como cancelada. El detalle
// temporal (SaleItems) se conserva en base de datos para trazabilidad,
// pero el status "cancelada" hace que nunca se cuente como venta real ni
// afecte el inventario (que de todas formas nunca se tocó).
func (s *saleService) CancelSale(saleID uuid.UUID) error {
	sale, err := s.saleRepo.FindByID(saleID)
	if err != nil {
		return ErrSaleNotFound
	}
	if sale.Status != "en_progreso" {
		return ErrSaleNotInProgress
	}

	sale.Status = "cancelada"
	return s.saleRepo.Update(sale)
}

func (s *saleService) GetSaleDetail(saleID uuid.UUID) (*domain.Sale, []domain.SaleItem, error) {
	sale, err := s.saleRepo.FindByID(saleID)
	if err != nil {
		return nil, nil, ErrSaleNotFound
	}
	items, err := s.saleRepo.FindItems(saleID)
	if err != nil {
		return nil, nil, err
	}
	return sale, items, nil
}

// FinalizeCash cubre RF-039 + RF-041: valida que el valor recibido alcance,
// calcula el cambio, y delega el cierre común (descuento real de
// inventario, timestamps) a finalizeCommon.
func (s *saleService) FinalizeCash(saleID uuid.UUID, amountReceived float64) (*domain.Sale, float64, error) {
	sale, items, err := s.prepareFinalize(saleID)
	if err != nil {
		return nil, 0, err
	}

	if amountReceived < sale.TotalAmount {
		return nil, 0, ErrInsufficientPayment
	}
	change := amountReceived - sale.TotalAmount

	method := "efectivo"
	sale.PaymentMethod = &method
	sale.AmountReceived = &amountReceived
	sale.ChangeGiven = &change

	if err := s.finalizeCommon(sale, items); err != nil {
		return nil, 0, err
	}
	return sale, change, nil
}

// FinalizeNequi cubre RF-040 + RF-041: solo registra el método y el valor
// pagado, sin integración real con la plataforma (tal como pide el RF).
func (s *saleService) FinalizeNequi(saleID uuid.UUID, amountPaid float64) (*domain.Sale, error) {
	sale, items, err := s.prepareFinalize(saleID)
	if err != nil {
		return nil, err
	}

	method := "nequi"
	sale.PaymentMethod = &method
	sale.AmountReceived = &amountPaid

	if err := s.finalizeCommon(sale, items); err != nil {
		return nil, err
	}
	return sale, nil
}

func (s *saleService) prepareFinalize(saleID uuid.UUID) (*domain.Sale, []domain.SaleItem, error) {
	sale, err := s.saleRepo.FindByID(saleID)
	if err != nil {
		return nil, nil, ErrSaleNotFound
	}
	if sale.Status != "en_progreso" {
		return nil, nil, ErrSaleNotInProgress
	}

	items, err := s.saleRepo.FindItems(saleID)
	if err != nil {
		return nil, nil, err
	}
	if len(items) == 0 {
		return nil, nil, ErrEmptySale
	}

	return sale, items, nil
}

// finalizeCommon cubre el resto de RF-041: descuenta inventario real
// (recién aquí, no antes), registra el movimiento por cada ítem, y marca
// fecha/hora de finalización.
func (s *saleService) finalizeCommon(sale *domain.Sale, items []domain.SaleItem) error {
	for _, item := range items {
		if err := s.batchRepo.IncreaseQuantity(item.BatchID, -item.UnitsConsumed); err != nil {
			return err
		}

		movement := &domain.InventoryMovement{
			ID: uuid.New(), MedicineID: item.MedicineID, BatchID: &item.BatchID,
			MovementType: "venta", Quantity: -item.UnitsConsumed,
			UserID: sale.UserID, ReferenceID: &sale.ID,
		}
		if err := s.movementRepo.Create(movement); err != nil {
			return err
		}
	}

	now := time.Now()
	sale.Status = "finalizada"
	sale.FinalizedAt = &now
	return s.saleRepo.Update(sale)
}

func (s *saleService) recalculateTotal(sale *domain.Sale) error {
	items, err := s.saleRepo.FindItems(sale.ID)
	if err != nil {
		return err
	}
	total := 0.0
	for _, it := range items {
		total += it.Subtotal
	}
	sale.TotalAmount = total
	return s.saleRepo.Update(sale)
}

type ReceiptItemData struct {
	MedicineName string
	Quantity     int
	SaleUnit     string
	UnitPrice    float64
	BasePrice    float64
	TaxAmount    float64
	HasIva       bool
	Subtotal     float64
}

type ReceiptData struct {
	SaleNumber     int
	CreatedAt      time.Time
	CashierName    string
	Items          []ReceiptItemData
	TotalAmount    float64
	TotalTax       float64
	PaymentMethod  string
	AmountReceived *float64
	ChangeGiven    *float64
}

// GetReceipt cubre RF-043: arma el comprobante completo de una venta ya
// finalizada — número, fecha, hora, productos con cantidades y precios
// (discriminando base + IVA cuando aplica), total y método de pago.
func (s *saleService) GetReceipt(saleID uuid.UUID) (*ReceiptData, error) {
	saleRecord, err := s.saleRepo.FindByID(saleID)
	if err != nil {
		return nil, ErrSaleNotFound
	}
	if saleRecord.Status != "finalizada" {
		return nil, errors.New("solo se puede generar comprobante de una venta finalizada")
	}

	saleItems, err := s.saleRepo.FindItems(saleID)
	if err != nil {
		return nil, err
	}

	cashierName := "Cajero"
	if u, err := s.userRepo.FindByID(saleRecord.UserID); err == nil {
		cashierName = u.FullName
	}

	receiptItems := make([]ReceiptItemData, 0, len(saleItems))
	totalTax := 0.0
	for _, it := range saleItems {
		medicine, err := s.medicineRepo.FindByID(it.MedicineID)
		name := ""
		hasIva := false
		if err == nil {
			name = medicine.Name
			hasIva = medicine.HasIva
		}
		base := it.Subtotal - it.TaxAmount
		totalTax += it.TaxAmount

		receiptItems = append(receiptItems, ReceiptItemData{
			MedicineName: name, Quantity: it.Quantity, SaleUnit: it.SaleUnit,
			BasePrice: base, TaxAmount: it.TaxAmount, HasIva: hasIva, Subtotal: it.Subtotal,
		})
	}

	method := ""
	if saleRecord.PaymentMethod != nil {
		method = *saleRecord.PaymentMethod
	}

	return &ReceiptData{
		SaleNumber: saleRecord.SaleNumber, CreatedAt: saleRecord.CreatedAt, CashierName: cashierName,
		Items: receiptItems, TotalAmount: saleRecord.TotalAmount, TotalTax: totalTax,
		PaymentMethod: method, AmountReceived: saleRecord.AmountReceived, ChangeGiven: saleRecord.ChangeGiven,
	}, nil
}
