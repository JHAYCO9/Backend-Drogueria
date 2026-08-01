package dto

// ReporteZResponse cubre RF-055: resumen consolidado de la jornada,
// calculado a partir de la caja cerrada indicada.
type ReporteZResponse struct {
	BusinessName    string  `json:"business_name"`
	OwnerName       string  `json:"owner_name"`
	Nit             string  `json:"nit"`
	Address         string  `json:"address"`
	DateTime        string  `json:"date_time"`
	RegistradoraNo  string  `json:"registradora_no"`
	RegistroInicial int     `json:"registro_inicial"`
	RegistroFinal   int     `json:"registro_final"`
	ZNumero         int     `json:"z_numero"`
	TotalExentos    float64 `json:"total_exentos"`
	ExemptCount     int     `json:"exempt_count"` // nuevo
	TotalGravados19 float64 `json:"total_gravados_19"`
	TotalIva19      float64 `json:"total_iva_19"`
	TaxedCount19    int     `json:"taxed_count_19"` // nuevo
	NumeroVentas    int     `json:"numero_ventas"`
	TotalVendido    float64 `json:"total_vendido"`
}
