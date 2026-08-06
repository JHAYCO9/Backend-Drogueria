package service

import (
	"fmt"

	"github.com/jung-kurt/gofpdf"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
)

// GenerateReporteZPDF cubre RF-073: genera el PDF y lo guarda en disco
// para servirlo públicamente vía HTTP (Twilio necesita una URL pública
// para descargar el archivo adjunto).
func GenerateReporteZPDF(report *dto.ReporteZResponse, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Courier", "B", 14)
	pdf.CellFormat(190, 10, "DROGUERIA MICKEY - REPORTE Z", "0", 1, "C", false, 0, "")

	pdf.SetFont("Courier", "", 11)
	pdf.Ln(6)
	pdf.CellFormat(190, 8, fmt.Sprintf("Fecha: %s", report.DateTime), "0", 1, "L", false, 0, "")
	pdf.CellFormat(190, 8, fmt.Sprintf("Z Numero: %d", report.ZNumero), "0", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Courier", "B", 12)
	pdf.CellFormat(190, 8, "RESUMEN DE LA JORNADA", "0", 1, "L", false, 0, "")
	pdf.SetFont("Courier", "", 11)
	pdf.CellFormat(190, 8, fmt.Sprintf("Numero de ventas: %d", report.NumeroVentas), "0", 1, "L", false, 0, "")
	pdf.CellFormat(190, 8, fmt.Sprintf("Total vendido: $%.0f", report.TotalVendido), "0", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Courier", "B", 12)
	pdf.CellFormat(190, 8, "DESGLOSE FISCAL", "0", 1, "L", false, 0, "")
	pdf.SetFont("Courier", "", 11)
	pdf.CellFormat(190, 8, fmt.Sprintf("Exento: $%.0f", report.TotalExentos), "0", 1, "L", false, 0, "")
	pdf.CellFormat(190, 8, fmt.Sprintf("Base gravada 19%%: $%.0f", report.TotalGravados19), "0", 1, "L", false, 0, "")
	pdf.CellFormat(190, 8, fmt.Sprintf("IVA 19%%: $%.0f", report.TotalIva19), "0", 1, "L", false, 0, "")

	return pdf.OutputFileAndClose(outputPath)
}
