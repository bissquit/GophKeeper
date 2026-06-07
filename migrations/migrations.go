// Package migrations embeds SQL migration files
package migrations

import (
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

//go:embed *.sql
var embeddedMigrations embed.FS

// InitializeDB applies every pending up-migration against databaseURL
func InitializeDB(databaseURL string) error {
	d, err := iofs.New(embeddedMigrations, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}
