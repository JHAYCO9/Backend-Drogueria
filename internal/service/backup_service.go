package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

type BackupService interface {
	CreateBackup() (fileName string, sizeBytes int64, err error)
	GetLastBackupTime() (*time.Time, error)
}

// Simplificamos: backupService solo necesita *gorm.DB para exportar
// tablas, y actualiza system_settings.last_backup_at directamente.
type backupServiceImpl struct {
	db *gorm.DB
}

func NewBackupService(db *gorm.DB) BackupService {
	return &backupServiceImpl{db: db}
}

// CreateBackup cubre RF-084: exporta las tablas críticas a un archivo
// JSON con marca de tiempo. RNF-023: se verifica que el archivo se
// escribió correctamente (tamaño > 0) antes de confirmar éxito, y se
// registra la fecha del respaldo solo si la escritura fue exitosa.
func (s *backupServiceImpl) CreateBackup() (string, int64, error) {
	tables := []string{
		"users", "roles", "medicines", "batches", "sales", "sale_items",
		"inventory_movements", "cash_registers", "cash_movements", "returns",
		"system_settings",
	}

	backup := map[string]interface{}{}
	for _, table := range tables {
		var rows []map[string]interface{}
		if err := s.db.Table(table).Find(&rows).Error; err != nil {
			return "", 0, fmt.Errorf("error exportando tabla %s: %w", table, err)
		}
		backup[table] = rows
	}

	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", 0, err
	}

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("backup_%s.json", timestamp)
	fullPath := filepath.Join(backupDir, fileName)

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", 0, err
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", 0, err
	}

	// RNF-023: verificación de integridad — confirma que el archivo
	// realmente se escribió con contenido antes de dar por exitoso el respaldo.
	info, err := os.Stat(fullPath)
	if err != nil || info.Size() == 0 {
		return "", 0, fmt.Errorf("el archivo de respaldo no se generó correctamente")
	}

	now := time.Now()
	_ = s.db.Table("system_settings").Where("1 = 1").Update("last_backup_at", now)

	return fileName, info.Size(), nil
}

func (s *backupServiceImpl) GetLastBackupTime() (*time.Time, error) {
	var result struct {
		LastBackupAt *time.Time
	}
	err := s.db.Table("system_settings").Select("last_backup_at").Scan(&result).Error
	return result.LastBackupAt, err
}
