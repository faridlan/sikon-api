package config // Sesuaikan dengan nama package-mu

import (
	"log"

	"github.com/faridlan/sikon-api/db/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func RunDBMigration(dbURL string) {
	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("Gagal membaca file migrasi embed: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		log.Fatalf("Gagal membuat instance migrasi: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Gagal menjalankan migrasi: %v", err)
	}

	log.Println("✅ Database migrasi berhasil dijalankan!")
}
