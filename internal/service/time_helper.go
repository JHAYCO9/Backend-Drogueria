package service

import "time"

var bogotaLocation *time.Location

func init() {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		loc = time.FixedZone("UTC-5", -5*60*60) // respaldo si el contenedor no tiene la base de datos de zonas horarias
	}
	bogotaLocation = loc
}

// NowBogota se usa al CREAR cualquier registro con timestamp visible
// al usuario (ventas, caja, movimientos, etc.), en vez de time.Now().
func NowBogota() time.Time {
	return time.Now().In(bogotaLocation)
}

// FormatBogota se usa al FORMATEAR cualquier timestamp para el JSON de
// respuesta. Re-proyecta el instante real a hora de Colombia sin
// importar en qué zona horaria haya quedado etiquetado el time.Time
// después de un viaje de ida y vuelta por la base de datos — esto es
// justo lo que corrige la inconsistencia (algunas ventas se formateaban
// justo al crearse, otras después de leerse de Postgres con otra zona).
func FormatBogota(t time.Time) string {
	return t.In(bogotaLocation).Format(time.RFC3339)
}