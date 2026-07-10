package main

import (
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/drogueria/drogueria-micky-backend/internal/config"
	"github.com/drogueria/drogueria-micky-backend/internal/database"
	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

// Este script se corre UNA sola vez por base de datos (local o Aiven)
// para crear el primer usuario Administrador. No pasa por HTTP ni por
// el middleware de auth, porque en este punto no existe nadie con quien
// loguearse todavía.
func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("error conectando a la base de datos: %v", err)
	}

	// evita crear el admin dos veces si ya corriste el seed antes
	var count int64
	db.Model(&domain.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		log.Println("el usuario 'admin' ya existe, no se hizo nada")
		return
	}

	var adminRole domain.Role
	if err := db.Where("name = ?", "Administrador").First(&adminRole).Error; err != nil {
		log.Fatalf("no se encontró el rol 'Administrador', ¿corriste el seed de roles del schema.sql?: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("Admin123!"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("error generando hash: %v", err)
	}

	admin := domain.User{
		ID:           uuid.New(),
		FullName:     "Administrador Prueba",
		DocumentID:   "0000000000",
		Username:     "admin",
		PasswordHash: string(hash),
		RoleID:       adminRole.ID,
		IsActive:     true,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("error creando el usuario admin: %v", err)
	}

	log.Println("usuario 'admin' creado correctamente. Usuario: admin / Contraseña: Admin123!")
}